# Roda todos os testes com o detector de data race (exige gcc no Windows).
Set-Location (Split-Path $PSScriptRoot -Parent)
go test -race ./...
exit $LASTEXITCODE
