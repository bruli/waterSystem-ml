import json
import sys

from data_loader import load_available_zones, load_prediction_data
from features import build_prediction_dataset
from model_io import load_models_for_zone


MIN_DAYS_BETWEEN_WATERING = 1.0
MAX_FORECAST_PRECIPITATION_PROBABILITY = 70.0
MIN_SECONDS_TO_WATER = 3.0


def should_water_by_rules(row) -> tuple[bool, str]:
    if row["weather_is_raining_last"] == 1:
        return False, "raining_now"

    if row["forecast_precipitation_probability"] >= MAX_FORECAST_PRECIPITATION_PROBABILITY:
        return False, "high_rain_probability"

    if row["days_since_last_watering"] < MIN_DAYS_BETWEEN_WATERING:
        return False, "too_soon_since_last_watering"

    return True, "ok"


def classifier_probability(clf, X):
    classes = list(clf.classes_)

    if len(classes) == 1:
        only_class = classes[0]
        return 1.0 if only_class == 1 else 0.0

    positive_class_index = classes.index(1)
    return float(clf.predict_proba(X)[0, positive_class_index])


def main():
    zones = load_available_zones(start="-180d")

    if not zones:
        print("No s'han trobat zones disponibles")
        return

    results = []

    for zone in zones:
        print(f"Predint zona: {zone}", file=sys.stderr)

        try:
            clf, reg, metadata = load_models_for_zone(zone)
        except FileNotFoundError:
            print(f"No hi ha models guardats per a {zone}, la salto")
            continue

        try:
            df = load_prediction_data(zone=zone, lookback="-30d")
        except ValueError as e:
            print(f"Error carregant dades per a {zone}: {e}")
            continue

        X, df = build_prediction_dataset(df, metadata)

        row = df.iloc[0]

        # classifier només com a informació auxiliar
        watering_proba = classifier_probability(clf, X)

        allowed, reason = should_water_by_rules(row)

        predicted_seconds = 0.0
        if allowed and reg is not None:
            predicted_seconds = float(reg.predict(X)[0])

        if predicted_seconds < MIN_SECONDS_TO_WATER:
            predicted_seconds = 0.0
            if reason == "ok":
                reason = "below_min_seconds"

        result = {
            "time": str(row["_time"]),
            "zone": row["zone"],
            "temperature": float(row["temperature"]) if row["temperature"] is not None else None,
            "weather_is_raining_last": int(row["weather_is_raining_last"]),
            "forecast_precipitation_probability": float(row["forecast_precipitation_probability"]),
            "days_since_last_watering": float(row["days_since_last_watering"]),
            "watering_proba": round(watering_proba, 4),
            "should_water": predicted_seconds > 0,
            "predicted_seconds": round(predicted_seconds, 1),
            "decision_reason": reason,
        }

        results.append(result)

    print(json.dumps(results, indent=2, default=str))


if __name__ == "__main__":
    main()