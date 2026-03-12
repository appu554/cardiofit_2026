#!/bin/bash

# KB-6 Formulary Elasticsearch Initialization
# Drug formulary search and coverage analysis setup

set -e

# Configuration
ELASTICSEARCH_URL=${ELASTICSEARCH_URL:-"http://localhost:9200"}
INDEX_PREFIX="kb6_formulary"

echo "Initializing KB-6 Formulary Elasticsearch indexes..."
echo "Elasticsearch URL: $ELASTICSEARCH_URL"

# Function to create index with mapping
create_index() {
    local index_name=$1
    local mapping_file=$2
    
    echo "Creating index: $index_name"
    
    # Check if index exists
    if curl -s -f -XHEAD "$ELASTICSEARCH_URL/$index_name" > /dev/null 2>&1; then
        echo "Index $index_name already exists, skipping creation"
        return 0
    fi
    
    # Create index
    curl -X PUT "$ELASTICSEARCH_URL/$index_name" \
        -H "Content-Type: application/json" \
        -d @"$mapping_file" \
        -w "\nHTTP Status: %{http_code}\n"
        
    echo "Index $index_name created successfully"
}

# Function to bulk insert sample data
insert_sample_data() {
    local index_name=$1
    local data_file=$2
    
    echo "Inserting sample data into: $index_name"
    
    curl -X POST "$ELASTICSEARCH_URL/$index_name/_bulk" \
        -H "Content-Type: application/x-ndjson" \
        --data-binary @"$data_file" \
        -w "\nHTTP Status: %{http_code}\n"
        
    echo "Sample data inserted into $index_name"
}

# Create temporary mapping files
cat > /tmp/formulary_drugs_mapping.json << 'EOF'
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "drug_name_analyzer": {
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding", "drug_synonyms"]
        },
        "autocomplete_analyzer": {
          "tokenizer": "autocomplete_tokenizer",
          "filter": ["lowercase", "asciifolding"]
        }
      },
      "tokenizer": {
        "autocomplete_tokenizer": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 10,
          "token_chars": ["letter", "digit"]
        }
      },
      "filter": {
        "drug_synonyms": {
          "type": "synonym",
          "synonyms": [
            "acetaminophen,paracetamol,tylenol",
            "ibuprofen,advil,motrin",
            "aspirin,acetylsalicylic acid,asa",
            "metformin,glucophage",
            "atorvastatin,lipitor",
            "amlodipine,norvasc",
            "metoprolol,lopressor,toprol",
            "hydrochlorothiazide,hctz,microzide"
          ]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "drug_id": {"type": "keyword"},
      "drug_name": {
        "type": "text",
        "analyzer": "drug_name_analyzer",
        "fields": {
          "keyword": {"type": "keyword"},
          "autocomplete": {
            "type": "text",
            "analyzer": "autocomplete_analyzer"
          }
        }
      },
      "generic_name": {
        "type": "text",
        "analyzer": "drug_name_analyzer",
        "fields": {"keyword": {"type": "keyword"}}
      },
      "brand_names": {
        "type": "text",
        "analyzer": "drug_name_analyzer"
      },
      "rxnorm_code": {"type": "keyword"},
      "ndc_codes": {"type": "keyword"},
      "therapeutic_class": {"type": "keyword"},
      "drug_class": {"type": "keyword"},
      "route_of_administration": {"type": "keyword"},
      "dosage_forms": {"type": "keyword"},
      "strengths": {"type": "keyword"},
      "formulary_status": {"type": "keyword"},
      "tier": {"type": "integer"},
      "coverage_status": {"type": "keyword"},
      "prior_auth_required": {"type": "boolean"},
      "step_therapy_required": {"type": "boolean"},
      "quantity_limits": {
        "properties": {
          "limit_type": {"type": "keyword"},
          "limit_value": {"type": "integer"},
          "limit_period": {"type": "keyword"}
        }
      },
      "copay_info": {
        "properties": {
          "tier_1": {"type": "float"},
          "tier_2": {"type": "float"},
          "tier_3": {"type": "float"},
          "specialty": {"type": "float"}
        }
      },
      "alternatives": {
        "properties": {
          "drug_id": {"type": "keyword"},
          "drug_name": {"type": "text"},
          "tier": {"type": "integer"},
          "cost_difference": {"type": "float"}
        }
      },
      "contraindications": {"type": "text"},
      "age_restrictions": {
        "properties": {
          "min_age": {"type": "integer"},
          "max_age": {"type": "integer"}
        }
      },
      "pregnancy_category": {"type": "keyword"},
      "formulary_id": {"type": "keyword"},
      "effective_date": {"type": "date"},
      "expiration_date": {"type": "date"},
      "last_updated": {"type": "date"},
      "created_at": {"type": "date"}
    }
  }
}
EOF

