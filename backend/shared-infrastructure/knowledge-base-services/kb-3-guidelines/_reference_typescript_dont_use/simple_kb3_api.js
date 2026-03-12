const express = require('express');
const { Pool } = require('pg');

const app = express();
const port = 8084;

const pool = new Pool({
  host: 'localhost',
  port: 5435,
  user: 'kb3admin',
  password: 'kb3_postgres_password',
  database: 'kb3_guidelines'
});

app.use(express.json());

// Health check endpoint
app.get('/health', async (req, res) => {
  try {
    const client = await pool.connect();
    await client.query('SELECT 1');
    client.release();
    res.json({ status: 'healthy', service: 'KB-3 Guidelines', timestamp: new Date().toISOString() });
  } catch (error) {
    res.status(500).json({ status: 'unhealthy', error: error.message });
  }
});

// Get all guidelines
app.get('/api/guidelines', async (req, res) => {
  try {
    const client = await pool.connect();
    const result = await client.query(`
      SELECT guideline_id, organization, region, condition_primary, 
             icd10_codes, version, effective_date, status, 
             evidence_summary, quality_metrics
      FROM guideline_evidence.guidelines 
      ORDER BY guideline_id
    `);
    client.release();
    res.json({ 
      count: result.rows.length, 
      guidelines: result.rows 
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get guidelines by condition
app.get('/api/guidelines/condition/:condition', async (req, res) => {
  try {
    const condition = req.params.condition.toLowerCase();
    const client = await pool.connect();
    const result = await client.query(`
      SELECT guideline_id, organization, condition_primary, 
             evidence_summary->>'recommendation' as recommendation,
             evidence_summary->>'evidence_grade' as evidence_grade,
             quality_metrics->>'methodology_score' as methodology_score
      FROM guideline_evidence.guidelines 
      WHERE LOWER(condition_primary) LIKE $1
      ORDER BY quality_metrics->>'methodology_score' DESC
    `, [`%${condition}%`]);
    client.release();
    res.json({ 
      condition: req.params.condition,
      count: result.rows.length, 
      guidelines: result.rows 
    });
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

// Get specific guideline
app.get('/api/guidelines/:id', async (req, res) => {
  try {
    const client = await pool.connect();
    const result = await client.query(`
      SELECT * FROM guideline_evidence.guidelines 
      WHERE guideline_id = $1
    `, [req.params.id]);
    client.release();
    
    if (result.rows.length === 0) {
      res.status(404).json({ error: 'Guideline not found' });
    } else {
      res.json(result.rows[0]);
    }
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

app.listen(port, () => {
  console.log(`🚀 KB-3 Guidelines API running on port ${port}`);
  console.log(`📊 Health check: http://localhost:${port}/health`);
  console.log(`📋 Guidelines: http://localhost:${port}/api/guidelines`);
  console.log(`🔍 Search by condition: http://localhost:${port}/api/guidelines/condition/diabetes`);
});
