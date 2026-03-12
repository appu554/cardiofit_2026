#!/usr/bin/env python3
"""
KB2 Phenotype Data Insertion Script

This script inserts clinical phenotype definitions from JSON files into KB2's MongoDB database.
It connects to the MongoDB instance used by the KB2 Clinical Context service and populates
the phenotype_definitions collection with the converted YAML data.

Usage:
    python insert_phenotypes.py [--dry-run] [--collection-name COLLECTION_NAME]

Environment Variables:
    MONGODB_URI: MongoDB connection string (default: mongodb://kb2admin:kb2_mongodb_password@localhost:27018/kb2_clinical_context?authSource=admin)
"""

import json
import os
import sys
import glob
import argparse
from datetime import datetime
from typing import List, Dict, Any
from pathlib import Path

try:
    from pymongo import MongoClient
    from pymongo.errors import ConnectionFailure, DuplicateKeyError
except ImportError:
    print("Error: pymongo is required. Install with: pip install pymongo")
    sys.exit(1)


class KB2PhenotypeInserter:
    """Handles insertion of phenotype data into KB2's MongoDB database."""

    def __init__(self, mongodb_uri: str, collection_name: str = "phenotype_definitions"):
        """
        Initialize the inserter with MongoDB connection.

        Args:
            mongodb_uri: MongoDB connection string
            collection_name: Name of the collection to insert data into
        """
        self.mongodb_uri = mongodb_uri
        self.collection_name = collection_name
        self.client = None
        self.db = None
        self.collection = None

    def connect(self) -> bool:
        """
        Establish connection to MongoDB.

        Returns:
            True if connection successful, False otherwise
        """
        try:
            print(f"Connecting to MongoDB: {self.mongodb_uri}")
            self.client = MongoClient(self.mongodb_uri, serverSelectionTimeoutMS=5000)

            # Test the connection
            self.client.admin.command('ping')

            # Get database and collection
            self.db = self.client.get_default_database()
            self.collection = self.db[self.collection_name]

            print(f"✓ Connected to database: {self.db.name}")
            print(f"✓ Using collection: {self.collection_name}")
            return True

        except ConnectionFailure as e:
            print(f"✗ Failed to connect to MongoDB: {e}")
            return False
        except Exception as e:
            print(f"✗ Unexpected error during connection: {e}")
            return False

    def load_json_files(self, json_dir: str) -> List[Dict[str, Any]]:
        """
        Load all JSON phenotype files from directory.

        Args:
            json_dir: Directory containing JSON phenotype files

        Returns:
            List of phenotype dictionaries
        """
        json_files = glob.glob(os.path.join(json_dir, "kb2-phenotype-*.json"))

        if not json_files:
            print(f"✗ No JSON files found in {json_dir}")
            return []

        phenotypes = []

        for json_file in sorted(json_files):
            try:
                with open(json_file, 'r', encoding='utf-8') as f:
                    phenotype_data = json.load(f)
                    phenotypes.append(phenotype_data)
                    print(f"✓ Loaded: {os.path.basename(json_file)}")

            except json.JSONDecodeError as e:
                print(f"✗ Invalid JSON in {json_file}: {e}")
            except Exception as e:
                print(f"✗ Error loading {json_file}: {e}")

        print(f"✓ Loaded {len(phenotypes)} phenotype definitions")
        return phenotypes

    def validate_phenotype(self, phenotype: Dict[str, Any]) -> bool:
        """
        Validate phenotype data structure.

        Args:
            phenotype: Phenotype dictionary to validate

        Returns:
            True if valid, False otherwise
        """
        required_fields = [
            'phenotype_id', 'name', 'description', 'category', 'severity',
            'criteria', 'icd10_codes', 'snomed_codes', 'algorithm_type',
            'algorithm', 'validation_data', 'status'
        ]

        for field in required_fields:
            if field not in phenotype:
                print(f"✗ Missing required field '{field}' in phenotype {phenotype.get('phenotype_id', 'UNKNOWN')}")
                return False

        # Validate criteria structure
        criteria = phenotype.get('criteria', {})
        if not isinstance(criteria, dict):
            print(f"✗ Invalid criteria structure in phenotype {phenotype['phenotype_id']}")
            return False

        # Validate algorithm structure
        algorithm = phenotype.get('algorithm', {})
        if not isinstance(algorithm, dict) or 'type' not in algorithm:
            print(f"✗ Invalid algorithm structure in phenotype {phenotype['phenotype_id']}")
            return False

        return True

    def insert_phenotypes(self, phenotypes: List[Dict[str, Any]], dry_run: bool = False) -> Dict[str, int]:
        """
        Insert phenotypes into MongoDB collection.

        Args:
            phenotypes: List of phenotype dictionaries
            dry_run: If True, validate only without inserting

        Returns:
            Dictionary with insertion statistics
        """
        stats = {
            'total': len(phenotypes),
            'inserted': 0,
            'updated': 0,
            'skipped': 0,
            'errors': 0
        }

        if dry_run:
            print(f"\n🔍 DRY RUN: Validating {len(phenotypes)} phenotypes...")
        else:
            print(f"\n📥 Inserting {len(phenotypes)} phenotypes...")

        for phenotype in phenotypes:
            phenotype_id = phenotype.get('phenotype_id', 'UNKNOWN')

            try:
                # Validate phenotype structure
                if not self.validate_phenotype(phenotype):
                    stats['errors'] += 1
                    continue

                if dry_run:
                    print(f"✓ Valid: {phenotype_id} - {phenotype['name']}")
                    stats['inserted'] += 1
                    continue

                # Add insertion metadata
                phenotype['_inserted_at'] = datetime.utcnow()
                phenotype['_inserted_by'] = 'kb2_phenotype_inserter'

                # Try to insert or update
                result = self.collection.replace_one(
                    {'phenotype_id': phenotype_id},
                    phenotype,
                    upsert=True
                )

                if result.upserted_id:
                    print(f"✓ Inserted: {phenotype_id} - {phenotype['name']}")
                    stats['inserted'] += 1
                elif result.modified_count > 0:
                    print(f"✓ Updated: {phenotype_id} - {phenotype['name']}")
                    stats['updated'] += 1
                else:
                    print(f"• Unchanged: {phenotype_id} - {phenotype['name']}")
                    stats['skipped'] += 1

            except DuplicateKeyError:
                print(f"⚠ Duplicate key for {phenotype_id}")
                stats['errors'] += 1
            except Exception as e:
                print(f"✗ Error inserting {phenotype_id}: {e}")
                stats['errors'] += 1

        return stats

    def create_indexes(self) -> bool:
        """
        Create necessary indexes for the phenotype collection.

        Returns:
            True if indexes created successfully, False otherwise
        """
        try:
            print("\n📊 Creating database indexes...")

            # Create indexes based on KB2's query patterns
            indexes = [
                ('phenotype_id', 1),  # Primary lookup
                ('category', 1),      # Category filtering
                ('status', 1),        # Active/inactive filtering
                ('icd10_codes', 1),   # ICD-10 code lookup
                ('snomed_codes', 1),  # SNOMED code lookup
            ]

            for field, direction in indexes:
                self.collection.create_index([(field, direction)])
                print(f"✓ Created index on: {field}")

            # Create compound indexes for common queries
            self.collection.create_index([('category', 1), ('status', 1)])
            print("✓ Created compound index on: category + status")

            return True

        except Exception as e:
            print(f"✗ Error creating indexes: {e}")
            return False

    def verify_insertion(self) -> Dict[str, Any]:
        """
        Verify the inserted data by querying the collection.

        Returns:
            Dictionary with verification results
        """
        try:
            total_count = self.collection.count_documents({})
            active_count = self.collection.count_documents({'status': 'active'})

            # Count by category
            pipeline = [
                {'$group': {'_id': '$category', 'count': {'$sum': 1}}},
                {'$sort': {'_id': 1}}
            ]
            category_counts = list(self.collection.aggregate(pipeline))

            # Get sample phenotypes
            sample_phenotypes = list(self.collection.find({}, {'phenotype_id': 1, 'name': 1, 'category': 1}).limit(5))

            return {
                'total_count': total_count,
                'active_count': active_count,
                'category_counts': category_counts,
                'sample_phenotypes': sample_phenotypes
            }

        except Exception as e:
            print(f"✗ Error during verification: {e}")
            return {}

    def close(self):
        """Close the MongoDB connection."""
        if self.client:
            self.client.close()
            print("✓ MongoDB connection closed")


