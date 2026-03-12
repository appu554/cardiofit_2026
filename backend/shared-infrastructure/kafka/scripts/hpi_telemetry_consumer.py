#!/usr/bin/env python3
"""
Simple telemetry consumer for local testing.

Usage:
  pip install confluent-kafka
  python scripts/hpi_telemetry_consumer.py --topic hpi.session.events
"""

import argparse
import json
import sys
from datetime import datetime

from confluent_kafka import Consumer, KafkaException


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Consume HPI telemetry events.")
    parser.add_argument("--bootstrap", default="localhost:9092", help="Bootstrap servers")
    parser.add_argument("--group", default="hpi-telemetry-debug", help="Consumer group id")
    parser.add_argument("--topic", default="hpi.session.events", help="Topic to consume")
    parser.add_argument("--timeout", type=float, default=1.0, help="Poll timeout in seconds")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    conf = {
        "bootstrap.servers": args.bootstrap,
        "group.id": args.group,
        "auto.offset.reset": "earliest",
    }
    consumer = Consumer(conf)
    consumer.subscribe([args.topic])

    print(f"Listening on {args.topic} (bootstrap={args.bootstrap}) … Ctrl+C to stop")
    try:
        while True:
            msg = consumer.poll(args.timeout)
            if msg is None:
                continue
            if msg.error():
                raise KafkaException(msg.error())
            ts = datetime.utcfromtimestamp(msg.timestamp()[1] / 1000).isoformat()
            value = msg.value()
            try:
                decoded = json.loads(value)
            except Exception:
                decoded = value.decode("utf-8", errors="replace")
            print(f"[{ts}] {msg.topic()}[{msg.partition()}] offset={msg.offset()} -> {decoded}")
    except KeyboardInterrupt:
        pass
    finally:
        consumer.close()


if __name__ == "__main__":
    main()
