#!/usr/bin/env python3
"""
GraphDB Import Script for CAE Clinical Schema and Data

This script imports the RDF schema and sample data into GraphDB.
It handles repository creation, schema import, and data loading.
"""

import requests
import json
import os
import sys
from pathlib import Path

class GraphDBImporter:
    def __init__(self, graphdb_url="http://localhost:7200", repository_id="cae-clinical-intelligence"):
        self.graphdb_url = graphdb_url
        self.repository_id = repository_id
        self.headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        }
        
    def create_repository(self):
        """Create a new GraphDB repository for CAE clinical intelligence"""
        repository_config = {
            "id": self.repository_id,
            "title": "CAE Clinical Intelligence Repository",
            "type": "graphdb",
            "params": {
                "ruleset": {
                    "label": "Ruleset",
                    "name": "ruleset",
                    "value": "rdfs-plus"
                },
                "storageFolder": {
                    "label": "Storage folder",
                    "name": "storageFolder", 
                    "value": "storage"
                },
                "enableContextIndex": {
                    "label": "Use context index",
                    "name": "enableContextIndex",
                    "value": "false"
                },
                "enablePredicateList": {
                    "label": "Use predicate indices",
                    "name": "enablePredicateList", 
                    "value": "true"
                },
                "inMemoryLiteralProperties": {
                    "label": "Cache literal language tags",
                    "name": "inMemoryLiteralProperties",
                    "value": "true"
                },
                "enableLiteralIndex": {
                    "label": "Enable literal index",
                    "name": "enableLiteralIndex",
                    "value": "true"
                }
            }
        }
        
        url = f"{self.graphdb_url}/rest/repositories"
        
        try:
            response = requests.post(url, json=repository_config, headers=self.headers)
            if response.status_code == 201:
                print(f"✅ Repository '{self.repository_id}' created successfully")
                return True
            elif response.status_code == 409:
                print(f"ℹ️  Repository '{self.repository_id}' already exists")
                return True
            else:
                print(f"❌ Failed to create repository: {response.status_code} - {response.text}")
                return False
        except requests.exceptions.RequestException as e:
            print(f"❌ Error connecting to GraphDB: {e}")
            return False
    
    def import_rdf_file(self, file_path, context=None):
        """Import an RDF file into the repository"""
        if not os.path.exists(file_path):
            print(f"❌ File not found: {file_path}")
            return False
            
        # Determine content type based on file extension
        if file_path.endswith('.ttl'):
            content_type = 'text/turtle'
        elif file_path.endswith('.rdf'):
            content_type = 'application/rdf+xml'
        elif file_path.endswith('.n3'):
            content_type = 'text/n3'
        else:
            content_type = 'text/turtle'  # default
            
        url = f"{self.graphdb_url}/repositories/{self.repository_id}/statements"
        
        # Add context parameter if provided
        if context:
            url += f"?context={context}"
            
        headers = {
            'Content-Type': content_type
        }
        
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                data = f.read()
                
            response = requests.post(url, data=data, headers=headers)
            
            if response.status_code == 204:
                print(f"✅ Successfully imported: {os.path.basename(file_path)}")
                return True
            else:
                print(f"❌ Failed to import {file_path}: {response.status_code} - {response.text}")
                return False
                
        except requests.exceptions.RequestException as e:
            print(f"❌ Error importing {file_path}: {e}")
            return False
        except Exception as e:
            print(f"❌ Error reading {file_path}: {e}")
            return False
    
    def clear_repository(self):
        """Clear all data from the repository"""
        url = f"{self.graphdb_url}/repositories/{self.repository_id}/statements"
        
        try:
            response = requests.delete(url)
            if response.status_code == 204:
                print(f"✅ Repository '{self.repository_id}' cleared successfully")
                return True
            else:
                print(f"❌ Failed to clear repository: {response.status_code} - {response.text}")
                return False
        except requests.exceptions.RequestException as e:
            print(f"❌ Error clearing repository: {e}")
            return False
    
    def test_connection(self):
        """Test connection to GraphDB"""
        try:
            response = requests.get(f"{self.graphdb_url}/rest/repositories")
            if response.status_code == 200:
                print("✅ Successfully connected to GraphDB")
                return True
            else:
                print(f"❌ Failed to connect to GraphDB: {response.status_code}")
                return False
        except requests.exceptions.RequestException as e:
            print(f"❌ Error connecting to GraphDB: {e}")
            return False
    
    def query_repository(self, sparql_query):
        """Execute a SPARQL query against the repository"""
        url = f"{self.graphdb_url}/repositories/{self.repository_id}"
        
        headers = {
            'Content-Type': 'application/sparql-query',
            'Accept': 'application/sparql-results+json'
        }
        
        try:
            response = requests.post(url, data=sparql_query, headers=headers)
            if response.status_code == 200:
                return response.json()
            else:
                print(f"❌ Query failed: {response.status_code} - {response.text}")
                return None
        except requests.exceptions.RequestException as e:
            print(f"❌ Error executing query: {e}")
            return None

def main():
    """Main import process"""
    print("🚀 CAE Clinical Intelligence - GraphDB Import")
    print("=" * 50)
    
    # Initialize importer
    importer = GraphDBImporter()
    
    # Test connection
    if not importer.test_connection():
        print("❌ Cannot connect to GraphDB. Please ensure GraphDB is running on http://localhost:7200")
        sys.exit(1)
    
    # Create repository
    if not importer.create_repository():
        print("❌ Failed to create repository")
        sys.exit(1)
    
    # Get current directory
    current_dir = Path(__file__).parent
    
    # Import schema file
    schema_file = current_dir / "cae-clinical-schema.ttl"
    if schema_file.exists():
        print("\n📋 Importing clinical schema...")
        if not importer.import_rdf_file(str(schema_file)):
            print("❌ Failed to import schema")
            sys.exit(1)
    else:
        print(f"⚠️  Schema file not found: {schema_file}")
    
    # Import sample data file
    data_file = current_dir / "cae-sample-data.ttl"
    if data_file.exists():
        print("\n📊 Importing sample clinical data...")
        if not importer.import_rdf_file(str(data_file)):
            print("❌ Failed to import sample data")
            sys.exit(1)
    else:
        print(f"⚠️  Sample data file not found: {data_file}")
    
    # Test with a simple query
    print("\n🔍 Testing with sample query...")
    test_query = """
    PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
    SELECT ?patient ?medication WHERE {
        ?patient a cae:Patient .
        ?patient cae:prescribedMedication ?medication .
    } LIMIT 5
    """
    
    result = importer.query_repository(test_query)
    if result and 'results' in result and 'bindings' in result['results']:
        bindings = result['results']['bindings']
        print(f"✅ Found {len(bindings)} patient-medication relationships")
        for binding in bindings[:3]:  # Show first 3
            patient = binding['patient']['value'].split('/')[-1]
            medication = binding['medication']['value'].split('/')[-1]
            print(f"   • {patient} → {medication}")
    else:
        print("⚠️  No results from test query")
    
    print("\n🎉 Import completed successfully!")
    print(f"📍 Repository: {importer.repository_id}")
    print(f"🌐 GraphDB URL: {importer.graphdb_url}")
    print("\n💡 You can now query the repository using SPARQL or connect your CAE service to GraphDB")

if __name__ == "__main__":
    main()
