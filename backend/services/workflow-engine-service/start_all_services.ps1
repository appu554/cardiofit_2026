# PowerShell script to start all required services for workflow integration testing
# Run this from the vaidshala root directory

Write-Host "🚀 Starting All Services for Workflow Integration Testing" -ForegroundColor Green
Write-Host "=" * 70 -ForegroundColor Green

# Check if we're in the right directory
if (-not (Test-Path "backend\services")) {
    Write-Host "❌ Please run this script from the vaidshala root directory" -ForegroundColor Red
    exit 1
}

# Function to start a service in a new PowerShell window
function Start-ServiceInNewWindow {
    param(
        [string]$ServiceName,
        [string]$Directory,
        [string]$Command,
        [int]$Port
    )
    
    Write-Host "🔄 Starting $ServiceName on port $Port..." -ForegroundColor Yellow
    
    $scriptBlock = @"
Set-Location '$Directory'
Write-Host '🚀 Starting $ServiceName...' -ForegroundColor Green
Write-Host 'Directory: $Directory' -ForegroundColor Cyan
Write-Host 'Command: $Command' -ForegroundColor Cyan
Write-Host 'Port: $Port' -ForegroundColor Cyan
Write-Host '=' * 50 -ForegroundColor Green
$Command
Write-Host '❌ $ServiceName has stopped. Press any key to close...' -ForegroundColor Red
Read-Host
"@
    
    Start-Process powershell -ArgumentList "-NoExit", "-Command", $scriptBlock
    Start-Sleep -Seconds 2
}

# Start services in order
Write-Host "`n1. Starting Patient Service (Port 8003)..." -ForegroundColor Cyan
Start-ServiceInNewWindow -ServiceName "Patient Service" -Directory "backend\services\patient-service" -Command "python start_service.py" -Port 8003

Write-Host "`n2. Starting Observation Service (Port 8007)..." -ForegroundColor Cyan
Start-ServiceInNewWindow -ServiceName "Observation Service" -Directory "backend\services\observation-service" -Command "python start_service.py" -Port 8007

Write-Host "`n3. Starting Encounter Service (Port 8020)..." -ForegroundColor Cyan
Start-ServiceInNewWindow -ServiceName "Encounter Service" -Directory "backend\services\encounter-service" -Command "python start_service.py" -Port 8020

Write-Host "`n4. Starting Workflow Engine Service (Port 8015)..." -ForegroundColor Cyan
Start-ServiceInNewWindow -ServiceName "Workflow Engine Service" -Directory "backend\services\workflow-engine-service" -Command "python start_service.py" -Port 8015

Write-Host "`n5. Starting Auth Service (Port 8001)..." -ForegroundColor Cyan
Start-ServiceInNewWindow -ServiceName "Auth Service" -Directory "backend\services\auth-service" -Command "npm start" -Port 8001

Write-Host "`n6. Starting API Gateway (Port 8005)..." -ForegroundColor Cyan
Start-ServiceInNewWindow -ServiceName "API Gateway" -Directory "backend\services\api-gateway" -Command "npm start" -Port 8005

Write-Host "`n7. Starting Apollo Federation Gateway (Port 4000)..." -ForegroundColor Cyan
Start-ServiceInNewWindow -ServiceName "Apollo Federation Gateway" -Directory "apollo-federation" -Command "npm start" -Port 4000

Write-Host "`n✅ All services are starting up..." -ForegroundColor Green
Write-Host "⏳ Please wait 30-60 seconds for all services to be ready" -ForegroundColor Yellow

Write-Host "`n📋 Service URLs:" -ForegroundColor Cyan
Write-Host "• Patient Service: http://localhost:8003/health" -ForegroundColor White
Write-Host "• Observation Service: http://localhost:8007/health" -ForegroundColor White
Write-Host "• Encounter Service: http://localhost:8020/health" -ForegroundColor White
Write-Host "• Workflow Service: http://localhost:8015/health" -ForegroundColor White
Write-Host "• Auth Service: http://localhost:8001/health" -ForegroundColor White
Write-Host "• API Gateway: http://localhost:8005/health" -ForegroundColor White
Write-Host "• Federation Gateway: http://localhost:4000/health" -ForegroundColor White

Write-Host "`n🧪 To test the integration:" -ForegroundColor Green
Write-Host "cd backend\services\workflow-engine-service" -ForegroundColor White
Write-Host "python test_real_services_integration.py" -ForegroundColor White

Write-Host "`n🔍 To check service health:" -ForegroundColor Green
Write-Host "curl http://localhost:4000/health" -ForegroundColor White

Write-Host "`n⚠️  Note: Each service will open in a separate PowerShell window" -ForegroundColor Yellow
Write-Host "Close individual windows to stop services, or use Ctrl+C in each window" -ForegroundColor Yellow

Write-Host "`nPress any key to continue..." -ForegroundColor Green
Read-Host
