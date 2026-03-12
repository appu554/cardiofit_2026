# NidaanKosha-100k Dataset Analysis

**Dataset**: ekacare/NidaanKosha-100k-V1.0 (HuggingFace)
**Purpose**: Indian clinical cases for fine-tuning MIMIC-IV models

## Dataset Overview

- Total records analyzed: 100
- Total columns: 9
- Mapped to MIMIC-IV: 2/37 features (5.4%)

## Available Columns

- `document_id`
- `age`
- `gender`
- `test_name`
- `display_ranges`
- `value`
- `unit`
- `specimen`
- `loinc`

## Feature Mapping to MIMIC-IV

| MIMIC-IV Feature | NidaanKosha Column |
|-----------------|--------------------|
| age | age |
| gender_male | gender |

## Next Steps

1. Download full dataset (100k records)
2. Implement feature extraction pipeline
3. Handle missing features with imputation
4. Generate training/validation/test splits
5. Fine-tune MIMIC-IV models on Indian data
