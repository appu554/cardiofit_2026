import { ApolloClient, InMemoryCache, HttpLink, split } from '@apollo/client';
import { getMainDefinition } from '@apollo/client/utilities';
import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { createClient } from 'graphql-ws';

// Analytics Service endpoint - defaults to port 8050
const httpLink = new HttpLink({
  uri: import.meta.env.VITE_GRAPHQL_URL || 'http://localhost:8050/graphql',
  credentials: 'include',
});

// WebSocket link for subscriptions (if enabled)
const wsLink = import.meta.env.VITE_ENABLE_WEBSOCKET === 'true'
  ? new GraphQLWsLink(
      createClient({
        url: import.meta.env.VITE_WS_URL || 'ws://localhost:8050/graphql',
        connectionParams: {
          reconnect: true,
        },
        retryAttempts: 5,
        retryWait: async (retries) => {
          await new Promise((resolve) =>
            setTimeout(resolve, Math.min(1000 * 2 ** retries, 10000))
          );
        },
      })
    )
  : null;

// Split between HTTP and WebSocket based on operation type
const splitLink = wsLink
  ? split(
      ({ query }) => {
        const definition = getMainDefinition(query);
        return (
          definition.kind === 'OperationDefinition' &&
          definition.operation === 'subscription'
        );
      },
      wsLink,
      httpLink
    )
  : httpLink;

const client = new ApolloClient({
  link: splitLink,
  cache: new InMemoryCache({
    typePolicies: {
      Query: {
        fields: {
          // High-risk patients list - always replace with incoming data
          highRiskPatients: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
          // Department metrics - always replace with incoming data
          allDepartmentMetrics: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
          // Patient risk profiles - merge by patientId
          patientRiskProfile: {
            merge(existing, incoming) {
              return incoming;
            },
          },
          // Active alerts - always replace with incoming data
          activeAlerts: {
            merge(existing = [], incoming) {
              return incoming;
            },
          },
        },
      },
    },
  }),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'cache-and-network',
      errorPolicy: 'all',
    },
    query: {
      fetchPolicy: 'network-only',
      errorPolicy: 'all',
    },
    mutate: {
      errorPolicy: 'all',
    },
  },
});

export default client;
