import json
import os
import random
import threading
import time
from datetime import datetime, timezone

import paho.mqtt.client as mqtt

DEVICE_ID = os.getenv("DEVICE_ID", "d-123")
BROKER = os.getenv("MQTT_BROKER_HOST", "mosquitto")
PORT = int(os.getenv("MQTT_BROKER_PORT", "1883"))
TELEMETRY_INTERVAL_SEC = float(os.getenv("TELEMETRY_INTERVAL_SEC", "1.0"))

PROFILE = os.getenv("DEVICE_PROFILE", "normal")
SCENARIOS = [s.strip() for s in os.getenv("SCENARIOS", "").split(",") if s.strip()]

PROFILES = {
    "normal": {
        "base_latency": 20.0,
        "base_loss": 0.01,
        "base_jitter": 3.0,
        "base_rssi": -65.0,
        "base_battery": 95.0,
        "battery_drain": 0.2,
        "disconnect_prob": 0.0,
    },
    "weak_battery": {
        "base_latency": 22.0,
        "base_loss": 0.015,
        "base_jitter": 4.0,
        "base_rssi": -68.0,
        "base_battery": 20.0,
        "battery_drain": 0.8,
        "disconnect_prob": 0.01,
    },
    "bad_network": {
        "base_latency": 45.0,
        "base_loss": 0.05,
        "base_jitter": 7.0,
        "base_rssi": -88.0,
        "base_battery": 80.0,
        "battery_drain": 0.3,
        "disconnect_prob": 0.03,
    },
    "unstable": {
        "base_latency": 30.0,
        "base_loss": 0.03,
        "base_jitter": 6.0,
        "base_rssi": -75.0,
        "base_battery": 70.0,
        "battery_drain": 0.4,
        "disconnect_prob": 0.12,
    },
}

profile = PROFILES.get(PROFILE, PROFILES["normal"])

current_config = {"rate": 10}
current_version = None

device_state = {
    "battery": profile["base_battery"],
    "online": True,
    "last_disconnect_until": 0.0,
}


def now():
    return datetime.now(timezone.utc).isoformat()


def has_scenario(name: str) -> bool:
    return name in SCENARIOS


def build_metrics():
    rate = current_config.get("rate", 10)

    latency = profile["base_latency"] + random.gauss(0, 3.0)
    loss = profile["base_loss"] + random.uniform(0.0, 0.01)
    jitter = profile["base_jitter"] + random.gauss(0, 0.8)
    rssi = profile["base_rssi"] + random.gauss(0, 2.5)

    if rate > 10:
        latency += (rate - 10) * 0.8
        loss += (rate - 10) * 0.001
        jitter += (rate - 10) * 0.15
        device_state["battery"] -= profile["battery_drain"] * 1.5
    else:
        device_state["battery"] -= profile["battery_drain"]

    if has_scenario("latency_spike"):
        latency += random.uniform(30, 90)
        jitter += random.uniform(3, 10)

    if has_scenario("packet_loss_burst"):
        loss += random.uniform(0.08, 0.25)

    if has_scenario("battery_drain"):
        device_state["battery"] -= random.uniform(1.0, 3.0)

    if has_scenario("network_disconnect"):
        if random.random() < 0.15:
            device_state["last_disconnect_until"] = time.time() + random.uniform(5, 15)

    if random.random() < profile["disconnect_prob"]:
        device_state["last_disconnect_until"] = time.time() + random.uniform(3, 10)

    device_state["online"] = time.time() >= device_state["last_disconnect_until"]

    device_state["battery"] = max(5.0, min(100.0, device_state["battery"]))
    loss = max(0.0, min(0.5, loss))
    latency = max(1.0, latency)
    jitter = max(0.1, jitter)
    rssi = max(-100.0, min(-40.0, rssi))

    return {
        "latency_ms": round(latency, 3),
        "loss": round(loss, 5),
        "jitter_ms": round(jitter, 3),
        "rssi": round(rssi, 3),
        "battery": round(device_state["battery"], 3),
    }


def telemetry_loop(client):
    while True:
        metrics = build_metrics()

        if device_state["online"]:
            payload = {
                "deviceId": DEVICE_ID,
                "ts": now(),
                "metrics": metrics,
                "configVersionId": current_version,
                "profile": PROFILE,
                "scenarios": SCENARIOS,
            }
            client.publish(f"telemetry/{DEVICE_ID}/metrics", json.dumps(payload), qos=0, retain=False)
        else:
            print(f"[{DEVICE_ID}] offline, telemetry skipped")

        time.sleep(TELEMETRY_INTERVAL_SEC)


def on_connect(client, userdata, flags, rc, properties=None):
    topic = f"config/desired/{DEVICE_ID}"
    client.subscribe(topic, qos=1)
    print(f"[{DEVICE_ID}] connected rc={rc}, subscribed {topic}")


def on_message(client, userdata, msg):
    global current_config, current_version

    payload = msg.payload.decode("utf-8")
    print(f"[{DEVICE_ID}] desired received: {payload}")
    desired = json.loads(payload)

    version = desired.get("version")
    version_id = desired.get("configVersionId")
    cfg = desired.get("payload", {})

    time.sleep(random.uniform(0.1, 0.5))

    fail_prob = 0.05
    if PROFILE == "unstable":
        fail_prob = 0.15
    if PROFILE == "bad_network":
        fail_prob = 0.10
    if device_state["battery"] < 10:
        fail_prob += 0.10

    fail = random.random() < fail_prob
    status = "FAILED" if fail else "APPLIED"
    err = "simulated apply error" if fail else ""

    ack = {
        "deviceId": DEVICE_ID,
        "version": version,
        "configVersionId": version_id,
        "status": status,
        "error": err,
        "ts": now()
    }
    client.publish(f"config/ack/{DEVICE_ID}", json.dumps(ack), qos=1, retain=False)

    if not fail:
        current_config = cfg
        current_version = version_id

        reported = {
            "deviceId": DEVICE_ID,
            "version": version,
            "configVersionId": version_id,
            "state": {
                "config": cfg,
                "battery": device_state["battery"],
                "online": device_state["online"],
                "profile": PROFILE,
                "scenarios": SCENARIOS,
            },
            "ts": now()
        }
        client.publish(f"state/reported/{DEVICE_ID}", json.dumps(reported), qos=0, retain=False)


def main():
    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
    client.on_connect = on_connect
    client.on_message = on_message
    client.connect(BROKER, PORT, 60)

    t = threading.Thread(target=telemetry_loop, args=(client,), daemon=True)
    t.start()

    client.loop_forever()


if __name__ == "__main__":
    main()