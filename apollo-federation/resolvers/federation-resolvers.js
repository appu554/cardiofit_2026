// Federation resolvers for Apollo Federation Gateway
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

// Helper function to make authenticated requests to the Patient Service
async function fetchFromPatientService(path, options = {}, context) {
  const url = `${PATIENT_SERVICE_URL}${path}`;

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

  // Add roles and permissions if available
  if (context && context.userRoles) {
    headers['X-User-Roles'] = Array.isArray(context.userRoles)
      ? context.userRoles.join(',')
      : context.userRoles;
  }

  if (context && context.userPermissions) {
    headers['X-User-Permissions'] = Array.isArray(context.userPermissions)
      ? context.userPermissions.join(',')
      : context.userPermissions;
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`HTTP error ${response.status}: ${errorText}`);
    }

    return await response.json();
  } catch (error) {
    logger.error(`Error fetching from Patient Service: ${error.message}`, error);
    throw error;
  }
}

// Federation resolvers
const federationResolvers = {
  // Resolver for Patient references
  Patient: {
    __resolveReference: async (reference, context) => {
      const { id } = reference;
      logger.debug(`Resolving reference for Patient with ID: ${id}`);

      try {
        const data = await fetchFromPatientService(`/api/direct-fhir/Patient/${id}`, { method: 'GET' }, context);
        logger.debug(`Successfully resolved reference for Patient with ID: ${id}`);
        return data;
      } catch (error) {
        logger.error(`Error resolving reference for Patient with ID ${id}:`, error);
        return null;
      }
    }
  }
};

module.exports = federationResolvers;
