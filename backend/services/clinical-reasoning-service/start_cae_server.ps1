# Clinical Assertion Engine (CAE) gRPC Server Startup Script
# PowerShell script to start the CAE gRPC server on Windows

Write-Host "🚀 Starting Clinical Assertion Engine (CAE) gRPC Server..." -ForegroundColor Green

# Change to the app directory
Set-Location -Path "app"

# Try different Python commands
$pythonCommands = @("python", "py", "python3", "C:\Python312\python.exe", "C:\Users\$env:USERNAME\AppData\Local\Programs\Python\Python312\python.exe")

$pythonFound = $false
foreach ($cmd in $pythonCommands) {
    try {
        $version = & $cmd --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Found Python: $cmd ($version)" -ForegroundColor Green
            Write-Host "🔧 Starting CAE gRPC Server on port 8027..." -ForegroundColor Yellow
            
            # Start the gRPC server
            & $cmd grpc_server.py
            $pythonFound = $true
            break
        }
    }
    catch {
        continue
    }
}

if (-not $pythonFound) {
    Write-Host "❌ Python not found. Please install Python or add it to PATH." -ForegroundColor Red
    Write-Host "💡 Try installing Python from: https://www.python.org/downloads/" -ForegroundColor Yellow
    exit 1
}
