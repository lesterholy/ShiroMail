$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot\..
docker compose down -v --remove-orphans
docker compose up -d --build
cmd /c "if exist frontend\\tsconfig.tsbuildinfo del /f /q frontend\\tsconfig.tsbuildinfo"
cmd /c "if exist frontend\\playwright-report rmdir /s /q frontend\\playwright-report"
cmd /c "if exist frontend\\test-results rmdir /s /q frontend\\test-results"
