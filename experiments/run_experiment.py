import csv
import json
import os
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

import psycopg2
import requests


ROOT = Path(__file__).resolve().parent
RESULTS_DIR = ROOT / "results"
RESULTS_DIR.mkdir(parents=True, exist_ok=True)
RESULTS_CSV = RESULTS_DIR / "experiment_results.csv"

BASE_URLS = {
    "device_registry": "http://localhost:8081",
    "config_service": "http://localhost:8082",
    "orchestrator": "http://localhost:8084",
}

PG_CONN = {
    "host": os.getenv("PG_HOST", "localhost"),
    "port": int(os.getenv("PG_PORT", "55432")),
    "user": os.getenv("POSTGRES_USER", "pinchik"),
    "password": os.getenv("POSTGRES_PASSWORD", "pass_iot_core"),
    "dbname": os.getenv("POSTGRES_DB", "iot_core"),
}

MONGO_USER = "pinchik"
MONGO_PASS = "pass_iot_configs"
MONGO_DB = "iot_configs"

COMPOSE_1 = "docker-compose.yml"
COMPOSE_2 = "docker-compose.services.yml"


def now_utc():
    return datetime.now(timezone.utc)


def iso_to_dt(s: str | None):
    if not s:
        return None
    return datetime.fromisoformat(s.replace("Z", "+00:00"))


def sh(cmd: list[str], check: bool = True, capture: bool = True) -> str:
    result = subprocess.run(cmd, check=check, capture_output=capture, text=True)
    return result.stdout.strip() if capture else ""


def http_json(method: str, url: str, payload=None):
    if method == "GET":
        resp = requests.get(url, timeout=20)
    else:
        resp = requests.request(method, url, json=payload, timeout=20)

    if resp.status_code < 200 or resp.status_code >= 300:
        raise RuntimeError(f"{method} {url} -> {resp.status_code}: {resp.text}")

    return resp.json()


def get_network_name():
    container_id = sh([
        "docker", "compose", "-f", COMPOSE_1, "-f", COMPOSE_2,
        "ps", "-q", "postgres"
    ])
    return sh([
        "docker", "inspect", container_id,
        "--format", "{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}"
    ])


def pg_conn():
    return psycopg2.connect(**PG_CONN)


def create_device(protocol: str, run_id: str, n: int):
    payload = {
        "external_id": f"{run_id}-dev-{n}",
        "device_type": "sensor-v1",
        "protocol": protocol,
        "network_profile": "wifi",
        "location_zone": "lab",
        "tags": {"run": run_id, "n": n},
    }
    data = http_json("POST", f"{BASE_URLS['device_registry']}/v1/devices", payload)
    return data["id"]


def create_template(run_id: str):
    payload = {
        "name": f"exp-config-{run_id}",
        "deviceType": "sensor-v1",
        "schema": {
            "type": "object",
            "properties": {"rate": {"type": "integer"}},
            "required": ["rate"],
        },
        "default": {"rate": 10},
    }
    data = http_json("POST", f"{BASE_URLS['config_service']}/v1/templates", payload)
    return data["id"]


def create_version(template_id: str, rate: int):
    payload = {
        "templateId": template_id,
        "payload": {"rate": rate},
    }
    data = http_json("POST", f"{BASE_URLS['config_service']}/v1/versions", payload)
    return data["id"]


def start_simulators(protocol: str, sim_image: str, device_ids: list[str], telemetry_interval: float, run_id: str):
    net = get_network_name()
    names = []

    for i, device_id in enumerate(device_ids, start=1):
        name = f"{protocol}_sim_{run_id}_{i}"
        names.append(name)

        # remove if exists
        subprocess.run(["docker", "rm", "-f", name], capture_output=True, text=True)

        env = [
            "-e", f"DEVICE_ID={device_id}",
            "-e", "MQTT_BROKER_HOST=mosquitto",
            "-e", "MQTT_BROKER_PORT=1883",
        ]

        if protocol == "mqtt":
            env += ["-e", f"TELEMETRY_INTERVAL_SEC={telemetry_interval}"]

        sh([
            "docker", "run", "-d",
            "--name", name,
            "--network", net,
            *env,
            sim_image
        ])
    return names


def stop_simulators(names: list[str]):
    for name in names:
        subprocess.run(["docker", "rm", "-f", name], capture_output=True, text=True)


def create_deployment(config_version_id: str, device_ids: list[str], strategy: dict):
    payload = {
        "configVersionId": config_version_id,
        "deviceIds": device_ids,
        "strategy": strategy,
    }
    data = http_json("POST", f"{BASE_URLS['orchestrator']}/v1/deployments", payload)
    return data["deploymentId"]


