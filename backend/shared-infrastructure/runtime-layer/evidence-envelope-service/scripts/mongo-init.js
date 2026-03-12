// MongoDB initialization script for Evidence Envelope Service
// Creates database, collections, and indexes for optimal performance

// Switch to evidence_envelopes database
db = db.getSiblingDB('evidence_envelopes');

// Create collections with validation
db.createCollection('envelopes', {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["envelope_id", "proposal_id", "status", "created_at"],
      properties: {
        envelope_id: {
          bsonType: "string",
          description: "Unique envelope identifier"
        },
        proposal_id: {
          bsonType: "string",
          description: "Clinical proposal identifier"
        },
        status: {
          bsonType: "string",
          enum: ["active", "finalized"],
          description: "Envelope status"
        },
        created_at: {
          bsonType: "date",
          description: "Creation timestamp"
        }
      }
    }
  }
});

db.createCollection('audit_records', {
  validator: {
    $jsonSchema: {
      bsonType: "object",
      required: ["envelope_id", "audit_type", "timestamp"],
      properties: {
        envelope_id: {
          bsonType: "string",
          description: "Related envelope identifier"
        },
        audit_type: {
          bsonType: "string",
          description: "Type of audit record"
        },
        timestamp: {
          bsonType: "date",
          description: "Audit event timestamp"
        }
      }
    }
  }
});

// Create indexes for envelopes collection
db.envelopes.createIndex(
  { "envelope_id": 1 },
  { unique: true, name: "envelope_id_unique" }
);

db.envelopes.createIndex(
  { "proposal_id": 1, "created_at": -1 },
  { name: "proposal_created_idx" }
);

db.envelopes.createIndex(
  { "clinical_context.patient_id": 1, "clinical_context.workflow_type": 1, "status": 1 },
  { name: "patient_workflow_status_idx" }
);

db.envelopes.createIndex(
  { "status": 1, "created_at": -1 },
  { name: "status_created_idx" }
);

// TTL index for automated cleanup (7 years = 220898400 seconds)
db.envelopes.createIndex(
  { "created_at": 1 },
  {
    expireAfterSeconds: 220898400,
    name: "created_at_ttl"
  }
);

// Create indexes for audit_records collection
db.audit_records.createIndex(
  { "envelope_id": 1, "timestamp": -1 },
  { name: "audit_envelope_time_idx" }
);

db.audit_records.createIndex(
  { "audit_type": 1, "timestamp": -1 },
  { name: "audit_type_time_idx" }
);

// TTL index for audit records (7 years)
db.audit_records.createIndex(
  { "timestamp": 1 },
  {
    expireAfterSeconds: 220898400,
    name: "audit_timestamp_ttl"
  }
);

// Create application user with appropriate permissions
db.createUser({
  user: "evidence_service",
  pwd: "evidence_service_password_change_in_production",
  roles: [
    {
      role: "readWrite",
      db: "evidence_envelopes"
    }
  ]
});

print("Evidence Envelope Service MongoDB initialization completed successfully");
print("Created collections: envelopes, audit_records");
print("Created indexes for optimal query performance");
print("Created application user: evidence_service");
print("Configured TTL indexes for automatic data retention compliance");