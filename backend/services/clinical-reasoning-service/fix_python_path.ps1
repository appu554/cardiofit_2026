# PowerShell script to fix Python PATH issue permanently
# Run this script as Administrator

Write-Host "🔧 Fixing Python PATH Issue" -ForegroundColor Green
Write-Host "=" * 40

# Define the Python Scripts path
$pythonScriptsPath = "C:\Users\apoor\AppData\Local\Packages\PythonSoftwareFoundation.Python.3.12_qbz5n2kfra8p0\LocalCache\local-packages\Python312\Scripts"

# Check if the path exists
if (Test-Path $pythonScriptsPath) {
    Write-Host "✅ Python Scripts directory found: $pythonScriptsPath" -ForegroundColor Green
} else {
    Write-Host "❌ Python Scripts directory not found: $pythonScriptsPath" -ForegroundColor Red
    Write-Host "Please check your Python installation path" -ForegroundColor Yellow
    exit 1
}

# Get current user PATH
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")

# Check if the path is already in PATH
if ($currentPath -like "*$pythonScriptsPath*") {
    Write-Host "✅ Python Scripts path is already in PATH" -ForegroundColor Green
} else {
    Write-Host "📝 Adding Python Scripts path to user PATH..." -ForegroundColor Yellow
    
    # Add the path to user PATH
    $newPath = $currentPath + ";" + $pythonScriptsPath
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
}

# Set up PYTHONPATH for the project
$projectRoot = "D:\angular project\clinical-synthesis-hub\vaidshala"
$backendRoot = "$projectRoot\backend"
$serviceRoot = "$backendRoot\services\clinical-reasoning-service"

Write-Host "\n🔧 Setting up PYTHONPATH for the project" -ForegroundColor Green
Write-Host "=" * 40

# Get current PYTHONPATH
$currentPythonPath = [Environment]::GetEnvironmentVariable("PYTHONPATH", "User")

# Check if paths are already in PYTHONPATH
$pathsToAdd = @($projectRoot, $backendRoot, $serviceRoot)
$pathsAdded = $false

foreach ($path in $pathsToAdd) {
    if ($currentPythonPath -notlike "*$path*") {
        Write-Host "📝 Adding $path to PYTHONPATH..." -ForegroundColor Yellow
        if ($currentPythonPath) {
            $currentPythonPath = $currentPythonPath + ";" + $path
        } else {
            $currentPythonPath = $path
        }
        $pathsAdded = $true
    } else {
        Write-Host "✅ $path is already in PYTHONPATH" -ForegroundColor Green
    }
}

if ($pathsAdded) {
    [Environment]::SetEnvironmentVariable("PYTHONPATH", $currentPythonPath, "User")
    Write-Host "✅ PYTHONPATH updated successfully" -ForegroundColor Green
} else {
    Write-Host "✅ All required paths are already in PYTHONPATH" -ForegroundColor Green
}
    
    Write-Host "✅ Python Scripts path added to PATH successfully!" -ForegroundColor Green
}

# Also add Python executable paths if not present
$pythonExePath = "C:\Users\apoor\AppData\Local\Microsoft\WindowsApps\PythonSoftwareFoundation.Python.3.12_qbz5n2kfra8p0"

if (Test-Path $pythonExePath) {
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$pythonExePath*") {
        Write-Host "📝 Adding Python executable path to user PATH..." -ForegroundColor Yellow
        $newPath = $currentPath + ";" + $pythonExePath
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        Write-Host "✅ Python executable path added to PATH!" -ForegroundColor Green
    } else {
        Write-Host "✅ Python executable path is already in PATH" -ForegroundColor Green
    }
}

Write-Host ""
Write-Host "🎉 PATH configuration complete!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Next steps:" -ForegroundColor Yellow
Write-Host "1. Close and reopen your terminal/command prompt" -ForegroundColor White
Write-Host "2. Test with: py --version" -ForegroundColor White
Write-Host "3. Test with: pip --version" -ForegroundColor White
Write-Host ""
Write-Host "🔄 If you're using Git Bash, you may need to restart it completely" -ForegroundColor Cyan

# Display current PATH for verification
Write-Host ""
Write-Host "📊 Current User PATH:" -ForegroundColor Cyan
$finalPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathEntries = $finalPath -split ";"
foreach ($entry in $pathEntries) {
    if ($entry -like "*Python*") {
        Write-Host "  ✅ $entry" -ForegroundColor Green
    } else {
        Write-Host "  - $entry" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "✨ Python PATH fix completed!" -ForegroundColor Green
