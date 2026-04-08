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

SENSOR_MEASUREMENT_TO_ZONE = {
    "sensor.bonsai_big_bonsai_big_soil_moisture": "Bonsai big",
    "sensor.bonsai_big_bonsai_big_soil_temperature": "Bonsai big",
    "sensor.bonsai_small_bonsai_small_soil_moisture": "Bonsai small",
    "sensor.bonsai_small_bonsai_small_soil_temperature": "Bonsai small",
}


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
    df["_time"] = pd.to_datetime(df["_time"], utc=True)
    df["seconds"] = pd.to_numeric(df["_value"], errors="coerce")
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


def load_forecast_data(start: str = "-90d") -> pd.DataFrame:
    query = f'''
from(bucket: "{INFLUXDB_BUCKET}")
  |> range(start: {start})
  |> filter(fn: (r) => r._measurement == "forecast_v2")
  |> filter(fn: (r) =>
      r._field == "temperature" or
      r._field == "relative_humidity" or
      r._field == "precipitation_probability" or
      r._field == "cloud_cover" or
      r._field == "shortwave_radiation" or
      r._field == "drying_factor" or
      r._field == "forecast_generated_at"
  )
  |> keep(columns: ["_time", "_field", "_value", "location"])
  |> pivot(rowKey: ["_time", "location"], columnKey: ["_field"], valueColumn: "_value")
  |> sort(columns: ["_time"])
'''

    df = _query_to_df(query)

    if df.empty:
        return pd.DataFrame(
            columns=[
                "_time",
                "location",
                "forecast_temperature",
                "forecast_relative_humidity",
                "forecast_precipitation_probability",
                "forecast_cloud_cover",
                "forecast_shortwave_radiation",
                "forecast_drying_factor",
                "forecast_generated_at",
            ]
        )

    df = df.copy()
    df["_time"] = pd.to_datetime(df["_time"], utc=True)

    numeric_defaults = {
        "temperature": None,
        "relative_humidity": 0.0,
        "precipitation_probability": 0.0,
        "cloud_cover": 0.0,
        "shortwave_radiation": 0.0,
        "drying_factor": 0.0,
    }

    for col, default in numeric_defaults.items():
        if col not in df.columns:
            df[col] = default

    if "forecast_generated_at" not in df.columns:
        df["forecast_generated_at"] = None

    df["forecast_temperature"] = pd.to_numeric(df["temperature"], errors="coerce")
    df["forecast_relative_humidity"] = pd.to_numeric(df["relative_humidity"], errors="coerce").fillna(0.0)
    df["forecast_precipitation_probability"] = pd.to_numeric(
        df["precipitation_probability"], errors="coerce"
    ).fillna(0.0)
    df["forecast_cloud_cover"] = pd.to_numeric(df["cloud_cover"], errors="coerce").fillna(0.0)
    df["forecast_shortwave_radiation"] = pd.to_numeric(
        df["shortwave_radiation"], errors="coerce"
    ).fillna(0.0)
    df["forecast_drying_factor"] = pd.to_numeric(
        df["drying_factor"], errors="coerce"
    ).fillna(0.0)

    df["forecast_generated_at"] = pd.to_numeric(df["forecast_generated_at"], errors="coerce")
    df["forecast_generated_at"] = pd.to_datetime(
        df["forecast_generated_at"], unit="s", utc=True, errors="coerce"
    )

    return df[
        [
            "_time",
            "location",
            "forecast_temperature",
            "forecast_relative_humidity",
            "forecast_precipitation_probability",
            "forecast_cloud_cover",
            "forecast_shortwave_radiation",
            "forecast_drying_factor",
            "forecast_generated_at",
        ]
    ].sort_values(["_time", "forecast_generated_at"])