def get_deployment(deployment_id: str):
    return http_json("GET", f"{BASE_URLS['orchestrator']}/v1/deployments/{deployment_id}")


def wait_deployment_finished(deployment_id: str, timeout_sec: int = 30):
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        data = get_deployment(deployment_id)
        if data["status"] in {"DONE", "ROLLED_BACK", "FAILED", "FAILED_POLICY"}:
            return data
        time.sleep(1)
    raise TimeoutError(f"deployment {deployment_id} did not finish in time")


def fetch_avg_metrics(device_ids: list[str], ts_from: datetime, ts_to: datetime):
    sql = """
        SELECT
            AVG(latency_ms),
            AVG(loss),
            AVG(jitter_ms)
        FROM telemetry.metrics_raw
        WHERE device_id = ANY(%s::uuid[])
        AND ts >= %s
        AND ts <= %s
    """

    with pg_conn() as conn:
        with conn.cursor() as cur:
            cur.execute(sql, (device_ids, ts_from, ts_to))
            row = cur.fetchone()

    latency = float(row[0]) if row[0] is not None else 0.0
    loss = float(row[1]) if row[1] is not None else 0.0
    jitter = float(row[2]) if row[2] is not None else 0.0
    return latency, loss, jitter


def cleanup_run(state: dict):
    # simulators
    stop_simulators(state.get("sim_names", []))

    deployment_id = state.get("deployment_id")
    if deployment_id:
        with pg_conn() as conn:
            with conn.cursor() as cur:
                cur.execute("DELETE FROM deploy.deployment_targets WHERE deployment_id=%s", (deployment_id,))
                cur.execute("DELETE FROM deploy.deployments WHERE id=%s", (deployment_id,))
            conn.commit()

    config_version_id = state.get("config_version_id")
    if config_version_id:
        with pg_conn() as conn:
            with conn.cursor() as cur:
                cur.execute("DELETE FROM cfg.config_apply_log WHERE config_version_id=%s", (config_version_id,))
            conn.commit()

    template_id = state.get("template_id")
    if template_id:
        with pg_conn() as conn:
            with conn.cursor() as cur:
                cur.execute("""
                    DELETE FROM cfg.config_assignments
                    WHERE config_version_id IN (
                        SELECT id FROM cfg.config_versions WHERE template_id=%s
                    )
                """, (template_id,))
                cur.execute("DELETE FROM cfg.config_versions WHERE template_id=%s", (template_id,))
                cur.execute("DELETE FROM cfg.config_templates WHERE id=%s", (template_id,))
            conn.commit()

        # Mongo cleanup via container
        js = f"""
            db = db.getSiblingDB('{MONGO_DB}');
            db.config_versions.deleteMany({{ templateId: '{template_id}' }});
            db.config_templates.deleteMany({{ name: '{state['template_name']}', deviceType: 'sensor-v1' }});
        """
        subprocess.run([
            "docker", "exec", "-i", "iot_mongo",
            "mongosh",
            "-u", MONGO_USER,
            "-p", MONGO_PASS,
            "--authenticationDatabase", "admin",
            "--quiet",
            "--eval", js
        ], capture_output=True, text=True)

    device_ids = state.get("device_ids", [])
    if device_ids:
        with pg_conn() as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "DELETE FROM telemetry.metrics_raw WHERE device_id = ANY(%s::uuid[])",
                    (device_ids,)
                )
                cur.execute("DELETE FROM registry.device_group_members WHERE device_id = ANY(%s::uuid[])", (device_ids,))
                cur.execute("DELETE FROM registry.devices WHERE id = ANY(%s::uuid[])", (device_ids,))
            conn.commit()


def ensure_results_header():
    if RESULTS_CSV.exists():
        return
    with open(RESULTS_CSV, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            "scenario",
            "repeat",
            "protocol",
            "devices",
            "config_rate",
            "deployment_status",
            "applied",
            "failed",
            "rolled_back",
            "pending",
            "failure_rate",
            "avg_latency_before",
            "avg_latency_after",
            "avg_loss_before",
            "avg_loss_after",
            "avg_jitter_before",
            "avg_jitter_after",
        ])


def append_result(row: dict):
    with open(RESULTS_CSV, "a", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            row["scenario"],
            row["repeat"],
            row["protocol"],
            row["devices"],
            row["config_rate"],
            row["deployment_status"],
            row["applied"],
            row["failed"],
            row["rolled_back"],
            row["pending"],
            row["failure_rate"],
            row["avg_latency_before"],
            row["avg_latency_after"],
            row["avg_loss_before"],
            row["avg_loss_after"],
            row["avg_jitter_before"],
            row["avg_jitter_after"],
        ])


