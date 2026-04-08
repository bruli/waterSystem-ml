import json
import os
import re
import warnings

import joblib
from sklearn.exceptions import InconsistentVersionWarning


MODEL_DIR = os.getenv("MODEL_DIR", "./models")


def slugify_zone(zone: str) -> str:
    value = zone.strip().lower()
    value = value.replace(" ", "_")
    value = re.sub(r"[^a-z0-9_]+", "", value)
    return value


def _safe_joblib_load(path: str):
    with warnings.catch_warnings():
        warnings.filterwarnings("error", category=InconsistentVersionWarning)
        try:
            return joblib.load(path)
        except InconsistentVersionWarning as e:
            raise RuntimeError(
                f"El model '{path}' es va guardar amb una altra versió de scikit-learn. "
                f"Esborra els models antics i torna a entrenar."
            ) from e


def save_models_for_zone(zone: str, clf, reg, metadata: dict) -> None:
    zone_slug = slugify_zone(zone)
    zone_dir = os.path.join(MODEL_DIR, zone_slug)
    os.makedirs(zone_dir, exist_ok=True)

    classifier_path = os.path.join(zone_dir, "classifier.joblib")
    regressor_path = os.path.join(zone_dir, "regressor.joblib")
    metadata_path = os.path.join(zone_dir, "metadata.json")

    joblib.dump(clf, classifier_path)

    if reg is not None:
        joblib.dump(reg, regressor_path)
    else:
        if os.path.exists(regressor_path):
            os.remove(regressor_path)

    metadata_to_save = dict(metadata)
    metadata_to_save["zone"] = zone

    with open(metadata_path, "w", encoding="utf-8") as f:
        json.dump(metadata_to_save, f, indent=2)


def load_models_for_zone(zone: str):
    zone_slug = slugify_zone(zone)
    zone_dir = os.path.join(MODEL_DIR, zone_slug)

    classifier_path = os.path.join(zone_dir, "classifier.joblib")
    regressor_path = os.path.join(zone_dir, "regressor.joblib")
    metadata_path = os.path.join(zone_dir, "metadata.json")

    clf = _safe_joblib_load(classifier_path)

    reg = _safe_joblib_load(regressor_path) if os.path.exists(regressor_path) else None

    with open(metadata_path, encoding="utf-8") as f:
        metadata = json.load(f)

    return clf, reg, metadata