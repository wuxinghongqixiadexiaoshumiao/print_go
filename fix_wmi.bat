@echo off
title WMI Repair Script (Corrected)
setlocal

:: Verify Administrator Privileges
>nul 2>&1 "%SYSTEMROOT%\system32\cacls.exe" "%SYSTEMROOT%\system32\config\system"
if '%errorlevel%' NEQ '0' (
    echo.
    echo [ERROR] Administrator privileges are required to run this script.
    echo Please right-click this file and select 'Run as administrator'.
    echo.
    pause
    exit /b 1
)

cls
echo =================================================================
echo  Windows Management Instrumentation (WMI) Repair Tool
echo =================================================================
echo.
echo  This script will attempt to repair a potentially corrupted
echo  WMI repository, which is a likely cause of the application crash.
echo.
echo  The following steps will be performed:
echo    1. Stop the WMI service.
echo    2. Reset the WMI repository to its default state.
echo    3. Restart the WMI service.
echo.
echo  IMPORTANT: Please RESTART your computer after this script is done.
echo.
echo =================================================================
pause
cls

echo --- Step 1: Stopping the WMI service (winmgmt) ---
echo.
net stop winmgmt
if errorlevel 1 (
    echo [WARNING] Could not stop the service. It might already be stopped.
) else (
    echo [SUCCESS] WMI service stopped.
)
echo.
pause

echo --- Step 2: Resetting the WMI repository ---
echo This may take a moment...
echo.
winmgmt /resetrepository
if errorlevel 1 (
    echo [FATAL ERROR] Failed to reset the WMI repository. Cannot continue.
    pause
    exit /b 1
) else (
    echo [SUCCESS] WMI repository has been reset.
)
echo.
pause

echo --- Step 3: Restarting the WMI service ---
echo.
net start winmgmt
if errorlevel 1 (
    echo [FATAL ERROR] Failed to restart the WMI service.
    echo Please restart your computer manually.
) else (
    echo [SUCCESS] WMI service started.
)
echo.

echo =================================================================
echo  Repair process complete.
echo =================================================================
echo.
echo  IMPORTANT: Please RESTART YOUR COMPUTER now to ensure all
echo  changes take effect properly.
echo.
echo  After restarting, please try to run your Go application again.
echo.
pause