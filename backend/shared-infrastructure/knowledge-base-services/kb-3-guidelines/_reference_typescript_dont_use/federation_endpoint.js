const express = require('express');
const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const { buildSubgraphSchema } = require('@apollo/subgraph');
const gql = require('graphql-tag');
const { Pool } = require('pg');
const cors = require('cors');
const { json } = require('body-parser');

const app = express();
const port = 8085;

// PostgreSQL connection
const pool = new Pool({
  host: 'localhost',
  port: 5435,
  user: 'kb3admin',
  password: 'kb3_postgres_password',
  database: 'kb3_guidelines'
});

// GraphQL Schema Definition with Federation v2 directives
const typeDefs = gql`
  extend schema
    @link(url: "https://specs.apollographql.org/federation/v2.0",
          import: ["@key", "@shareable", "@external", "@requires", "@provides"])

  type Guideline @key(fields: "guideline_id") {
    guideline_id: String!
    organization: String!
    region: String!
    condition_primary: String!
    icd10_codes: [String!]!
    version: String!
    effective_date: String!
    status: String!
    approval_status: String!
    evidence_summary: EvidenceSummary!
    quality_metrics: QualityMetrics!
    created_at: String!
    updated_at: String!
    created_by: String
    approved_by: String
  }

  type EvidenceSummary {
    recommendation: String!
    evidence_grade: String!
    strength_of_recommendation: String!
  }

  type QualityMetrics {
    methodology_score: Float!
    bias_risk: String!
    consistency: String!
  }

  type GuidelineConnection {
    totalCount: Int!
    guidelines: [Guideline!]!
  }

  type Query {
    # Federated queries for guidelines
    guidelines(
      condition: String
      organization: String
      status: String
      limit: Int = 20
      offset: Int = 0
    ): GuidelineConnection!

    guideline(guideline_id: String!): Guideline

    # Search guidelines by clinical condition
    guidelinesByCondition(condition: String!): [Guideline!]!

    # Get guidelines by evidence grade
    guidelinesByEvidenceGrade(grade: String!): [Guideline!]!

    # Get highest quality guidelines
    topQualityGuidelines(limit: Int = 10): [Guideline!]!
  }
`;

