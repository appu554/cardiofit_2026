"""
Elasticsearch Index Templates and Mappings
Production-grade index configurations for clinical data
"""
from typing import Dict, Any


# Clinical Events Index - Main event stream with full enrichments
CLINICAL_EVENTS_TEMPLATE = {
    "index_patterns": ["clinical_events-*"],
    "template": {
        "settings": {
            "number_of_shards": 3,
            "number_of_replicas": 1,
            "refresh_interval": "5s",
            "analysis": {
                "analyzer": {
                    "clinical_analyzer": {
                        "type": "custom",
                        "tokenizer": "standard",
                        "filter": ["lowercase", "asciifolding", "porter_stem"]
                    }
                }
            }
        },
        "mappings": {
            "properties": {
                "eventId": {"type": "keyword"},
                "patientId": {"type": "keyword"},
                "deviceId": {"type": "keyword"},
                "timestamp": {"type": "date"},
                "eventType": {"type": "keyword"},
                "stage": {"type": "keyword"},

                # Raw data from device
                "rawData": {
                    "type": "object",
                    "enabled": True,
                    "properties": {
                        "heartRate": {"type": "integer"},
                        "bloodPressure": {
                            "properties": {
                                "systolic": {"type": "integer"},
                                "diastolic": {"type": "integer"}
                            }
                        },
                        "oxygenSaturation": {"type": "float"},
                        "temperature": {"type": "float"},
                        "respiratoryRate": {"type": "integer"},
                        "glucoseLevel": {"type": "float"},
                        "activityLevel": {"type": "keyword"},
                        "steps": {"type": "integer"}
                    }
                },

                # FHIR transformations
                "enrichments": {
                    "type": "object",
                    "properties": {
                        "fhirResources": {
                            "type": "object",
                            "enabled": True
                        },
                        "clinicalContext": {
                            "type": "object",
                            "enabled": True
                        }
                    }
                },

                # Semantic annotations
                "semanticAnnotations": {
                    "type": "object",
                    "properties": {
                        "medicalConcepts": {
                            "type": "nested",
                            "properties": {
                                "code": {"type": "keyword"},
                                "system": {"type": "keyword"},
                                "display": {"type": "text", "analyzer": "clinical_analyzer"},
                                "category": {"type": "keyword"}
                            }
                        },
                        "conditions": {
                            "type": "nested",
                            "properties": {
                                "name": {"type": "text", "analyzer": "clinical_analyzer"},
                                "severity": {"type": "keyword"},
                                "onsetDate": {"type": "date"}
                            }
                        }
                    }
                },

                # ML predictions
                "mlPredictions": {
                    "type": "object",
                    "properties": {
                        "riskScore": {"type": "float"},
                        "riskLevel": {"type": "keyword"},
                        "predictions": {
                            "type": "nested",
                            "properties": {
                                "condition": {"type": "keyword"},
                                "probability": {"type": "float"},
                                "confidence": {"type": "float"}
                            }
                        },
                        "recommendations": {
                            "type": "text",
                            "analyzer": "clinical_analyzer"
                        }
                    }
                },

                # Metadata
                "processingTime": {"type": "date"},
                "version": {"type": "keyword"}
            }
        }
    }
}


# Patients Index - Current patient state with demographics
PATIENTS_TEMPLATE = {
    "index_patterns": ["patients"],
    "template": {
        "settings": {
            "number_of_shards": 1,
            "number_of_replicas": 1,
            "refresh_interval": "10s"
        },
        "mappings": {
            "properties": {
                "patientId": {"type": "keyword"},
                "demographics": {
                    "properties": {
                        "name": {"type": "text"},
                        "age": {"type": "integer"},
                        "gender": {"type": "keyword"},
                        "dateOfBirth": {"type": "date"}
                    }
                },
                "currentState": {
                    "properties": {
                        "latestEventId": {"type": "keyword"},
                        "latestEventTime": {"type": "date"},
                        "currentRiskLevel": {"type": "keyword"},
                        "currentRiskScore": {"type": "float"},
                        "activeConditions": {"type": "keyword"},
                        "deviceIds": {"type": "keyword"}
                    }
                },
                "vitalsSummary": {
                    "properties": {
                        "latestHeartRate": {"type": "integer"},
                        "latestBP": {
                            "properties": {
                                "systolic": {"type": "integer"},
                                "diastolic": {"type": "integer"}
                            }
                        },
                        "latestO2Sat": {"type": "float"},
                        "latestTemp": {"type": "float"}
                    }
                },
                "updatedAt": {"type": "date"}
            }
        }
    }
}


