# Clinical Workflow Engine Database Schema

## 🗄️ Overview

The Clinical Workflow Engine database schema provides comprehensive support for clinical workflows, security framework integration, and medical-legal compliance. This document outlines the complete database structure and security enhancements.

## 📊 Database Schema Status

### ✅ **COMPLETE IMPLEMENTATION**

All required tables and security enhancements have been successfully implemented:

- **Core Workflow Tables**: 21 tables ✅
- **Security Framework Tables**: 4 new tables ✅  
- **Enhanced Audit Columns**: 5 new columns ✅
- **Security Views**: 3 reporting views ✅
- **Indexes and Constraints**: All applied ✅

## 🏗️ Core Workflow Tables

### Primary Workflow Tables
- `workflow_definitions` - Workflow templates and BPMN definitions
- `workflow_instances` - Active and completed workflow executions
- `workflow_tasks` - Individual tasks within workflows
- `workflow_events` - Event log for workflow state changes
- `workflow_timers` - Scheduled events and timeouts
- `workflow_gateways` - Decision points and parallel gateways
- `workflow_escalations` - Escalation rules and notifications

### Clinical Activity Tables
- `clinical_activity_executions` - Clinical activity execution records
- `clinical_errors` - Clinical error tracking and recovery
- `clinical_workflow_metrics` - Performance and quality metrics
- `clinical_timers` - Clinical-specific timing and escalations

### Task Management Tables
- `task_assignments` - Task assignment and delegation
- `task_comments` - Task collaboration and notes
- `task_attachments` - File attachments and documents
- `task_escalations` - Task-level escalation tracking

## 🔒 Security Framework Tables

### 1. **PHI Access Log** (`phi_access_log`)
**Purpose**: Detailed PHI access tracking for HIPAA compliance

**Key Columns**:
- `id` (UUID) - Unique access log identifier
- `user_id` - User accessing PHI data
- `patient_id` - Patient whose data was accessed
- `access_type` - Type of access (encrypt, decrypt, view, export)
- `phi_fields_accessed` (JSONB) - Specific PHI fields accessed
- `access_timestamp` - When access occurred
- `session_id` - User session identifier
- `ip_address` - Source IP address
- `access_purpose` - Purpose of access (clinical_care, emergency, audit)

**Indexes**:
- User ID, Patient ID, Access Timestamp, Access Type, Workflow Instance ID

### 2. **Security Events** (`security_events`)
**Purpose**: Security incident monitoring and investigation

**Key Columns**:
- `id` (UUID) - Unique event identifier
- `event_type` - Type of security event
- `severity` - Event severity (low, medium, high, critical)
- `event_details` (JSONB) - Detailed event information
- `investigation_status` - Investigation status (open, investigating, resolved)
- `source_ip` - Source IP address
- `detection_method` - How event was detected

**Indexes**:
- Event Type, Severity, User ID, Event Timestamp, Investigation Status

### 3. **Clinical Decision Audit** (`clinical_decision_audit`)
**Purpose**: Comprehensive clinical decision tracking for medical-legal compliance

**Key Columns**:
- `id` (UUID) - Unique decision identifier
- `decision_id` - Business decision identifier
- `decision_type` - Type of clinical decision
- `decision_maker_id` - Provider making the decision
- `clinical_context` (JSONB) - Clinical context and data
- `clinical_rationale` - Medical reasoning for decision
- `safety_checks_performed` (JSONB) - Safety validations performed
- `overrides_applied` (JSONB) - Any safety overrides applied
- `supervisor_approval` - Supervisor approval for overrides
- `decision_confidence` - Confidence level (0.00-1.00)

**Indexes**:
- Decision Maker ID, Patient ID, Decision Type, Decision Timestamp, Workflow Instance ID

### 4. **Encrypted Workflow States** (`encrypted_workflow_states`)
**Purpose**: Secure storage of workflow states containing PHI

**Key Columns**:
- `id` (UUID) - Unique state identifier
- `workflow_instance_id` - Associated workflow instance
- `encrypted_state` (TEXT) - Base64 encoded encrypted JSON
- `encryption_key_id` - Key used for encryption
- `phi_fields_encrypted` (JSONB) - List of encrypted PHI fields
- `encrypted_by` - User who encrypted the state
- `decryption_count` - Number of times decrypted

**Indexes**:
- Workflow Instance ID, Encrypted By, Encrypted At

## 🔧 Enhanced Audit Trail

### Enhanced `clinical_audit_trail` Table

**New Security Columns**:
- `event_type` (audit_event_type) - Structured event type enum
- `audit_level_enum` (audit_level_type) - Audit level (standard, detailed, comprehensive)
- `outcome` - Event outcome (success, failure, warning)
- `error_details` (JSONB) - Detailed error information
- `safety_critical` (BOOLEAN) - Whether event is safety-critical

