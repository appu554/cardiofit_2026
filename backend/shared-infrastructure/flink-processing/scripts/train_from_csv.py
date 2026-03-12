#!/usr/bin/env python3
"""
Train MIMIC-IV Models from Exported CSV Files

This script trains XGBoost models on MIMIC-IV data that was exported
from BigQuery console (no authentication needed).

Usage:
    python3 scripts/train_from_csv.py
"""

import os
import sys
from pathlib import Path
import pandas as pd
import numpy as np
import xgboost as xgb
from sklearn.model_selection import train_test_split
from sklearn.metrics import roc_auc_score, classification_report, confusion_matrix
from sklearn.preprocessing import StandardScaler
import onnx
from skl2onnx import convert_sklearn
from skl2onnx.common.data_types import FloatTensorType
from onnxmltools.convert import convert_xgboost
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime

# Add script directory to path
sys.path.append(str(Path(__file__).parent))
from mimic_iv_config import OUTPUT_DIRS, MODEL_OUTPUTS, TRAINING_CONFIG


class MIMICModelTrainer:
    """Train XGBoost models from exported MIMIC-IV CSV files."""

    def __init__(self, model_name: str, csv_file: str, label_col: str):
        """
        Initialize trainer.

        Args:
            model_name: Name of model (sepsis, deterioration, mortality, readmission)
            csv_file: Path to CSV file with features
            label_col: Name of label column
        """
        self.model_name = model_name
        self.csv_file = csv_file
        self.label_col = label_col
        self.model = None
        self.scaler = StandardScaler()

        print(f"\n{'='*70}")
        print(f"  Training {model_name.upper()} Model")
        print(f"{'='*70}\n")

    def load_and_prepare_data(self):
        """Load CSV and prepare features."""
        print(f"📂 Loading data from: {self.csv_file}")

        # Load CSV
        df = pd.read_csv(self.csv_file)
        print(f"   Loaded {len(df):,} samples")

        # Separate features and labels
        self.y = df[self.label_col].values

        # Select feature columns (exclude IDs and labels)
        exclude_cols = ['subject_id', 'hadm_id', 'stay_id', 'icu_intime', 'icu_outtime',
                        self.label_col, 'died_in_hospital', 'admission_type']
        feature_cols = [c for c in df.columns if c not in exclude_cols]

        self.X = df[feature_cols].values
        self.feature_names = feature_cols

        # Handle missing values (fill with median)
        self.X = pd.DataFrame(self.X, columns=self.feature_names).fillna(
            pd.DataFrame(self.X, columns=self.feature_names).median()
        ).values

        print(f"   Features: {len(self.feature_names)}")
        print(f"   Positive samples: {self.y.sum():,} ({100*self.y.mean():.1f}%)")

        # Split data
        self.X_train, self.X_temp, self.y_train, self.y_temp = train_test_split(
            self.X, self.y,
            test_size=0.30,
            stratify=self.y,
            random_state=42
        )

        self.X_val, self.X_test, self.y_val, self.y_test = train_test_split(
            self.X_temp, self.y_temp,
            test_size=0.50,
            stratify=self.y_temp,
            random_state=42
        )

        print(f"   Train: {len(self.X_train):,} samples")
        print(f"   Val:   {len(self.X_val):,} samples")
        print(f"   Test:  {len(self.X_test):,} samples")

        # Normalize features
        self.X_train = self.scaler.fit_transform(self.X_train)
        self.X_val = self.scaler.transform(self.X_val)
        self.X_test = self.scaler.transform(self.X_test)

        return True

    def train_model(self):
        """Train XGBoost model."""
        print("\n🎯 Training XGBoost model...")

        # Calculate scale_pos_weight for class imbalance
        pos_weight = (self.y_train == 0).sum() / (self.y_train == 1).sum()

        # XGBoost configuration
        config = TRAINING_CONFIG['xgboost'].copy()
        config['scale_pos_weight'] = pos_weight

        self.model = xgb.XGBClassifier(**config)

        # Train with early stopping
        self.model.fit(
            self.X_train, self.y_train,
            eval_set=[(self.X_train, self.y_train), (self.X_val, self.y_val)],
            early_stopping_rounds=10,
            verbose=False
        )

        print(f"✅ Training complete")
        print(f"   Best iteration: {self.model.best_iteration}")

        return True

    def evaluate_model(self):
        """Evaluate model performance."""
        print("\n📊 Evaluating model...")

        # Predictions
        y_train_pred_proba = self.model.predict_proba(self.X_train)[:, 1]
        y_val_pred_proba = self.model.predict_proba(self.X_val)[:, 1]
        y_test_pred_proba = self.model.predict_proba(self.X_test)[:, 1]

        # AUROC scores
        train_auroc = roc_auc_score(self.y_train, y_train_pred_proba)
        val_auroc = roc_auc_score(self.y_val, y_val_pred_proba)
        test_auroc = roc_auc_score(self.y_test, y_test_pred_proba)

        print(f"   Train AUROC: {train_auroc:.4f}")
        print(f"   Val AUROC:   {val_auroc:.4f}")
        print(f"   Test AUROC:  {test_auroc:.4f}")

        # Test set predictions (binary)
        y_test_pred = (y_test_pred_proba >= 0.5).astype(int)

        # Classification report
        report = classification_report(self.y_test, y_test_pred, output_dict=True)

        sensitivity = report['1']['recall']  # True positive rate
        specificity = report['0']['recall']  # True negative rate

        print(f"   Test Sensitivity: {sensitivity:.4f}")
        print(f"   Test Specificity: {specificity:.4f}")

        # Feature importance
        feature_importance = pd.DataFrame({
            'feature': self.feature_names,
            'importance': self.model.feature_importances_
        }).sort_values('importance', ascending=False)

        print(f"\n   Top 10 Important Features:")
        for idx, row in feature_importance.head(10).iterrows():
            print(f"      {row['feature']}: {row['importance']:.4f}")

        return {
            'train_auroc': train_auroc,
            'val_auroc': val_auroc,
            'test_auroc': test_auroc,
            'test_sensitivity': sensitivity,
            'test_specificity': specificity,
            'feature_importance': feature_importance
        }

    def export_to_onnx(self, results):
        """Export model to ONNX format."""
        print(f"\n💾 Exporting to ONNX...")

        # Convert to ONNX
        n_features = len(self.feature_names)
        initial_types = [('float_input', FloatTensorType([None, n_features]))]

        onnx_model = convert_xgboost(
            self.model,
            initial_types=initial_types,
            target_opset=12
        )

        # Add metadata
        onnx_model.producer_name = "CardioFit-MIMIC-IV"
        onnx_model.producer_version = "2.0.0"

        # Add training metadata
        metadata = {
            'is_mock_model': 'false',
            'training_data': 'MIMIC-IV v3.1',
            'model_type': self.model_name,
            'train_samples': str(len(self.X_train)),
            'test_auroc': f"{results['test_auroc']:.4f}",
            'test_sensitivity': f"{results['test_sensitivity']:.4f}",
            'test_specificity': f"{results['test_specificity']:.4f}",
            'created_date': datetime.now().isoformat(),
            'n_features': str(n_features)
        }

        for key, value in metadata.items():
            meta = onnx_model.metadata_props.add()
            meta.key = key
            meta.value = value

        # Save ONNX model
        output_path = MODEL_OUTPUTS[self.model_name]
        with open(output_path, 'wb') as f:
            f.write(onnx_model.SerializeToString())

        print(f"✅ ONNX model saved: {output_path}")

        # Verify ONNX model
        onnx_model = onnx.load(output_path)
        onnx.checker.check_model(onnx_model)
        print(f"✅ ONNX model validated")

        return output_path