# Clinical Documents Index - Full-text searchable clinical notes
CLINICAL_DOCUMENTS_TEMPLATE = {
    "index_patterns": ["clinical_documents-*"],
    "template": {
        "settings": {
            "number_of_shards": 2,
            "number_of_replicas": 1,
            "analysis": {
                "analyzer": {
                    "clinical_text_analyzer": {
                        "type": "custom",
                        "tokenizer": "standard",
                        "filter": [
                            "lowercase",
                            "asciifolding",
                            "porter_stem",
                            "clinical_synonyms"
                        ]
                    }
                },
                "filter": {
                    "clinical_synonyms": {
                        "type": "synonym",
                        "synonyms": [
                            "bp,blood pressure",
                            "hr,heart rate",
                            "o2,oxygen,spo2",
                            "temp,temperature",
                            "rr,respiratory rate"
                        ]
                    }
                }
            }
        },
        "mappings": {
            "properties": {
                "documentId": {"type": "keyword"},
                "eventId": {"type": "keyword"},
                "patientId": {"type": "keyword"},
                "documentType": {"type": "keyword"},
                "title": {"type": "text", "analyzer": "clinical_text_analyzer"},
                "content": {
                    "type": "text",
                    "analyzer": "clinical_text_analyzer",
                    "fields": {
                        "keyword": {"type": "keyword", "ignore_above": 256}
                    }
                },
                "author": {"type": "keyword"},
                "createdAt": {"type": "date"},
                "tags": {"type": "keyword"}
            }
        }
    }
}


# Alerts Index - Real-time clinical alerts and notifications
ALERTS_TEMPLATE = {
    "index_patterns": ["alerts-*"],
    "template": {
        "settings": {
            "number_of_shards": 1,
            "number_of_replicas": 1,
            "refresh_interval": "1s"  # Real-time refresh for alerts
        },
        "mappings": {
            "properties": {
                "alertId": {"type": "keyword"},
                "eventId": {"type": "keyword"},
                "patientId": {"type": "keyword"},
                "alertType": {"type": "keyword"},
                "severity": {"type": "keyword"},  # LOW, MEDIUM, HIGH, CRITICAL
                "riskScore": {"type": "float"},
                "title": {"type": "text"},
                "description": {"type": "text", "analyzer": "clinical_analyzer"},
                "triggeredBy": {
                    "properties": {
                        "metric": {"type": "keyword"},
                        "value": {"type": "float"},
                        "threshold": {"type": "float"}
                    }
                },
                "recommendations": {"type": "text"},
                "acknowledged": {"type": "boolean"},
                "acknowledgedBy": {"type": "keyword"},
                "acknowledgedAt": {"type": "date"},
                "resolvedAt": {"type": "date"},
                "createdAt": {"type": "date"},
                "expiresAt": {"type": "date"}
            }
        }
    }
}


def get_all_templates() -> Dict[str, Dict[str, Any]]:
    """Get all index templates for initialization"""
    return {
        "clinical_events": CLINICAL_EVENTS_TEMPLATE,
        "patients": PATIENTS_TEMPLATE,
        "clinical_documents": CLINICAL_DOCUMENTS_TEMPLATE,
        "alerts": ALERTS_TEMPLATE
    }