def run_one_experiment(scenario: dict, repeat_index: int):
    run_id = f"{scenario['name']}_{repeat_index}_{uuid.uuid4().hex[:8]}"
    print(f"\n=== RUN {scenario['name']} repeat={repeat_index} run_id={run_id} ===")

    state = {
        "device_ids": [],
        "sim_names": [],
        "template_id": None,
        "template_name": f"exp-config-{run_id}",
        "config_version_id": None,
        "deployment_id": None,
    }

    try:
        # 1. create devices
        for i in range(1, scenario["devices"] + 1):
            did = create_device(scenario["protocol"], run_id, i)
            state["device_ids"].append(did)

        print(f"created devices: {len(state['device_ids'])}")

        # 2. start simulators
        sim_names = start_simulators(
            protocol=scenario["protocol"],
            sim_image=scenario["simImage"],
            device_ids=state["device_ids"],
            telemetry_interval=scenario.get("telemetryIntervalSec", 0.5),
            run_id=run_id,
        )
        state["sim_names"] = sim_names
        print("simulators started")

        # 3. wait for pre-telemetry
        pre_sec = scenario.get("preTelemetrySec", 5)
        t0 = now_utc()
        time.sleep(pre_sec)
        t1 = now_utc()

        avg_latency_before, avg_loss_before, avg_jitter_before = fetch_avg_metrics(
            state["device_ids"], t0, t1
        )

        # 4. create template + version
        template_id = create_template(run_id)
        state["template_id"] = template_id
        state["template_name"] = f"exp-config-{run_id}"

        cfg_id = create_version(template_id, scenario["configRate"])
        state["config_version_id"] = cfg_id

        print(f"template={template_id}")
        print(f"config_version={cfg_id}")

        # 5. deployment
        dep_id = create_deployment(cfg_id, state["device_ids"], scenario["strategy"])
        state["deployment_id"] = dep_id
        print(f"deployment={dep_id}")

        dep = wait_deployment_finished(dep_id, timeout_sec=40)

        # 6. post metrics
        post_sec = scenario.get("postDeploySec", 6)
        started_at = iso_to_dt(dep.get("startedAt"))
        if started_at is None:
            started_at = now_utc()

        time.sleep(post_sec)
        t2 = now_utc()

        avg_latency_after, avg_loss_after, avg_jitter_after = fetch_avg_metrics(
            state["device_ids"], started_at, t2
        )

        counts = dep["counts"]
        total = counts["total"]
        failed = counts["failed"]
        applied = counts["applied"]
        rolled_back = counts["rolledBack"]
        pending = counts["pending"]

        failure_rate = (failed / total) if total else 0.0

        result = {
            "scenario": scenario["name"],
            "repeat": repeat_index,
            "protocol": scenario["protocol"],
            "devices": scenario["devices"],
            "config_rate": scenario["configRate"],
            "deployment_status": dep["status"],
            "applied": applied,
            "failed": failed,
            "rolled_back": rolled_back,
            "pending": pending,
            "failure_rate": round(failure_rate, 4),
            "avg_latency_before": round(avg_latency_before, 4),
            "avg_latency_after": round(avg_latency_after, 4),
            "avg_loss_before": round(avg_loss_before, 6),
            "avg_loss_after": round(avg_loss_after, 6),
            "avg_jitter_before": round(avg_jitter_before, 4),
            "avg_jitter_after": round(avg_jitter_after, 4),
        }

        print("result:", json.dumps(result, indent=2, ensure_ascii=False))
        append_result(result)

    finally:
        cleanup_run(state)


def load_scenario(name: str):
    path = ROOT / "scenarios" / f"{name}.json"
    if not path.exists():
        raise FileNotFoundError(f"scenario not found: {path}")
    return json.loads(path.read_text(encoding="utf-8"))


def main():
    if len(sys.argv) != 2:
        print("Usage: python experiments/run_experiment.py <scenario>")
        print("Scenarios: baseline | canary | smart")
        sys.exit(1)

    scenario_name = sys.argv[1]
    scenario = load_scenario(scenario_name)

    ensure_results_header()

    for i in range(1, scenario.get("repeats", 1) + 1):
        run_one_experiment(scenario, i)

    print("\nAll runs finished.")
    print(f"Results saved to: {RESULTS_CSV}")


if __name__ == "__main__":
    main()