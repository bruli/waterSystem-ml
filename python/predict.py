import json
import numpy as np
import pandas as pd

from data_loader import load_available_zones, load_prediction_data
from features import build_prediction_dataset
from model_io import load_models_for_zone

SOIL_MOISTURE_THRESHOLD = 40.0
MIN_SECONDS = 3.0
MAX_SECONDS = 45.0


def safe_predict_proba(clf, X_predict):
    proba = clf.predict_proba(X_predict)

    if len(clf.classes_) == 1:
        only_class = clf.classes_[0]
        value = 1.0 if only_class == 1 else 0.0
        return np.full(len(X_predict), value, dtype=float)

    class_index = list(clf.classes_).index(1)
    return proba[:, class_index]


def row_has_valid_soil_data(row) -> bool:
    soil_moisture = pd.to_numeric(pd.Series([row.get("soil_moisture")]), errors="coerce").iloc[0]
    soil_temperature = pd.to_numeric(pd.Series([row.get("soil_temperature")]), errors="coerce").iloc[0]

    if pd.isna(soil_moisture) or pd.isna(soil_temperature):
        return False

    if soil_moisture <= 0 or soil_temperature <= 0:
        return False

    return True


def apply_blocking_rules(row, seconds: float) -> tuple[float, str]:
    weather_is_raining_last = int(row.get("weather_is_raining_last", 0) or 0)
    soil_temp_is_extreme = int(row.get("soil_temp_is_extreme", 0) or 0)
    forecast_rain_prob = float(row.get("forecast_precipitation_probability", 0.0) or 0.0)

    if not row.get("has_valid_soil_data", False):
        return 0.0, "missing_soil_data"

    if float(row.get("soil_moisture", 0.0) or 0.0) >= SOIL_MOISTURE_THRESHOLD:
        return 0.0, "soil_moisture_ok"

    if weather_is_raining_last == 1:
        return 0.0, "raining_now_or_recently"

    if soil_temp_is_extreme == 1:
        return 0.0, "soil_temp_extreme"

    if forecast_rain_prob >= 80:
        return 0.0, "high_rain_probability"

    return float(seconds), "soil_moisture_below_threshold"


def adjust_seconds_by_conditions(row, predicted_seconds: float) -> float:
    seconds = float(predicted_seconds)

    forecast_rain_prob = float(row.get("forecast_precipitation_probability", 0.0) or 0.0)
    drying_factor = float(row.get("forecast_drying_factor", 0.0) or 0.0)
    temperature = float(row.get("temperature", 0.0) or 0.0)

    if forecast_rain_prob >= 50:
        seconds *= 0.7

    if drying_factor >= 0.7:
        seconds *= 1.25
    elif drying_factor >= 0.4:
        seconds *= 1.10
    elif drying_factor <= 0.15:
        seconds *= 0.85

    if temperature >= 30:
        seconds *= 1.15
    elif temperature <= 10:
        seconds *= 0.85

    return seconds


def build_decision_reason(row) -> str:
    return str(row.get("decision_reason_raw", "unknown"))


def log_zone_prediction(zone: str, row: dict) -> None:
    print(
        (
            f"[{zone}] "
            f"soil_moisture={row.get('soil_moisture')} "
            f"threshold={SOIL_MOISTURE_THRESHOLD} "
            f"valid_soil={row.get('has_valid_soil_data')} "
            f"watering_proba={row.get('watering_proba')} "
            f"raw_seconds={row.get('raw_predicted_seconds')} "
            f"adjusted_seconds={row.get('predicted_seconds')} "
            f"should_water={row.get('should_water')} "
            f"reason={row.get('decision_reason')}"
        )
    )


