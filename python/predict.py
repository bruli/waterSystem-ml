import json

from data_loader import load_available_zones, load_prediction_data
from features import build_prediction_dataset
from model_io import load_models_for_zone


def build_decision_reason(row, threshold: float) -> str:
    if row["weather_is_raining_last"] == 1:
        return "raining_now_or_recently"

    if row["forecast_precipitation_probability"] >= 70:
        return "high_rain_probability"

    if row["days_since_last_watering"] < 0.10:
        return "too_soon_since_last_watering"

    if row["watering_proba"] < threshold:
        return "low_model_probability"

    if row["predicted_seconds"] <= 0:
        return "model_predicted_no_watering"

    return "model_predicted_watering"


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

        df = load_prediction_data(zone=zone, lookback="-30d")

        if df.empty:
            print(f"Sense dades de predicció per a la zona {zone}, la salto")
            continue

        X_predict, df_predict = build_prediction_dataset(df, metadata)

        if X_predict.empty:
            print(f"Sense features per a la zona {zone}, la salto")
            continue

        df_predict = df_predict.copy()

        threshold = float(metadata.get("classification_threshold", 0.3))

        df_predict["watering_proba"] = clf.predict_proba(X_predict)[:, 1]
        df_predict["should_water"] = df_predict["watering_proba"] >= threshold
        df_predict["predicted_seconds"] = 0.0

        mask = df_predict["should_water"]

        if reg is not None and mask.any():
            df_predict.loc[mask, "predicted_seconds"] = reg.predict(X_predict[mask])

        df_predict["predicted_seconds"] = df_predict["predicted_seconds"].clip(lower=0.0)

        # Regles de seguretat
        df_predict.loc[df_predict["weather_is_raining_last"] == 1, "predicted_seconds"] = 0.0
        df_predict.loc[df_predict["forecast_precipitation_probability"] >= 70, "predicted_seconds"] = 0.0
        df_predict.loc[df_predict["days_since_last_watering"] < 0.10, "predicted_seconds"] = 0.0
        df_predict.loc[df_predict["predicted_seconds"] < 3.0, "predicted_seconds"] = 0.0

        df_predict["should_water"] = df_predict["predicted_seconds"] > 0
        df_predict["predicted_seconds"] = df_predict["predicted_seconds"].round(1)

        df_predict["decision_reason"] = df_predict.apply(
            lambda row: build_decision_reason(row, threshold),
            axis=1,
        )

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
            "should_water",
            "predicted_seconds",
            "decision_reason",
        ]

        for col in output_columns:
            if col not in df_predict.columns:
                df_predict[col] = None

        records = df_predict[output_columns].copy().to_dict(orient="records")
        results.extend(records)

    print(json.dumps(results, default=str, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()