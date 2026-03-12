@echo off
REM Create a symbolic link to the shared directory

echo Creating symbolic link to shared directory...
mklink /D app\shared ..\..\shared

echo Done!
pause
