# Camunda Cloud Setup Guide

This guide will help you set up Camunda Cloud integration for the Workflow Engine Service.

## 🌟 Why Camunda Cloud?

- **No Infrastructure Management** - Fully managed service
- **Enterprise Features** - Built-in monitoring, scaling, and reliability
- **Web Modeler** - Design BPMN workflows directly in the browser
- **REST API** - Perfect integration with your existing service
- **Free Tier** - Great for development and testing

## 🚀 Step-by-Step Setup

### Step 1: Create Camunda Cloud Account

1. **Visit Camunda Cloud**: https://camunda.com/products/cloud/
2. **Click "Start Free Trial"**
3. **Fill out registration form**:
   - Email address
   - Company name: `Clinical Synthesis Hub`
   - Use case: `Healthcare Workflow Management`
4. **Verify your email** and complete registration

### Step 2: Create a Cluster

1. **Login to Camunda Cloud Console**: https://console.cloud.camunda.io/
2. **Click "Create New Cluster"**
3. **Configure cluster settings**:
   - **Name**: `clinical-synthesis-hub`
   - **Plan**: `Development` (free tier)
   - **Region**: Choose closest to your location (e.g., `us-east-1`)
4. **Click "Create Cluster"**
5. **Wait 2-3 minutes** for cluster creation

### Step 3: Get API Credentials

1. **Go to your cluster dashboard**
2. **Click "API" tab**
3. **Click "Create New Client"**
4. **Configure client**:
   - **Name**: `workflow-engine-service`
   - **Scopes**: Select all available scopes:
     - ✅ Zeebe
     - ✅ Tasklist
     - ✅ Operate
     - ✅ Optimize
5. **Click "Create"**
6. **Copy credentials** (you'll need these):
   ```
   Client ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
   Client Secret: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   Cluster ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
   Region: us-east-1
   Authorization Server URL: https://login.cloud.camunda.io/oauth/token
   ```

### Step 4: Configure Workflow Engine Service

1. **Copy environment file**:
   ```bash
   cd backend/services/workflow-engine-service
   cp .env.example .env
   ```

2. **Edit `.env` file** with your Camunda Cloud credentials:
   ```bash
   # Workflow Engine Configuration
   USE_CAMUNDA_CLOUD=true
   CAMUNDA_CLOUD_CLIENT_ID=your_client_id_here
   CAMUNDA_CLOUD_CLIENT_SECRET=your_client_secret_here
   CAMUNDA_CLOUD_CLUSTER_ID=your_cluster_id_here
   CAMUNDA_CLOUD_REGION=your_region_here
   CAMUNDA_CLOUD_AUTHORIZATION_SERVER_URL=https://login.cloud.camunda.io/oauth/token
   ```

### Step 5: Install Dependencies

```bash
cd backend/services/workflow-engine-service
pip install -r requirements.txt
```

### Step 6: Test Connection

1. **Start the service**:
   ```bash
   python run_service.py
   ```

2. **Check health endpoint**:
   ```bash
   curl http://localhost:8015/health
   ```

3. **Look for these status indicators**:
   ```json
   {
     "status": "healthy",
     "use_camunda_cloud": true,
     "camunda_cloud_initialized": true,
     "workflow_engine_initialized": true
   }
   ```

## 🎯 Deploy Your First Workflow

### Step 7: Deploy Sample Workflow

1. **Use the Web Modeler**:
   - Go to your Camunda Cloud console
   - Click "Modeler" tab
   - Click "Create New File" → "BPMN Diagram"
   - Import the sample workflow: `workflows/patient-admission-workflow.bpmn`

2. **Or deploy via API**:
   ```python
   # Example deployment code
   from app.services.camunda_cloud_service import camunda_cloud_service
   
   # Read BPMN file
   with open('workflows/patient-admission-workflow.bpmn', 'r') as f:
       bpmn_xml = f.read()
   
   # Deploy workflow
   process_key = await camunda_cloud_service.deploy_workflow(
       bpmn_xml=bpmn_xml,
       workflow_name="patient-admission-workflow"
   )
   ```

### Step 8: Start a Workflow Instance

```python
# Start workflow instance
instance_key = await camunda_cloud_service.start_process_instance(
    process_key="patient-admission-workflow",
    variables={
        "patientData": {
            "name": "John Doe",
            "dateOfBirth": "1990-01-01",
            "mrn": "MRN123456"
        },
        "assignee": "doctor@hospital.com"
    }
)
```

## 🔧 GraphQL Integration

### Test Workflow Operations

```graphql
# Start a workflow
mutation {
  startWorkflow(
    definitionId: 1
    patientId: "patient-123"
    initialVariables: [
      { key: "patientData", value: "{\"name\":\"John Doe\"}" }
    ]
  ) {
    id
    status
    startTime
  }
}

# Get workflow instances
query {
  workflowInstances(status: ACTIVE) {
    id
    status
    startTime
    variables
  }
}

# Get tasks for a user
query {
  tasks(assignee: "doctor@hospital.com", status: READY) {
    id
    name
    description
    priority
    dueDate
  }
}
```

## 📊 Monitoring and Management

### Camunda Cloud Console Features

1. **Operate**: Monitor running workflow instances
2. **Tasklist**: Manage human tasks
3. **Optimize**: Analyze workflow performance
4. **Modeler**: Design and edit BPMN workflows

### Access URLs
- **Console**: https://console.cloud.camunda.io/
- **Operate**: https://operate.camunda.io/
- **Tasklist**: https://tasklist.camunda.io/
- **Optimize**: https://optimize.camunda.io/

## 🚨 Troubleshooting

### Common Issues

1. **Authentication Failed**
   - Check client ID and secret
   - Verify cluster ID and region
   - Ensure scopes are correctly set

2. **Connection Timeout**
   - Check internet connectivity
   - Verify cluster is running
   - Check firewall settings

3. **Workflow Deployment Failed**
   - Validate BPMN XML syntax
   - Check process ID uniqueness
   - Verify task types are supported

### Debug Mode

Enable debug logging in `.env`:
```bash
LOG_LEVEL=DEBUG
```

### Health Check

Monitor service health:
```bash
curl http://localhost:8015/health | jq
```

## 🎉 Next Steps

1. **Design Custom Workflows** using Camunda Web Modeler
2. **Integrate with FHIR Resources** for healthcare-specific workflows
3. **Set up Monitoring** and alerts
4. **Scale to Production** with Camunda Cloud's enterprise features

## 📚 Resources

- [Camunda Cloud Documentation](https://docs.camunda.io/)
- [BPMN 2.0 Tutorial](https://camunda.com/bpmn/)
- [Zeebe Client Documentation](https://docs.camunda.io/docs/apis-clients/python-client/)
- [Healthcare Workflow Patterns](https://camunda.com/solutions/healthcare/)

## 💡 Pro Tips

1. **Use meaningful process keys** for easy identification
2. **Design workflows with error handling** using BPMN error events
3. **Leverage Camunda's built-in retry mechanisms**
4. **Monitor workflow performance** using Optimize
5. **Use correlation keys** for message-based workflows
