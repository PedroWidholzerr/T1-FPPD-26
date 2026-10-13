#!/usr/bin/env bash
# Lê os CSVs de relatorio/dados e gera tabelas Markdown (mediana e [mín–máx]) em tabelas.md.
# Só precisa de bash e awk (POSIX); mesma saída do agrega.ps1.
set -eu
cd "$(dirname "$0")/.."
export LC_ALL=C   # ponto decimal, independente do locale
D=relatorio/dados
OUT=$D/tabelas.md

# Funções awk comuns: mediana e "mediana [mín–máx]" com `casas` decimais.
FUNCS='
function ordena(a, n,   i, j, t) {
    for (i = 2; i <= n; i++) { t = a[i]; for (j = i - 1; j >= 1 && a[j] > t; j--) a[j+1] = a[j]; a[j+1] = t }
}
function resumo(a, n, casas,   o, i, m, med, f) {
    for (i = 1; i <= n; i++) o[i] = a[i]
    ordena(o, n)
    m = int(n / 2)
    med = (n % 2 == 1) ? o[m+1] : (o[m] + o[m+1]) / 2
    f = "%." casas "f [%." casas "f–%." casas "f]"
    return sprintf(f, med, o[1], o[n])
}
'

{
cat <<'CAB'
# Tabelas dos experimentos

Gerado por `scripts/agrega.ps1` a partir dos CSVs desta pasta. Cada célula é **mediana [mín–máx]** sobre as repetições.

CAB

# ---------------------------------------------------------------- Problema 1
if [ -f "$D/filosofos.csv" ]; then
cat <<'CAB'
## Problema 1 — Jantar dos Filósofos (R=100, pensar=comer=10ms)

Cada execução vira um ponto: espera média = média das esperas médias dos filósofos; espera máx = maior espera de qualquer filósofo.

| estratégia | N | reps | duração (ms) | espera média (ms) | espera máx (ms) | desvio das médias (ms) | Jain |
|---|---|---|---|---|---|---|---|
CAB
awk -F, "$FUNCS"'
NR == 1 { next }
{
    # uma "execução" = mesmo (estratégia, N, seed)
    run = $1 SUBSEP ($2 + 0) SUBSEP $6
    g = $1 SUBSEP ($2 + 0)
    if (!(run in vis)) {
        vis[run] = 1
        if (!(g in gv)) { gv[g] = 1; gl[++ng] = g; est[g] = $1; nn[g] = $2 + 0 }
        c = ++cnt[g]
        dur[g, c] = $14; dp[g, c] = $16; jain[g, c] = $15
        runs[g, c] = run
        mx[run] = -1
    }
    soma[run] += $9; qtd[run]++
    if ($10 + 0 > mx[run]) mx[run] = $10 + 0
}
END {
    # ordena grupos por N e depois por estratégia
    for (i = 2; i <= ng; i++) {
        t = gl[i]
        for (j = i - 1; j >= 1 && (nn[gl[j]] > nn[t] || (nn[gl[j]] == nn[t] && est[gl[j]] > est[t])); j--) gl[j+1] = gl[j]
        gl[j+1] = t
    }
    for (i = 1; i <= ng; i++) {
        g = gl[i]; n = cnt[g]
        for (c = 1; c <= n; c++) {
            d[c] = dur[g, c] + 0; p[c] = dp[g, c] + 0; j2[c] = jain[g, c] + 0
            r = runs[g, c]; m[c] = soma[r] / qtd[r]; x[c] = mx[r]
        }
        printf "| %s | %d | %d | %s | %s | %s | %s | %s |\n", est[g], nn[g], n, resumo(d, n, 0), resumo(m, n, 2), resumo(x, n, 2), resumo(p, n, 3), resumo(j2, n, 4)
    }
}' "$D/filosofos.csv"
echo
if [ -f "$D/base-deadlock.txt" ]; then
    echo "### Versão base (deadlock)"
    echo
    echo '```'
    tr -d '\r' < "$D/base-deadlock.txt" | sed '1s/^\xEF\xBB\xBF//'
    echo '```'
    echo
fi
fi

# ---------------------------------------------------------------- Problema 2
if [ -f "$D/pc.csv" ]; then
echo "## Problema 2 — Produtor/Consumidor (P=4, C=4)"
echo
awk -F, "$FUNCS"'
BEGIN {
    rot["1|3"] = "A (produtores rápidos: prod 1ms, cons 3ms)"
    rot["3|1"] = "B (consumidores rápidos: prod 3ms, cons 1ms)"
    rot["0|0"] = "C (sem atraso)"
}
NR == 1 { next }
{
    cen = ($6 + 0) "|" ($7 + 0)
    if (!(cen in cv)) { cv[cen] = 1; cl[++nc] = cen; itens[cen] = $5 }
    g = cen SUBSEP ($4 + 0) SUBSEP $1
    if (!(g in gv)) { gv[g] = 1; gl[cen, ++ngc[cen]] = g; kk[g] = $4 + 0; ver[g] = $1 }
    c = ++cnt[g]
    thr[g, c] = $11; dur[g, c] = $10; oc[g, c] = $12; to[g, c] = $13
}
END {
    for (i = 2; i <= nc; i++) {
        t = cl[i]
        for (j = i - 1; j >= 1 && rot[cl[j]] > rot[t]; j--) cl[j+1] = cl[j]
        cl[j+1] = t
    }
    for (a = 1; a <= nc; a++) {
        cen = cl[a]; nome = (cen in rot) ? rot[cen] : cen
        printf "### Cenário %s — itens por produtor: %s\n\n", nome, itens[cen]
        print "| versão | K | reps | throughput (itens/s) | duração (ms) | ocupação média | timeouts do cons. 0 |"
        print "|---|---|---|---|---|---|---|"
        n = ngc[cen]
        # ordena por K e depois por versão
        for (i = 2; i <= n; i++) {
            t = gl[cen, i]
            for (j = i - 1; j >= 1 && (kk[gl[cen, j]] > kk[t] || (kk[gl[cen, j]] == kk[t] && ver[gl[cen, j]] > ver[t])); j--) gl[cen, j+1] = gl[cen, j]
            gl[cen, j+1] = t
        }
        for (i = 1; i <= n; i++) {
            g = gl[cen, i]; r = cnt[g]
            for (c = 1; c <= r; c++) { A[c] = thr[g, c] + 0; B[c] = dur[g, c] + 0; C[c] = oc[g, c] + 0; E[c] = to[g, c] + 0 }
            printf "| %s | %d | %d | %s | %s | %s | %s |\n", ver[g], kk[g], r, resumo(A, r, 0), resumo(B, r, 1), resumo(C, r, 2), resumo(E, r, 0)
        }
        print ""
    }
}' "$D/pc.csv"
fi
} > "$OUT"
echo "Gerado $OUT"
