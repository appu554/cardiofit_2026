// Test script to verify Apollo Federation service mappings
const fetch = require('node-fetch');

const federationServices = [
  { name: 'patients', url: 'http://localhost:8003/health' },
  { name: 'medications', url: 'http://localhost:8004/health' },
  { name: 'kb1-drug-rules', url: 'http://localhost:8081/health' },
  { name: 'kb2-clinical-context', url: 'http://localhost:8082/health' },
  { name: 'kb3-guidelines', url: 'http://localhost:8083/health' },
  { name: 'kb4-patient-safety', url: 'http://localhost:8084/health' },
  { name: 'kb5-ddi', url: 'http://localhost:8085/health' },
  { name: 'kb6-formulary', url: 'http://localhost:8086/health' },
  { name: 'kb7-terminology', url: 'http://localhost:8087/health' },
  { name: 'evidence-envelope', url: 'http://localhost:8088/health' }
];

async function testServices() {
  console.log('🔍 Testing KB Service Health Endpoints...\n');
  
  for (const service of federationServices) {
    try {
      const response = await fetch(service.url, { timeout: 2000 });
      if (response.ok) {
        console.log(`✅ ${service.name.padEnd(25)} - HEALTHY`);
      } else {
        console.log(`❌ ${service.name.padEnd(25)} - HTTP ${response.status}`);
      }
    } catch (error) {
      console.log(`❌ ${service.name.padEnd(25)} - UNREACHABLE (${error.message})`);
    }
  }
  
  console.log('\n📊 Database Services (External):');
  const dbServices = [
    { name: 'MongoDB', url: 'http://localhost:27017' },
    { name: 'Neo4j', url: 'http://localhost:7474' },
    { name: 'TimescaleDB', url: 'postgresql://localhost:5434' },
    { name: 'Elasticsearch', url: 'http://localhost:9200' }
  ];
  
  for (const db of dbServices) {
    try {
      if (db.name === 'Neo4j') {
        const response = await fetch('http://localhost:7474/db/data/', { timeout: 2000 });
        console.log(`${response.ok ? '✅' : '❌'} ${db.name.padEnd(25)} - ${response.ok ? 'HEALTHY' : 'UNHEALTHY'}`);
      } else if (db.name === 'Elasticsearch') {
        const response = await fetch(db.url, { timeout: 2000 });
        console.log(`${response.ok ? '✅' : '❌'} ${db.name.padEnd(25)} - ${response.ok ? 'HEALTHY' : 'UNHEALTHY'}`);
      } else {
        console.log(`⏳ ${db.name.padEnd(25)} - Requires specific client to test`);
      }
    } catch (error) {
      console.log(`❌ ${db.name.padEnd(25)} - UNREACHABLE`);
    }
  }
}

testServices().catch(console.error);