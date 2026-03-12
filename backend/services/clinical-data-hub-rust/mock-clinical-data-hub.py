#!/usr/bin/env python3
"""
Mock Clinical Data Hub Service
Temporary HTTP service to simulate the Clinical Data Hub on port 8118
until the full Rust implementation is ready.
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import time
from datetime import datetime
import urllib.parse as urlparse

class MockClinicalDataHubHandler(BaseHTTPRequestHandler):
    def _set_cors_headers(self):
        """Set CORS headers for all responses"""
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type, Authorization')

    def _send_json_response(self, status_code, data):
        """Send JSON response with proper headers"""
        self.send_response(status_code)
        self.send_header('Content-type', 'application/json')
        self._set_cors_headers()
        self.end_headers()
        self.wfile.write(json.dumps(data, indent=2).encode('utf-8'))

    def do_OPTIONS(self):
        """Handle OPTIONS requests for CORS"""
        self.send_response(200)
        self._set_cors_headers()
        self.end_headers()

    def do_GET(self):
        """Handle GET requests"""
        path = urlparse.urlparse(self.path).path

        if path == '/health':
            self._handle_health()
        elif path == '/ready':
            self._handle_ready()
        elif path == '/metrics':
            self._handle_metrics()
        elif path == '/api/federation':
            self._handle_federation_sdl()
        else:
            self._send_json_response(404, {
                "error": "Not Found",
                "message": f"Endpoint {path} not found",
                "available_endpoints": [
                    "/health",
                    "/ready",
                    "/metrics",
                    "/api/federation"
                ]
            })

    def do_POST(self):
        """Handle POST requests"""
        path = urlparse.urlparse(self.path).path

        if path == '/api/federation':
            self._handle_federation_graphql()
        else:
            self._send_json_response(404, {
                "error": "Not Found",
                "message": f"POST endpoint {path} not found"
            })

    def _handle_health(self):
        """Health check endpoint"""
        health_data = {
            "status": "healthy",
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "service": "clinical-data-hub-mock",
            "version": "1.0.0",
            "checks": {
                "cache": {"status": "healthy", "type": "mock"},
                "database": {"status": "healthy", "type": "mock"},
                "grpc_service": {"status": "healthy", "port": 8018}
            }
        }
        self._send_json_response(200, health_data)

    def _handle_ready(self):
        """Readiness probe endpoint"""
        ready_data = {
            "status": "ready",
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "service": "clinical-data-hub-mock"
        }
        self._send_json_response(200, ready_data)

    def _handle_metrics(self):
        """Prometheus metrics endpoint"""
        metrics = """# HELP clinical_data_hub_cache_hits_total Total number of cache hits
# TYPE clinical_data_hub_cache_hits_total counter
clinical_data_hub_cache_hits_total{layer="l1"} 1234
clinical_data_hub_cache_hits_total{layer="l2"} 567
clinical_data_hub_cache_hits_total{layer="l3"} 89

# HELP clinical_data_hub_requests_total Total number of requests
# TYPE clinical_data_hub_requests_total counter
clinical_data_hub_requests_total{method="GET"} 4321
clinical_data_hub_requests_total{method="POST"} 1234

# HELP clinical_data_hub_response_time_seconds Response time in seconds
# TYPE clinical_data_hub_response_time_seconds histogram
clinical_data_hub_response_time_seconds_bucket{le="0.001"} 100
clinical_data_hub_response_time_seconds_bucket{le="0.005"} 200
clinical_data_hub_response_time_seconds_bucket{le="0.01"} 300
clinical_data_hub_response_time_seconds_bucket{le="0.05"} 400
clinical_data_hub_response_time_seconds_bucket{le="+Inf"} 500
clinical_data_hub_response_time_seconds_sum 1.25
clinical_data_hub_response_time_seconds_count 500
"""
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self._set_cors_headers()
        self.end_headers()
        self.wfile.write(metrics.encode('utf-8'))

    def _handle_federation_sdl(self):
        """Handle federation SDL requests"""
        sdl_response = {
            "data": {
                "_service": {
                    "sdl": '''
                        directive @key(fields: String!) on OBJECT | INTERFACE
                        directive @external on FIELD_DEFINITION | OBJECT

                        scalar JSON
                        scalar DateTime

                        type ClinicalData @key(fields: "patientId") {
                            patientId: ID!
                            aggregatedData: JSON
                            cacheLayer: String
                            lastUpdated: DateTime
                        }

                        type PerformanceMetrics {
                            cacheHitRate: Float!
                            averageResponseTime: Float!
                            throughput: Int!
                        }

                        type Query {
                            _entities(representations: [JSON!]!): [ClinicalData]!
                            _service: _Service!
                            performanceMetrics: PerformanceMetrics!
                        }

                        type _Service {
                            sdl: String
                        }
                    '''
                }
            }
        }
        self._send_json_response(200, sdl_response)

    def _handle_federation_graphql(self):
        """Handle GraphQL federation requests"""
        try:
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            request_data = json.loads(post_data.decode('utf-8'))

            query = request_data.get('query', '')

            if '_service' in query and 'sdl' in query:
                response_data = {
                    "data": {
                        "_service": {
                            "sdl": '''
                                directive @key(fields: String!) on OBJECT | INTERFACE
                                directive @external on FIELD_DEFINITION | OBJECT

                                scalar JSON
                                scalar DateTime

                                type ClinicalData @key(fields: "patientId") {
                                    patientId: ID!
                                    aggregatedData: JSON
                                    cacheLayer: String
                                    lastUpdated: DateTime
                                }

                                type PerformanceMetrics {
                                    cacheHitRate: Float!
                                    averageResponseTime: Float!
                                    throughput: Int!
                                }

                                type Query {
                                    _entities(representations: [JSON!]!): [ClinicalData]!
                                    _service: _Service!
                                    performanceMetrics: PerformanceMetrics!
                                }

                                type _Service {
                                    sdl: String
                                }
                            '''
                        }
                    }
                }
            elif '_entities' in query:
                response_data = {
                    "data": {
                        "_entities": []
                    }
                }
            elif 'performanceMetrics' in query:
                response_data = {
                    "data": {
                        "performanceMetrics": {
                            "cacheHitRate": 0.85,
                            "averageResponseTime": 12.5,
                            "throughput": 1500
                        }
                    }
                }
            else:
                response_data = {
                    "errors": [
                        {
                            "message": "Query not supported in mock service",
                            "query": query
                        }
                    ]
                }

            self._send_json_response(200, response_data)

        except Exception as e:
            self._send_json_response(500, {
                "errors": [{"message": f"Internal server error: {str(e)}"}]
            })

def run_mock_server(port=8118):
    """Run the mock Clinical Data Hub server"""
    server_address = ('', port)
    httpd = HTTPServer(server_address, MockClinicalDataHubHandler)

    print(f"🚀 Mock Clinical Data Hub Service starting on port {port}")
    print(f"📡 Health endpoint: http://localhost:{port}/health")
    print(f"🔗 Federation endpoint: http://localhost:{port}/api/federation")
    print(f"📊 Metrics endpoint: http://localhost:{port}/metrics")
    print("=" * 60)

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print(f"\n🛑 Mock Clinical Data Hub Service stopped")
        httpd.server_close()

if __name__ == "__main__":
    run_mock_server(8118)