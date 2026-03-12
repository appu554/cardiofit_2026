#!/usr/bin/env python3
"""
Script to check what phenotype data is actually in the MongoDB database
"""

import pymongo
import json
from datetime import datetime

def check_phenotypes():
    # MongoDB connection using the same URI as the KB2 service
    MONGODB_URI = "mongodb://kb2admin:kb2_mongodb_password@localhost:27018/kb2_clinical_context?authSource=admin"

    try:
        client = pymongo.MongoClient(MONGODB_URI)
        db = client.kb2_clinical_context

        print("=" * 80)
        print("CHECKING MONGODB DATABASE FOR PHENOTYPE DATA")
        print("=" * 80)

        # Check available collections
        collections = db.list_collection_names()
        print(f"\nAvailable collections: {collections}")

        # Check phenotype_definitions collection specifically
        phenotype_collection = db.phenotype_definitions

        print(f"\nCollection: phenotype_definitions")
        print(f"Document count: {phenotype_collection.count_documents({})}")

        # Get all documents
        all_phenotypes = list(phenotype_collection.find({}))

        if all_phenotypes:
            print(f"\nFound {len(all_phenotypes)} phenotype documents:")
            print("-" * 50)

            for i, phenotype in enumerate(all_phenotypes, 1):
                phenotype_id = phenotype.get('phenotype_id', 'NO_ID')
                name = phenotype.get('name', 'NO_NAME')
                category = phenotype.get('category', 'NO_CATEGORY')
                status = phenotype.get('status', 'NO_STATUS')

                print(f"{i}. ID: {phenotype_id}")
                print(f"   Name: {name}")
                print(f"   Category: {category}")
                print(f"   Status: {status}")

                # Show first few keys to understand structure
                keys = list(phenotype.keys())[:10]
                print(f"   Keys: {keys}")
                print()
        else:
            print("\nNo phenotype documents found!")

        # Check if there are any documents with our HTN IDs
        htn_query = {"phenotype_id": {"$regex": "^PHE-HTN"}}
        htn_docs = list(phenotype_collection.find(htn_query))

        print(f"\nDocuments matching HTN pattern (PHE-HTN*): {len(htn_docs)}")
        if htn_docs:
            for doc in htn_docs:
                print(f"  - {doc.get('phenotype_id')}: {doc.get('name')}")

        # Check indexes
        indexes = list(phenotype_collection.list_indexes())
        print(f"\nIndexes on phenotype_definitions collection:")
        for idx in indexes:
            print(f"  - {idx}")

        client.close()

    except Exception as e:
        print(f"Error connecting to MongoDB: {e}")
        return False

    return True

if __name__ == "__main__":
    check_phenotypes()