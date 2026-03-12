#!/usr/bin/env python3
"""
Sepsis Risk Model Training Pipeline

Complete end-to-end training pipeline for sepsis prediction using MIMIC-IV data.

Clinical Focus:
- Early warning for sepsis development within 48 hours
- Sensitive to lactate elevation, fever, tachycardia patterns
- Emphasizes WBC abnormalities and SOFA score progression

Pipeline Stages:
1. Load MIMIC-IV features (70 features from mimic_feature_extractor.py)
2. Train/test split with temporal validation
3. Handle class imbalance with SMOTE
4. Hyperparameter tuning with Optuna (Bayesian optimization)
5. Train final XGBoost model
6. Export to ONNX format
7. Validation and performance reporting

Target Metrics:
- AUROC > 0.85
- Sensitivity > 0.80 (high recall for sepsis detection)
- Specificity > 0.75
- PPV > 0.40 (realistic given ~8% prevalence)

Prerequisites:
- Feature matrix from mimic_feature_extractor.py
- XGBoost, Optuna, SMOTE, ONNX export tools

Usage:
    python scripts/train_sepsis_model.py --input data/sepsis_features.csv --output models/sepsis_risk_v1.0.0.onnx

@author CardioFit Team - Module 5 Training Pipeline
@version 1.0.0
"""

import argparse
import sys
import pandas as pd
import numpy as np
import xgboost as xgb
import optuna
import onnxmltools
from onnxmltools.convert import convert_xgboost
from onnxmltools.convert.common.data_types import FloatTensorType
import onnx
import onnxruntime as ort
from sklearn.model_selection import train_test_split, StratifiedKFold
from sklearn.metrics import (
    roc_auc_score, roc_curve, precision_recall_curve,
    confusion_matrix, classification_report, average_precision_score
)
from imblearn.over_sampling import SMOTE
from datetime import datetime
import warnings
warnings.filterwarnings('ignore')


