import pandas as pd

FEATURE_COLUMNS = [
    "temperature",
    "weather_is_raining_last",
    "forecast_temperature",
    "forecast_relative_humidity",
    "forecast_precipitation_probability",
    "forecast_cloud_cover",
    "forecast_shortwave_radiation",
    "forecast_drying_factor",
    "soil_moisture",
    "soil_temperature",
    "soil_moisture_diff",
    "soil_temp_is_extreme",
    "hour",
    "day_of_week",
    "month",
    "days_since_last_watering",
]


def normalize_zone(zone: str) -> str:
    if not zone:
        return zone

    zone = zone.strip()

    if zone.endswith(" with fertilizer"):
        zone = zone.replace(" with fertilizer", "")

    return zone


def add_time_features(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    df["_time"] = pd.to_datetime(df["_time"], utc=True)

    df["hour"] = df["_time"].dt.hour
    df["day_of_week"] = df["_time"].dt.dayofweek
    df["month"] = df["_time"].dt.month

    return df


def add_watering_history(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy().sort_values("_time")

    if "last_watering_time" in df.columns:
        df["last_watering_time"] = pd.to_datetime(
            df["last_watering_time"], utc=True, errors="coerce"
        )
    else:
        df["last_watering_time"] = df["_time"].where(df["seconds"] > 0).ffill()

    df["days_since_last_watering"] = (
            (df["_time"] - df["last_watering_time"]).dt.total_seconds() / 86400
    )

    df["days_since_last_watering"] = df["days_since_last_watering"].clip(lower=0)
    df["days_since_last_watering"] = df["days_since_last_watering"].fillna(999)

    return df


def add_sensor_features(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy().sort_values("_time")

    df["soil_moisture"] = pd.to_numeric(
        df.get("soil_moisture", 0.0), errors="coerce"
    )
    df["soil_temperature"] = pd.to_numeric(
        df.get("soil_temperature", 0.0), errors="coerce"
    )

    df["soil_moisture"] = df["soil_moisture"].fillna(0.0)
    df["soil_temperature"] = df["soil_temperature"].fillna(0.0)

    df["soil_moisture_diff"] = df["soil_moisture"].diff().fillna(0.0)

    df["soil_temp_is_extreme"] = (
            (df["soil_temperature"] < 5.0) | (df["soil_temperature"] > 30.0)
    ).astype(int)

    return df


def build_training_dataset(df: pd.DataFrame):
    df = add_time_features(df)
    df = add_watering_history(df)
    df = add_sensor_features(df)

    df = df.dropna(subset=["temperature"])

    if "forecast_temperature" not in df.columns:
        df["forecast_temperature"] = df["temperature"]

    df["forecast_relative_humidity"] = pd.to_numeric(
        df.get("forecast_relative_humidity", 0.0), errors="coerce"
    ).fillna(0.0)

    df["forecast_precipitation_probability"] = pd.to_numeric(
        df.get("forecast_precipitation_probability", 0.0), errors="coerce"
    ).fillna(0.0)

    df["forecast_cloud_cover"] = pd.to_numeric(
        df.get("forecast_cloud_cover", 0.0), errors="coerce"
    ).fillna(0.0)

    df["forecast_shortwave_radiation"] = pd.to_numeric(
        df.get("forecast_shortwave_radiation", 0.0), errors="coerce"
    ).fillna(0.0)

    df["forecast_drying_factor"] = pd.to_numeric(
        df.get("forecast_drying_factor", 0.0), errors="coerce"
    ).fillna(0.0)

    df["soil_moisture"] = pd.to_numeric(
        df.get("soil_moisture", 0.0), errors="coerce"
    ).fillna(0.0)

    df["soil_temperature"] = pd.to_numeric(
        df.get("soil_temperature", 0.0), errors="coerce"
    ).fillna(0.0)

    df["soil_moisture_diff"] = pd.to_numeric(
        df.get("soil_moisture_diff", 0.0), errors="coerce"
    ).fillna(0.0)

    df["soil_temp_is_extreme"] = pd.to_numeric(
        df.get("soil_temp_is_extreme", 0), errors="coerce"
    ).fillna(0).astype(int)

    if df.empty:
        return {
            "X_classifier": pd.DataFrame(),
            "y_classifier": pd.Series(dtype="int64"),
            "X_regressor": pd.DataFrame(),
            "y_regressor": pd.Series(dtype="float64"),
            "metadata": {
                "feature_columns": FEATURE_COLUMNS,
                "classification_threshold": 0.3,
                "model_version": "v4",
            },
        }

    X = df[FEATURE_COLUMNS].copy()
    y_classifier = (df["seconds"] > 0).astype(int)

    df_reg = df[df["seconds"] > 0].copy()
    X_reg = df_reg[FEATURE_COLUMNS].copy()
    y_reg = df_reg["seconds"].copy()

    metadata = {
        "feature_columns": FEATURE_COLUMNS,
        "classification_threshold": 0.3,
        "model_version": "v4",
    }

    return {
        "X_classifier": X,
        "y_classifier": y_classifier,
        "X_regressor": X_reg,
        "y_regressor": y_reg,
        "metadata": metadata,
    }


def build_prediction_dataset(df: pd.DataFrame, metadata: dict):
    df = add_time_features(df)
    df = add_watering_history(df)
    df = add_sensor_features(df)

    if "forecast_temperature" not in df.columns:
        df["forecast_temperature"] = df["temperature"]

    for col in metadata["feature_columns"]:
        if col not in df.columns:
            if col == "forecast_temperature":
                df[col] = df["temperature"]
            elif col == "soil_temp_is_extreme":
                df[col] = 0
            else:
                df[col] = 0.0

    X = df[metadata["feature_columns"]].copy()

    return X, df