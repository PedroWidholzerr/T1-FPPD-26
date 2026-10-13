# Falha (exit 1) se algum .go importar sync, sync/atomic ou context; depois roda go vet.
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)

$proibidos = Get-ChildItem -Recurse -Filter *.go | Select-String -Pattern '"(sync|sync/atomic|context)"'
if ($proibidos) {
    Write-Host "ERRO: imports proibidos encontrados:" -ForegroundColor Red
    $proibidos | ForEach-Object { Write-Host "  $($_.Path):$($_.LineNumber): $($_.Line.Trim())" }
    exit 1
}
Write-Host "OK: nenhum import de sync, sync/atomic ou context."

go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "OK: go vet limpo."
