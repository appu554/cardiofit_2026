#!/usr/bin/env python3
"""
Generate 4 mock ONNX models for Module 5 infrastructure testing.

This script creates realistic clinical prediction models for:
- Sepsis risk
- Patient deterioration
- Mortality risk
- 30-day readmission

Models are trained on synthetic data but configured to behave like
production clinical models (70 features → binary classification).

Usage:
    python scripts/create_mock_onnx_models.py

Requirements:
    pip install xgboost scikit-learn onnx onnxruntime skl2onnx numpy

Output:
    models/sepsis_risk_v1.0.0.onnx
    models/deterioration_risk_v1.0.0.onnx
    models/mortality_risk_v1.0.0.onnx
    models/readmission_risk_v1.0.0.onnx
"""

import numpy as np
import xgboost as xgb
from sklearn.datasets import make_classification
import onnxmltools
from onnxmltools.convert import convert_xgboost
from onnxmltools.convert.common.data_types import FloatTensorType
import onnx
import onnxruntime as ort
from datetime import datetime
import os
import sys


class MockModelGenerator:
    """Generate realistic mock clinical prediction models."""

    def __init__(self, output_dir='models'):
        self.output_dir = output_dir
        os.makedirs(output_dir, exist_ok=True)

    def generate_sepsis_model(self):
        """
        Generate sepsis risk prediction model.

        Clinical focus:
        - High sensitivity to lactate elevation
        - Responds to fever + tachycardia patterns
        - Emphasizes WBC abnormalities
        - SOFA score progression
        """
        print("🔬 Generating Sepsis Risk Model...")

        # Generate synthetic clinical data (70 features, 5000 patients)
        X, y = make_classification(
            n_samples=5000,
            n_features=70,
            n_informative=25,  # 25 features truly predictive
            n_redundant=10,    # 10 redundant features (correlated)
            n_classes=2,
            weights=[0.92, 0.08],  # 8% sepsis rate (realistic)
            flip_y=0.03,  # 3% label noise (clinical reality)
            random_state=42
        )

        # Train XGBoost model with clinical-realistic hyperparameters
        model = xgb.XGBClassifier(
            n_estimators=100,       # 100 trees (balance accuracy/speed)
            max_depth=6,            # Depth 6 (prevents overfitting)
            learning_rate=0.1,      # Standard learning rate
            subsample=0.8,          # 80% row sampling
            colsample_bytree=0.8,   # 80% feature sampling
            scale_pos_weight=11.5,  # Balance 92:8 class ratio
            random_state=42,
            eval_metric='logloss',
            n_jobs=-1
        )

        model.fit(X, y, verbose=False)

        # Calculate performance metrics
        y_pred_proba = model.predict_proba(X)[:, 1]
        from sklearn.metrics import roc_auc_score
        train_auroc = roc_auc_score(y, y_pred_proba)

        # Export to ONNX
        output_path = self._export_to_onnx(
            model,
            model_name='sepsis_risk',
            version='1.0.0',
            description='Sepsis risk prediction (early warning for sepsis development within 48 hours)',
            clinical_focus='Lactate, WBC, temperature, heart rate, SOFA score',
            train_auroc=train_auroc
        )

        return output_path

    def generate_deterioration_model(self):
        """
        Generate patient deterioration prediction model.

        Clinical focus:
        - Vital sign trends (6-hour changes)
        - NEWS2 and qSOFA score progression
        - Unplanned ICU transfer risk
        - Respiratory distress indicators
        """
        print("📉 Generating Deterioration Risk Model...")

        X, y = make_classification(
            n_samples=5000,
            n_features=70,
            n_informative=20,
            n_redundant=15,
            n_classes=2,
            weights=[0.94, 0.06],  # 6% deterioration rate
            flip_y=0.02,
            random_state=43
        )

        model = xgb.XGBClassifier(
            n_estimators=100,
            max_depth=6,
            learning_rate=0.1,
            subsample=0.8,
            colsample_bytree=0.8,
            scale_pos_weight=15.7,  # Balance 94:6 ratio
            random_state=43,
            eval_metric='logloss',
            n_jobs=-1
        )

        model.fit(X, y, verbose=False)

        y_pred_proba = model.predict_proba(X)[:, 1]
        from sklearn.metrics import roc_auc_score
        train_auroc = roc_auc_score(y, y_pred_proba)

        output_path = self._export_to_onnx(
            model,
            model_name='deterioration_risk',
            version='1.0.0',
            description='Clinical deterioration risk (6-24 hour prediction window)',
            clinical_focus='Vital trends, NEWS2, respiratory rate, MAP, lactate',
            train_auroc=train_auroc
        )

        return output_path

    def generate_mortality_model(self):
        """
        Generate in-hospital mortality prediction model.

        Clinical focus:
        - Age and comorbidity burden
        - APACHE II score
        - Organ dysfunction markers
        - Severity of illness indicators
        """
        print("💀 Generating Mortality Risk Model...")

        X, y = make_classification(
            n_samples=5000,
            n_features=70,
            n_informative=18,
            n_redundant=12,
            n_classes=2,
            weights=[0.96, 0.04],  # 4% mortality rate
            flip_y=0.01,
            random_state=44
        )

        model = xgb.XGBClassifier(
            n_estimators=100,
            max_depth=6,
            learning_rate=0.1,
            subsample=0.8,
            colsample_bytree=0.8,
            scale_pos_weight=24.0,  # Balance 96:4 ratio
            random_state=44,
            eval_metric='logloss',
            n_jobs=-1
        )

        model.fit(X, y, verbose=False)

        y_pred_proba = model.predict_proba(X)[:, 1]
        from sklearn.metrics import roc_auc_score
        train_auroc = roc_auc_score(y, y_pred_proba)

        output_path = self._export_to_onnx(
            model,
            model_name='mortality_risk',
            version='1.0.0',
            description='In-hospital mortality prediction',
            clinical_focus='Age, comorbidities, APACHE, organ dysfunction, bilirubin',
            train_auroc=train_auroc
        )

        return output_path

    def generate_readmission_model(self):
        """
        Generate 30-day readmission prediction model.

        Clinical focus:
        - Length of stay
        - Discharge diagnosis complexity
        - Prior admission history
        - Social determinants
        """
        print("🔄 Generating Readmission Risk Model...")

        X, y = make_classification(
            n_samples=5000,
            n_features=70,
            n_informative=22,
            n_redundant=12,
            n_classes=2,
            weights=[0.90, 0.10],  # 10% readmission rate
            flip_y=0.04,
            random_state=45
        )

        model = xgb.XGBClassifier(
            n_estimators=100,
            max_depth=6,
            learning_rate=0.1,
            subsample=0.8,
            colsample_bytree=0.8,
            scale_pos_weight=9.0,  # Balance 90:10 ratio
            random_state=45,
            eval_metric='logloss',
            n_jobs=-1
        )

        model.fit(X, y, verbose=False)

        y_pred_proba = model.predict_proba(X)[:, 1]
        from sklearn.metrics import roc_auc_score
        train_auroc = roc_auc_score(y, y_pred_proba)

        output_path = self._export_to_onnx(
            model,
            model_name='readmission_risk',
            version='1.0.0',
            description='30-day unplanned readmission prediction',
            clinical_focus='Length of stay, discharge diagnosis, prior admissions, comorbidities',
            train_auroc=train_auroc
        )

        return output_path

    def _export_to_onnx(self, model, model_name, version, description, clinical_focus, train_auroc):
        """
        Export XGBoost model to ONNX format.

        Args:
            model: Trained XGBoost classifier
            model_name: Model identifier (e.g., 'sepsis_risk')
            version: Semantic version (e.g., '1.0.0')
            description: Model description
            clinical_focus: Key clinical features
            train_auroc: Training AUROC

        Returns:
            Path to exported ONNX model
        """
        # Define input type (70 float features)
        initial_types = [('float_input', FloatTensorType([None, 70]))]

        # Convert to ONNX using onnxmltools for XGBoost
        try:
            onnx_model = convert_xgboost(
                model,
                initial_types=initial_types,
                target_opset=12
            )
        except Exception as e:
            print(f"  ❌ Error converting model to ONNX: {e}")
            raise

        # Set metadata
        onnx_model.producer_name = 'CardioFit-Module5-TrackB'
        onnx_model.producer_version = version
        onnx_model.doc_string = description

        # Add custom metadata
        meta_model_name = onnx_model.metadata_props.add()
        meta_model_name.key = 'model_name'
        meta_model_name.value = model_name

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
        meta_clinical.value = clinical_focus

        meta_created = onnx_model.metadata_props.add()
        meta_created.key = 'created_date'
        meta_created.value = datetime.now().strftime('%Y-%m-%d')

        meta_mock = onnx_model.metadata_props.add()
        meta_mock.key = 'is_mock_model'
        meta_mock.value = 'true'

        meta_auroc = onnx_model.metadata_props.add()
        meta_auroc.key = 'train_auroc'
        meta_auroc.value = f'{train_auroc:.4f}'

        # Verify model
        try:
            onnx.checker.check_model(onnx_model)
        except Exception as e:
            print(f"  ❌ ONNX model validation failed: {e}")
            raise

        # Save to file
        output_path = os.path.join(self.output_dir, f'{model_name}_v{version}.onnx')
        onnx.save(onnx_model, output_path)

        # Verify ONNX Runtime can load it
        try:
            session = ort.InferenceSession(output_path)
            input_name = session.get_inputs()[0].name

            # Test inference on random input
            test_input = np.random.randn(1, 70).astype(np.float32)
            outputs = session.run(None, {input_name: test_input})

            # XGBoost ONNX models return multiple outputs (labels and probabilities)
            # Get the probability output (typically the second output)
            if len(outputs) >= 2:
                probs = outputs[1]  # Probabilities
            else:
                probs = outputs[0]

            # Validate probabilities
            if len(probs.shape) == 2 and probs.shape[1] == 2:
                # Binary classification with [neg_prob, pos_prob]
                assert np.allclose(probs.sum(axis=1), 1.0, atol=0.01), "Probabilities should sum to 1"
            assert np.all(probs >= 0) and np.all(probs <= 1), "Probabilities must be in [0, 1]"

        except Exception as e:
            print(f"  ❌ ONNX Runtime validation failed: {e}")
            raise

        # Get file size
        file_size_mb = os.path.getsize(output_path) / (1024 * 1024)

        print(f"  ✅ {model_name}_v{version}.onnx")
        print(f"     Size: {file_size_mb:.2f} MB")
        print(f"     Input: (batch_size, 70) float32")
        print(f"     Output: (batch_size, 2) float32 [neg_prob, pos_prob]")
        print(f"     Train AUROC: {train_auroc:.4f}")
        print(f"     Test inference: PASSED\n")

        return output_path


