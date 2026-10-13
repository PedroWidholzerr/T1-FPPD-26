# Roda os experimentos SEM -race (o detector distorce os tempos) e gera os CSVs em relatorio/dados/.
# A corretude já foi validada com -race (scripts\test.ps1 e go run -race).
# Uso: .\scripts\bench.ps1 [-Reps 5]
param([int]$Reps = 5)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)

# Compila antes e mede com os binários (go run incluiria o tempo de compilação).
New-Item -ItemType Directory -Force bin, relatorio\dados | Out-Null
go build -o bin\filosofos.exe ./filosofos
go build -o bin\pc.exe ./produtor-consumidor

[Console]::OutputEncoding = [Text.Encoding]::UTF8   # a saída do Go é UTF-8; sem isto os acentos do arquivo da base saem corrompidos
$csvF = 'relatorio\dados\filosofos.csv'
$csvP = 'relatorio\dados\pc.csv'
Remove-Item $csvF, $csvP, 'relatorio\dados\base-deadlock.txt' -ErrorAction SilentlyContinue

# ---------------------------------------------------------------- Problema 1
# Estratégias corretas: N em {5,10}, R=100, durações default, $Reps repetições (seed = nº da repetição).
foreach ($est in 'hierarquia', 'limite', 'garcom') {
    foreach ($n in 5, 10) {
        foreach ($rep in 1..$Reps) {
            Write-Host "filosofos: $est n=$n rep=$rep"
            .\bin\filosofos.exe "-estrategia=$est" -n $n -r 100 -seed $rep -csv $csvF | Out-Null
            if ($LASTEXITCODE -ne 0) { throw "filosofos falhou ($est n=$n rep=$rep)" }
        }
    }
}
# Base: uma execução mostrando o deadlock (a saída vai para um arquivo).
Write-Host "filosofos: base (deadlock esperado)"
.\bin\filosofos.exe -estrategia=base -n 5 -r 100 | Out-File -Encoding utf8 relatorio\dados\base-deadlock.txt

# ---------------------------------------------------------------- Problema 2
# P=4, C=4. Cenários (atrasos por item):
#   A: produtores mais rápidos (prod 1ms, cons 3ms)   -> buffer tende a encher
#   B: consumidores mais rápidos (prod 3ms, cons 1ms) -> buffer tende a esvaziar
#   C: sem atraso nenhum                              -> mede só o custo da sincronização
$cenarios = @(
    @{ nome = 'A'; prod = '1ms'; cons = '3ms'; itens = 300 },
    @{ nome = 'B'; prod = '3ms'; cons = '1ms'; itens = 300 },
    @{ nome = 'C'; prod = '0ms'; cons = '0ms'; itens = 50000 }
)
foreach ($c in $cenarios) {
    foreach ($versao in 'channel', 'sem') {
        foreach ($k in 1, 10, 100) {
            foreach ($rep in 1..$Reps) {
                Write-Host "pc: cenario=$($c.nome) $versao k=$k rep=$rep"
                .\bin\pc.exe "-versao=$versao" -p 4 -c 4 -k $k -itens $c.itens `
                    "-prod-delay=$($c.prod)" "-cons-delay=$($c.cons)" -seed $rep -csv $csvP | Out-Null
                if ($LASTEXITCODE -ne 0) { throw "pc falhou (cenario $($c.nome) $versao k=$k rep=$rep)" }
            }
        }
    }
}
Write-Host "Pronto. CSVs em relatorio\dados\. Rode .\scripts\agrega.ps1 para gerar as tabelas."
