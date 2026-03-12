/**
 * Analytics Service for Apollo Federation
 *
 * Standalone GraphQL service for Module 6 Analytics Dashboard
 * Runs on port 8050 and exposes /api/federation endpoint
 *
 * Usage:
 *   node services/analytics-service.js
 */

const { ApolloServer } = require('@apollo/server');
const { startStandaloneServer } = require('@apollo/server/standalone');
const { buildSubgraphSchema } = require('@apollo/subgraph');
const { readFileSync } = require('fs');
const { join } = require('path');
const gql = require('graphql-tag');

const analyticsDataService = require('./analytics-data-service');
const analyticsResolvers = require('../resolvers/analytics-dashboard-resolvers');

// Load GraphQL schema
const schemaPath = join(__dirname, '../schemas/analytics-dashboard-schema.graphql');
const typeDefs = gql(readFileSync(schemaPath, 'utf8'));

// Logger
const logger = {
  info: (message, ...args) => console.log(`[Analytics Service] ${new Date().toISOString()} - ${message}`, ...args),
  error: (message, ...args) => console.error(`[Analytics Service ERROR] ${new Date().toISOString()} - ${message}`, ...args),
  warn: (message, ...args) => console.warn(`[Analytics Service WARN] ${new Date().toISOString()} - ${message}`, ...args)
};

async function startAnalyticsService() {
  try {
    // Initialize data service connections
    logger.info('Initializing Analytics Data Service...');
    await analyticsDataService.initialize();

    // Start Kafka consumers for real-time data population
    await startKafkaConsumers();

    // Create Apollo Server with Federation support
    const server = new ApolloServer({
      schema: buildSubgraphSchema({ typeDefs, resolvers: analyticsResolvers }),
      introspection: true,
      plugins: [
        {
          async requestDidStart() {
            return {
              async willSendResponse({ response }) {
                // Log errors
                if (response.errors) {
                  logger.error('GraphQL errors:', response.errors);
                }
              },
            };
          },
        },
      ],
    });

    // Start the server
    const { url } = await startStandaloneServer(server, {
      listen: { port: parseInt(process.env.ANALYTICS_SERVICE_PORT || '8050') },
      context: async ({ req }) => {
        return {
          token: req.headers.authorization || '',
          userId: req.headers['x-user-id'],
          userRole: req.headers['x-user-role']
        };
      },
    });

    logger.info(`🚀 Analytics Service ready at ${url}`);
    logger.info(`Federation endpoint: ${url}graphql`);
    logger.info('Ready to be added to Apollo Gateway with:');
    logger.info(`  { name: 'analytics', url: '${url}graphql' }`);

  } catch (error) {
    logger.error('Failed to start Analytics Service:', error);
    process.exit(1);
  }
}

/**
 * Start Kafka consumers to populate Redis cache with real-time data
 */
async function startKafkaConsumers() {
  try {
    if (!analyticsDataService.connected.kafka) {
      logger.warn('Kafka not available, skipping real-time data population');
      return;
    }

    logger.info('Starting Kafka consumers for real-time data...');

    // Subscribe to population health metrics
    await analyticsDataService.subscribeToTopic(
      'analytics-population-health',
      async (data) => {
        logger.info(`Received population health metrics for department: ${data.department}`);
        await analyticsDataService.cachePopulationMetrics(data);
      }
    );

    // Subscribe to ML predictions to build patient risk profiles
    await analyticsDataService.subscribeToTopic(
      'inference-results.v1',
      async (prediction) => {
        logger.info(`Received ML prediction for patient: ${prediction.patientId}`);

        // Build patient risk profile from prediction
        const profile = {
          patientId: prediction.patientId,
          department: extractDepartment(prediction),
          mortalityRisk: extractRiskScore(prediction, 'mortality'),
          sepsisRisk: extractRiskScore(prediction, 'sepsis'),
          readmissionRisk: extractRiskScore(prediction, 'readmission'),
          overallRiskScore: prediction.primaryScore || 0.0,
          riskLevel: calculateRiskLevel(prediction.primaryScore || 0.0),
          lastUpdated: new Date().toISOString()
        };

        await analyticsDataService.cachePatientRiskProfile(profile);
      }
    );

    // Aggregate hospital-wide KPIs every 60 seconds
    setInterval(async () => {
      try {
        await aggregateHospitalKPIs();
      } catch (error) {
        logger.error('Error aggregating hospital KPIs:', error.message);
      }
    }, 60000);

    logger.info('Kafka consumers started successfully');

  } catch (error) {
    logger.error('Error starting Kafka consumers:', error.message);
  }
}

