from fastapi import FastAPI, Depends, HTTPException, status, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.openapi.docs import get_swagger_ui_html
from fastapi.openapi.utils import get_openapi
from fastapi.staticfiles import StaticFiles
from fastapi.responses import HTMLResponse, FileResponse
from contextlib import asynccontextmanager
import os
from app.api.routes import router as auth_router
from app.config import settings
from app.security import get_token_from_header, verify_token

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup logic
    yield
    # Shutdown logic

# Initialize FastAPI app
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="""## Authentication Service for Clinical Synthesis Hub

    This API provides authentication and authorization services using Auth0 as the identity provider.

    ### Authentication Flows

    This service supports two main authentication flows:

    1. **Authorization Code Flow** (Recommended)
       - Secure flow for web applications
       - User is redirected to Auth0 for authentication
       - More secure and supports features like social login, MFA, etc.
       - Use `/api/auth/authorize` to start the flow
       - Use `/api/auth/callback` to exchange the code for tokens

    2. **Client Credentials Flow**
       - For machine-to-machine authentication
       - No user interaction required
       - Use `/api/auth/client-token` to get a token

    ### Token Management

    - Use `/api/auth/refresh` to refresh tokens when they expire
    - Use `/api/auth/logout` to log users out and revoke tokens
    - Use `/api/auth/verify` to verify tokens
    - Use `/api/auth/me` to get user information from a token

    ### Security

    All protected endpoints require a valid JWT token in the Authorization header:
    `Authorization: Bearer {token}`
    """,
    version="1.0.0",
    lifespan=lifespan,
    docs_url=None,  # Disable default docs
    redoc_url=None,  # Disable default redoc
    openapi_url="/api/auth/openapi.json",  # Custom OpenAPI URL
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Replace with specific origins in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(auth_router, prefix=settings.API_PREFIX)

# Custom Swagger UI route
@app.get(f"{settings.API_PREFIX}/auth/docs", include_in_schema=False)
async def custom_swagger_ui_html():
    return get_swagger_ui_html(
        openapi_url=app.openapi_url,
        title=f"{app.title} - API Documentation",
        oauth2_redirect_url=app.swagger_ui_oauth2_redirect_url,
        swagger_js_url="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui-bundle.js",
        swagger_css_url="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.9.0/swagger-ui.css",
        swagger_favicon_url="https://fastapi.tiangolo.com/img/favicon.png",
        swagger_ui_parameters={
            "docExpansion": "list",
            "defaultModelsExpandDepth": 1,
            "defaultModelExpandDepth": 2,
            "deepLinking": True,
            "displayRequestDuration": True,
            "syntaxHighlight.theme": "monokai"
        }
    )

# Custom OpenAPI schema with security definitions
@app.get(f"{settings.API_PREFIX}/auth/openapi.json", include_in_schema=False)
async def get_custom_openapi():
    if app.openapi_schema:
        return app.openapi_schema

    openapi_schema = get_openapi(
        title=app.title,
        version=app.version,
        description=app.description,
        routes=app.routes,
    )

    # Add security schemes
    openapi_schema["components"] = {
        "securitySchemes": {
            "BearerAuth": {
                "type": "http",
                "scheme": "bearer",
                "bearerFormat": "JWT",
                "description": "Enter JWT token",
            },
            "OAuth2": {
                "type": "oauth2",
                "flows": {
                    "password": {
                        "tokenUrl": f"https://{settings.AUTH0_DOMAIN}/oauth/token",
                        "scopes": {
                            "openid": "OpenID Connect",
                            "profile": "Profile information",
                            "email": "Email information",
                        },
                    },
                    "clientCredentials": {
                        "tokenUrl": f"https://{settings.AUTH0_DOMAIN}/oauth/token",
                        "scopes": {
                            "read:patients": "Read patient data",
                            "write:notes": "Write clinical notes",
                        },
                    },
                },
            },
        }
    }

    app.openapi_schema = openapi_schema
    return app.openapi_schema

# Health check endpoint
@app.get("/health")
async def health_check():
    return {"status": "ok"}

# Serve test pages
@app.get("/test-auth-code-flow.html", include_in_schema=False)
async def serve_test_auth_code_flow():
    test_page_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), "test-auth-code-flow.html")
    return FileResponse(test_page_path)