def main():
    """Main execution function."""
    parser = argparse.ArgumentParser(description='Insert KB2 phenotype data into MongoDB')
    parser.add_argument('--dry-run', action='store_true', help='Validate data without inserting')
    parser.add_argument('--collection-name', default='phenotype_definitions', help='MongoDB collection name')
    parser.add_argument('--json-dir', help='Directory containing JSON files (default: ../sample-data/phenotypes-json/)')

    args = parser.parse_args()

    # Set up paths
    script_dir = Path(__file__).parent
    if args.json_dir:
        json_dir = args.json_dir
    else:
        json_dir = script_dir.parent / "sample-data" / "phenotypes-json"

    if not os.path.exists(json_dir):
        print(f"✗ JSON directory not found: {json_dir}")
        sys.exit(1)

    # Get MongoDB URI from environment
    mongodb_uri = os.getenv(
        'MONGODB_URI',
        'mongodb://kb2admin:kb2_mongodb_password@localhost:27018/kb2_clinical_context?authSource=admin'
    )

    # Initialize inserter
    inserter = KB2PhenotypeInserter(mongodb_uri, args.collection_name)

    try:
        # Connect to MongoDB
        if not inserter.connect():
            sys.exit(1)

        # Load JSON files
        phenotypes = inserter.load_json_files(json_dir)
        if not phenotypes:
            sys.exit(1)

        # Insert phenotypes
        stats = inserter.insert_phenotypes(phenotypes, args.dry_run)

        # Print results
        print(f"\n📋 INSERTION SUMMARY:")
        print(f"   Total phenotypes: {stats['total']}")
        print(f"   Inserted: {stats['inserted']}")
        print(f"   Updated: {stats['updated']}")
        print(f"   Skipped: {stats['skipped']}")
        print(f"   Errors: {stats['errors']}")

        if not args.dry_run and stats['errors'] == 0:
            # Create indexes
            inserter.create_indexes()

            # Verify insertion
            verification = inserter.verify_insertion()
            if verification:
                print(f"\n✅ VERIFICATION RESULTS:")
                print(f"   Total documents: {verification['total_count']}")
                print(f"   Active phenotypes: {verification['active_count']}")
                print(f"   Categories:")
                for cat in verification['category_counts']:
                    print(f"     - {cat['_id']}: {cat['count']}")

                if verification['sample_phenotypes']:
                    print(f"   Sample phenotypes:")
                    for pheno in verification['sample_phenotypes']:
                        print(f"     - {pheno['phenotype_id']}: {pheno['name']} ({pheno['category']})")

        success = stats['errors'] == 0
        sys.exit(0 if success else 1)

    except KeyboardInterrupt:
        print("\n⚠ Operation cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"✗ Unexpected error: {e}")
        sys.exit(1)
    finally:
        inserter.close()


if __name__ == '__main__':
    main()