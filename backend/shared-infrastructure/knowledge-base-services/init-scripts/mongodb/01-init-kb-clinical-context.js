// Switch to the kb_clinical_context database
db = db.getSiblingDB('kb_clinical_context');

// Create user for the application
db.createUser({
  user: 'kb_context_user',
  pwd: 'kb_context_password',
  roles: [
    {
      role: 'readWrite',
      db: 'kb_clinical_context'
    }
  ]
});

// Create collections with validation
db.createCollection('phenotype_definitions', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['phenotype_id', 'name', 'version', 'criteria', 'status'],
      properties: {
        phenotype_id: {
          bsonType: 'string',
          description: 'Unique identifier for phenotype'
        },
        name: {
          bsonType: 'string',
          description: 'Human-readable phenotype name'
        },
        version: {
          bsonType: 'string',
          pattern: '^\\d+\\.\\d+\\.\\d+$'
        },
        criteria: {
          bsonType: 'object',
          description: 'Phenotype detection criteria'
        },
        status: {
          enum: ['active', 'draft', 'deprecated']
        }
      }
    }
  }
});

db.createCollection('patient_contexts', {
  validator: {
    $jsonSchema: {
      bsonType: 'object',
      required: ['patient_id', 'context_id', 'timestamp'],
      properties: {
        patient_id: { bsonType: 'string' },
        context_id: { bsonType: 'string' },
        timestamp: { bsonType: 'date' }
      }
    }
  }
});

// Create indexes
db.phenotype_definitions.createIndex({ 'phenotype_id': 1, 'version': -1 });
db.phenotype_definitions.createIndex({ 'status': 1 });
db.patient_contexts.createIndex({ 'patient_id': 1, 'timestamp': -1 });
db.patient_contexts.createIndex({ 'context_id': 1 }, { unique: true });

print('MongoDB initialization complete for KB-2 Clinical Context');