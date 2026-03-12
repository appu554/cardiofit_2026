// Test script to check the federation endpoint
// Import fetch with ESM syntax for node-fetch v3
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

async function testFederation() {
  try {
    console.log('Testing federation endpoint...');

    const response = await fetch('http://localhost:8003/api/federation', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkFwb2xsbyBGZWRlcmF0aW9uIiwicm9sZSI6ImFkbWluIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c',
        'X-User-ID': '123456',
        'X-User-Role': 'admin'
      },
      body: JSON.stringify({
        query: `
          query GetFederationSDL {
            _service {
              sdl
            }
          }
        `
      })
    });

    const data = await response.json();
    console.log('Response:', JSON.stringify(data, null, 2));

    if (data.data && data.data._service && data.data._service.sdl) {
      console.log('Federation endpoint is working correctly!');
      console.log('SDL length:', data.data._service.sdl.length);

      // Print the first 500 characters of the SDL
      console.log('SDL preview:', data.data._service.sdl.substring(0, 500) + '...');
    } else {
      console.error('Federation endpoint is not returning the expected data structure.');
      console.error('Expected: { data: { _service: { sdl: "..." } } }');
      console.error('Received:', data);
    }
  } catch (error) {
    console.error('Error testing federation endpoint:', error);
  }
}

testFederation();
