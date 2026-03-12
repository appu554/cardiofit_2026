@echo off
echo ===============================================
echo RUST ENGINE BUILD STATUS TEST
echo ===============================================
echo.

echo Testing Rust Engine Compilation...
echo.

cargo check --lib > build_output.txt 2>&1

if %ERRORLEVEL% EQU 0 (
    echo ✅ Compilation successful!
    echo Engine is ready for testing!
) else (
    echo ❌ Compilation failed!
    echo.
    echo Build errors saved to build_output.txt
    echo.
    echo First few lines of errors:
    type build_output.txt | findstr /C:"error[E"
)

echo.
echo ===============================================
echo BUILD TEST COMPLETE
echo ===============================================
pause
