const axios = require('axios');

const MEDICATION_SERVICE_URL = process.env.MEDICATION_SERVICE_URL || 'http://localhost:8009';

const resolvers = {
  Patient: {
    __resolveReference(patient, { dataSources }) {
      return { id: patient.id };
    },
    
    async medications(patient, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-requests/patient/${patient.id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching patient medications:', error);
        return [];
      }
    },
    
    async medicationStatements(patient, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-statements/patient/${patient.id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching patient medication statements:', error);
        return [];
      }
    },
    
    async medicationAdministrations(patient, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-administrations/patient/${patient.id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching patient medication administrations:', error);
        return [];
      }
    },
    
    async allergies(patient, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/allergies/patient/${patient.id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching patient allergies:', error);
        return [];
      }
    }
  },
  
  Query: {
    async medications(parent, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medications`,
          {
            params: args,
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medications:', error);
        return { items: [], total: 0, page: 1, count: 0 };
      }
    },
    
    async medication(parent, { id }, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medications/${id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medication:', error);
        return null;
      }
    },
    
    async medicationRequests(parent, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-requests`,
          {
            params: args,
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medication requests:', error);
        return { items: [], total: 0, page: 1, count: 0 };
      }
    },
    
    async medicationRequest(parent, { id }, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-requests/${id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medication request:', error);
        return null;
      }
    },
    
    async medicationStatements(parent, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-statements`,
          {
            params: args,
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medication statements:', error);
        return { items: [], total: 0, page: 1, count: 0 };
      }
    },
    
    async medicationStatement(parent, { id }, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-statements/${id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medication statement:', error);
        return null;
      }
    },
    
    async medicationAdministrations(parent, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-administrations`,
          {
            params: args,
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medication administrations:', error);
        return { items: [], total: 0, page: 1, count: 0 };
      }
    },
    
    async medicationAdministration(parent, { id }, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/medication-administrations/${id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching medication administration:', error);
        return null;
      }
    },
    
    async allergies(parent, args, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/allergies`,
          {
            params: args,
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching allergies:', error);
        return { items: [], total: 0, page: 1, count: 0 };
      }
    },
    
    async allergy(parent, { id }, context) {
      try {
        const response = await axios.get(
          `${MEDICATION_SERVICE_URL}/api/allergies/${id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching allergy:', error);
        return null;
      }
    }
  },
  
  Mutation: {
    async createMedication(parent, { medicationData }, context) {
      try {
        const response = await axios.post(
          `${MEDICATION_SERVICE_URL}/api/medications`,
          medicationData,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error creating medication:', error);
        throw new Error('Failed to create medication');
      }
    },
    
    async createMedicationRequest(parent, { medicationRequestData }, context) {
      try {
        const response = await axios.post(
          `${MEDICATION_SERVICE_URL}/api/medication-requests`,
          medicationRequestData,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error creating medication request:', error);
        throw new Error('Failed to create medication request');
      }
    },
    
    async createMedicationStatement(parent, { medicationStatementData }, context) {
      try {
        const response = await axios.post(
          `${MEDICATION_SERVICE_URL}/api/medication-statements`,
          medicationStatementData,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error creating medication statement:', error);
        throw new Error('Failed to create medication statement');
      }
    },
    
    async createMedicationAdministration(parent, { medicationAdministrationData }, context) {
      try {
        const response = await axios.post(
          `${MEDICATION_SERVICE_URL}/api/medication-administrations`,
          medicationAdministrationData,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error creating medication administration:', error);
        throw new Error('Failed to create medication administration');
      }
    },
    
    async createAllergy(parent, { allergyData }, context) {
      try {
        const response = await axios.post(
          `${MEDICATION_SERVICE_URL}/api/allergies`,
          allergyData,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error creating allergy:', error);
        throw new Error('Failed to create allergy');
      }
    }
  }
};

module.exports = resolvers;
