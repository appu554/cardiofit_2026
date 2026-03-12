#!/bin/bash

# Wait for Elasticsearch to be ready
until curl -s http://localhost:9200/_cluster/health | grep -q '"status":"yellow\|green"'; do
  echo "Waiting for Elasticsearch..."
  sleep 5
done

# Create formulary index with mappings
curl -X PUT "localhost:9200/formulary" -H 'Content-Type: application/json' -d'
{
  "settings": {
    "number_of_shards": 2,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "drug_name_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "synonym_filter", "edge_ngram_filter"]
        }
      },
      "filter": {
        "edge_ngram_filter": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 20
        },
        "synonym_filter": {
          "type": "synonym",
          "synonyms": [
            "acetaminophen,tylenol,paracetamol",
            "ibuprofen,advil,motrin",
            "asa,aspirin"
          ]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "drug_rxnorm": {
        "type": "keyword"
      },
      "drug_name": {
        "type": "text",
        "analyzer": "drug_name_analyzer",
        "fields": {
          "keyword": {
            "type": "keyword"
          }
        }
      },
      "generic_name": {
        "type": "text",
        "analyzer": "drug_name_analyzer"
      },
      "brand_names": {
        "type": "text",
        "analyzer": "drug_name_analyzer"
      },
      "drug_class": {
        "type": "keyword"
      },
      "tier": {
        "type": "keyword"
      },
      "payer_id": {
        "type": "keyword"
      },
      "plan_id": {
        "type": "keyword"
      },
      "copay_amount": {
        "type": "float"
      },
      "prior_authorization": {
        "type": "boolean"
      },
      "quantity_limit": {
        "type": "object"
      },
      "effective_date": {
        "type": "date"
      },
      "termination_date": {
        "type": "date"
      }
    }
  }
}'

# Create drug pricing index
curl -X PUT "localhost:9200/drug_pricing" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "drug_rxnorm": {
        "type": "keyword"
      },
      "price_type": {
        "type": "keyword"
      },
      "price": {
        "type": "float"
      },
      "effective_date": {
        "type": "date"
      }
    }
  }
}'

echo "Elasticsearch initialization complete"