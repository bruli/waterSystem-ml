import os
import sys
import warnings
from typing import Optional

import pandas as pd
from influxdb_client import InfluxDBClient
from influxdb_client.client.warnings import MissingPivotFunction
from features import normalize_zone

warnings.simplefilter("ignore", MissingPivotFunction)

INFLUXDB_URL = os.getenv("INFLUXDB_URL")
INFLUXDB_TOKEN = os.getenv("INFLUXDB_TOKEN")
INFLUXDB_ORG = os.getenv("INFLUXDB_ORG", "home")
INFLUXDB_BUCKET = os.getenv("INFLUXDB_BUCKET", "bonsai-data")


def _get_client() -> InfluxDBClient:
    if not INFLUXDB_URL or not INFLUXDB_TOKEN:
        raise ValueError("Falten INFLUXDB_URL o INFLUXDB_TOKEN")

    return InfluxDBClient(
        url=INFLUXDB_URL,
        token=INFLUXDB_TOKEN,
        org=INFLUXDB_ORG,
    )


def _query_to_df(query: str) -> pd.DataFrame:
    with _get_client() as client:
        query_api = client.query_api()
        df = query_api.query_data_frame(query)

    if isinstance(df, list):
        valid_dfs = [item for item in df if isinstance(item, pd.DataFrame) and not item.empty]
        if not valid_dfs:
            return pd.DataFrame()
        df = pd.concat(valid_dfs, ignore_index=True)

    if df is None or df.empty:
        return pd.DataFrame()

    return df


def load_available_zones(start: str = "-180d") -> list[str]:
    query = f'''
from(bucket: "{INFLUXDB_BUCKET}")
  |> range(start: {start})
  |> filter(fn: (r) => r._measurement == "logs")
  |> keep(columns: ["zone"])
  |> group()
  |> distinct(column: "zone")
'''

    df = _query_to_df(query)

    print("AVAILABLE ZONES DF:", file=sys.stderr)
    print(df.head(), file=sys.stderr)
    print("COLUMNS:", df.columns, file=sys.stderr)

    if df.empty:
        return []

    if "_value" in df.columns:
        values = df["_value"].dropna().tolist()
    else:
        return []

    zones = sorted(
        list({
            normalize_zone(str(v))
            for v in values
            if str(v).strip()
        })
    )

    return zones

def load_logs_data(zone: Optional[str] = None, start: str = "-90d") -> pd.DataFrame:
    # Si hi ha zona, filtre per prefix (inclou "with fertilizer")
    if zone:
        zone_filter = f'|> filter(fn: (r) => strings.hasPrefix(v: r.zone, prefix: "{zone}"))'
    else:
        zone_filter = ""

    query = f'''
import "strings"

from(bucket: "{INFLUXDB_BUCKET}")
  |> range(start: {start})
  |> filter(fn: (r) => r._measurement == "logs")
  |> filter(fn: (r) => r._field == "seconds")
  {zone_filter}
  |> keep(columns: ["_time", "_value", "zone"])
  |> sort(columns: ["_time"])
'''

    df = _query_to_df(query)

    if df.empty:
        return pd.DataFrame(columns=["_time", "seconds", "zone"])

    df = df[["_time", "_value", "zone"]].copy()

    # conversions
    df["_time"] = pd.to_datetime(df["_time"], utc=True)
    df["seconds"] = pd.to_numeric(df["_value"], errors="coerce")

    # 🔥 NORMALITZACIÓ DE ZONA (clau)
    df["zone"] = df["zone"].astype(str).apply(normalize_zone)

    df = df.drop(columns=["_value"])

    return df


def load_weather_data(start: str = "-90d") -> pd.DataFrame:
    query = f'''
from(bucket: "{INFLUXDB_BUCKET}")
  |> range(start: {start})
  |> filter(fn: (r) => r._measurement == "weather")
  |> filter(fn: (r) => r._field == "temperature" or r._field == "is_raining")
  |> keep(columns: ["_time", "_field", "_value"])
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
  |> sort(columns: ["_time"])
'''

    df = _query_to_df(query)

    if df.empty:
        return pd.DataFrame(columns=["_time", "temperature", "weather_is_raining_last"])

    df = df.copy()
    df["_time"] = pd.to_datetime(df["_time"], utc=True)

    if "temperature" not in df.columns:
        df["temperature"] = None

    if "is_raining" not in df.columns:
        df["is_raining"] = 0

    df["temperature"] = pd.to_numeric(df["temperature"], errors="coerce")

    df["weather_is_raining_last"] = (
        df["is_raining"]
        .astype(str)
        .str.lower()
        .map({"true": 1, "false": 0, "1": 1, "0": 0})
    )

    df["weather_is_raining_last"] = (
        pd.to_numeric(df["weather_is_raining_last"], errors="coerce")
        .fillna(0)
        .astype(int)
    )

    return df[["_time", "temperature", "weather_is_raining_last"]].sort_values("_time")


def load_training_data(zone: Optional[str] = None, start: str = "-90d") -> pd.DataFrame:
    df_logs = load_logs_data(zone=zone, start=start)
    df_weather = load_weather_data(start=start)

    if df_logs.empty:
        return pd.DataFrame()

    if df_weather.empty:
        return pd.DataFrame()

    df_logs = df_logs.sort_values("_time")
    df_weather = df_weather.sort_values("_time")

    df = pd.merge_asof(
        df_logs,
        df_weather,
        on="_time",
        direction="backward",
    )

    df["forecast_precipitation_probability"] = 0.0

    return df


def load_prediction_data(zone: Optional[str] = None, lookback: str = "-30d") -> pd.DataFrame:
    df_weather = load_weather_data(start=lookback)
    df_logs = load_logs_data(zone=zone, start=lookback)

    if df_weather.empty:
        raise ValueError("No hi ha dades de weather per predir")

    latest_weather = df_weather.sort_values("_time").tail(1).copy()

    prediction_time = latest_weather.iloc[0]["_time"]

    df_logs_before_prediction = df_logs[df_logs["_time"] <= prediction_time].sort_values("_time")

    if df_logs_before_prediction.empty:
        last_watering_time = pd.NaT
        last_seconds = 0.0
    else:
        last_log = df_logs_before_prediction.tail(1).iloc[0]
        last_watering_time = last_log["_time"]
        last_seconds = float(last_log["seconds"])

    latest_weather["zone"] = zone if zone else "unknown"
    latest_weather["seconds"] = last_seconds
    latest_weather["last_watering_time"] = last_watering_time
    latest_weather["forecast_precipitation_probability"] = 0.0

    return latest_weather.reset_index(drop=True)