// Resolvers
const resolvers = {
  Query: {
    guidelines: async (_, { condition, organization, status, limit, offset }) => {
      try {
        let query = 'SELECT * FROM guideline_evidence.guidelines WHERE 1=1';
        const params = [];
        let paramCount = 1;

        if (condition) {
          query += ` AND LOWER(condition_primary) LIKE $${paramCount}`;
          params.push(`%${condition.toLowerCase()}%`);
          paramCount++;
        }

        if (organization) {
          query += ` AND organization = $${paramCount}`;
          params.push(organization);
          paramCount++;
        }

        if (status) {
          query += ` AND status = $${paramCount}`;
          params.push(status);
          paramCount++;
        }

        query += ` ORDER BY guideline_id LIMIT $${paramCount} OFFSET $${paramCount + 1}`;
        params.push(limit, offset);

        const countQuery = 'SELECT COUNT(*) as total FROM guideline_evidence.guidelines WHERE 1=1' +
          (condition ? ' AND LOWER(condition_primary) LIKE $1' : '') +
          (organization ? ` AND organization = $${condition ? 2 : 1}` : '') +
          (status ? ` AND status = $${[condition, organization].filter(Boolean).length + 1}` : '');

        const countParams = [];
        if (condition) countParams.push(`%${condition.toLowerCase()}%`);
        if (organization) countParams.push(organization);
        if (status) countParams.push(status);

        const client = await pool.connect();
        const [guidelines, countResult] = await Promise.all([
          client.query(query, params),
          client.query(countQuery, countParams)
        ]);
        client.release();

        return {
          totalCount: parseInt(countResult.rows[0].total),
          guidelines: guidelines.rows
        };
      } catch (error) {
        console.error('Error fetching guidelines:', error);
        throw new Error('Failed to fetch guidelines');
      }
    },

    guideline: async (_, { guideline_id }) => {
      try {
        const client = await pool.connect();
        const result = await client.query(
          'SELECT * FROM guideline_evidence.guidelines WHERE guideline_id = $1',
          [guideline_id]
        );
        client.release();
        return result.rows[0] || null;
      } catch (error) {
        console.error('Error fetching guideline:', error);
        throw new Error('Failed to fetch guideline');
      }
    },

    guidelinesByCondition: async (_, { condition }) => {
      try {
        const client = await pool.connect();
        const result = await client.query(
          `SELECT * FROM guideline_evidence.guidelines
           WHERE LOWER(condition_primary) LIKE $1
           ORDER BY quality_metrics->>'methodology_score' DESC`,
          [`%${condition.toLowerCase()}%`]
        );
        client.release();
        return result.rows;
      } catch (error) {
        console.error('Error fetching guidelines by condition:', error);
        throw new Error('Failed to fetch guidelines by condition');
      }
    },

    guidelinesByEvidenceGrade: async (_, { grade }) => {
      try {
        const client = await pool.connect();
        const result = await client.query(
          `SELECT * FROM guideline_evidence.guidelines
           WHERE evidence_summary->>'evidence_grade' = $1
           ORDER BY quality_metrics->>'methodology_score' DESC`,
          [grade]
        );
        client.release();
        return result.rows;
      } catch (error) {
        console.error('Error fetching guidelines by evidence grade:', error);
        throw new Error('Failed to fetch guidelines by evidence grade');
      }
    },

    topQualityGuidelines: async (_, { limit }) => {
      try {
        const client = await pool.connect();
        const result = await client.query(
          `SELECT * FROM guideline_evidence.guidelines
           WHERE status = 'active'
           ORDER BY (quality_metrics->>'methodology_score')::float DESC
           LIMIT $1`,
          [limit]
        );
        client.release();
        return result.rows;
      } catch (error) {
        console.error('Error fetching top quality guidelines:', error);
        throw new Error('Failed to fetch top quality guidelines');
      }
    }
  },

  Guideline: {
    __resolveReference: async (guideline) => {
      try {
        const client = await pool.connect();
        const result = await client.query(
          'SELECT * FROM guideline_evidence.guidelines WHERE guideline_id = $1',
          [guideline.guideline_id]
        );
        client.release();
        return result.rows[0];
      } catch (error) {
        console.error('Error resolving guideline reference:', error);
        return null;
      }
    }
  }
};

// Create Apollo Server with Federation v2 subgraph
const server = new ApolloServer({
  schema: buildSubgraphSchema([{ typeDefs, resolvers }]),
  plugins: [
    {
      requestDidStart() {
        return {
          didResolveOperation(requestContext) {
            console.log(`[KB-3] Operation: ${requestContext.request.operationName || 'Anonymous'}`);
          },
          didEncounterErrors(requestContext) {
            console.error('[KB-3] GraphQL errors:', requestContext.errors);
          }
        };
      }
    }
  ]
});

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({
    status: 'healthy',
    service: 'KB-3 Guidelines Federation',
    timestamp: new Date().toISOString(),
    port: port
  });
});

// Initialize server
async function startServer() {
  try {
    await server.start();

    app.use(
      '/api/federation',
      cors(),
      json(),
      expressMiddleware(server, {
        context: async ({ req }) => ({
          token: req.headers.authorization,
          userId: req.headers['x-user-id'],
          userRole: req.headers['x-user-role']
        })
      })
    );

    app.listen(port, () => {
      console.log(`🚀 KB-3 Guidelines Federation subgraph ready at:`);
      console.log(`   GraphQL: http://localhost:${port}/api/federation`);
      console.log(`   Health:  http://localhost:${port}/health`);
      console.log(`📊 Database: PostgreSQL (port 5435)`);
      console.log(`🔗 Federation: Apollo Federation v2 enabled`);
    });

  } catch (error) {
    console.error('Failed to start KB-3 Federation server:', error);
    process.exit(1);
  }
}

startServer();