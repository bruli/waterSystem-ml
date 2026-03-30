from __future__ import annotations

import os
import random
from datetime import datetime, timedelta, timezone

from influxdb_client import InfluxDBClient, Point, WritePrecision
from influxdb_client.client.write_api import SYNCHRONOUS

INFLUXDB_URL = os.getenv("INFLUXDB_URL", "http://influxdb:8086")
INFLUXDB_TOKEN = os.getenv("INFLUXDB_TOKEN", "my-super-token")
INFLUXDB_ORG = os.getenv("INFLUXDB_ORG", "water-system")
INFLUXDB_BUCKET = os.getenv("INFLUXDB_BUCKET", "bonsai-data")


def main() -> None:
    client = InfluxDBClient(
        url=INFLUXDB_URL,
        token=INFLUXDB_TOKEN,
        org=INFLUXDB_ORG,
    )

    write_api = client.write_api(write_options=SYNCHRONOUS)

    now = datetime.now(timezone.utc).replace(minute=0, second=0, microsecond=0)

    points: list[Point] = []
    zones = ["Bonsai big", "Bonsai small"]

    for hours_ago in range(24 * 14, -1, -1):
        ts = now - timedelta(hours=hours_ago)

        temperature = round(random.uniform(8.0, 28.0), 2)
        rain_probability = round(random.uniform(0, 100), 1)
        is_raining = random.random() < 0.12

        points.append(
            Point("weather")
            .field("temperature", temperature)
            .field("forecast_precipitation_probability", rain_probability)
            .field("is_raining", is_raining)
            .time(ts, WritePrecision.S)
        )

        for zone in zones:
            watered = random.random() < 0.18
            seconds = random.randint(8, 45) if watered else 0

            points.append(
                Point("logs")
                .tag("zone", zone)
                .field("seconds", seconds)
                .field("executed", watered)
                .time(ts, WritePrecision.S)
            )

    write_api.write(bucket=INFLUXDB_BUCKET, org=INFLUXDB_ORG, record=points)
    client.close()

    print(f"Inserted {len(points)} points into bucket '{INFLUXDB_BUCKET}'")


if __name__ == "__main__":
    main()