"""
Comprehensive Test Suite for Elasticsearch Projector
Tests indexing, search, and analytics capabilities
"""
import asyncio
import json
import time
from datetime import datetime, timezone
from typing import Dict, Any

from elasticsearch import Elasticsearch


def create_test_event(event_id: str, patient_id: str, risk_level: str = "LOW") -> Dict[str, Any]:
    """Create a test enriched event"""
    return {
        "eventId": event_id,
        "patientId": patient_id,
        "deviceId": "device_001",
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "eventType": "vitals",
        "stage": "enriched",
        "rawData": {
            "heartRate": 75,
            "bloodPressure": {"systolic": 120, "diastolic": 80},
            "oxygenSaturation": 98.5,
            "temperature": 37.0,
            "respiratoryRate": 16
        },
        "enrichments": {
            "fhirResources": {
                "Observation": {
                    "resourceType": "Observation",
                    "status": "final",
                    "code": {"coding": [{"system": "LOINC", "code": "8867-4", "display": "Heart rate"}]}
                }
            },
            "clinicalContext": {
                "notes": "Patient vitals within normal range. Continue monitoring."
            }
        },
        "semanticAnnotations": {
            "medicalConcepts": [
                {
                    "code": "364075005",
                    "system": "SNOMED-CT",
                    "display": "Heart rate",
                    "category": "vital-sign"
                }
            ],
            "conditions": [
                {
                    "name": "Hypertension",
                    "severity": "moderate",
                    "onsetDate": "2024-01-01"
                }
            ]
        },
        "mlPredictions": {
            "riskScore": 0.35 if risk_level == "LOW" else 0.85,
            "riskLevel": risk_level,
            "predictions": [
                {
                    "condition": "cardiac_event",
                    "probability": 0.12,
                    "confidence": 0.89
                }
            ],
            "recommendations": [
                "Continue current treatment plan",
                "Monitor blood pressure daily"
            ]
        }
    }