# Create sample drug data
cat > /tmp/formulary_drugs_sample.ndjson << 'EOF'
{"index": {"_id": "DRUG_001"}}
{"drug_id": "DRUG_001", "drug_name": "Metformin", "generic_name": "Metformin", "brand_names": ["Glucophage", "Fortamet"], "rxnorm_code": "6809", "ndc_codes": ["0093-1045-01"], "therapeutic_class": "Antidiabetic", "drug_class": "Biguanide", "route_of_administration": ["oral"], "dosage_forms": ["tablet", "extended_release"], "strengths": ["500mg", "850mg", "1000mg"], "formulary_status": "preferred", "tier": 1, "coverage_status": "full", "prior_auth_required": false, "step_therapy_required": false, "copay_info": {"tier_1": 5.00, "tier_2": 15.00, "tier_3": 35.00, "specialty": 100.00}, "alternatives": [{"drug_id": "DRUG_002", "drug_name": "Glyburide", "tier": 1, "cost_difference": 2.50}], "formulary_id": "STANDARD_2024", "effective_date": "2024-01-01", "last_updated": "2024-01-15", "created_at": "2024-01-01"}

{"index": {"_id": "DRUG_002"}}
{"drug_id": "DRUG_002", "drug_name": "Atorvastatin", "generic_name": "Atorvastatin", "brand_names": ["Lipitor"], "rxnorm_code": "83367", "ndc_codes": ["0071-0155-23"], "therapeutic_class": "Antilipemic", "drug_class": "HMG-CoA Reductase Inhibitor", "route_of_administration": ["oral"], "dosage_forms": ["tablet"], "strengths": ["10mg", "20mg", "40mg", "80mg"], "formulary_status": "preferred", "tier": 1, "coverage_status": "full", "prior_auth_required": false, "step_therapy_required": false, "copay_info": {"tier_1": 5.00, "tier_2": 15.00, "tier_3": 35.00, "specialty": 100.00}, "alternatives": [{"drug_id": "DRUG_003", "drug_name": "Simvastatin", "tier": 1, "cost_difference": 0.00}], "formulary_id": "STANDARD_2024", "effective_date": "2024-01-01", "last_updated": "2024-01-15", "created_at": "2024-01-01"}

{"index": {"_id": "DRUG_003"}}
{"drug_id": "DRUG_003", "drug_name": "Lisinopril", "generic_name": "Lisinopril", "brand_names": ["Prinivil", "Zestril"], "rxnorm_code": "29046", "ndc_codes": ["0378-0110-01"], "therapeutic_class": "Antihypertensive", "drug_class": "ACE Inhibitor", "route_of_administration": ["oral"], "dosage_forms": ["tablet"], "strengths": ["2.5mg", "5mg", "10mg", "20mg", "40mg"], "formulary_status": "preferred", "tier": 1, "coverage_status": "full", "prior_auth_required": false, "step_therapy_required": false, "copay_info": {"tier_1": 5.00, "tier_2": 15.00, "tier_3": 35.00, "specialty": 100.00}, "alternatives": [{"drug_id": "DRUG_004", "drug_name": "Enalapril", "tier": 1, "cost_difference": 0.50}], "formulary_id": "STANDARD_2024", "effective_date": "2024-01-01", "last_updated": "2024-01-15", "created_at": "2024-01-01"}

{"index": {"_id": "DRUG_004"}}
{"drug_id": "DRUG_004", "drug_name": "Amlodipine", "generic_name": "Amlodipine", "brand_names": ["Norvasc"], "rxnorm_code": "17767", "ndc_codes": ["0069-1540-66"], "therapeutic_class": "Antihypertensive", "drug_class": "Calcium Channel Blocker", "route_of_administration": ["oral"], "dosage_forms": ["tablet"], "strengths": ["2.5mg", "5mg", "10mg"], "formulary_status": "preferred", "tier": 1, "coverage_status": "full", "prior_auth_required": false, "step_therapy_required": false, "copay_info": {"tier_1": 5.00, "tier_2": 15.00, "tier_3": 35.00, "specialty": 100.00}, "formulary_id": "STANDARD_2024", "effective_date": "2024-01-01", "last_updated": "2024-01-15", "created_at": "2024-01-01"}

