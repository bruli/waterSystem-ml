from __future__ import annotations

import os
import random
from datetime import datetime, timedelta, timezone

import requests
from influxdb_client import InfluxDBClient, Point, WritePrecision
from influxdb_client.client.write_api import SYNCHRONOUS

INFLUXDB_URL = os.getenv("INFLUXDB_URL", "http://influxdb:8086")
INFLUXDB_TOKEN = os.getenv("INFLUXDB_TOKEN", "my-super-token")
INFLUXDB_ORG = os.getenv("INFLUXDB_ORG", "water-system")
INFLUXDB_BUCKET = os.getenv("INFLUXDB_BUCKET", "bonsai-data")

OPEN_METEO_URL = "https://api.open-meteo.com/v1/forecast"
OPEN_METEO_LATITUDE = float(os.getenv("OPEN_METEO_LATITUDE", "41.5451707"))
OPEN_METEO_LONGITUDE = float(os.getenv("OPEN_METEO_LONGITUDE", "2.1032168"))
FORECAST_LOCATION = os.getenv("FORECAST_LOCATION", "terrace")


def calculate_drying_factor(
        temperature: float,
        relative_humidity: float,
        precipitation_probability: float,
        cloud_cover: float,
        shortwave_radiation: float,
) -> float:
    """
    Fórmula simple de prova.
    Pots canviar els pesos quan vulgues.
    """
    temp_score = max(0.0, min(temperature / 35.0, 1.0)) * 30.0
    humidity_score = max(0.0, min((100.0 - relative_humidity) / 100.0, 1.0)) * 25.0
    rain_score = max(0.0, min((100.0 - precipitation_probability) / 100.0, 1.0)) * 20.0
    cloud_score = max(0.0, min((100.0 - cloud_cover) / 100.0, 1.0)) * 10.0
    radiation_score = max(0.0, min(shortwave_radiation / 1000.0, 1.0)) * 15.0

    return round(temp_score + humidity_score + rain_score + cloud_score + radiation_score, 2)


def fetch_forecast(start_date: str, end_date: str) -> dict:
    params = {
        "latitude": OPEN_METEO_LATITUDE,
        "longitude": OPEN_METEO_LONGITUDE,
        "hourly": ",".join(
            [
                "temperature_2m",
                "relative_humidity_2m",
                "precipitation_probability",
                "cloud_cover",
                "shortwave_radiation",
            ]
        ),
        "start_date": start_date,
        "end_date": end_date,
        "timezone": "auto",
    }

    response = requests.get(OPEN_METEO_URL, params=params, timeout=30)
    response.raise_for_status()
    return response.json()


def build_forecast_points(forecast_data: dict) -> list[Point]:
    hourly = forecast_data.get("hourly", {})

    times = hourly.get("time", [])
    temperatures = hourly.get("temperature_2m", [])
    humidities = hourly.get("relative_humidity_2m", [])
    precipitation_probabilities = hourly.get("precipitation_probability", [])
    cloud_covers = hourly.get("cloud_cover", [])
    shortwave_radiations = hourly.get("shortwave_radiation", [])

    forecast_generated_at = datetime.now(timezone.utc).replace(microsecond=0).isoformat()

    points: list[Point] = []

    for i, time_str in enumerate(times):
        ts = datetime.fromisoformat(time_str)

        temperature = float(temperatures[i])
        relative_humidity = float(humidities[i])
        precipitation_probability = float(precipitation_probabilities[i])
        cloud_cover = float(cloud_covers[i])
        shortwave_radiation = float(shortwave_radiations[i])

        drying_factor = calculate_drying_factor(
            temperature=temperature,
            relative_humidity=relative_humidity,
            precipitation_probability=precipitation_probability,
            cloud_cover=cloud_cover,
            shortwave_radiation=shortwave_radiation,
        )

        point = (
            Point("forecast_v2")
            .tag("location", FORECAST_LOCATION)
            .field("temperature", temperature)
            .field("relative_humidity", relative_humidity)
            .field("precipitation_probability", precipitation_probability)
            .field("cloud_cover", cloud_cover)
            .field("shortwave_radiation", shortwave_radiation)
            .field("forecast_generated_at", forecast_generated_at)
            .field("drying_factor", drying_factor)
            .time(ts, WritePrecision.S)
        )

        points.append(point)

    return points


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

    # weather + logs de prova
    for hours_ago in range(24 * 14, -1, -1):
        ts = now - timedelta(hours=hours_ago)

        temperature = round(random.uniform(8.0, 28.0), 2)
        is_raining = random.random() < 0.12
        humidity = round(random.uniform(5.0, 35.0), 2)

        points.append(
            Point("weather")
            .field("temperature", temperature)
            .field("is_raining", is_raining)
            .field("humidity", humidity)
            .time(ts, WritePrecision.S)
        )

        for zone in zones:
            watered = random.random() < 0.18
            seconds = random.randint(8, 45) if watered else 0

            points.append(
                Point("logs")
                .tag("zone", zone)
                .field("seconds", seconds)
                .time(ts, WritePrecision.S)
            )

    # forecast per a demà
    tomorrow = (datetime.now().date() + timedelta(days=1)).isoformat()
    forecast_data = fetch_forecast(start_date=tomorrow, end_date=tomorrow)
    forecast_points = build_forecast_points(forecast_data)
    points.extend(forecast_points)

    write_api.write(bucket=INFLUXDB_BUCKET, org=INFLUXDB_ORG, record=points)
    client.close()

    print(f"Inserted {len(points)} points into bucket '{INFLUXDB_BUCKET}'")
    print(f"Inserted {len(forecast_points)} forecast points into measurement 'forecast_v2'")


if __name__ == "__main__":
    main()