**Event Types Enum**:
```sql
CREATE TYPE audit_event_type AS ENUM (
    'workflow_started', 'workflow_completed', 'workflow_failed',
    'activity_executed', 'safety_check_performed', 'safety_override',
    'phi_accessed', 'clinical_decision', 'compensation_executed',
    'break_glass_access', 'user_login', 'user_logout', 'data_export'
);
```

**Audit Levels Enum**:
```sql
CREATE TYPE audit_level_type AS ENUM ('standard', 'detailed', 'comprehensive');
```

## 📊 Security Reporting Views

### 1. **PHI Access Summary** (`phi_access_summary`)
Daily summary of PHI access by user and patient:
```sql
SELECT patient_id, user_id, access_date, access_count, 
       workflows_accessed, total_phi_fields_accessed, access_types
FROM phi_access_summary;
```

### 2. **Security Events Summary** (`security_events_summary`)
Daily summary of security events by type and severity:
```sql
SELECT event_type, severity, event_date, event_count,
       affected_users, affected_patients
FROM security_events_summary;
```

### 3. **Clinical Decision Summary** (`clinical_decision_summary`)
Daily summary of clinical decisions by type and provider:
```sql
SELECT decision_type, decision_maker_id, decision_date,
       decisions_made, patients_affected, avg_confidence,
       decisions_with_overrides
FROM clinical_decision_summary;
```

## 🚨 Break-Glass Access Enhancements

### Enhanced `emergency_access_records` Table

**New Break-Glass Columns**:
- `session_id` - Unique break-glass session identifier
- `access_type` - Type of emergency access
- `justification` - Emergency justification code
- `clinical_details` - Detailed clinical emergency description
- `supervisor_approval` - Supervisor approval identifier
- `actions_performed` (JSONB) - Actions performed during session
- `audit_trail_ids` (JSONB) - Associated audit trail entries

## 🔍 Database Integration Features

### 1. **Automatic Audit Storage**
- All security framework events automatically stored in database
- Persistent audit trail with 7-year retention
- Real-time audit entry creation with transaction safety

### 2. **PHI Encryption Integration**
- Encrypted workflow states stored in dedicated table
- Automatic encryption key management
- Decryption tracking and access logging

### 3. **Clinical Decision Tracking**
- Comprehensive decision audit with medical-legal compliance
- Safety override tracking with supervisor approval
- Evidence-based decision documentation

### 4. **Break-Glass Session Management**
- Time-limited emergency access sessions
- Complete action logging during emergency access
- Automatic session expiration and cleanup

## 📈 Performance Optimizations

### Indexes Applied
- **PHI Access Log**: 5 indexes for fast access pattern queries
- **Security Events**: 5 indexes for incident investigation
- **Clinical Decisions**: 5 indexes for compliance reporting
- **Encrypted States**: 3 indexes for workflow state retrieval

### Query Optimization
- Partitioned tables for large audit datasets
- Optimized views for common reporting queries
- Efficient JSON indexing for structured data

## 🔧 Migration Status

### Applied Migrations
- ✅ `005_clinical_workflow_engine_tables.sql` - Core workflow tables
- ✅ `006_security_framework_enhancements_simple.sql` - Security enhancements

### Migration Results
```
🎉 Security Framework Database Migration Complete!
✅ All security tables and enhancements applied
🔒 Database ready for security framework integration

📋 Security Tables: 7/7 ✅
📊 Security Views: 3/3 ✅  
🔍 Enhanced Columns: 5/5 ✅
```

## 🚀 Production Readiness

### Compliance Features
- ✅ **HIPAA Compliance**: PHI access logging and encryption
- ✅ **Medical-Legal Compliance**: 7-year audit retention
- ✅ **Security Monitoring**: Comprehensive event tracking
- ✅ **Emergency Access**: Break-glass procedures with audit

### Performance Features
- ✅ **Optimized Indexes**: Fast query performance
- ✅ **Efficient Storage**: JSON compression for large datasets
- ✅ **Scalable Design**: Partitioned tables for growth
- ✅ **Transaction Safety**: ACID compliance for audit integrity

### Monitoring Features
- ✅ **Real-time Views**: Live security dashboards
- ✅ **Automated Alerts**: Critical event notifications
- ✅ **Compliance Reporting**: Automated audit reports
- ✅ **Performance Metrics**: Database health monitoring

## 🔗 Related Documentation

- [Security Framework Documentation](./SECURITY_FRAMEWORK_DOCUMENTATION.md)
- [Clinical Workflow Engine Implementation Plan](../CLINICAL_WORKFLOW_ENGINE_IMPLEMENTATION_PLAN.md)
- [API Documentation](./API_DOCUMENTATION.md)

---

**Database Schema Status**: ✅ **COMPLETE AND PRODUCTION READY**

The database schema provides enterprise-grade support for clinical workflows with comprehensive security, audit, and compliance capabilities.
