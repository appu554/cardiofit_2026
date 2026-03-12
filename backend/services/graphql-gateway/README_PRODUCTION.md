# GraphQL Gateway Production Deployment Guide

This guide provides instructions for deploying the GraphQL Gateway to a production environment.

## Prerequisites

- Python 3.8 or higher
- pip (Python package manager)
- Access to the FHIR and Auth services
- MongoDB Atlas account (for direct DB access)

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/your-org/clinical-synthesis-hub.git
   cd clinical-synthesis-hub/backend/clinical-synthesis-hub-graphql
   ```

2. Create a virtual environment:
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   ```

3. Install dependencies:
   ```bash
   pip install -r requirements.txt
   ```

## Configuration

1. Create a `.env` file in the `services/graphql-gateway` directory with the following content:
   ```
   # Service URLs
   FHIR_SERVICE_URL=https://your-fhir-service-url
   AUTH_SERVICE_URL=https://your-auth-service-url

   # Environment
   ENVIRONMENT=production

   # CORS
   ALLOWED_ORIGINS=https://your-frontend-url,https://another-allowed-origin

   # Performance
   REQUEST_TIMEOUT=30

   # Rate limiting
   RATE_LIMIT_REQUESTS=100
   RATE_LIMIT_WINDOW=60

   # MongoDB connection (for direct DB access)
   MONGODB_URI=mongodb+srv://username:password@your-mongodb-cluster.mongodb.net/fhirdb
   ```

2. Adjust the values according to your production environment.

## Running in Production

### Option 1: Using the run.py script

```bash
cd services/graphql-gateway
python run.py
```

### Option 2: Using Uvicorn directly

```bash
cd services/graphql-gateway
python -m uvicorn standalone_server:app --host 0.0.0.0 --port 8006 --workers 4
```

### Option 3: Using Gunicorn with Uvicorn workers (recommended)

```bash
cd services/graphql-gateway
gunicorn standalone_server:app -w 4 -k uvicorn.workers.UvicornWorker -b 0.0.0.0:8006
```

### Option 4: Using Docker

1. Build the Docker image:
   ```bash
   docker build -t graphql-gateway -f services/graphql-gateway/Dockerfile .
   ```

2. Run the Docker container:
   ```bash
   docker run -d -p 8006:8006 --env-file services/graphql-gateway/.env --name graphql-gateway graphql-gateway
   ```

## Deployment Options

### Docker Compose

Create a `docker-compose.yml` file:

```yaml
version: '3'
services:
  graphql-gateway:
    build:
      context: .
      dockerfile: services/graphql-gateway/Dockerfile
    ports:
      - "8006:8006"
    env_file:
      - services/graphql-gateway/.env
    restart: always
```

Run with:
```bash
docker-compose up -d
```

### Kubernetes

1. Create a ConfigMap for the environment variables:
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: graphql-gateway-config
   data:
     FHIR_SERVICE_URL: "https://your-fhir-service-url"
     AUTH_SERVICE_URL: "https://your-auth-service-url"
     ENVIRONMENT: "production"
     ALLOWED_ORIGINS: "https://your-frontend-url,https://another-allowed-origin"
     REQUEST_TIMEOUT: "30"
     RATE_LIMIT_REQUESTS: "100"
     RATE_LIMIT_WINDOW: "60"
   ```

2. Create a Secret for sensitive information:
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: graphql-gateway-secrets
   type: Opaque
   stringData:
     MONGODB_URI: "mongodb+srv://username:password@your-mongodb-cluster.mongodb.net/fhirdb"
   ```

3. Create a Deployment:
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: graphql-gateway
   spec:
     replicas: 3
     selector:
       matchLabels:
         app: graphql-gateway
     template:
       metadata:
         labels:
           app: graphql-gateway
       spec:
         containers:
         - name: graphql-gateway
           image: your-registry/graphql-gateway:latest
           ports:
           - containerPort: 8006
           envFrom:
           - configMapRef:
               name: graphql-gateway-config
           - secretRef:
               name: graphql-gateway-secrets
           resources:
             limits:
               cpu: "1"
               memory: "512Mi"
             requests:
               cpu: "0.5"
               memory: "256Mi"
           livenessProbe:
             httpGet:
               path: /health
               port: 8006
             initialDelaySeconds: 30
             periodSeconds: 10
           readinessProbe:
             httpGet:
               path: /health
               port: 8006
             initialDelaySeconds: 5
             periodSeconds: 5
   ```

4. Create a Service:
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: graphql-gateway
   spec:
     selector:
       app: graphql-gateway
     ports:
     - port: 80
       targetPort: 8006
     type: ClusterIP
   ```

5. Create an Ingress (if needed):
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: graphql-gateway-ingress
     annotations:
       nginx.ingress.kubernetes.io/ssl-redirect: "true"
   spec:
     rules:
     - host: api.your-domain.com
       http:
         paths:
         - path: /graphql
           pathType: Prefix
           backend:
             service:
               name: graphql-gateway
               port:
                 number: 80
     tls:
     - hosts:
       - api.your-domain.com
       secretName: your-tls-secret
   ```

## Monitoring and Logging

### Prometheus Metrics

The GraphQL Gateway exposes metrics at the `/metrics` endpoint. You can configure Prometheus to scrape these metrics.

### Logging

Logs are written to `graphql_gateway.log` and to stdout. In a production environment, you should configure a logging service like ELK Stack, Graylog, or a cloud-based logging solution.

## Security Considerations

1. Always use HTTPS in production
2. Set appropriate CORS settings
3. Configure rate limiting based on your expected traffic
4. Use a reverse proxy like Nginx in front of the application
5. Implement proper authentication and authorization
6. Regularly update dependencies

## Troubleshooting

If you encounter issues, check:

1. The application logs
2. The `/health` endpoint for service status
3. The `/metrics` endpoint for performance metrics
4. Connection to the FHIR and Auth services
5. MongoDB connection (for direct DB access)

## Support

For support, contact the development team at your-email@example.com.
