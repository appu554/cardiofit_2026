const { Pool } = require("pg");

const pool = new Pool({
  host: "localhost",
  port: 5435,
  user: "kb3admin",
  password: "kb3_postgres_password", 
  database: "kb3_guidelines"
});

async function testConnection() {
  try {
    console.log("🔌 Testing KB-3 database connection...");
    
    const client = await pool.connect();
    console.log("✅ Connected to KB-3 database");

    const result = await client.query("SELECT COUNT(*) as total_guidelines FROM guideline_evidence.guidelines");
    console.log("📊 Found " + result.rows[0].total_guidelines + " guidelines in database");

    const guidelines = await client.query(`
      SELECT guideline_id, organization, condition_primary, status 
      FROM guideline_evidence.guidelines 
      ORDER BY guideline_id 
      LIMIT 5
    `);
    
    console.log("
📋 Sample Guidelines:");
    guidelines.rows.forEach(g => {
      console.log("  • " + g.guideline_id + " (" + g.organization + ") - " + g.condition_primary);
    });

    client.release();
    console.log("
✅ All tests passed! KB-3 database is working correctly.");
    
  } catch (error) {
    console.error("❌ Database connection failed:", error.message);
  } finally {
    await pool.end();
  }
}

testConnection();
