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

current_config = {"rate": 10}
current_version = None
battery_level = 0.95

def now():
    return datetime.now(timezone.utc).isoformat()

def build_metrics():
    global battery_level
    battery_level = max(0.05, battery_level - random.uniform(0.0005, 0.003))

    rate = current_config.get("rate", 10)

    # Чем чаще отчётность, тем условно выше нагрузка
    base_latency = 15 + max(0, 20 - rate) * 0.2
    latency = max(1.0, random.gauss(base_latency, 3.5))

    loss = min(0.2, max(0.0, random.random() * 0.02))
    jitter = max(0.1, random.gauss(4.0, 1.0))
    rssi = max(-95.0, min(-40.0, random.gauss(-63.0, 4.0)))

    return {
        "latency_ms": round(latency, 3),
        "loss": round(loss, 5),
        "jitter_ms": round(jitter, 3),
        "rssi": round(rssi, 3),
        "battery": round(battery_level, 5),
    }

def telemetry_loop(client):
    while True:
        payload = {
            "deviceId": DEVICE_ID,
            "ts": now(),
            "metrics": build_metrics(),
            "configVersionId": current_version,
        }
        client.publish(f"telemetry/{DEVICE_ID}/metrics", json.dumps(payload), qos=0, retain=False)
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

    version_id = desired["versionId"]
    cfg = desired.get("payload", {})

    time.sleep(random.uniform(0.1, 0.5))

    fail = random.random() < 0.05
    status = "FAILED" if fail else "APPLIED"
    err = "simulated apply error" if fail else ""

    ack = {
        "deviceId": DEVICE_ID,
        "versionId": version_id,
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
            "versionId": version_id,
            "state": cfg,
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