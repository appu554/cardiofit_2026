"""
GraphQL Schema Explorer and Query Builder for the Patient Service.

This module provides endpoints for exploring the GraphQL schema and building queries.
"""

from fastapi import APIRouter, Request
import logging
import json

# Import GraphQL schema
from app.graphql.schema import schema

# Configure logging
logger = logging.getLogger(__name__)

# Create router
router = APIRouter()

@router.get("/schema")
async def get_schema():
    """
    Get the GraphQL schema as JSON.
    
    This endpoint returns the GraphQL schema in a format that can be used by a query builder UI.
    """
    try:
        # Get schema types
        types = {}
        for type_name, type_obj in schema.get_type_map().items():
            # Skip internal types
            if type_name.startswith('__'):
                continue
                
            # Get type fields
            fields = {}
            if hasattr(type_obj, 'fields'):
                for field_name, field_obj in type_obj.fields.items():
                    # Get field arguments
                    args = {}
                    if hasattr(field_obj, 'args'):
                        for arg_name, arg_obj in field_obj.args.items():
                            args[arg_name] = {
                                'type': str(arg_obj.type),
                                'description': arg_obj.description,
                                'defaultValue': arg_obj.default_value if hasattr(arg_obj, 'default_value') else None
                            }
                            
                    # Add field to fields dict
                    fields[field_name] = {
                        'type': str(field_obj.type),
                        'description': field_obj.description,
                        'args': args
                    }
                    
            # Add type to types dict
            types[type_name] = {
                'kind': type_obj.__class__.__name__,
                'description': type_obj.description if hasattr(type_obj, 'description') else None,
                'fields': fields
            }
            
        # Get queries
        queries = {}
        if 'Query' in types:
            queries = types['Query']['fields']
            
        # Get mutations
        mutations = {}
        if 'Mutation' in types:
            mutations = types['Mutation']['fields']
            
        return {
            'types': types,
            'queries': queries,
            'mutations': mutations
        }
    except Exception as e:
        logger.error(f"Error getting GraphQL schema: {str(e)}")
        return {"error": str(e)}

