// Import fetch with ESM syntax for node-fetch v3
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

// Configure logging
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  debug: (message) => console.debug(`[DEBUG] ${new Date().toISOString()} - ${message}`)
};

// Base URL for the Patient Service
const PATIENT_SERVICE_URL = process.env.PATIENT_SERVICE_URL || 'http://localhost:8003';

// We'll use the patient service directly since the API Gateway is not accessible
const DIRECT_FHIR_URL = process.env.DIRECT_FHIR_URL || 'http://localhost:8003';

// Use the regular GraphQL endpoint for operations (not federation)
const PATIENT_GRAPHQL_URL = `${PATIENT_SERVICE_URL}/api/graphql`;

// Helper function to make a fetch request with authentication headers
async function makeFetchRequest(url, options = {}, context) {
  // Set up headers with authentication
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers
  };

  // Add authorization header if token is available
  if (context && context.token) {
    headers['Authorization'] = context.token;
    logger.debug('Authorization header added to request');
  }

  // Add user info headers if available
  if (context && context.userId) {
    headers['X-User-ID'] = context.userId;
  }
  if (context && context.userRole) {
    headers['X-User-Role'] = context.userRole;
  }
  if (context && context.userRoles) {
    headers['X-User-Roles'] = Array.isArray(context.userRoles)
      ? JSON.stringify(context.userRoles)
      : context.userRoles;
  }
  if (context && context.userPermissions) {
    headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
      ? JSON.stringify(context.userPermissions)
      : context.userPermissions;
  }

  try {
    const startTime = Date.now();
    const response = await fetch(url, {
      ...options,
      headers
    });
    const responseTime = Date.now() - startTime;

    logger.debug(`Response received in ${responseTime}ms with status ${response.status}`);

    if (!response.ok) {
      const errorText = await response.text();
      logger.error(`Service error: ${response.status}`, new Error(errorText));
      throw new Error(`Service error: ${response.status} ${errorText}`);
    }

    const data = await response.json();
    return data;
  } catch (error) {
    logger.error(`Error fetching from service: ${error.message}`, error);
    throw error;
  }
}

// Helper function to make authenticated requests to the Patient Service
async function fetchFromPatientService(path, options = {}, context) {
  // Remove any /api/graphql prefix that might be added
  let cleanPath = path;
  if (cleanPath.includes('/api/graphql')) {
    cleanPath = cleanPath.replace('/api/graphql', '');
  }

  // Make sure we don't have duplicate /api prefixes
  if (cleanPath.startsWith('/api') && PATIENT_SERVICE_URL.endsWith('/api')) {
    cleanPath = cleanPath.replace('/api', '');
  }

  // For FHIR endpoints, use the direct FHIR URL
  if (cleanPath.includes('/fhir/')) {
    const fhirPath = cleanPath.includes('/api/fhir/')
      ? cleanPath
      : `/api${cleanPath}`;
    const url = `${DIRECT_FHIR_URL}${fhirPath}`;
    logger.debug(`Making direct FHIR request to: ${options.method || 'GET'} ${url}`);
    return makeFetchRequest(url, options, context);
  }

  const url = `${PATIENT_SERVICE_URL}${cleanPath}`;
  logger.debug(`Making request to Patient Service: ${options.method || 'GET'} ${url}`);

  // Set up headers with authentication
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers
  };

  // Add authorization header if token is available
  if (context && context.token) {
    headers['Authorization'] = context.token;
    logger.debug('Authorization header added to request');
  }

  // Add user info headers if available
  if (context && context.userId) {
    headers['X-User-ID'] = context.userId;
  }
  if (context && context.userRole) {
    headers['X-User-Role'] = context.userRole;
  }
  if (context && context.userRoles) {
    headers['X-User-Roles'] = JSON.stringify(context.userRoles);
  }
  if (context && context.userPermissions) {
    headers['X-User-Permissions'] = JSON.stringify(context.userPermissions);
  }

  try {
    const startTime = Date.now();
    const response = await fetch(url, {
      ...options,
      headers
    });
    const responseTime = Date.now() - startTime;

    logger.debug(`Response received in ${responseTime}ms with status ${response.status}`);

    if (!response.ok) {
      const errorText = await response.text();
      logger.error(`Patient service error: ${response.status}`, new Error(errorText));
      throw new Error(`Patient service error: ${response.status} ${errorText}`);
    }

    const data = await response.json();
    return data;
  } catch (error) {
    logger.error(`Error fetching from patient service: ${error.message}`, error);
    throw error;
  }
}

