# Lê os CSVs de relatorio/dados e gera tabelas Markdown (mediana e [mín–máx]) em tabelas.md.
$ErrorActionPreference = 'Stop'
Set-Location (Split-Path $PSScriptRoot -Parent)
$ci = [Globalization.CultureInfo]::InvariantCulture   # ponto decimal, independente do Windows

function Num($s) { [double]::Parse($s, $ci) }
function Mediana($xs) {
    $o = @($xs | Sort-Object)
    $m = [int][math]::Floor($o.Count / 2)
    if ($o.Count % 2 -eq 1) { $o[$m] } else { ($o[$m - 1] + $o[$m]) / 2 }
}
# "mediana [mín–máx]" com `casas` decimais
function Resumo($xs, $casas) {
    $f = "F$casas"
    $med = (Mediana $xs).ToString($f, $ci)
    $mn = ($xs | Measure-Object -Minimum).Minimum.ToString($f, $ci)
    $mx = ($xs | Measure-Object -Maximum).Maximum.ToString($f, $ci)
    "$med [$mn–$mx]"
}

$saida = New-Object System.Collections.Generic.List[string]
$saida.Add("# Tabelas dos experimentos")
$saida.Add("")
$saida.Add("Gerado por ``scripts/agrega.ps1`` a partir dos CSVs desta pasta. Cada célula é **mediana [mín–máx]** sobre as repetições.")
$saida.Add("")

# ---------------------------------------------------------------- Problema 1
if (Test-Path relatorio\dados\filosofos.csv) {
    $linhas = Import-Csv relatorio\dados\filosofos.csv
    $saida.Add("## Problema 1 — Jantar dos Filósofos (R=100, pensar=comer=10ms)")
    $saida.Add("")
    $saida.Add("Cada execução vira um ponto: espera média = média das esperas médias dos filósofos; espera máx = maior espera de qualquer filósofo.")
    $saida.Add("")
    $saida.Add("| estratégia | N | reps | duração (ms) | espera média (ms) | espera máx (ms) | desvio das médias (ms) | Jain |")
    $saida.Add("|---|---|---|---|---|---|---|---|")
    foreach ($g in $linhas | Group-Object { "$($_.estrategia)|$([int]$_.n)" } | Sort-Object { [int]($_.Name.Split('|')[1]) }, Name) {
        # uma "execução" = mesmo (estratégia, N, seed)
        $execs = $g.Group | Group-Object seed
        $dur = @(); $med = @(); $mx = @(); $dp = @(); $j = @()
        foreach ($e in $execs) {
            $dur += Num $e.Group[0].duracao_ms
            $dp += Num $e.Group[0].desvio_medias_ms
            $j += Num $e.Group[0].jain
            $med += ($e.Group | ForEach-Object { Num $_.espera_media_ms } | Measure-Object -Average).Average
            $mx += ($e.Group | ForEach-Object { Num $_.espera_max_ms } | Measure-Object -Maximum).Maximum
        }
        $p = $g.Name.Split('|')
        $saida.Add("| $($p[0]) | $($p[1]) | $($execs.Count) | $(Resumo $dur 0) | $(Resumo $med 2) | $(Resumo $mx 2) | $(Resumo $dp 3) | $(Resumo $j 4) |")
    }
    $saida.Add("")
    if (Test-Path relatorio\dados\base-deadlock.txt) {
        $saida.Add("### Versão base (deadlock)")
        $saida.Add("")
        $saida.Add('```')
        Get-Content relatorio\dados\base-deadlock.txt | ForEach-Object { $saida.Add($_) }
        $saida.Add('```')
        $saida.Add("")
    }
}

# ---------------------------------------------------------------- Problema 2
if (Test-Path relatorio\dados\pc.csv) {
    $linhas = Import-Csv relatorio\dados\pc.csv
    $rotulo = @{ '1|3' = 'A (produtores rápidos: prod 1ms, cons 3ms)'; '3|1' = 'B (consumidores rápidos: prod 3ms, cons 1ms)'; '0|0' = 'C (sem atraso)' }
    $saida.Add("## Problema 2 — Produtor/Consumidor (P=4, C=4)")
    $saida.Add("")
    foreach ($cen in $linhas | Group-Object { "$([double]$_.prod_delay_ms)|$([double]$_.cons_delay_ms)" } | Sort-Object { $rotulo[$_.Name] }) {
        $nome = $rotulo[$cen.Name]; if (-not $nome) { $nome = $cen.Name }
        $saida.Add("### Cenário $nome — itens por produtor: $($cen.Group[0].itens)")
        $saida.Add("")
        $saida.Add("| versão | K | reps | throughput (itens/s) | duração (ms) | ocupação média | timeouts do cons. 0 |")
        $saida.Add("|---|---|---|---|---|---|---|")
        foreach ($g in $cen.Group | Group-Object { "$([int]$_.k)|$($_.versao)" } | Sort-Object { [int]($_.Name.Split('|')[0]) }, Name) {
            $p = $g.Name.Split('|')
            $thr = $g.Group | ForEach-Object { Num $_.throughput }
            $dur = $g.Group | ForEach-Object { Num $_.duracao_ms }
            $oc = $g.Group | ForEach-Object { Num $_.ocupacao_media }
            $to = $g.Group | ForEach-Object { Num $_.timeouts }
            $saida.Add("| $($p[1]) | $($p[0]) | $($g.Count) | $(Resumo $thr 0) | $(Resumo $dur 1) | $(Resumo $oc 2) | $(Resumo $to 0) |")
        }
        $saida.Add("")
    }
}

$saida | Out-File -Encoding utf8 relatorio\dados\tabelas.md
Write-Host "Gerado relatorio\dados\tabelas.md"
