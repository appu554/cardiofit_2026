// Test script to check the FHIR response format
// Import fetch with ESM syntax for node-fetch v3
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

async function testFHIR() {
  try {
    console.log('Testing FHIR endpoint...');
    
    // Make a request to the patient service's FHIR endpoint
    const response = await fetch('http://localhost:8003/api/fhir/Patient?_count=10&_page=1', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkFwb2xsbyBGZWRlcmF0aW9uIiwicm9sZSI6ImFkbWluIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c',
        'X-User-ID': '123456',
        'X-User-Role': 'admin'
      }
    });
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error(`HTTP error ${response.status}: ${errorText}`);
      return;
    }
    
    const data = await response.json();
    console.log('Response:', JSON.stringify(data, null, 2));
    
    // Check if the response has the expected format
    if (data.entry && Array.isArray(data.entry)) {
      console.log('Found', data.entry.length, 'patients');
      
      // Print the first patient
      if (data.entry.length > 0) {
        console.log('First patient:', JSON.stringify(data.entry[0], null, 2));
      }
    } else {
      console.error('Response does not have the expected format');
      console.error('Expected: { entry: [...] }');
      console.error('Received:', data);
    }
  } catch (error) {
    console.error('Error testing FHIR endpoint:', error);
  }
}

testFHIR();