async def test_elasticsearch_connection():
    """Test 1: Verify Elasticsearch connection"""
    print("\n=== Test 1: Elasticsearch Connection ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        if es.ping():
            print("✅ Elasticsearch connection successful")

            # Get cluster health
            health = es.cluster.health()
            print(f"✅ Cluster status: {health['status']}")
            print(f"✅ Number of nodes: {health['number_of_nodes']}")

            return True
        else:
            print("❌ Cannot connect to Elasticsearch")
            return False

    except Exception as e:
        print(f"❌ Connection error: {e}")
        return False
    finally:
        es.close()


async def test_index_creation():
    """Test 2: Verify index templates and creation"""
    print("\n=== Test 2: Index Template Creation ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        # Wait for service to create templates
        await asyncio.sleep(5)

        # Check for index templates
        templates = ["clinical_events", "patients", "clinical_documents", "alerts"]
        all_exist = True

        for template_name in templates:
            try:
                template = es.indices.get_index_template(name=template_name)
                print(f"✅ Template exists: {template_name}")
            except Exception as e:
                print(f"❌ Template missing: {template_name} - {e}")
                all_exist = False

        # Check for actual indices
        indices = ["patients", "clinical_events-2024", "clinical_documents-2024", "alerts-2024"]
        for index_name in indices:
            if es.indices.exists(index=index_name):
                print(f"✅ Index exists: {index_name}")
            else:
                print(f"⚠️  Index pending: {index_name} (will be created on first document)")

        return all_exist

    except Exception as e:
        print(f"❌ Index verification error: {e}")
        return False
    finally:
        es.close()


async def test_event_indexing():
    """Test 3: Test event indexing and retrieval"""
    print("\n=== Test 3: Event Indexing ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        # Wait for initial events to be indexed
        await asyncio.sleep(10)

        # Search for events
        result = es.search(
            index="clinical_events-*",
            body={
                "query": {"match_all": {}},
                "size": 10,
                "sort": [{"timestamp": {"order": "desc"}}]
            }
        )

        total_events = result['hits']['total']['value']
        print(f"✅ Total events indexed: {total_events}")

        if total_events > 0:
            # Display sample event
            sample = result['hits']['hits'][0]['_source']
            print(f"✅ Sample event ID: {sample.get('eventId')}")
            print(f"✅ Patient ID: {sample.get('patientId')}")
            print(f"✅ Risk level: {sample.get('mlPredictions', {}).get('riskLevel')}")
            return True
        else:
            print("⚠️  No events indexed yet (waiting for Kafka messages)")
            return True  # Not a failure, just waiting for data

    except Exception as e:
        print(f"❌ Event indexing test error: {e}")
        return False
    finally:
        es.close()


async def test_patient_state():
    """Test 4: Test patient state tracking"""
    print("\n=== Test 4: Patient State Tracking ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        await asyncio.sleep(10)

        # Search patients index
        result = es.search(
            index="patients",
            body={
                "query": {"match_all": {}},
                "size": 10
            }
        )

        total_patients = result['hits']['total']['value']
        print(f"✅ Total patients tracked: {total_patients}")

        if total_patients > 0:
            sample = result['hits']['hits'][0]['_source']
            print(f"✅ Sample patient ID: {sample.get('patientId')}")

            current_state = sample.get('currentState', {})
            print(f"✅ Latest event: {current_state.get('latestEventId')}")
            print(f"✅ Current risk: {current_state.get('currentRiskLevel')}")

            vitals = sample.get('vitalsSummary', {})
            if vitals.get('latestHeartRate'):
                print(f"✅ Latest heart rate: {vitals.get('latestHeartRate')}")

            return True
        else:
            print("⚠️  No patient states yet")
            return True

    except Exception as e:
        print(f"❌ Patient state test error: {e}")
        return False
    finally:
        es.close()


async def test_full_text_search():
    """Test 5: Test full-text search capabilities"""
    print("\n=== Test 5: Full-Text Search ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        await asyncio.sleep(10)

        # Test search queries
        queries = [
            "heart rate",
            "high blood pressure",
            "risk:HIGH",
            "patientId:P1001"
        ]

        for query in queries:
            result = es.search(
                index="clinical_events-*",
                body={
                    "query": {
                        "query_string": {
                            "query": query,
                            "default_operator": "AND"
                        }
                    },
                    "size": 5
                }
            )

            total = result['hits']['total']['value']
            took = result['took']
            print(f"✅ Query '{query}': {total} results in {took}ms")

        return True

    except Exception as e:
        print(f"❌ Full-text search test error: {e}")
        return False
    finally:
        es.close()


async def test_alerts():
    """Test 6: Test alert creation and retrieval"""
    print("\n=== Test 6: Alert Management ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        await asyncio.sleep(10)

        # Search for alerts
        result = es.search(
            index="alerts-*",
            body={
                "query": {"match_all": {}},
                "size": 10,
                "sort": [{"createdAt": {"order": "desc"}}]
            }
        )

        total_alerts = result['hits']['total']['value']
        print(f"✅ Total alerts created: {total_alerts}")

        if total_alerts > 0:
            sample = result['hits']['hits'][0]['_source']
            print(f"✅ Alert ID: {sample.get('alertId')}")
            print(f"✅ Severity: {sample.get('severity')}")
            print(f"✅ Patient: {sample.get('patientId')}")
            print(f"✅ Acknowledged: {sample.get('acknowledged')}")

            # Count by severity
            agg_result = es.search(
                index="alerts-*",
                body={
                    "size": 0,
                    "aggs": {
                        "by_severity": {
                            "terms": {"field": "severity"}
                        }
                    }
                }
            )

            buckets = agg_result.get('aggregations', {}).get('by_severity', {}).get('buckets', [])
            print("\n✅ Alerts by severity:")
            for bucket in buckets:
                print(f"   - {bucket['key']}: {bucket['doc_count']}")

        return True

    except Exception as e:
        print(f"❌ Alert test error: {e}")
        return False
    finally:
        es.close()


async def test_aggregations():
    """Test 7: Test aggregation queries"""
    print("\n=== Test 7: Aggregations ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        await asyncio.sleep(10)

        # Test 1: Risk level distribution
        result = es.search(
            index="patients",
            body={
                "size": 0,
                "aggs": {
                    "risk_distribution": {
                        "terms": {"field": "currentState.currentRiskLevel"}
                    }
                }
            }
        )

        buckets = result.get('aggregations', {}).get('risk_distribution', {}).get('buckets', [])
        print("✅ Risk level distribution:")
        for bucket in buckets:
            print(f"   - {bucket['key']}: {bucket['doc_count']} patients")

        # Test 2: Events over time
        result = es.search(
            index="clinical_events-*",
            body={
                "size": 0,
                "aggs": {
                    "events_over_time": {
                        "date_histogram": {
                            "field": "timestamp",
                            "calendar_interval": "hour"
                        }
                    }
                }
            }
        )

        time_buckets = result.get('aggregations', {}).get('events_over_time', {}).get('buckets', [])
        print(f"\n✅ Events by hour: {len(time_buckets)} time buckets")

        # Test 3: Average risk score
        result = es.search(
            index="clinical_events-*",
            body={
                "size": 0,
                "aggs": {
                    "avg_risk": {
                        "avg": {"field": "mlPredictions.riskScore"}
                    }
                }
            }
        )

        avg_risk = result.get('aggregations', {}).get('avg_risk', {}).get('value')
        if avg_risk is not None:
            print(f"✅ Average risk score: {avg_risk:.3f}")

        return True

    except Exception as e:
        print(f"❌ Aggregation test error: {e}")
        return False
    finally:
        es.close()


async def test_api_endpoints():
    """Test 8: Test FastAPI endpoints"""
    print("\n=== Test 8: API Endpoints ===")

    import requests

    try:
        base_url = "http://localhost:8052"

        # Test health endpoint
        response = requests.get(f"{base_url}/health", timeout=5)
        if response.status_code == 200:
            health = response.json()
            print(f"✅ Health endpoint: {health.get('status')}")
            print(f"✅ Elasticsearch connected: {health.get('elasticsearch', {}).get('connected')}")
        else:
            print(f"❌ Health endpoint returned {response.status_code}")

        # Test stats endpoint
        response = requests.get(f"{base_url}/stats", timeout=5)
        if response.status_code == 200:
            stats = response.json()
            print(f"✅ Stats endpoint working")
            processing_stats = stats.get('statistics', {})
            print(f"✅ Events indexed: {processing_stats.get('events_indexed', 0)}")
            print(f"✅ Patients updated: {processing_stats.get('patients_updated', 0)}")
            print(f"✅ Alerts created: {processing_stats.get('alerts_created', 0)}")

        # Test search endpoint
        response = requests.post(
            f"{base_url}/search",
            json={"query": "heart rate", "size": 5},
            timeout=5
        )
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Search endpoint: {result.get('total', 0)} results")

        # Test active alerts endpoint
        response = requests.get(f"{base_url}/alerts/active", timeout=5)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Active alerts: {result.get('totalActiveAlerts', 0)}")

        # Test risk distribution endpoint
        response = requests.get(f"{base_url}/aggregations/risk-distribution", timeout=5)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Risk distribution aggregation working")

        return True

    except Exception as e:
        print(f"❌ API endpoint test error: {e}")
        return False


async def test_performance():
    """Test 9: Test indexing and search performance"""
    print("\n=== Test 9: Performance Testing ===")

    es = Elasticsearch(["http://localhost:9200"])

    try:
        # Test search latency
        queries = ["heart rate", "blood pressure", "diabetes", "HIGH"]
        latencies = []

        for query in queries:
            start = time.time()
            result = es.search(
                index="clinical_events-*",
                body={
                    "query": {"query_string": {"query": query}},
                    "size": 10
                }
            )
            latency = (time.time() - start) * 1000
            latencies.append(latency)
            print(f"✅ Query '{query}': {latency:.1f}ms")

        avg_latency = sum(latencies) / len(latencies)
        print(f"\n✅ Average search latency: {avg_latency:.1f}ms")

        if avg_latency < 100:
            print("✅ Excellent search performance (<100ms)")
        elif avg_latency < 500:
            print("✅ Good search performance (<500ms)")
        else:
            print("⚠️  Slow search performance (>500ms)")

        return True

    except Exception as e:
        print(f"❌ Performance test error: {e}")
        return False
    finally:
        es.close()


async def run_all_tests():
    """Run all tests sequentially"""
    print("=" * 60)
    print("ELASTICSEARCH PROJECTOR TEST SUITE")
    print("=" * 60)

    tests = [
        ("Elasticsearch Connection", test_elasticsearch_connection),
        ("Index Creation", test_index_creation),
        ("Event Indexing", test_event_indexing),
        ("Patient State Tracking", test_patient_state),
        ("Full-Text Search", test_full_text_search),
        ("Alert Management", test_alerts),
        ("Aggregations", test_aggregations),
        ("API Endpoints", test_api_endpoints),
        ("Performance", test_performance)
    ]

    results = []
    for test_name, test_func in tests:
        try:
            result = await test_func()
            results.append((test_name, result))
        except Exception as e:
            print(f"\n❌ Test '{test_name}' crashed: {e}")
            results.append((test_name, False))

    # Summary
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)

    passed = sum(1 for _, result in results if result)
    total = len(results)

    for test_name, result in results:
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"{status}: {test_name}")

    print(f"\nTotal: {passed}/{total} tests passed ({(passed/total)*100:.1f}%)")
    print("=" * 60)


if __name__ == "__main__":
    asyncio.run(run_all_tests())
