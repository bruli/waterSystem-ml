import json
import os
import re

import joblib


MODEL_DIR = os.getenv("MODEL_DIR", "./models")


def slugify_zone(zone: str) -> str:
    value = zone.strip().lower()
    value = value.replace(" ", "_")
    value = re.sub(r"[^a-z0-9_]+", "", value)
    return value


def save_models_for_zone(zone: str, clf, reg, metadata: dict) -> None:
    zone_slug = slugify_zone(zone)
    zone_dir = os.path.join(MODEL_DIR, zone_slug)
    os.makedirs(zone_dir, exist_ok=True)

    joblib.dump(clf, os.path.join(zone_dir, "classifier.joblib"))

    if reg is not None:
        joblib.dump(reg, os.path.join(zone_dir, "regressor.joblib"))

    metadata_to_save = dict(metadata)
    metadata_to_save["zone"] = zone

    with open(os.path.join(zone_dir, "metadata.json"), "w", encoding="utf-8") as f:
        json.dump(metadata_to_save, f, indent=2)


def load_models_for_zone(zone: str):
    zone_slug = slugify_zone(zone)
    zone_dir = os.path.join(MODEL_DIR, zone_slug)

    clf = joblib.load(os.path.join(zone_dir, "classifier.joblib"))

    reg_path = os.path.join(zone_dir, "regressor.joblib")
    reg = joblib.load(reg_path) if os.path.exists(reg_path) else None

    with open(os.path.join(zone_dir, "metadata.json"), encoding="utf-8") as f:
        metadata = json.load(f)

    return clf, reg, metadata