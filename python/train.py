from sklearn.dummy import DummyClassifier
from sklearn.ensemble import RandomForestClassifier, RandomForestRegressor

from data_loader import load_available_zones, load_training_data
from features import build_training_dataset
from model_io import save_models_for_zone


def main():
    zones = load_available_zones(start="-180d")

    if not zones:
        print("No s'han trobat zones disponibles")
        return

    print("Zones trobades:", zones)

    for zone in zones:
        print(f"\n=== Entrenant zona: {zone} ===")

        df = load_training_data(zone=zone, start="-90d")

        if df.empty:
            print(f"Sense dades per a la zona {zone}, la salto")
            continue

        data = build_training_dataset(df)

        print("X_classifier shape:", data["X_classifier"].shape)
        print("y_classifier counts:")
        print(data["y_classifier"].value_counts(dropna=False))
        print("Feature columns:", data["metadata"]["feature_columns"])

        if data["X_classifier"].empty or data["y_classifier"].empty:
            print(f"Sense dades útils per classifier a {zone}, la salto")
            continue

        unique_classes = sorted(data["y_classifier"].dropna().unique().tolist())

        metadata = dict(data["metadata"])
        metadata["classifier_classes"] = unique_classes
        metadata["samples_classifier"] = int(len(data["X_classifier"]))
        metadata["samples_regressor"] = int(len(data["X_regressor"]))

        if len(unique_classes) < 2:
            print(f"[WARN] Zona {zone}: només hi ha una classe a y_classifier: {unique_classes}")
            print("[WARN] Es guardarà un DummyClassifier constant.")

            constant_class = int(unique_classes[0])
            clf = DummyClassifier(strategy="constant", constant=constant_class)
            clf.fit(data["X_classifier"], data["y_classifier"])
        else:
            clf = RandomForestClassifier(
                n_estimators=100,
                random_state=42,
                class_weight="balanced",
            )
            clf.fit(data["X_classifier"], data["y_classifier"])

        reg = None
        if not data["X_regressor"].empty and not data["y_regressor"].empty:
            reg = RandomForestRegressor(
                n_estimators=100,
                random_state=42,
            )
            reg.fit(data["X_regressor"], data["y_regressor"])
        else:
            print(f"Sense dades suficients per al regressor a {zone}")

        save_models_for_zone(zone, clf, reg, metadata)
        print(f"Models guardats per a {zone}")


if __name__ == "__main__":
    main()