def main():
    zones = load_available_zones(start="-180d")

    if not zones:
        print("[]")
        return

    results = []

    for zone in zones:
        print(f"Predint zona: {zone}")

        try:
            clf, reg, metadata = load_models_for_zone(zone)
        except FileNotFoundError:
            print(f"No s'han trobat models per a la zona {zone}, la salto")
            continue
        except RuntimeError as e:
            print(f"Error carregant models per a {zone}: {e}")
            continue

        df = load_prediction_data(zone=zone, lookback="-30d")

        if df.empty:
            print(f"Sense dades de predicció per a la zona {zone}, la salto")
            continue

        X_predict, df_predict = build_prediction_dataset(df, metadata)

        if X_predict.empty:
            print(f"Sense features per a la zona {zone}, la salto")
            continue

        df_predict = df_predict.copy()

        df_predict["watering_proba"] = safe_predict_proba(clf, X_predict)
        df_predict["raw_predicted_seconds"] = 0.0

        if reg is not None:
            try:
                df_predict["raw_predicted_seconds"] = reg.predict(X_predict)
            except Exception as e:
                print(f"Error calculant predicted_seconds per a {zone}: {e}")
                df_predict["raw_predicted_seconds"] = 0.0

        df_predict["raw_predicted_seconds"] = pd.to_numeric(
            df_predict["raw_predicted_seconds"], errors="coerce"
        ).fillna(0.0)

        df_predict["raw_predicted_seconds"] = df_predict["raw_predicted_seconds"].clip(lower=0.0)

        if "soil_temp_is_extreme" not in df_predict.columns:
            df_predict["soil_temp_is_extreme"] = 0

        df_predict["soil_temp_is_extreme"] = pd.to_numeric(
            df_predict["soil_temp_is_extreme"], errors="coerce"
        ).fillna(0).astype(int)

        df_predict["soil_moisture"] = pd.to_numeric(
            df_predict.get("soil_moisture", 0.0), errors="coerce"
        )

        df_predict["soil_temperature"] = pd.to_numeric(
            df_predict.get("soil_temperature", 0.0), errors="coerce"
        )

        df_predict["forecast_precipitation_probability"] = pd.to_numeric(
            df_predict.get("forecast_precipitation_probability", 0.0), errors="coerce"
        ).fillna(0.0)

        df_predict["forecast_drying_factor"] = pd.to_numeric(
            df_predict.get("forecast_drying_factor", 0.0), errors="coerce"
        ).fillna(0.0)

        df_predict["weather_is_raining_last"] = pd.to_numeric(
            df_predict.get("weather_is_raining_last", 0), errors="coerce"
        ).fillna(0).astype(int)

        df_predict["temperature"] = pd.to_numeric(
            df_predict.get("temperature", 0.0), errors="coerce"
        ).fillna(0.0)

        df_predict["has_valid_soil_data"] = df_predict.apply(row_has_valid_soil_data, axis=1)

        df_predict["predicted_seconds"] = 0.0
        df_predict["decision_reason_raw"] = "unknown"

        for idx, row in df_predict.iterrows():
            base_seconds, reason = apply_blocking_rules(row, row["raw_predicted_seconds"])

            if base_seconds > 0:
                adjusted_seconds = adjust_seconds_by_conditions(row, base_seconds)
            else:
                adjusted_seconds = 0.0

            adjusted_seconds = max(0.0, min(float(adjusted_seconds), MAX_SECONDS))

            if adjusted_seconds < MIN_SECONDS:
                adjusted_seconds = 0.0
                if reason == "soil_moisture_below_threshold":
                    reason = "below_minimum_runtime"

            df_predict.at[idx, "predicted_seconds"] = adjusted_seconds
            df_predict.at[idx, "decision_reason_raw"] = reason

        df_predict["should_water"] = df_predict["predicted_seconds"] > 0.0
        df_predict["predicted_seconds"] = df_predict["predicted_seconds"].round(1)
        df_predict["decision_reason"] = df_predict.apply(build_decision_reason, axis=1)

        output_columns = [
            "_time",
            "zone",
            "temperature",
            "weather_is_raining_last",
            "forecast_temperature",
            "forecast_relative_humidity",
            "forecast_precipitation_probability",
            "forecast_cloud_cover",
            "forecast_shortwave_radiation",
            "forecast_drying_factor",
            "days_since_last_watering",
            "soil_moisture",
            "soil_temperature",
            "soil_moisture_diff",
            "watering_proba",
            "raw_predicted_seconds",
            "should_water",
            "predicted_seconds",
            "decision_reason",
        ]

        for col in output_columns:
            if col not in df_predict.columns:
                df_predict[col] = None

        records = df_predict[output_columns].copy().to_dict(orient="records")

        for record in records:
            log_zone_prediction(zone, record)

        results.extend(records)

    print(json.dumps(results, default=str, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()