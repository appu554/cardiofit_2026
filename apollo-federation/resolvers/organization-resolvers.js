const axios = require('axios');

const ORGANIZATION_SERVICE_URL = process.env.ORGANIZATION_SERVICE_URL || 'http://localhost:8012';

const resolvers = {
  Organization: {
    __resolveReference(organization, { dataSources }) {
      return { id: organization.id };
    }
  },

  Query: {
    async organization(_, { id }, context) {
      try {
        const response = await axios.get(
          `${ORGANIZATION_SERVICE_URL}/api/organizations/${id}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'X-User-Roles': context.userRoles?.join(','),
              'X-User-Permissions': context.userPermissions?.join(',')
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error fetching organization:', error);
        if (error.response?.status === 404) {
          return null;
        }
        throw new Error(`Failed to fetch organization: ${error.message}`);
      }
    },

    async organizations(_, args, context) {
      try {
        const params = new URLSearchParams();
        
        if (args.name) params.append('name', args.name);
        if (args.organizationType) params.append('type', args.organizationType);
        if (args.status) params.append('status', args.status);
        if (args.active !== undefined) params.append('active', args.active.toString());

        const response = await axios.get(
          `${ORGANIZATION_SERVICE_URL}/api/organizations?${params.toString()}`,
          {
            headers: {
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'X-User-Roles': context.userRoles?.join(','),
              'X-User-Permissions': context.userPermissions?.join(',')
            }
          }
        );

        const organizations = response.data;
        return {
          organizations,
          totalCount: organizations.length,
          hasMore: false // TODO: Implement pagination
        };
      } catch (error) {
        console.error('Error searching organizations:', error);
        return {
          organizations: [],
          totalCount: 0,
          hasMore: false
        };
      }
    }
  },

  Mutation: {
    async createOrganization(_, { organizationData }, context) {
      try {
        const response = await axios.post(
          `${ORGANIZATION_SERVICE_URL}/api/organizations`,
          organizationData,
          {
            headers: {
              'Content-Type': 'application/json',
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'X-User-Roles': context.userRoles?.join(','),
              'X-User-Permissions': context.userPermissions?.join(',')
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error creating organization:', error);
        throw new Error(`Failed to create organization: ${error.response?.data?.detail || error.message}`);
      }
    },

    async updateOrganization(_, { id, updateData }, context) {
      try {
        const response = await axios.put(
          `${ORGANIZATION_SERVICE_URL}/api/organizations/${id}`,
          updateData,
          {
            headers: {
              'Content-Type': 'application/json',
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'X-User-Roles': context.userRoles?.join(','),
              'X-User-Permissions': context.userPermissions?.join(',')
            }
          }
        );
        return response.data;
      } catch (error) {
        console.error('Error updating organization:', error);
        if (error.response?.status === 404) {
          throw new Error('Organization not found');
        }
        throw new Error(`Failed to update organization: ${error.response?.data?.detail || error.message}`);
      }
    },

    async submitOrganizationForVerification(_, { id, documents }, context) {
      try {
        const response = await axios.post(
          `${ORGANIZATION_SERVICE_URL}/api/organizations/${id}/verify`,
          { documents: documents || [] },
          {
            headers: {
              'Content-Type': 'application/json',
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'X-User-Roles': context.userRoles?.join(','),
              'X-User-Permissions': context.userPermissions?.join(',')
            }
          }
        );
        return true;
      } catch (error) {
        console.error('Error submitting organization for verification:', error);
        return false;
      }
    },

    async approveOrganization(_, { id, notes }, context) {
      try {
        const response = await axios.post(
          `${ORGANIZATION_SERVICE_URL}/api/organizations/${id}/approve`,
          { notes },
          {
            headers: {
              'Content-Type': 'application/json',
              Authorization: context.token,
              'X-User-ID': context.userId,
              'X-User-Role': context.userRole,
              'X-User-Roles': context.userRoles?.join(','),
              'X-User-Permissions': context.userPermissions?.join(',')
            }
          }
        );
        return true;
      } catch (error) {
        console.error('Error approving organization:', error);
        return false;
      }
    }
  }
};

module.exports = resolvers;