def load_soil_moisture_data(zone: Optional[str] = None, start: str = "-90d") -> pd.DataFrame:
    query = f'''
from(bucket: "{INFLUXDB_BUCKET}")
  |> range(start: {start})
  |> filter(fn: (r) => r._field == "value")
  |> filter(fn: (r) => r.domain == "sensor")
  |> filter(fn: (r) =>
      r._measurement == "sensor.bonsai_big_bonsai_big_soil_moisture" or
      r._measurement == "sensor.bonsai_small_bonsai_small_soil_moisture"
  )
  |> keep(columns: ["_time", "_measurement", "_value"])
  |> sort(columns: ["_time"])
'''

    df = _query_to_df(query)

    if df.empty:
        return pd.DataFrame(columns=["_time", "zone", "soil_moisture"])

    df = df[["_time", "_measurement", "_value"]].copy()
    df["_time"] = pd.to_datetime(df["_time"], utc=True)
    df["zone"] = df["_measurement"].map(SENSOR_MEASUREMENT_TO_ZONE)
    df["soil_moisture"] = pd.to_numeric(df["_value"], errors="coerce")
    df = df.drop(columns=["_measurement", "_value"])
    df = df.dropna(subset=["zone"])
    df["zone"] = df["zone"].astype(str).apply(normalize_zone)

    if zone:
        df = df[df["zone"] == normalize_zone(zone)]

    return df.sort_values(["zone", "_time"]).reset_index(drop=True)


def load_soil_temperature_data(zone: Optional[str] = None, start: str = "-90d") -> pd.DataFrame:
    query = f'''
from(bucket: "{INFLUXDB_BUCKET}")
  |> range(start: {start})
  |> filter(fn: (r) => r._field == "value")
  |> filter(fn: (r) => r.domain == "sensor")
  |> filter(fn: (r) =>
      r._measurement == "sensor.bonsai_big_bonsai_big_soil_temperature" or
      r._measurement == "sensor.bonsai_small_bonsai_small_soil_temperature"
  )
  |> keep(columns: ["_time", "_measurement", "_value"])
  |> sort(columns: ["_time"])
'''

    df = _query_to_df(query)

    if df.empty:
        return pd.DataFrame(columns=["_time", "zone", "soil_temperature"])

    df = df[["_time", "_measurement", "_value"]].copy()
    df["_time"] = pd.to_datetime(df["_time"], utc=True)
    df["zone"] = df["_measurement"].map(SENSOR_MEASUREMENT_TO_ZONE)
    df["soil_temperature"] = pd.to_numeric(df["_value"], errors="coerce")
    df = df.drop(columns=["_measurement", "_value"])
    df = df.dropna(subset=["zone"])
    df["zone"] = df["zone"].astype(str).apply(normalize_zone)

    if zone:
        df = df[df["zone"] == normalize_zone(zone)]

    return df.sort_values(["zone", "_time"]).reset_index(drop=True)


def _empty_forecast_row() -> dict:
    return {
        "forecast_temperature": None,
        "forecast_relative_humidity": 0.0,
        "forecast_precipitation_probability": 0.0,
        "forecast_cloud_cover": 0.0,
        "forecast_shortwave_radiation": 0.0,
        "forecast_drying_factor": 0.0,
    }


def _latest_forecast_row_for_time(
        df_forecast: pd.DataFrame,
        target_time: pd.Timestamp,
) -> dict:
    if df_forecast.empty:
        return _empty_forecast_row()

    valid_forecasts = df_forecast[
        (df_forecast["_time"] <= target_time) &
        (df_forecast["forecast_generated_at"] <= target_time)
        ].sort_values(["forecast_generated_at", "_time"])

    if valid_forecasts.empty:
        return _empty_forecast_row()

    row = valid_forecasts.iloc[-1]

    return {
        "forecast_temperature": (
            float(row["forecast_temperature"])
            if pd.notna(row["forecast_temperature"])
            else None
        ),
        "forecast_relative_humidity": float(row["forecast_relative_humidity"]),
        "forecast_precipitation_probability": float(row["forecast_precipitation_probability"]),
        "forecast_cloud_cover": float(row["forecast_cloud_cover"]),
        "forecast_shortwave_radiation": float(row["forecast_shortwave_radiation"]),
        "forecast_drying_factor": float(row["forecast_drying_factor"]),
    }


def _merge_sensor_asof_by_zone(
        df_base: pd.DataFrame,
        df_sensor: pd.DataFrame,
        value_columns: list[str],
        tolerance: str = "24h",
) -> pd.DataFrame:
    if df_base.empty or df_sensor.empty:
        return df_base

    parts = []

    for zone in df_base["zone"].dropna().unique():
        base_zone = df_base[df_base["zone"] == zone].copy().sort_values("_time")
        sensor_zone = df_sensor[df_sensor["zone"] == zone].copy().sort_values("_time")

        if sensor_zone.empty:
            parts.append(base_zone)
            continue

        merged = pd.merge_asof(
            base_zone,
            sensor_zone[["_time"] + value_columns].sort_values("_time"),
            on="_time",
            direction="backward",
            tolerance=pd.Timedelta(tolerance),
        )
        parts.append(merged)

    if not parts:
        return df_base

    return pd.concat(parts, ignore_index=True).sort_values(["zone", "_time"]).reset_index(drop=True)