{"index": {"_id": "DRUG_005"}}
{"drug_id": "DRUG_005", "drug_name": "Adalimumab", "generic_name": "Adalimumab", "brand_names": ["Humira"], "rxnorm_code": "323272", "ndc_codes": ["0074-3799-02"], "therapeutic_class": "Antirheumatic", "drug_class": "TNF Blocker", "route_of_administration": ["subcutaneous"], "dosage_forms": ["injection"], "strengths": ["40mg/0.8mL"], "formulary_status": "covered", "tier": 4, "coverage_status": "partial", "prior_auth_required": true, "step_therapy_required": true, "quantity_limits": {"limit_type": "monthly", "limit_value": 2, "limit_period": "month"}, "copay_info": {"tier_1": 5.00, "tier_2": 15.00, "tier_3": 35.00, "specialty": 150.00}, "age_restrictions": {"min_age": 18}, "formulary_id": "STANDARD_2024", "effective_date": "2024-01-01", "last_updated": "2024-01-15", "created_at": "2024-01-01"}
EOF

# Create coverage rules mapping
cat > /tmp/coverage_rules_mapping.json << 'EOF'
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1
  },
  "mappings": {
    "properties": {
      "rule_id": {"type": "keyword"},
      "rule_name": {"type": "text"},
      "rule_type": {"type": "keyword"},
      "drug_criteria": {
        "properties": {
          "drug_ids": {"type": "keyword"},
          "therapeutic_classes": {"type": "keyword"},
          "generic_required": {"type": "boolean"}
        }
      },
      "patient_criteria": {
        "properties": {
          "age_min": {"type": "integer"},
          "age_max": {"type": "integer"},
          "gender": {"type": "keyword"},
          "diagnosis_codes": {"type": "keyword"},
          "prior_medications": {"type": "keyword"}
        }
      },
      "coverage_decision": {"type": "keyword"},
      "prior_auth_required": {"type": "boolean"},
      "step_therapy_drugs": {"type": "keyword"},
      "effective_date": {"type": "date"},
      "expiration_date": {"type": "date"},
      "priority": {"type": "integer"},
      "status": {"type": "keyword"}
    }
  }
}
EOF

# Create sample coverage rules data
cat > /tmp/coverage_rules_sample.ndjson << 'EOF'
{"index": {"_id": "RULE_001"}}
{"rule_id": "RULE_001", "rule_name": "Diabetes Medication Coverage", "rule_type": "coverage", "drug_criteria": {"therapeutic_classes": ["Antidiabetic"], "generic_required": false}, "patient_criteria": {"diagnosis_codes": ["E11", "E10"]}, "coverage_decision": "approved", "prior_auth_required": false, "effective_date": "2024-01-01", "priority": 10, "status": "active"}

{"index": {"_id": "RULE_002"}}  
{"rule_id": "RULE_002", "rule_name": "Specialty Drug Prior Auth", "rule_type": "prior_auth", "drug_criteria": {"therapeutic_classes": ["Antirheumatic", "Oncology"]}, "patient_criteria": {}, "coverage_decision": "conditional", "prior_auth_required": true, "effective_date": "2024-01-01", "priority": 20, "status": "active"}

{"index": {"_id": "RULE_003"}}
{"rule_id": "RULE_003", "rule_name": "ACE Inhibitor Step Therapy", "rule_type": "step_therapy", "drug_criteria": {"therapeutic_classes": ["Antihypertensive"], "drug_ids": ["ARB_CLASS"]}, "patient_criteria": {"diagnosis_codes": ["I10"]}, "step_therapy_drugs": ["DRUG_003", "DRUG_004"], "coverage_decision": "conditional", "effective_date": "2024-01-01", "priority": 15, "status": "active"}
EOF

# Create prior auth rules mapping  
cat > /tmp/prior_auth_rules_mapping.json << 'EOF'
{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1
  },
  "mappings": {
    "properties": {
      "rule_id": {"type": "keyword"},
      "drug_id": {"type": "keyword"},
      "drug_name": {"type": "text"},
      "criteria_type": {"type": "keyword"},
      "clinical_criteria": {
        "properties": {
          "diagnosis_required": {"type": "keyword"},
          "failed_therapies": {"type": "keyword"},
          "contraindications": {"type": "keyword"},
          "lab_requirements": {"type": "text"}
        }
      },
      "documentation_required": {"type": "text"},
      "approval_duration": {"type": "integer"},
      "renewal_criteria": {"type": "text"},
      "override_codes": {"type": "keyword"},
      "emergency_override": {"type": "boolean"},
      "effective_date": {"type": "date"},
      "review_date": {"type": "date"},
      "status": {"type": "keyword"}
    }
  }
}
EOF

