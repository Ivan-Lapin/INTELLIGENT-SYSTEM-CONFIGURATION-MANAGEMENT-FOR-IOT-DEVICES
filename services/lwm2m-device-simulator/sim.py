import json
import os
import random
import time
from datetime import datetime, timezone

import paho.mqtt.client as mqtt

DEVICE_ID = os.getenv("DEVICE_ID", "lwm2m-device-1")
BROKER = os.getenv("MQTT_BROKER_HOST", "mosquitto")
PORT = int(os.getenv("MQTT_BROKER_PORT", "1883"))

current_config = {"rate": 10}
current_version = None


def now():
    return datetime.now(timezone.utc).isoformat()


def on_connect(client, userdata, flags, rc, properties=None):
    topic = f"lwm2m/desired/{DEVICE_ID}"
    client.subscribe(topic, qos=1)
    print(f"[LwM2M {DEVICE_ID}] connected rc={rc}, subscribed {topic}")


def on_message(client, userdata, msg):
    global current_config, current_version

    payload = msg.payload.decode("utf-8")
    print(f"[LwM2M {DEVICE_ID}] desired received: {payload}")
    desired = json.loads(payload)

    version_id = desired["versionId"]
    cfg = desired.get("payload", {})

    time.sleep(random.uniform(0.1, 0.4))

    fail = random.random() < 0.03
    status = "FAILED" if fail else "APPLIED"
    err = "simulated lwm2m apply error" if fail else ""

    ack = {
        "deviceId": DEVICE_ID,
        "versionId": version_id,
        "status": status,
        "error": err,
        "ts": now(),
        "protocol": "lwm2m"
    }
    client.publish(f"lwm2m/ack/{DEVICE_ID}", json.dumps(ack), qos=1, retain=False)

    if not fail:
        current_config = cfg
        current_version = version_id

        reported = {
            "deviceId": DEVICE_ID,
            "versionId": version_id,
            "state": cfg,
            "ts": now(),
            "protocol": "lwm2m"
        }
        client.publish(f"lwm2m/reported/{DEVICE_ID}", json.dumps(reported), qos=0, retain=False)


def main():
    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
    client.on_connect = on_connect
    client.on_message = on_message
    client.connect(BROKER, PORT, 60)
    client.loop_forever()


if __name__ == "__main__":
    main()