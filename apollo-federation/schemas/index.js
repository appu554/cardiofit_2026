// Import all schema definitions
const patientSchema = require('./patient-schema');
const kb2ClinicalContextSchema = require('./kb2-clinical-context-schema');
const kb7TerminologySchema = require('./kb7-terminology-schema');

// Export all schemas
module.exports = {
  patientSchema,
  kb2ClinicalContextSchema,
  kb7TerminologySchema
};