@router.get("/query-builder")
async def query_builder(request: Request):
    """
    GraphQL query builder UI.
    
    This endpoint serves a web-based UI for building GraphQL queries.
    """
    return {
        "html": """
        <!DOCTYPE html>
        <html>
        <head>
            <title>GraphQL Query Builder</title>
            <style>
                body {
                    font-family: Arial, sans-serif;
                    margin: 0;
                    padding: 0;
                    display: flex;
                    height: 100vh;
                }
                #sidebar {
                    width: 300px;
                    background-color: #f5f5f5;
                    padding: 20px;
                    overflow-y: auto;
                    border-right: 1px solid #ddd;
                }
                #main {
                    flex: 1;
                    display: flex;
                    flex-direction: column;
                }
                #query-editor {
                    flex: 1;
                    padding: 20px;
                    border-bottom: 1px solid #ddd;
                    display: flex;
                }
                #query-input {
                    flex: 1;
                    font-family: monospace;
                    padding: 10px;
                    border: 1px solid #ddd;
                    border-radius: 4px;
                    resize: none;
                }
                #result-viewer {
                    flex: 1;
                    padding: 20px;
                    overflow-y: auto;
                }
                #result-output {
                    font-family: monospace;
                    padding: 10px;
                    border: 1px solid #ddd;
                    border-radius: 4px;
                    background-color: #f9f9f9;
                    min-height: 200px;
                    white-space: pre-wrap;
                }
                .controls {
                    padding: 10px 20px;
                    background-color: #f0f0f0;
                    border-bottom: 1px solid #ddd;
                    display: flex;
                    align-items: center;
                }
                button {
                    padding: 8px 16px;
                    background-color: #4CAF50;
                    color: white;
                    border: none;
                    border-radius: 4px;
                    cursor: pointer;
                    margin-right: 10px;
                }
                button:hover {
                    background-color: #45a049;
                }
                .token-input {
                    display: flex;
                    align-items: center;
                    margin-left: auto;
                }
                .token-input input {
                    padding: 6px;
                    border: 1px solid #ccc;
                    border-radius: 4px;
                    width: 250px;
                    margin-left: 8px;
                }
                h2 {
                    margin-top: 0;
                }
                .type-item, .query-item, .mutation-item {
                    margin-bottom: 10px;
                    cursor: pointer;
                }
                .type-name, .query-name, .mutation-name {
                    font-weight: bold;
                    color: #333;
                }
                .type-name:hover, .query-name:hover, .mutation-name:hover {
                    text-decoration: underline;
                }
                .type-description, .query-description, .mutation-description {
                    font-size: 0.9em;
                    color: #666;
                    margin-top: 4px;
                }
                .field-list {
                    margin-left: 20px;
                    margin-top: 8px;
                    display: none;
                }
                .field-item {
                    margin-bottom: 6px;
                }
                .field-name {
                    font-weight: bold;
                    color: #0066cc;
                }
                .field-type {
                    color: #009900;
                    margin-left: 6px;
                }
                .field-description {
                    font-size: 0.9em;
                    color: #666;
                    margin-top: 2px;
                    margin-left: 10px;
                }
                .arg-list {
                    margin-left: 20px;
                    margin-top: 4px;
                }
                .arg-item {
                    margin-bottom: 4px;
                }
                .arg-name {
                    color: #cc6600;
                }
                .arg-type {
                    color: #990099;
                    margin-left: 6px;
                }
                .arg-description {
                    font-size: 0.9em;
                    color: #666;
                    margin-top: 2px;
                    margin-left: 10px;
                }
                .section {
                    margin-bottom: 20px;
                }
                .section-title {
                    font-size: 1.2em;
                    font-weight: bold;
                    margin-bottom: 10px;
                    color: #333;
                    border-bottom: 1px solid #ddd;
                    padding-bottom: 5px;
                }
            </style>
        </head>
        <body>
            <div id="sidebar">
                <h2>GraphQL Schema</h2>
                <div class="section">
                    <div class="section-title">Queries</div>
                    <div id="queries-list"></div>
                </div>
                <div class="section">
                    <div class="section-title">Mutations</div>
                    <div id="mutations-list"></div>
                </div>
                <div class="section">
                    <div class="section-title">Types</div>
                    <div id="types-list"></div>
                </div>
            </div>
            <div id="main">
                <div class="controls">
                    <button id="run-query">Run Query</button>
                    <button id="clear-query">Clear</button>
                    <div class="token-input">
                        <label for="token">Auth Token:</label>
                        <input type="text" id="token" placeholder="Bearer your-token-here" />
                    </div>
                </div>
                <div id="query-editor">
                    <textarea id="query-input" placeholder="Enter your GraphQL query here..."></textarea>
                </div>
                <div id="result-viewer">
                    <pre id="result-output">// Results will appear here</pre>
                </div>
            </div>
            
            <script>
                // Load schema on page load
                document.addEventListener('DOMContentLoaded', async () => {
                    try {
                        // Load schema
                        const response = await fetch('/api/graphql/explorer/schema');
                        const schema = await response.json();
                        
                        // Render queries
                        const queriesList = document.getElementById('queries-list');
                        for (const [queryName, query] of Object.entries(schema.queries)) {
                            const queryItem = document.createElement('div');
                            queryItem.className = 'query-item';
                            
                            const queryNameElem = document.createElement('div');
                            queryNameElem.className = 'query-name';
                            queryNameElem.textContent = queryName;
                            queryNameElem.onclick = () => toggleFields(queryItem);
                            queryItem.appendChild(queryNameElem);
                            
                            if (query.description) {
                                const queryDesc = document.createElement('div');
                                queryDesc.className = 'query-description';
                                queryDesc.textContent = query.description;
                                queryItem.appendChild(queryDesc);
                            }
                            
                            // Add fields
                            const fieldList = document.createElement('div');
                            fieldList.className = 'field-list';
                            
                            // Add arguments
                            if (query.args && Object.keys(query.args).length > 0) {
                                const argList = document.createElement('div');
                                argList.className = 'arg-list';
                                
                                for (const [argName, arg] of Object.entries(query.args)) {
                                    const argItem = document.createElement('div');
                                    argItem.className = 'arg-item';
                                    
                                    const argNameElem = document.createElement('span');
                                    argNameElem.className = 'arg-name';
                                    argNameElem.textContent = argName;
                                    argItem.appendChild(argNameElem);
                                    
                                    const argTypeElem = document.createElement('span');
                                    argTypeElem.className = 'arg-type';
                                    argTypeElem.textContent = arg.type;
                                    argItem.appendChild(argTypeElem);
                                    
                                    if (arg.description) {
                                        const argDesc = document.createElement('div');
                                        argDesc.className = 'arg-description';
                                        argDesc.textContent = arg.description;
                                        argItem.appendChild(argDesc);
                                    }
                                    
                                    argList.appendChild(argItem);
                                }
                                
                                fieldList.appendChild(argList);
                            }
                            
                            // Add "Add to Query" button
                            const addButton = document.createElement('button');
                            addButton.textContent = 'Add to Query';
                            addButton.onclick = () => addQueryToEditor(queryName, query);
                            fieldList.appendChild(addButton);
                            
                            queryItem.appendChild(fieldList);
                            queriesList.appendChild(queryItem);
                        }
                        
                        // Render mutations
                        const mutationsList = document.getElementById('mutations-list');
                        for (const [mutationName, mutation] of Object.entries(schema.mutations)) {
                            const mutationItem = document.createElement('div');
                            mutationItem.className = 'mutation-item';
                            
                            const mutationNameElem = document.createElement('div');
                            mutationNameElem.className = 'mutation-name';
                            mutationNameElem.textContent = mutationName;
                            mutationNameElem.onclick = () => toggleFields(mutationItem);
                            mutationItem.appendChild(mutationNameElem);
                            
                            if (mutation.description) {
                                const mutationDesc = document.createElement('div');
                                mutationDesc.className = 'mutation-description';
                                mutationDesc.textContent = mutation.description;
                                mutationItem.appendChild(mutationDesc);
                            }
                            
                            // Add fields
                            const fieldList = document.createElement('div');
                            fieldList.className = 'field-list';
                            
                            // Add arguments
                            if (mutation.args && Object.keys(mutation.args).length > 0) {
                                const argList = document.createElement('div');
                                argList.className = 'arg-list';
                                
                                for (const [argName, arg] of Object.entries(mutation.args)) {
                                    const argItem = document.createElement('div');
                                    argItem.className = 'arg-item';
                                    
                                    const argNameElem = document.createElement('span');
                                    argNameElem.className = 'arg-name';
                                    argNameElem.textContent = argName;
                                    argItem.appendChild(argNameElem);
                                    
                                    const argTypeElem = document.createElement('span');
                                    argTypeElem.className = 'arg-type';
                                    argTypeElem.textContent = arg.type;
                                    argItem.appendChild(argTypeElem);
                                    
                                    if (arg.description) {
                                        const argDesc = document.createElement('div');
                                        argDesc.className = 'arg-description';
                                        argDesc.textContent = arg.description;
                                        argItem.appendChild(argDesc);
                                    }
                                    
                                    argList.appendChild(argItem);
                                }
                                
                                fieldList.appendChild(argList);
                            }
                            
                            // Add "Add to Query" button
                            const addButton = document.createElement('button');
                            addButton.textContent = 'Add to Query';
                            addButton.onclick = () => addMutationToEditor(mutationName, mutation);
                            fieldList.appendChild(addButton);
                            
                            mutationItem.appendChild(fieldList);
                            mutationsList.appendChild(mutationItem);
                        }
                        
                        // Render types
                        const typesList = document.getElementById('types-list');
                        for (const [typeName, type] of Object.entries(schema.types)) {
                            // Skip Query and Mutation types
                            if (typeName === 'Query' || typeName === 'Mutation') {
                                continue;
                            }
                            
                            const typeItem = document.createElement('div');
                            typeItem.className = 'type-item';
                            
                            const typeNameElem = document.createElement('div');
                            typeNameElem.className = 'type-name';
                            typeNameElem.textContent = typeName;
                            typeNameElem.onclick = () => toggleFields(typeItem);
                            typeItem.appendChild(typeNameElem);
                            
                            if (type.description) {
                                const typeDesc = document.createElement('div');
                                typeDesc.className = 'type-description';
                                typeDesc.textContent = type.description;
                                typeItem.appendChild(typeDesc);
                            }
                            
                            // Add fields
                            if (type.fields && Object.keys(type.fields).length > 0) {
                                const fieldList = document.createElement('div');
                                fieldList.className = 'field-list';
                                
                                for (const [fieldName, field] of Object.entries(type.fields)) {
                                    const fieldItem = document.createElement('div');
                                    fieldItem.className = 'field-item';
                                    
                                    const fieldNameElem = document.createElement('span');
                                    fieldNameElem.className = 'field-name';
                                    fieldNameElem.textContent = fieldName;
                                    fieldItem.appendChild(fieldNameElem);
                                    
                                    const fieldTypeElem = document.createElement('span');
                                    fieldTypeElem.className = 'field-type';
                                    fieldTypeElem.textContent = field.type;
                                    fieldItem.appendChild(fieldTypeElem);
                                    
                                    if (field.description) {
                                        const fieldDesc = document.createElement('div');
                                        fieldDesc.className = 'field-description';
                                        fieldDesc.textContent = field.description;
                                        fieldItem.appendChild(fieldDesc);
                                    }
                                    
                                    fieldList.appendChild(fieldItem);
                                }
                                
                                typeItem.appendChild(fieldList);
                            }
                            
                            typesList.appendChild(typeItem);
                        }
                    } catch (error) {
                        console.error('Error loading schema:', error);
                    }
                });
                
                // Toggle field list visibility
                function toggleFields(item) {
                    const fieldList = item.querySelector('.field-list');
                    if (fieldList) {
                        fieldList.style.display = fieldList.style.display === 'none' || fieldList.style.display === '' ? 'block' : 'none';
                    }
                }
                
                // Add query to editor
                function addQueryToEditor(queryName, query) {
                    const queryInput = document.getElementById('query-input');
                    const queryText = `query {
  ${queryName}${getQueryArgs(query.args)} {
    # Add fields here
  }
}`;
                    queryInput.value = queryText;
                }
                
                // Add mutation to editor
                function addMutationToEditor(mutationName, mutation) {
                    const queryInput = document.getElementById('query-input');
                    const queryText = `mutation {
  ${mutationName}${getQueryArgs(mutation.args)} {
    # Add fields here
  }
}`;
                    queryInput.value = queryText;
                }
                
                // Get query arguments
                function getQueryArgs(args) {
                    if (!args || Object.keys(args).length === 0) {
                        return '';
                    }
                    
                    const argStrings = [];
                    for (const [argName, arg] of Object.entries(args)) {
                        argStrings.push(`${argName}: $${argName}`);
                    }
                    
                    return `(${argStrings.join(', ')})`;
                }
                
                // Run query
                document.getElementById('run-query').addEventListener('click', async () => {
                    const queryInput = document.getElementById('query-input');
                    const resultOutput = document.getElementById('result-output');
                    const token = document.getElementById('token').value;
                    
                    try {
                        const response = await fetch('/api/graphql', {
                            method: 'POST',
                            headers: {
                                'Content-Type': 'application/json',
                                'Authorization': token
                            },
                            body: JSON.stringify({
                                query: queryInput.value
                            })
                        });
                        
                        const result = await response.json();
                        resultOutput.textContent = JSON.stringify(result, null, 2);
                    } catch (error) {
                        resultOutput.textContent = `Error: ${error.message}`;
                    }
                });
                
                // Clear query
                document.getElementById('clear-query').addEventListener('click', () => {
                    document.getElementById('query-input').value = '';
                    document.getElementById('result-output').textContent = '// Results will appear here';
                });
                
                // Initialize token from localStorage if available
                document.addEventListener('DOMContentLoaded', () => {
                    const token = localStorage.getItem('auth_token');
                    if (token) {
                        document.getElementById('token').value = token;
                    }
                    
                    // Save token to localStorage when changed
                    document.getElementById('token').addEventListener('change', (e) => {
                        localStorage.setItem('auth_token', e.target.value);
                    });
                });
            </script>
        </body>
        </html>
        """
    }
