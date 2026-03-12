// Import all resolvers
const patientResolvers = require('./patient-resolvers');
const kb2ClinicalContextResolvers = require('./kb2-clinical-context-resolvers');
const kb7TerminologyResolvers = require('./kb7-terminology-resolvers');

// Export all resolvers
module.exports = {
  patientResolvers,
  kb2ClinicalContextResolvers,
  kb7TerminologyResolvers
};