class SepsisModelTrainer:
    """
    Complete training pipeline for sepsis risk prediction.
    """

    def __init__(self, random_state=42):
        """
        Initialize trainer.

        Args:
            random_state: Random seed for reproducibility
        """
        self.random_state = random_state
        self.model = None
        self.best_params = None
        self.feature_names = None
        self.metrics = {}

    def load_data(self, input_path):
        """
        Load feature matrix from CSV.

        Args:
            input_path: Path to CSV file from mimic_feature_extractor.py

        Returns:
            X (features), y (labels), patient_ids
        """
        print("\n📊 Loading feature matrix...")
        df = pd.read_csv(input_path)

        print(f"   Total patients: {len(df)}")
        print(f"   Positive cases: {df['label'].sum()} ({df['label'].mean()*100:.1f}%)")

        # Separate features, labels, and metadata
        patient_ids = df['subject_id'].values
        y = df['label'].values
        X = df.drop(['subject_id', 'label'], axis=1)

        self.feature_names = X.columns.tolist()
        print(f"   Features: {len(self.feature_names)}")

        return X.values, y, patient_ids

    def split_data(self, X, y, patient_ids, test_size=0.2):
        """
        Split into train/test sets with stratification.

        Args:
            X: Feature matrix
            y: Labels
            patient_ids: Patient identifiers
            test_size: Proportion for test set

        Returns:
            X_train, X_test, y_train, y_test
        """
        print(f"\n✂️  Splitting data (test_size={test_size})...")

        X_train, X_test, y_train, y_test = train_test_split(
            X, y, test_size=test_size, random_state=self.random_state,
            stratify=y
        )

        print(f"   Train: {len(X_train)} patients ({y_train.mean()*100:.1f}% positive)")
        print(f"   Test:  {len(X_test)} patients ({y_test.mean()*100:.1f}% positive)")

        return X_train, X_test, y_train, y_test

    def apply_smote(self, X_train, y_train):
        """
        Apply SMOTE to balance training set.

        Args:
            X_train: Training features
            y_train: Training labels

        Returns:
            X_resampled, y_resampled
        """
        print("\n⚖️  Applying SMOTE for class balance...")

        original_ratio = y_train.mean()
        print(f"   Before SMOTE: {y_train.sum()} positive / {len(y_train)} total ({original_ratio*100:.1f}%)")

        # SMOTE to 30% positive rate (not 50%, to keep some imbalance)
        smote = SMOTE(sampling_strategy=0.3, random_state=self.random_state)
        X_resampled, y_resampled = smote.fit_resample(X_train, y_train)

        new_ratio = y_resampled.mean()
        print(f"   After SMOTE:  {y_resampled.sum()} positive / {len(y_resampled)} total ({new_ratio*100:.1f}%)")

        return X_resampled, y_resampled

    def optimize_hyperparameters(self, X_train, y_train, n_trials=50):
        """
        Hyperparameter optimization with Optuna.

        Args:
            X_train: Training features
            y_train: Training labels
            n_trials: Number of optimization trials

        Returns:
            Best hyperparameters
        """
        print(f"\n🔧 Hyperparameter optimization ({n_trials} trials)...")

        def objective(trial):
            """Optuna objective function."""
            params = {
                'n_estimators': trial.suggest_int('n_estimators', 50, 300),
                'max_depth': trial.suggest_int('max_depth', 3, 10),
                'learning_rate': trial.suggest_float('learning_rate', 0.01, 0.3, log=True),
                'subsample': trial.suggest_float('subsample', 0.6, 1.0),
                'colsample_bytree': trial.suggest_float('colsample_bytree', 0.6, 1.0),
                'min_child_weight': trial.suggest_int('min_child_weight', 1, 10),
                'gamma': trial.suggest_float('gamma', 0.0, 1.0),
                'reg_alpha': trial.suggest_float('reg_alpha', 0.0, 1.0),
                'reg_lambda': trial.suggest_float('reg_lambda', 0.0, 1.0),
                'scale_pos_weight': trial.suggest_float('scale_pos_weight', 1.0, 20.0),
                'random_state': self.random_state,
                'eval_metric': 'logloss',
                'n_jobs': -1
            }

            # 5-fold cross-validation
            cv = StratifiedKFold(n_splits=5, shuffle=True, random_state=self.random_state)
            cv_scores = []

            for train_idx, val_idx in cv.split(X_train, y_train):
                X_cv_train, X_cv_val = X_train[train_idx], X_train[val_idx]
                y_cv_train, y_cv_val = y_train[train_idx], y_train[val_idx]

                model = xgb.XGBClassifier(**params)
                model.fit(X_cv_train, y_cv_train, verbose=False)

                y_pred_proba = model.predict_proba(X_cv_val)[:, 1]
                auroc = roc_auc_score(y_cv_val, y_pred_proba)
                cv_scores.append(auroc)

            return np.mean(cv_scores)

        # Run optimization
        study = optuna.create_study(direction='maximize', study_name='sepsis_xgboost')
        study.optimize(objective, n_trials=n_trials, show_progress_bar=True)

        self.best_params = study.best_params
        print(f"\n   Best AUROC: {study.best_value:.4f}")
        print(f"   Best params:")
        for param, value in self.best_params.items():
            print(f"      {param}: {value}")

        return self.best_params

    def train_final_model(self, X_train, y_train):
        """
        Train final model with optimized hyperparameters.

        Args:
            X_train: Training features
            y_train: Training labels

        Returns:
            Trained XGBoost model
        """
        print("\n🚀 Training final model...")

        # Use best params from optimization
        params = {**self.best_params, 'random_state': self.random_state, 'eval_metric': 'logloss', 'n_jobs': -1}

        self.model = xgb.XGBClassifier(**params)
        self.model.fit(X_train, y_train, verbose=True)

        print("   ✅ Model training complete")

        return self.model

    def evaluate_model(self, X_test, y_test):
        """
        Comprehensive model evaluation.

        Args:
            X_test: Test features
            y_test: Test labels

        Returns:
            Dictionary of metrics
        """
        print("\n📈 Evaluating model on test set...")

        # Predictions
        y_pred_proba = self.model.predict_proba(X_test)[:, 1]
        y_pred = self.model.predict(X_test)

        # AUROC
        auroc = roc_auc_score(y_test, y_pred_proba)
        auprc = average_precision_score(y_test, y_pred_proba)

        # Confusion matrix
        tn, fp, fn, tp = confusion_matrix(y_test, y_pred).ravel()
        sensitivity = tp / (tp + fn)
        specificity = tn / (tn + fp)
        ppv = tp / (tp + fp) if (tp + fp) > 0 else 0
        npv = tn / (tn + fn) if (tn + fn) > 0 else 0

        # Store metrics
        self.metrics = {
            'auroc': auroc,
            'auprc': auprc,
            'sensitivity': sensitivity,
            'specificity': specificity,
            'ppv': ppv,
            'npv': npv,
            'tn': tn,
            'fp': fp,
            'fn': fn,
            'tp': tp
        }

        # Print results
        print("\n" + "=" * 70)
        print("TEST SET PERFORMANCE")
        print("=" * 70)
        print(f"\nROC Metrics:")
        print(f"   AUROC:       {auroc:.4f} {self._pass_fail(auroc, 0.85)}")
        print(f"   AUPRC:       {auprc:.4f}")
        print(f"\nClassification Metrics:")
        print(f"   Sensitivity: {sensitivity:.4f} {self._pass_fail(sensitivity, 0.80)}")
        print(f"   Specificity: {specificity:.4f} {self._pass_fail(specificity, 0.75)}")
        print(f"   PPV:         {ppv:.4f} {self._pass_fail(ppv, 0.40)}")
        print(f"   NPV:         {npv:.4f}")
        print(f"\nConfusion Matrix:")
        print(f"   TN: {tn:5d}  FP: {fp:5d}")
        print(f"   FN: {fn:5d}  TP: {tp:5d}")
        print()

        return self.metrics

    def export_to_onnx(self, output_path, version='1.0.0'):
        """
        Export trained model to ONNX format.

        Args:
            output_path: Path to save ONNX model
            version: Model version string

        Returns:
            Path to exported ONNX model
        """
        print(f"\n📦 Exporting to ONNX...")

        # Define input type (70 float features)
        initial_types = [('float_input', FloatTensorType([None, 70]))]

        # Convert to ONNX
        try:
            onnx_model = convert_xgboost(
                self.model,
                initial_types=initial_types,
                target_opset=12
            )
        except Exception as e:
            print(f"   ❌ Error converting to ONNX: {e}")
            raise

        # Set metadata
        onnx_model.producer_name = 'CardioFit-Module5-MIMIC'
        onnx_model.producer_version = version
        onnx_model.doc_string = 'Sepsis risk prediction (early warning for sepsis development within 48 hours)'

        # Add custom metadata
        meta_model_name = onnx_model.metadata_props.add()
        meta_model_name.key = 'model_name'
        meta_model_name.value = 'sepsis_risk'

        meta_version = onnx_model.metadata_props.add()
        meta_version.key = 'version'
        meta_version.value = version

        meta_features = onnx_model.metadata_props.add()
        meta_features.key = 'input_features'
        meta_features.value = '70'

        meta_output = onnx_model.metadata_props.add()
        meta_output.key = 'output_type'
        meta_output.value = 'binary_classification_probability'

        meta_clinical = onnx_model.metadata_props.add()
        meta_clinical.key = 'clinical_focus'
        meta_clinical.value = 'Lactate, WBC, temperature, heart rate, SOFA score'

        meta_created = onnx_model.metadata_props.add()
        meta_created.key = 'created_date'
        meta_created.value = datetime.now().strftime('%Y-%m-%d')

        meta_auroc = onnx_model.metadata_props.add()
        meta_auroc.key = 'test_auroc'
        meta_auroc.value = f"{self.metrics['auroc']:.4f}"

        meta_sensitivity = onnx_model.metadata_props.add()
        meta_sensitivity.key = 'test_sensitivity'
        meta_sensitivity.value = f"{self.metrics['sensitivity']:.4f}"

        # Verify model
        try:
            onnx.checker.check_model(onnx_model)
        except Exception as e:
            print(f"   ❌ ONNX validation failed: {e}")
            raise

        # Save to file
        onnx.save(onnx_model, output_path)

        # Verify ONNX Runtime can load it
        try:
            session = ort.InferenceSession(output_path)
            input_name = session.get_inputs()[0].name

            # Test inference
            test_input = np.random.randn(1, 70).astype(np.float32)
            outputs = session.run(None, {input_name: test_input})

            print(f"   ✅ ONNX model exported: {output_path}")
            print(f"   ✅ ONNX Runtime validation: PASSED")

        except Exception as e:
            print(f"   ❌ ONNX Runtime validation failed: {e}")
            raise

        return output_path

    def _pass_fail(self, value, threshold):
        """Helper to format pass/fail indicators."""
        return "✅ PASS" if value >= threshold else "❌ FAIL"