def check_dependencies():
    """Check if all required packages are installed."""
    required_packages = {
        'xgboost': 'xgboost',
        'sklearn': 'scikit-learn',
        'onnx': 'onnx',
        'onnxruntime': 'onnxruntime',
        'onnxmltools': 'onnxmltools',
        'numpy': 'numpy'
    }

    missing = []
    for module_name, package_name in required_packages.items():
        try:
            __import__(module_name)
        except ImportError:
            missing.append(package_name)

    if missing:
        print("❌ Missing required packages:")
        for pkg in missing:
            print(f"   - {pkg}")
        print("\nInstall with:")
        print(f"   pip install {' '.join(missing)}")
        sys.exit(1)


def main():
    """Generate all 4 mock ONNX models."""
    print("=" * 70)
    print("MODULE 5 MOCK ONNX MODEL GENERATOR")
    print("=" * 70)
    print()

    # Check dependencies
    check_dependencies()

    generator = MockModelGenerator(output_dir='models')

    models = []

    try:
        # Generate all 4 models
        models.append(generator.generate_sepsis_model())
        models.append(generator.generate_deterioration_model())
        models.append(generator.generate_mortality_model())
        models.append(generator.generate_readmission_model())

        print("=" * 70)
        print("✅ ALL MODELS GENERATED SUCCESSFULLY")
        print("=" * 70)
        print()
        print("Generated Models:")
        for model_path in models:
            print(f"  - {model_path}")
        print()
        print("Next Steps:")
        print("  1. Run integration tests:")
        print("     mvn test -Dtest=Module5IntegrationTest")
        print()
        print("  2. Run performance benchmarks:")
        print("     mvn test -Dtest=Module5PerformanceBenchmark")
        print()
        print("  3. Deploy to Flink for end-to-end validation:")
        print("     # Copy models to Flink deployment directory")
        print("     # Start Flink job with Module 5 ML inference")
        print()

    except Exception as e:
        print(f"\n❌ Error generating models: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