// Resolvers for the Patient Service
const resolvers = {
  Query: {
    // Get patients with pagination
    patients: async (_, { page = 1, limit = 10, generalPractitioner }, context) => {
      logger.info(`Query: patients(page: ${page}, limit: ${limit}, generalPractitioner: ${generalPractitioner || 'none'})`);

      try {
        // Construct the URL directly using the patient service's FHIR endpoint
        const url = `${DIRECT_FHIR_URL}/api/fhir/Patient?_count=${limit}&_page=${page}${
          generalPractitioner ? `&generalPractitioner=${encodeURIComponent(generalPractitioner)}` : ''
        }`;

        if (generalPractitioner) {
          logger.debug(`Added generalPractitioner filter: ${generalPractitioner}`);
        }

        logger.debug(`Making direct request to: ${url}`);

        // Set up headers with authentication
        const headers = {
          'Content-Type': 'application/json',
        };

        // Add authorization header if token is available
        if (context && context.token) {
          headers['Authorization'] = context.token;
          logger.debug('Authorization header added to request');
        }

        // Add user info headers if available
        if (context && context.userId) {
          headers['X-User-ID'] = context.userId;
        }
        if (context && context.userRole) {
          headers['X-User-Role'] = context.userRole;
        }
        if (context && context.userRoles) {
          headers['X-User-Roles'] = Array.isArray(context.userRoles)
            ? JSON.stringify(context.userRoles)
            : context.userRoles;
        }
        if (context && context.userPermissions) {
          headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
            ? JSON.stringify(context.userPermissions)
            : context.userPermissions;
        }

        // Make the request directly
        const response = await fetch(url, {
          method: 'GET',
          headers
        });

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`Error retrieving patients: ${response.status}`, new Error(errorText));
          throw new Error(`Error retrieving patients: ${response.status} ${errorText}`);
        }

        const data = await response.json();

        // Log the raw response for debugging
        logger.debug(`Raw response from patient service: ${JSON.stringify(data).substring(0, 500)}...`);

        // Format response to match GraphQL schema
        // Check if data is an array (direct list of patients) or a Bundle with entry array
        const patients = Array.isArray(data) ? data : (data.entry ? data.entry.map(entry => entry.resource) : []);
        const total = Array.isArray(data) ? data.length : (data.total || 0);
        const count = patients.length;

        const result = {
          items: patients,
          total: total,
          page: page,
          count: count
        };

        logger.debug(`Returning ${result.items.length} patients out of ${result.total} total`);
        return result;
      } catch (error) {
        logger.error(`Error in patients query:`, error);
        // Return empty result on error
        return {
          items: [],
          total: 0,
          page: page,
          count: 0
        };
      }
    },

    // Get a single patient by ID
    patient: async (_, { id }, context) => {
      logger.info(`Query: patient(id: ${id})`);

      try {
        // Use direct FHIR URL to avoid path issues
        const url = `${DIRECT_FHIR_URL}/api/fhir/Patient/${id}`;
        logger.debug(`Making direct request to: ${url}`);

        // Set up headers with authentication
        const headers = {
          'Content-Type': 'application/json',
        };

        // Add authorization header if token is available
        if (context && context.token) {
          headers['Authorization'] = context.token;
          logger.debug('Authorization header added to request');
        }

        // Add user info headers if available
        if (context && context.userId) {
          headers['X-User-ID'] = context.userId;
        }
        if (context && context.userRole) {
          headers['X-User-Role'] = context.userRole;
        }
        if (context && context.userRoles) {
          headers['X-User-Roles'] = Array.isArray(context.userRoles)
            ? JSON.stringify(context.userRoles)
            : context.userRoles;
        }
        if (context && context.userPermissions) {
          headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
            ? JSON.stringify(context.userPermissions)
            : context.userPermissions;
        }

        // Make the request directly
        const response = await fetch(url, {
          method: 'GET',
          headers
        });

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`Error retrieving patient: ${response.status}`, new Error(errorText));
          throw new Error(`Error retrieving patient: ${response.status} ${errorText}`);
        }

        const data = await response.json();
        logger.debug(`Retrieved patient with ID ${id}`);

        // Log the raw response for debugging
        logger.debug(`Raw response from patient service: ${JSON.stringify(data).substring(0, 500)}...`);

        return data;
      } catch (error) {
        logger.error(`Error retrieving patient with ID ${id}:`, error);
        return null;
      }
    },

    // Search patients by name
    searchPatients: async (_, { name }, context) => {
      logger.info(`Query: searchPatients(name: ${name})`);

      try {
        // Use direct FHIR URL to avoid path issues
        const url = `${DIRECT_FHIR_URL}/api/fhir/Patient?name=${encodeURIComponent(name)}`;
        logger.debug(`Making direct request to: ${url}`);

        // Set up headers with authentication
        const headers = {
          'Content-Type': 'application/json',
        };

        // Add authorization header if token is available
        if (context && context.token) {
          headers['Authorization'] = context.token;
          logger.debug('Authorization header added to request');
        }

        // Add user info headers if available
        if (context && context.userId) {
          headers['X-User-ID'] = context.userId;
        }
        if (context && context.userRole) {
          headers['X-User-Role'] = context.userRole;
        }
        if (context && context.userRoles) {
          headers['X-User-Roles'] = Array.isArray(context.userRoles)
            ? JSON.stringify(context.userRoles)
            : context.userRoles;
        }
        if (context && context.userPermissions) {
          headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
            ? JSON.stringify(context.userPermissions)
            : context.userPermissions;
        }

        // Make the request directly
        const response = await fetch(url, {
          method: 'GET',
          headers
        });

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`Error searching patients: ${response.status}`, new Error(errorText));
          throw new Error(`Error searching patients: ${response.status} ${errorText}`);
        }

        const data = await response.json();

        // Return array of patient resources
        const patients = data.entry ? data.entry.map(entry => entry.resource) : [];
        logger.debug(`Found ${patients.length} patients matching name "${name}"`);
        return patients;
      } catch (error) {
        logger.error(`Error searching patients by name "${name}":`, error);
        return [];
      }
    }
  },

  Mutation: {
    // Create a new patient
    createPatient: async (_, { input }, context) => {
      logger.info(`Mutation: createPatient`);
      logger.debug(`Patient data: ${JSON.stringify(input).substring(0, 200)}...`);

      try {
        // Use direct FHIR URL to avoid path issues
        const url = `${DIRECT_FHIR_URL}/api/fhir/Patient`;
        logger.debug(`Making direct request to: ${url}`);

        // Set up headers with authentication
        const headers = {
          'Content-Type': 'application/json',
        };

        // Add authorization header if token is available
        if (context && context.token) {
          headers['Authorization'] = context.token;
          logger.debug('Authorization header added to request');
        }

        // Add user info headers if available
        if (context && context.userId) {
          headers['X-User-ID'] = context.userId;
        }
        if (context && context.userRole) {
          headers['X-User-Role'] = context.userRole;
        }
        if (context && context.userRoles) {
          headers['X-User-Roles'] = Array.isArray(context.userRoles)
            ? JSON.stringify(context.userRoles)
            : context.userRoles;
        }
        if (context && context.userPermissions) {
          headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
            ? JSON.stringify(context.userPermissions)
            : context.userPermissions;
        }

        // Make the request directly
        const response = await fetch(url, {
          method: 'POST',
          headers,
          body: JSON.stringify(input)
        });

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`Error creating patient: ${response.status}`, new Error(errorText));
          throw new Error(`Error creating patient: ${response.status} ${errorText}`);
        }

        const data = await response.json();
        logger.info(`Created patient with ID ${data.id}`);
        return { patient: data };
      } catch (error) {
        logger.error(`Error creating patient:`, error);
        throw error;
      }
    },

    // Update an existing patient
    updatePatient: async (_, { id, input }, context) => {
      logger.info(`Mutation: updatePatient(id: ${id})`);

      try {
        // Ensure ID is included in the resource
        const resourceWithId = {
          ...input,
          id
        };

        logger.debug(`Updating patient with data: ${JSON.stringify(resourceWithId).substring(0, 200)}...`);

        // Use direct FHIR URL to avoid path issues
        const url = `${DIRECT_FHIR_URL}/api/fhir/Patient/${id}`;
        logger.debug(`Making direct request to: ${url}`);

        // Set up headers with authentication
        const headers = {
          'Content-Type': 'application/json',
        };

        // Add authorization header if token is available
        if (context && context.token) {
          headers['Authorization'] = context.token;
          logger.debug('Authorization header added to request');
        }

        // Add user info headers if available
        if (context && context.userId) {
          headers['X-User-ID'] = context.userId;
        }
        if (context && context.userRole) {
          headers['X-User-Role'] = context.userRole;
        }
        if (context && context.userRoles) {
          headers['X-User-Roles'] = Array.isArray(context.userRoles)
            ? JSON.stringify(context.userRoles)
            : context.userRoles;
        }
        if (context && context.userPermissions) {
          headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
            ? JSON.stringify(context.userPermissions)
            : context.userPermissions;
        }

        // Make the request directly
        const response = await fetch(url, {
          method: 'PUT',
          headers,
          body: JSON.stringify(resourceWithId)
        });

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`Error updating patient: ${response.status}`, new Error(errorText));
          throw new Error(`Error updating patient: ${response.status} ${errorText}`);
        }

        const data = await response.json();
        logger.info(`Updated patient with ID ${id}`);
        return { patient: data };
      } catch (error) {
        logger.error(`Error updating patient with ID ${id}:`, error);
        throw error;
      }
    },

    // Delete a patient
    deletePatient: async (_, { id }, context) => {
      logger.info(`Mutation: deletePatient(id: ${id})`);

      try {
        // Use direct FHIR URL to avoid path issues
        const url = `${DIRECT_FHIR_URL}/api/fhir/Patient/${id}`;
        logger.debug(`Making direct request to: ${url}`);

        // Set up headers with authentication
        const headers = {
          'Content-Type': 'application/json',
        };

        // Add authorization header if token is available
        if (context && context.token) {
          headers['Authorization'] = context.token;
          logger.debug('Authorization header added to request');
        }

        // Add user info headers if available
        if (context && context.userId) {
          headers['X-User-ID'] = context.userId;
        }
        if (context && context.userRole) {
          headers['X-User-Role'] = context.userRole;
        }
        if (context && context.userRoles) {
          headers['X-User-Roles'] = Array.isArray(context.userRoles)
            ? JSON.stringify(context.userRoles)
            : context.userRoles;
        }
        if (context && context.userPermissions) {
          headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
            ? JSON.stringify(context.userPermissions)
            : context.userPermissions;
        }

        // Make the request directly
        const response = await fetch(url, {
          method: 'DELETE',
          headers
        });

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`Error deleting patient: ${response.status}`, new Error(errorText));
          throw new Error(`Error deleting patient: ${response.status} ${errorText}`);
        }

        logger.info(`Successfully deleted patient with ID ${id}`);
        return {
          success: true,
          message: `Patient with ID ${id} successfully deleted`
        };
      } catch (error) {
        logger.error(`Error deleting patient with ID ${id}:`, error);
        return {
          success: false,
          message: error.message
        };
      }
    }
  },

  // Resolve references if needed
  Patient: {
    __resolveReference: async (reference, { context }) => {
      const { id } = reference;
      logger.debug(`Resolving reference for Patient with ID: ${id}`);

      try {
        // Use direct FHIR URL to avoid path issues
        const url = `${DIRECT_FHIR_URL}/api/fhir/Patient/${id}`;
        logger.debug(`Making direct request to: ${url}`);

        // Set up headers with authentication
        const headers = {
          'Content-Type': 'application/json',
        };

        // Add authorization header if token is available
        if (context && context.token) {
          headers['Authorization'] = context.token;
          logger.debug('Authorization header added to request');
        }

        // Add user info headers if available
        if (context && context.userId) {
          headers['X-User-ID'] = context.userId;
        }
        if (context && context.userRole) {
          headers['X-User-Role'] = context.userRole;
        }
        if (context && context.userRoles) {
          headers['X-User-Roles'] = Array.isArray(context.userRoles)
            ? JSON.stringify(context.userRoles)
            : context.userRoles;
        }
        if (context && context.userPermissions) {
          headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
            ? JSON.stringify(context.userPermissions)
            : context.userPermissions;
        }

        // Make the request directly
        const response = await fetch(url, {
          method: 'GET',
          headers
        });

        if (!response.ok) {
          const errorText = await response.text();
          logger.error(`Error resolving reference: ${response.status}`, new Error(errorText));
          throw new Error(`Error resolving reference: ${response.status} ${errorText}`);
        }

        const data = await response.json();
        logger.debug(`Successfully resolved reference for Patient with ID: ${id}`);
        return data;
      } catch (error) {
        logger.error(`Error resolving reference for Patient with ID ${id}:`, error);
        return null;
      }
    }
  }
};

module.exports = resolvers;