def load_training_data(zone: Optional[str] = None, start: str = "-90d") -> pd.DataFrame:
    df_logs = load_logs_data(zone=zone, start=start)
    df_weather = load_weather_data(start=start)
    df_forecast = load_forecast_data(start=start)
    df_soil_moisture = load_soil_moisture_data(zone=zone, start=start)
    df_soil_temperature = load_soil_temperature_data(zone=zone, start=start)

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

    if df_forecast.empty:
        df["forecast_temperature"] = None
        df["forecast_relative_humidity"] = 0.0
        df["forecast_precipitation_probability"] = 0.0
        df["forecast_cloud_cover"] = 0.0
        df["forecast_shortwave_radiation"] = 0.0
        df["forecast_drying_factor"] = 0.0
    else:
        df_forecast = df_forecast.sort_values(["forecast_generated_at", "_time"]).copy()

        forecast_rows = []
        for _, row in df.iterrows():
            forecast_rows.append(_latest_forecast_row_for_time(df_forecast, row["_time"]))

        df_forecast_selected = pd.DataFrame(forecast_rows, index=df.index)

        for col in df_forecast_selected.columns:
            df[col] = df_forecast_selected[col]

    df = _merge_sensor_asof_by_zone(df, df_soil_moisture, ["soil_moisture"])
    df = _merge_sensor_asof_by_zone(df, df_soil_temperature, ["soil_temperature"])

    return df


def load_prediction_data(zone: Optional[str] = None, lookback: str = "-30d") -> pd.DataFrame:
    df_weather = load_weather_data(start=lookback)
    df_logs = load_logs_data(zone=zone, start=lookback)
    df_forecast = load_forecast_data(start=lookback)
    df_soil_moisture = load_soil_moisture_data(zone=zone, start=lookback)
    df_soil_temperature = load_soil_temperature_data(zone=zone, start=lookback)

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

    forecast_data = _empty_forecast_row()

    if not df_forecast.empty:
        df_forecast_before_prediction = df_forecast[
            (df_forecast["_time"] >= prediction_time.floor("h")) &
            (df_forecast["forecast_generated_at"] <= prediction_time)
            ].sort_values(["forecast_generated_at", "_time"])

        if not df_forecast_before_prediction.empty:
            latest_forecast = df_forecast_before_prediction.iloc[-1]
            forecast_data = {
                "forecast_temperature": (
                    float(latest_forecast["forecast_temperature"])
                    if pd.notna(latest_forecast["forecast_temperature"])
                    else None
                ),
                "forecast_relative_humidity": float(latest_forecast["forecast_relative_humidity"]),
                "forecast_precipitation_probability": float(
                    latest_forecast["forecast_precipitation_probability"]
                ),
                "forecast_cloud_cover": float(latest_forecast["forecast_cloud_cover"]),
                "forecast_shortwave_radiation": float(latest_forecast["forecast_shortwave_radiation"]),
                "forecast_drying_factor": float(latest_forecast["forecast_drying_factor"]),
            }

    latest_weather["zone"] = normalize_zone(zone) if zone else "unknown"
    latest_weather["seconds"] = last_seconds
    latest_weather["last_watering_time"] = last_watering_time

    for key, value in forecast_data.items():
        latest_weather[key] = value

    zone_normalized = normalize_zone(zone) if zone else None

    if zone_normalized and not df_soil_moisture.empty:
        moisture_zone = df_soil_moisture[df_soil_moisture["zone"] == zone_normalized].sort_values("_time")
        latest_weather["soil_moisture"] = (
            float(moisture_zone.iloc[-1]["soil_moisture"])
            if not moisture_zone.empty else None
        )
    else:
        latest_weather["soil_moisture"] = None

    if zone_normalized and not df_soil_temperature.empty:
        temp_zone = df_soil_temperature[df_soil_temperature["zone"] == zone_normalized].sort_values("_time")
        latest_weather["soil_temperature"] = (
            float(temp_zone.iloc[-1]["soil_temperature"])
            if not temp_zone.empty else None
        )
    else:
        latest_weather["soil_temperature"] = None

    return latest_weather.reset_index(drop=True)