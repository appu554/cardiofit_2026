# GraphQL Explorer for Clinical Synthesis Hub

This directory contains the GraphQL Explorer, a web-based interface for exploring and interacting with the Clinical Synthesis Hub's GraphQL API.

## Features

- **Interactive GraphQL IDE**: Built on GraphiQL, a popular GraphQL IDE
- **Schema Explorer**: Browse the GraphQL schema and documentation
- **Query Builder**: Write and execute GraphQL queries and mutations
- **Example Queries**: Pre-built example queries for common operations
- **Authentication Support**: Set JWT tokens for authenticated requests

## Usage

The GraphQL Explorer is available at `/graphql-explorer` in the API Gateway.

### Authentication

To authenticate your requests:

1. Obtain a JWT token from the Auth Service using the `/api/auth/token` endpoint
2. Enter the token in the input field at the top of the GraphQL Explorer
3. Click "Set Token" to apply the token to all requests

### Building Queries

The GraphQL Explorer provides a user-friendly interface for building queries:

1. Use the Documentation Explorer (button in the top right) to browse the schema
2. Write your query in the left panel
3. Set variables in the bottom panel (click "Variables" to expand)
4. Click the "Play" button to execute the query
5. View the results in the right panel

### Example Queries

The GraphQL Explorer includes example queries for common operations:

1. Click the "Query Examples" tab to view the examples
2. Click "Load Example" to load an example into the query editor
3. Modify the example as needed
4. Execute the query by clicking the "Play" button

## Implementation Details

The GraphQL Explorer is implemented as a static HTML page that uses:

- **GraphiQL**: A popular GraphQL IDE
- **React**: For the user interface
- **Fetch API**: For making GraphQL requests to the API Gateway

The GraphQL Explorer is served by the API Gateway at the `/graphql-explorer` endpoint and does not require authentication to access.

## Customization

To customize the GraphQL Explorer:

1. Edit the `graphql-explorer.html` file in this directory
2. Add or modify example queries in the "examples" section
3. Update the documentation in the "docs" section
4. Restart the API Gateway to apply the changes

## Troubleshooting

If you encounter issues with the GraphQL Explorer:

1. Check that the API Gateway is running
2. Verify that the GraphQL endpoint is accessible at `/graphql`
3. Check that your JWT token is valid
4. Check the browser console for JavaScript errors
5. Check the API Gateway logs for server-side errors
