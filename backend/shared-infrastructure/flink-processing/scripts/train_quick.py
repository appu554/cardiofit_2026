#!/usr/bin/env python3
"""Quick MIMIC-IV model training - simplified version"""

import pandas as pd
import xgboost as xgb
from sklearn.model_selection import train_test_split
from sklearn.metrics import roc_auc_score, classification_report
import onnx
from onnxmltools.convert import convert_xgboost
from onnxmltools.convert.common.data_types import FloatTensorType
from datetime import datetime
import os

# Output directories
os.makedirs("models", exist_ok=True)
os.makedirs("results/mimic_iv", exist_ok=True)

def train_model(cohort_name, label_col):
    print(f"\n{'='*70}")
    print(f"  Training {cohort_name.upper()} Model")
    print(f"{'='*70}\n")

    # Load data
    csv_file = f"data/mimic_iv/{cohort_name}_cohort_with_features.csv"
    print(f"📂 Loading: {csv_file}")
    df = pd.read_csv(csv_file)
    print(f"   Loaded {len(df):,} samples")

    # Prepare features and labels
    y = df[label_col].values
    exclude_cols = ['stay_id', label_col]
    feature_cols = [c for c in df.columns if c not in exclude_cols]
    X = df[feature_cols].fillna(0).values  # Simple imputation

    print(f"   Features: {len(feature_cols)}")
    print(f"   Positive samples: {y.sum():,} ({100*y.mean():.1f}%)")

    # Split data
    X_train, X_temp, y_train, y_temp = train_test_split(
        X, y, test_size=0.30, random_state=42, stratify=y if y.mean() < 1.0 else None
    )
    X_val, X_test, y_val, y_test = train_test_split(
        X_temp, y_temp, test_size=0.50, random_state=42, stratify=y_temp if y_temp.mean() < 1.0 else None
    )

    print(f"   Train: {len(X_train):,}")
    print(f"   Val:   {len(X_val):,}")
    print(f"   Test:  {len(X_test):,}")

    # Train XGBoost
    print("\n🎯 Training XGBoost...")
    model = xgb.XGBClassifier(
        n_estimators=100,
        max_depth=6,
        learning_rate=0.1,
        random_state=42,
        n_jobs=-1
    )

    model.fit(
        X_train, y_train,
        eval_set=[(X_val, y_val)],
        verbose=False
    )

    print("✅ Training complete")

    # Evaluate
    print("\n📊 Evaluating...")
    y_test_pred_proba = model.predict_proba(X_test)[:, 1]
    y_test_pred = (y_test_pred_proba >= 0.5).astype(int)

    auroc = roc_auc_score(y_test, y_test_pred_proba)
    print(f"   Test AUROC: {auroc:.4f}")

    report = classification_report(y_test, y_test_pred, output_dict=True, zero_division=0)
    if '1' in report:
        sensitivity = report['1']['recall']
        print(f"   Test Sensitivity: {sensitivity:.4f}")
    if '0' in report:
        specificity = report['0']['recall']
        print(f"   Test Specificity: {specificity:.4f}")

    # Export to ONNX
    print("\n💾 Exporting to ONNX...")
    initial_types = [('float_input', FloatTensorType([None, len(feature_cols)]))]
    onnx_model = convert_xgboost(model, initial_types=initial_types, target_opset=12)

    # Add metadata
    onnx_model.producer_name = "CardioFit-MIMIC-IV"
    onnx_model.producer_version = "2.0.0"

    meta = onnx_model.metadata_props.add()
    meta.key = "is_mock_model"
    meta.value = "false"

    meta = onnx_model.metadata_props.add()
    meta.key = "training_data"
    meta.value = "MIMIC-IV v3.1"

    meta = onnx_model.metadata_props.add()
    meta.key = "test_auroc"
    meta.value = f"{auroc:.4f}"

    meta = onnx_model.metadata_props.add()
    meta.key = "created_date"
    meta.value = datetime.now().isoformat()

    # Save
    output_path = f"models/{cohort_name}_risk_v2.0.0_mimic.onnx"
    with open(output_path, 'wb') as f:
        f.write(onnx_model.SerializeToString())

    print(f"✅ Saved: {output_path}")

    # Verify
    onnx_model_check = onnx.load(output_path)
    onnx.checker.check_model(onnx_model_check)
    print("✅ ONNX model validated")

    return auroc

# Train all 3 models
print("\n" + "="*70)
print("  MIMIC-IV MODEL TRAINING")
print("="*70)

results = {}
for cohort_name, label_col in [
    ('sepsis', 'sepsis_label'),
    ('deterioration', 'deterioration_label'),
    ('mortality', 'mortality_label')
]:
    try:
        auroc = train_model(cohort_name, label_col)
        results[cohort_name] = auroc
    except Exception as e:
        print(f"\n❌ Error training {cohort_name}: {e}")
        import traceback
        traceback.print_exc()

# Summary
print("\n" + "="*70)
print("  🎉 TRAINING COMPLETE!")
print("="*70)
for name, auroc in results.items():
    print(f"{name.upper()}: AUROC = {auroc:.4f}")

print("\n📁 Models saved in models/")
print("✅ Ready to replace mock models!")