/**
 * Aggregate hospital-wide KPIs from department metrics
 */
async function aggregateHospitalKPIs() {
  try {
    const departments = await analyticsDataService.getAllDepartmentMetrics();

    if (departments.length === 0) {
      logger.warn('No department metrics available for aggregation');
      return;
    }

    let totalPatients = 0;
    let highRiskPatients = 0;
    let criticalPatients = 0;
    let mortalitySum = 0;
    let sepsisSum = 0;
    let readmissionSum = 0;

    for (const dept of departments) {
      totalPatients += dept.totalPatients || 0;
      highRiskPatients += dept.highRiskPatients || 0;
      criticalPatients += dept.criticalPatients || 0;
      mortalitySum += (dept.avgMortalityRisk || 0) * (dept.totalPatients || 0);
      sepsisSum += (dept.avgSepsisRisk || 0) * (dept.totalPatients || 0);
      readmissionSum += (dept.avgReadmissionRisk || 0) * (dept.totalPatients || 0);
    }

    const kpis = {
      timestamp: new Date().toISOString(),
      totalPatients,
      highRiskPatients,
      criticalPatients,
      avgMortalityRisk: totalPatients > 0 ? mortalitySum / totalPatients : 0.0,
      avgSepsisRisk: totalPatients > 0 ? sepsisSum / totalPatients : 0.0,
      avgReadmissionRisk: totalPatients > 0 ? readmissionSum / totalPatients : 0.0,
      activeAlerts: 0, // TODO: Query from alert system
      criticalAlerts: 0, // TODO: Query from alert system
      modelAccuracy: null, // TODO: Query from ML performance metrics
      patientsOnProtocols: 0 // TODO: Query from protocol tracking
    };

    // Cache hospital KPIs in Redis
    if (analyticsDataService.connected.redis) {
      await analyticsDataService.redis.setex(
        'analytics:hospital:kpis',
        300, // 5 min TTL
        JSON.stringify(kpis)
      );
      logger.info('Hospital KPIs aggregated and cached');
    }

  } catch (error) {
    logger.error('Error aggregating hospital KPIs:', error.message);
  }
}

/**
 * Extract department from ML prediction metadata
 */
function extractDepartment(prediction) {
  const metadata = prediction.modelMetadata || {};
  return metadata.department || 'UNKNOWN';
}

/**
 * Extract specific risk score from ML prediction
 */
function extractRiskScore(prediction, riskType) {
  const scores = prediction.predictionScores || {};
  const modelName = prediction.modelName || '';

  // Check if model name matches risk type
  if (modelName.toLowerCase().includes(riskType)) {
    return prediction.primaryScore || 0.0;
  }

  // Check prediction scores map
  const patterns = [
    riskType,
    `${riskType}_risk`,
    `${riskType}_score`,
    `${riskType}_probability`
  ];

  for (const pattern of patterns) {
    if (scores[pattern] !== undefined) {
      return scores[pattern];
    }
  }

  return 0.0;
}

/**
 * Calculate risk level category from score
 */
function calculateRiskLevel(score) {
  if (score >= 0.75) return 'CRITICAL';
  if (score >= 0.50) return 'HIGH';
  if (score >= 0.25) return 'MODERATE';
  return 'LOW';
}

// Graceful shutdown
process.on('SIGTERM', async () => {
  logger.info('SIGTERM received, shutting down gracefully...');
  await analyticsDataService.shutdown();
  process.exit(0);
});

process.on('SIGINT', async () => {
  logger.info('SIGINT received, shutting down gracefully...');
  await analyticsDataService.shutdown();
  process.exit(0);
});

// Start the service
startAnalyticsService().catch((error) => {
  logger.error('Fatal error starting Analytics Service:', error);
  process.exit(1);
});