# Auth0 callback endpoint
@app.get(f"{settings.API_PREFIX}/auth/callback", include_in_schema=False)
async def auth0_callback():
    html_content = """
    <!DOCTYPE html>
    <html>
    <head>
        <title>Auth0 Callback</title>
        <script>
            // Extract token from URL hash and redirect back to the login page with the token
            window.onload = function() {
                const hash = window.location.hash;
                if (hash) {
                    // Parse the hash to extract the id_token
                    const hashParams = {};
                    hash.substring(1).split('&').forEach(function(item) {
                        const parts = item.split('=');
                        hashParams[parts[0]] = decodeURIComponent(parts[1]);
                    });

                    // Verify the nonce if id_token is present
                    if (hashParams.id_token) {
                        try {
                            // In a real implementation, you would verify the nonce in the id_token
                            // For now, we'll just pass the token back to the login page
                            console.log('ID token received');
                        } catch (error) {
                            console.error('Error verifying nonce:', error);
                        }
                    }

                    window.location.href = '/api/auth/test' + hash;
                } else {
                    window.location.href = '/api/auth/test';
                }
            };
        </script>
    </head>
    <body>
        <p>Processing authentication response...</p>
    </body>
    </html>
    """
    return HTMLResponse(content=html_content)

# Auth0 test page
@app.get("/api/auth/test", include_in_schema=False)
async def auth0_test_page():
    html_content = f"""
    <!DOCTYPE html>
    <html>
    <head>
        <title>Auth0 Test Page</title>
        <script src="https://cdn.auth0.com/js/auth0-spa-js/2.0/auth0-spa-js.production.js"></script>
        <style>
            body {{ font-family: Arial, sans-serif; margin: 20px; }}
            button {{ padding: 10px; margin: 5px; cursor: pointer; }}
            pre {{ background-color: #f0f0f0; padding: 10px; border-radius: 5px; overflow: auto; }}
        </style>
    </head>
    <body>
        <h1>Auth0 Test Page</h1>
        <div>
            <button id="login">Login</button>
            <button id="logout">Logout</button>
            <button id="get-token">Get Token</button>
            <button id="call-api">Call API</button>
        </div>
        <div>
            <h3>User Profile:</h3>
            <pre id="profile">Not logged in</pre>
            <h3>API Response:</h3>
            <pre id="api-response">No data</pre>
        </div>

        <script>
            let auth0Client = null;
            let token = null;

            window.onload = async function() {{
                auth0Client = await createAuth0Client({{
                    domain: '{settings.AUTH0_DOMAIN}',
                    clientId: '{settings.AUTH0_CLIENT_ID}',
                    authorizationParams: {{
                        audience: '{settings.AUTH0_API_AUDIENCE}',
                        redirect_uri: window.location.origin + '/api/auth/test'
                    }}
                }});

                // Check for authentication callback
                if (window.location.search.includes("code=")) {{
                    await auth0Client.handleRedirectCallback();
                    window.history.replaceState({{}}, document.title, window.location.pathname);
                    updateUI();
                }}

                document.getElementById('login').addEventListener('click', login);
                document.getElementById('logout').addEventListener('click', logout);
                document.getElementById('get-token').addEventListener('click', getToken);
                document.getElementById('call-api').addEventListener('click', callApi);

                updateUI();
            }};

            async function updateUI() {{
                const isAuthenticated = await auth0Client.isAuthenticated();
                if (isAuthenticated) {{
                    const user = await auth0Client.getUser();
                    document.getElementById('profile').textContent = JSON.stringify(user, null, 2);
                }} else {{
                    document.getElementById('profile').textContent = 'Not logged in';
                }}
            }}

            async function login() {{
                await auth0Client.loginWithRedirect();
            }}

            async function logout() {{
                auth0Client.logout({{
                    logoutParams: {{
                        returnTo: window.location.origin + '/api/auth/test'
                    }}
                }});
            }}

            async function getToken() {{
                try {{
                    token = await auth0Client.getTokenSilently();
                    alert('Token acquired! Check console for details.');
                    console.log('Token:', token);
                }} catch (error) {{
                    console.error('Error getting token:', error);
                    alert('Error getting token: ' + error.message);
                }}
            }}

            async function callApi() {{
                if (!token) {{
                    alert('Please get a token first');
                    return;
                }}

                try {{
                    const response = await fetch('/api/auth/me', {{
                        headers: {{
                            'Authorization': 'Bearer ' + token
                        }}
                    }});
                    const data = await response.json();
                    document.getElementById('api-response').textContent = JSON.stringify(data, null, 2);
                }} catch (error) {{
                    console.error('API Error:', error);
                    document.getElementById('api-response').textContent = 'Error: ' + error.message;
                }}
            }}
        </script>
    </body>
    </html>
    """
    from fastapi.responses import HTMLResponse
    return HTMLResponse(content=html_content)