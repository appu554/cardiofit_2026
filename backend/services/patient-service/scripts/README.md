# Patient Service Scripts

This directory contains utility scripts for the Patient Service.

## Available Scripts

### Fix Duplicate IDs

The `fix_duplicate_ids.py` script finds and fixes duplicate Patient IDs in the MongoDB database.

#### Usage

```bash
python fix_duplicate_ids.py
```

#### What it does

1. Connects to the MongoDB database
2. Finds all Patient resources with duplicate IDs
3. Keeps the first occurrence of each duplicate ID
4. Assigns new UUIDs to all other occurrences
5. Updates the database with the new IDs

#### When to use

Use this script when:
- You have duplicate Patient IDs in your database
- You're experiencing errors related to duplicate keys
- You're migrating data from another system that might have duplicate IDs

#### Example output

```
2023-05-06 12:34:56 - __main__ - INFO - Starting fix_duplicate_ids script...
2023-05-06 12:34:56 - __main__ - INFO - Connecting to MongoDB...
2023-05-06 12:34:56 - __main__ - INFO - Connected to MongoDB. Database: clinical_synthesis_hub
2023-05-06 12:34:56 - __main__ - INFO - Checking for duplicate Patient IDs in the database...
2023-05-06 12:34:56 - __main__ - WARNING - Found 1 groups of duplicate Patient IDs
2023-05-06 12:34:56 - __main__ - WARNING - Fixing 2 patients with duplicate ID 'example'
2023-05-06 12:34:56 - __main__ - INFO - Updated Patient ID from 'example' to '3f8e7d6c-5b4a-9c8d-7e6f-5d4c3b2a1098'
2023-05-06 12:34:56 - __main__ - INFO - Finished fixing duplicate Patient IDs
2023-05-06 12:34:56 - __main__ - INFO - Script completed.
```

## Adding New Scripts

When adding new scripts to this directory:

1. Follow the same pattern as existing scripts
2. Add proper documentation and logging
3. Update this README with information about the new script
4. Make sure the script can be run independently
5. Add error handling and proper cleanup