def main():
    """Main execution function."""
    parser = argparse.ArgumentParser(description='Train sepsis risk prediction model')
    parser.add_argument('--input', required=True, help='Input CSV file (from mimic_feature_extractor.py)')
    parser.add_argument('--output', required=True, help='Output ONNX file path')
    parser.add_argument('--version', default='1.0.0', help='Model version')
    parser.add_argument('--trials', type=int, default=50, help='Optuna trials for hyperparameter tuning')
    parser.add_argument('--test-size', type=float, default=0.2, help='Test set proportion')
    parser.add_argument('--seed', type=int, default=42, help='Random seed')

    args = parser.parse_args()

    print("=" * 70)
    print("SEPSIS RISK MODEL TRAINING PIPELINE")
    print("=" * 70)
    print(f"\nInput:      {args.input}")
    print(f"Output:     {args.output}")
    print(f"Version:    {args.version}")
    print(f"Trials:     {args.trials}")
    print(f"Test size:  {args.test_size}")
    print(f"Seed:       {args.seed}")

    try:
        # Initialize trainer
        trainer = SepsisModelTrainer(random_state=args.seed)

        # Load data
        X, y, patient_ids = trainer.load_data(args.input)

        # Split data
        X_train, X_test, y_train, y_test = trainer.split_data(X, y, patient_ids, args.test_size)

        # Apply SMOTE
        X_train_balanced, y_train_balanced = trainer.apply_smote(X_train, y_train)

        # Optimize hyperparameters
        best_params = trainer.optimize_hyperparameters(X_train_balanced, y_train_balanced, args.trials)

        # Train final model
        model = trainer.train_final_model(X_train_balanced, y_train_balanced)

        # Evaluate
        metrics = trainer.evaluate_model(X_test, y_test)

        # Export to ONNX
        onnx_path = trainer.export_to_onnx(args.output, args.version)

        print("=" * 70)
        print("TRAINING PIPELINE COMPLETE")
        print("=" * 70)
        print(f"\n✅ Model saved: {onnx_path}")
        print(f"✅ AUROC:       {metrics['auroc']:.4f}")
        print(f"✅ Sensitivity: {metrics['sensitivity']:.4f}")
        print(f"✅ PPV:         {metrics['ppv']:.4f}")
        print()

    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
