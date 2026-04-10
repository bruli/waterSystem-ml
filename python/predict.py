import argparse
import json
import math
import os

import numpy as np
import pandas as pd

from data_loader import load_available_zones, load_prediction_data
from features import build_prediction_dataset
from model_io import load_models_for_zone, slugify_zone

MIN_SECONDS = 13.0
MAX_SECONDS = 60.0
DEFAULT_THRESHOLD = 0.30
MODEL_DIR = os.getenv("MODEL_DIR", "./models")


def safe_predict_proba(clf, X_predict):
    proba = clf.predict_proba(X_predict)

    if len(clf.classes_) == 1:
        only_class = clf.classes_[0]
        value = 1.0 if only_class == 1 else 0.0
        return np.full(len(X_predict), value, dtype=float)

    class_index = list(clf.classes_).index(1)
    return proba[:, class_index]


def clamp_seconds(value: float) -> float:
    if value is None:
        return MIN_SECONDS

    try:
        value = float(value)
    except (TypeError, ValueError):
        return MIN_SECONDS

    if math.isnan(value):
        return MIN_SECONDS

    return max(MIN_SECONDS, min(MAX_SECONDS, value))


def row_has_valid_context_data(row) -> bool:
    temperature = pd.to_numeric(pd.Series([row.get("temperature")]), errors="coerce").iloc[0]
    drying = pd.to_numeric(pd.Series([row.get("forecast_drying_factor")]), errors="coerce").iloc[0]

    if pd.isna(temperature):
        return False

    if pd.isna(drying):
        return False

    return True


def fallback_seconds_from_row(row) -> float:
    drying = pd.to_numeric(pd.Series([row.get("forecast_drying_factor")]), errors="coerce").iloc[0]
    temp = pd.to_numeric(pd.Series([row.get("temperature")]), errors="coerce").iloc[0]
    raining = pd.to_numeric(pd.Series([row.get("weather_is_raining_last")]), errors="coerce").iloc[0]
    precip = pd.to_numeric(
        pd.Series([row.get("forecast_precipitation_probability")]), errors="coerce"
    ).iloc[0]

    if pd.isna(drying):
        drying = 0.0
    if pd.isna(temp):
        temp = 20.0
    if pd.isna(raining):
        raining = 0
    if pd.isna(precip):
        precip = 0.0

    seconds = 13.0

    if drying >= 0.75:
        seconds += 15.0
    elif drying >= 0.55:
        seconds += 10.0
    elif drying >= 0.35:
        seconds += 5.0

    if temp >= 30:
        seconds += 10.0
    elif temp >= 25:
        seconds += 5.0

    if raining == 1:
        seconds -= 10.0

    if precip >= 70:
        seconds -= 10.0
    elif precip >= 40:
        seconds -= 5.0

    return clamp_seconds(seconds)


def zone_model_exists(zone: str) -> bool:
    zone_slug = slugify_zone(zone)
    classifier_path = os.path.join(MODEL_DIR, zone_slug, "classifier.joblib")
    metadata_path = os.path.join(MODEL_DIR, zone_slug, "metadata.json")
    return os.path.exists(classifier_path) and os.path.exists(metadata_path)


def build_no_model_result(zone: str) -> dict:
    return {
        "zone": zone,
        "should_water": False,
        "decision_reason": "No hi ha model entrenat per a esta zona",
        "predicted_seconds": 0.0,
        "probability": 0.0,
        "used_model": False,
    }


def build_error_result(zone: str, message: str) -> dict:
    return {
        "zone": zone,
        "should_water": False,
        "decision_reason": message,
        "predicted_seconds": 0.0,
        "probability": 0.0,
        "used_model": False,
    }


def predict_zone(zone: str) -> dict:
    if not zone_model_exists(zone):
        return build_no_model_result(zone)

    try:
        clf, reg, metadata = load_models_for_zone(zone)
    except FileNotFoundError:
        return build_no_model_result(zone)
    except Exception as e:
        return build_error_result(zone, f"Error carregant el model: {str(e)}")

    try:
        df = load_prediction_data(zone=zone, lookback="-30d")
    except Exception as e:
        return build_error_result(zone, f"Error carregant dades de predicció: {str(e)}")

    if df.empty:
        return build_error_result(zone, "No hi ha dades de predicció disponibles")

    try:
        X_predict, df_enriched = build_prediction_dataset(df, metadata)
    except Exception as e:
        return build_error_result(zone, f"Error construint el dataset de predicció: {str(e)}")

    if X_predict.empty:
        return build_error_result(zone, "No hi ha features suficients per fer la predicció")

    row = df_enriched.iloc[-1]

    if not row_has_valid_context_data(row):
        return build_error_result(zone, "Falten dades de context (weather/forecast) per a la predicció")

    try:
        proba = float(safe_predict_proba(clf, X_predict)[-1])
    except Exception as e:
        return build_error_result(zone, f"Error calculant la probabilitat: {str(e)}")

    threshold = float(metadata.get("classification_threshold", DEFAULT_THRESHOLD))
    should_water = proba >= threshold

    predicted_seconds = 0.0
    if should_water:
        if reg is not None:
            try:
                reg_pred = float(reg.predict(X_predict)[-1])
                predicted_seconds = clamp_seconds(reg_pred)
            except Exception:
                predicted_seconds = fallback_seconds_from_row(row)
        else:
            predicted_seconds = fallback_seconds_from_row(row)

    reasons = []

    drying = pd.to_numeric(pd.Series([row.get("forecast_drying_factor")]), errors="coerce").iloc[0]
    temp = pd.to_numeric(pd.Series([row.get("temperature")]), errors="coerce").iloc[0]
    precip = pd.to_numeric(
        pd.Series([row.get("forecast_precipitation_probability")]), errors="coerce"
    ).iloc[0]
    raining = pd.to_numeric(pd.Series([row.get("weather_is_raining_last")]), errors="coerce").iloc[0]

    if not pd.isna(drying):
        reasons.append(f"drying_factor={float(drying):.2f}")
    if not pd.isna(temp):
        reasons.append(f"temperature={float(temp):.1f}")
    if not pd.isna(precip):
        reasons.append(f"forecast_precipitation_probability={float(precip):.1f}")
    if not pd.isna(raining):
        reasons.append(f"weather_is_raining_last={int(raining)}")

    decision_reason = f"Predicció per franja intermèdia (p={proba:.2f}, threshold={threshold:.2f})"
    if reasons:
        decision_reason += " | " + ", ".join(reasons)

    return {
        "zone": zone,
        "should_water": bool(should_water),
        "decision_reason": decision_reason,
        "predicted_seconds": float(predicted_seconds),
        "probability": proba,
        "used_model": True,
    }


def parse_args():
    parser = argparse.ArgumentParser()
    parser.add_argument("--zone", help="Prediu només una zona concreta")
    parser.add_argument("--json", action="store_true", help="Força eixida JSON")
    return parser.parse_args()


def main():
    args = parse_args()

    if args.zone:
        zones = [args.zone]
    else:
        zones = load_available_zones(start="-180d")

    if not zones:
        print("[]")
        return

    results = [predict_zone(zone) for zone in zones]

    # Només zones que realment s'han de regar
    watering_results = [result for result in results if result.get("should_water") is True]

    print(json.dumps(watering_results, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()