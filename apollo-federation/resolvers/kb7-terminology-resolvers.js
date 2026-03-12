const axios = require('axios');

// KB-7 Terminology Service URL - configurable via environment
const KB7_TERMINOLOGY_URL = process.env.KB7_TERMINOLOGY_URL || 'http://localhost:8087';

const kb7TerminologyResolvers = {
  Query: {
    // Terminology System Operations
    async terminologySystems(parent, args, context) {
      try {
        const response = await axios.get(`${KB7_TERMINOLOGY_URL}/v1/systems`, {
          params: { status: args.status },
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId
          }
        });
        return response.data.systems || [];
      } catch (error) {
        console.error('Error fetching terminology systems:', error);
        throw new Error(`Failed to fetch terminology systems: ${error.message}`);
      }
    },

    async terminologySystem(parent, args, context) {
      try {
        const response = await axios.get(`${KB7_TERMINOLOGY_URL}/v1/systems/${args.identifier}`, {
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId
          }
        });
        return response.data;
      } catch (error) {
        if (error.response && error.response.status === 404) {
          return null;
        }
        console.error('Error fetching terminology system:', error);
        throw new Error(`Failed to fetch terminology system: ${error.message}`);
      }
    },

    // Concept Operations
    async searchConcepts(parent, args, context) {
      try {
        const { input } = args;
        const response = await axios.get(`${KB7_TERMINOLOGY_URL}/v1/concepts`, {
          params: {
            q: input.query,
            system: input.systemUri,
            count: input.count,
            offset: input.offset,
            include_designations: input.includeDesignations,
            include_facets: input.includeFacets
          },
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId
          }
        });
        
        return {
          total: response.data.total || 0,
          concepts: response.data.concepts || [],
          facets: response.data.facets || []
        };
      } catch (error) {
        console.error('Error searching concepts:', error);
        throw new Error(`Failed to search concepts: ${error.message}`);
      }
    },

    async lookupConcept(parent, args, context) {
      try {
        const { system, code } = args;
        const response = await axios.get(`${KB7_TERMINOLOGY_URL}/v1/concepts/${system}/${code}`, {
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId
          }
        });
        return response.data;
      } catch (error) {
        if (error.response && error.response.status === 404) {
          return null;
        }
        console.error('Error looking up concept:', error);
        throw new Error(`Failed to lookup concept: ${error.message}`);
      }
    },

    async validateCode(parent, args, context) {
      try {
        const { input } = args;
        const response = await axios.post(`${KB7_TERMINOLOGY_URL}/v1/concepts/validate`, input, {
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId,
            'Content-Type': 'application/json'
          }
        });
        return response.data;
      } catch (error) {
        console.error('Error validating code:', error);
        throw new Error(`Failed to validate code: ${error.message}`);
      }
    },

    // Value Set Operations
    async valueSets(parent, args, context) {
      try {
        const response = await axios.get(`${KB7_TERMINOLOGY_URL}/v1/valuesets`, {
          params: { 
            domain: args.domain,
            status: args.status 
          },
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId
          }
        });
        return response.data.valueSets || [];
      } catch (error) {
        console.error('Error fetching value sets:', error);
        throw new Error(`Failed to fetch value sets: ${error.message}`);
      }
    },

    async valueSet(parent, args, context) {
      try {
        const { url, version } = args;
        const response = await axios.get(`${KB7_TERMINOLOGY_URL}/v1/valuesets/${encodeURIComponent(url)}`, {
          params: { version },
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId
          }
        });
        return response.data;
      } catch (error) {
        if (error.response && error.response.status === 404) {
          return null;
        }
        console.error('Error fetching value set:', error);
        throw new Error(`Failed to fetch value set: ${error.message}`);
      }
    },

    async expandValueSet(parent, args, context) {
      try {
        const { url, version, filter } = args;
        const response = await axios.post(`${KB7_TERMINOLOGY_URL}/v1/valuesets/${encodeURIComponent(url)}/expand`, 
          { version, filter },
          {
            headers: { 
              'Authorization': context.authHeader,
              'X-Request-ID': context.requestId,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error expanding value set:', error);
        throw new Error(`Failed to expand value set: ${error.message}`);
      }
    },

    // Concept Mapping Operations
    async conceptMappings(parent, args, context) {
      try {
        const response = await axios.get(`${KB7_TERMINOLOGY_URL}/v1/mappings`, {
          params: {
            source_system: args.sourceSystem,
            target_system: args.targetSystem,
            equivalence: args.equivalence
          },
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId
          }
        });
        return response.data.mappings || [];
      } catch (error) {
        console.error('Error fetching concept mappings:', error);
        throw new Error(`Failed to fetch concept mappings: ${error.message}`);
      }
    },

    async translateConcept(parent, args, context) {
      try {
        const { input } = args;
        const response = await axios.post(`${KB7_TERMINOLOGY_URL}/v1/concepts/translate`, input, {
          headers: { 
            'Authorization': context.authHeader,
            'X-Request-ID': context.requestId,
            'Content-Type': 'application/json'
          }
        });
        return response.data;
      } catch (error) {
        console.error('Error translating concept:', error);
        throw new Error(`Failed to translate concept: ${error.message}`);
      }
    },

    // Batch Operations
    async batchLookupConcepts(parent, args, context) {
      try {
        const { requests } = args;
        const response = await axios.post(`${KB7_TERMINOLOGY_URL}/v1/concepts/batch-lookup`, 
          { requests },
          {
            headers: { 
              'Authorization': context.authHeader,
              'X-Request-ID': context.requestId,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data.results || [];
      } catch (error) {
        console.error('Error in batch lookup concepts:', error);
        throw new Error(`Failed to batch lookup concepts: ${error.message}`);
      }
    },

    async batchValidateCodes(parent, args, context) {
      try {
        const { requests } = args;
        const response = await axios.post(`${KB7_TERMINOLOGY_URL}/v1/concepts/batch-validate`, 
          { requests },
          {
            headers: { 
              'Authorization': context.authHeader,
              'X-Request-ID': context.requestId,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data.results || [];
      } catch (error) {
        console.error('Error in batch validate codes:', error);
        throw new Error(`Failed to batch validate codes: ${error.message}`);
      }
    }
  },

  Mutation: {
    async refreshTerminologySystem(parent, args, context) {
      try {
        const { systemUri } = args;
        const response = await axios.post(`${KB7_TERMINOLOGY_URL}/v1/admin/refresh-system`, 
          { systemUri },
          {
            headers: { 
              'Authorization': context.authHeader,
              'X-Request-ID': context.requestId,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error refreshing terminology system:', error);
        return {
          success: false,
          message: `Failed to refresh terminology system: ${error.message}`
        };
      }
    },

    async rebuildValueSetExpansion(parent, args, context) {
      try {
        const { valueSetUrl } = args;
        const response = await axios.post(`${KB7_TERMINOLOGY_URL}/v1/admin/rebuild-expansion`, 
          { valueSetUrl },
          {
            headers: { 
              'Authorization': context.authHeader,
              'X-Request-ID': context.requestId,
              'Content-Type': 'application/json'
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error rebuilding value set expansion:', error);
        return {
          success: false,
          message: `Failed to rebuild value set expansion: ${error.message}`
        };
      }
    }
  },

  // Type Resolvers
  TerminologySystem: {
    conceptCount: (parent) => parent.concept_count,
    hierarchyMeaning: (parent) => parent.hierarchy_meaning,
    versionNeeded: (parent) => parent.version_needed,
    supportedRegions: (parent) => parent.supported_regions || [],
    createdAt: (parent) => parent.created_at,
    updatedAt: (parent) => parent.updated_at
  },

  TerminologyConcept: {
    systemId: (parent) => parent.system_id,
    parentCodes: (parent) => parent.parent_codes || [],
    childCodes: (parent) => parent.child_codes || [],
    clinicalDomain: (parent) => parent.clinical_domain,
    createdAt: (parent) => parent.created_at,
    updatedAt: (parent) => parent.updated_at
  },

  ValueSet: {
    useContext: (parent) => parent.use_context || [],
    clinicalDomain: (parent) => parent.clinical_domain,
    supportedRegions: (parent) => parent.supported_regions || [],
    createdAt: (parent) => parent.created_at,
    updatedAt: (parent) => parent.updated_at,
    expiredAt: (parent) => parent.expired_at
  },

  ConceptMapping: {
    sourceSystemId: (parent) => parent.source_system_id,
    sourceCode: (parent) => parent.source_code,
    targetSystemId: (parent) => parent.target_system_id,
    targetCode: (parent) => parent.target_code,
    mappingType: (parent) => parent.mapping_type,
    mappedBy: (parent) => parent.mapped_by,
    verifiedBy: (parent) => parent.verified_by,
    verifiedAt: (parent) => parent.verified_at,
    usageCount: (parent) => parent.usage_count || 0,
    lastUsedAt: (parent) => parent.last_used_at,
    createdAt: (parent) => parent.created_at,
    updatedAt: (parent) => parent.updated_at
  }
};

module.exports = kb7TerminologyResolvers;