def main():
    """Main training pipeline."""
    print("\n" + "="*70)
    print("  MIMIC-IV Model Training from Exported CSV Files")
    print("="*70)

    # Check if CSV files exist
    data_dir = Path(OUTPUT_DIRS['data'])
    csv_files = {
        'sepsis': data_dir / 'sepsis_cohort_with_features.csv',
        'deterioration': data_dir / 'deterioration_cohort_with_features.csv',
        'mortality': data_dir / 'mortality_cohort_with_features.csv',
        'readmission': data_dir / 'readmission_cohort_with_features.csv'
    }

    label_columns = {
        'sepsis': 'sepsis_label',
        'deterioration': 'deterioration_label',
        'mortality': 'mortality_label',
        'readmission': 'readmission_label'
    }

    # Check which files exist
    missing_files = []
    for model_name, csv_path in csv_files.items():
        if not csv_path.exists():
            missing_files.append(str(csv_path))

    if missing_files:
        print("\n❌ Missing CSV files:")
        for f in missing_files:
            print(f"   {f}")
        print("\nPlease export data from BigQuery first using:")
        print("   BIGQUERY_EXPORT_QUERIES.sql")
        return 1

    # Train each model
    results_summary = {}

    for model_name in ['sepsis', 'deterioration', 'mortality', 'readmission']:
        try:
            trainer = MIMICModelTrainer(
                model_name=model_name,
                csv_file=str(csv_files[model_name]),
                label_col=label_columns[model_name]
            )

            # Train pipeline
            trainer.load_and_prepare_data()
            trainer.train_model()
            results = trainer.evaluate_model()
            trainer.export_to_onnx(results)

            results_summary[model_name] = results

        except Exception as e:
            print(f"\n❌ Error training {model_name} model: {e}")
            import traceback
            traceback.print_exc()
            continue

    # Print final summary
    print("\n" + "="*70)
    print("  🎉 TRAINING COMPLETE!")
    print("="*70)

    for model_name, results in results_summary.items():
        print(f"\n{model_name.upper()} Model:")
        print(f"   AUROC:       {results['test_auroc']:.4f}")
        print(f"   Sensitivity: {results['test_sensitivity']:.4f}")
        print(f"   Specificity: {results['test_specificity']:.4f}")

    print("\n📁 Models saved to:")
    for model_name in results_summary.keys():
        print(f"   {MODEL_OUTPUTS[model_name]}")

    print("\n🎯 Next: Test the real models!")
    print("   mvn test -Dtest=CustomPatientMLTest")

    return 0


if __name__ == "__main__":
    exit(main())