# Create sample prior auth rules
cat > /tmp/prior_auth_rules_sample.ndjson << 'EOF'
{"index": {"_id": "PA_001"}}
{"rule_id": "PA_001", "drug_id": "DRUG_005", "drug_name": "Adalimumab", "criteria_type": "clinical", "clinical_criteria": {"diagnosis_required": ["M05", "M06", "K50", "K51"], "failed_therapies": ["methotrexate", "sulfasalazine"], "lab_requirements": "Recent CBC, LFTs, TB screening"}, "documentation_required": "Clinical notes documenting diagnosis and failed conventional therapies", "approval_duration": 180, "renewal_criteria": "Documented clinical response and absence of adverse events", "emergency_override": false, "effective_date": "2024-01-01", "review_date": "2024-12-31", "status": "active"}

{"index": {"_id": "PA_002"}}
{"rule_id": "PA_002", "drug_id": "SPECIALTY_001", "drug_name": "Sofosbuvir", "criteria_type": "clinical", "clinical_criteria": {"diagnosis_required": ["B18.2"], "lab_requirements": "HCV RNA quantitative, genotype, LFTs"}, "documentation_required": "Hepatitis C diagnosis confirmation and treatment history", "approval_duration": 84, "renewal_criteria": "Completion of treatment course", "emergency_override": false, "effective_date": "2024-01-01", "review_date": "2024-12-31", "status": "active"}
EOF

# Function to check Elasticsearch health
check_elasticsearch() {
    echo "Checking Elasticsearch health..."
    
    for i in {1..30}; do
        if curl -s "$ELASTICSEARCH_URL/_cluster/health" > /dev/null 2>&1; then
            echo "Elasticsearch is ready"
            return 0
        fi
        echo "Waiting for Elasticsearch to be ready... ($i/30)"
        sleep 2
    done
    
    echo "ERROR: Elasticsearch is not responding"
    exit 1
}

# Main execution
echo "=== KB-6 Formulary Elasticsearch Initialization ==="

# Check Elasticsearch availability
check_elasticsearch

# Create indexes
create_index "${INDEX_PREFIX}_drugs" "/tmp/formulary_drugs_mapping.json"
create_index "${INDEX_PREFIX}_coverage_rules" "/tmp/coverage_rules_mapping.json"  
create_index "${INDEX_PREFIX}_prior_auth_rules" "/tmp/prior_auth_rules_mapping.json"

# Insert sample data
insert_sample_data "${INDEX_PREFIX}_drugs" "/tmp/formulary_drugs_sample.ndjson"
insert_sample_data "${INDEX_PREFIX}_coverage_rules" "/tmp/coverage_rules_sample.ndjson"
insert_sample_data "${INDEX_PREFIX}_prior_auth_rules" "/tmp/prior_auth_rules_sample.ndjson"

# Refresh indexes
echo "Refreshing indexes..."
curl -X POST "$ELASTICSEARCH_URL/${INDEX_PREFIX}_*/_refresh" -w "\nHTTP Status: %{http_code}\n"

# Verify data
echo "Verifying data insertion..."
echo "Drugs count:"
curl -s "$ELASTICSEARCH_URL/${INDEX_PREFIX}_drugs/_count" | grep -o '"count":[0-9]*'

echo "Coverage rules count:"  
curl -s "$ELASTICSEARCH_URL/${INDEX_PREFIX}_coverage_rules/_count" | grep -o '"count":[0-9]*'

echo "Prior auth rules count:"
curl -s "$ELASTICSEARCH_URL/${INDEX_PREFIX}_prior_auth_rules/_count" | grep -o '"count":[0-9]*'

# Clean up temporary files
rm -f /tmp/formulary_drugs_mapping.json
rm -f /tmp/formulary_drugs_sample.ndjson
rm -f /tmp/coverage_rules_mapping.json
rm -f /tmp/coverage_rules_sample.ndjson  
rm -f /tmp/prior_auth_rules_mapping.json
rm -f /tmp/prior_auth_rules_sample.ndjson

echo ""
echo "=== KB-6 Formulary Elasticsearch Initialization Complete ==="
echo "Indexes created:"
echo "  - ${INDEX_PREFIX}_drugs (drug formulary data)"
echo "  - ${INDEX_PREFIX}_coverage_rules (coverage decision rules)"
echo "  - ${INDEX_PREFIX}_prior_auth_rules (prior authorization rules)"
echo ""
echo "Sample data inserted and ready for use!"
echo "Access the data at: $ELASTICSEARCH_URL"