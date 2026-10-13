#!/usr/bin/env bash
# Roda os experimentos SEM -race (o detector distorce os tempos) e gera os CSVs em relatorio/dados/.
# A corretude já foi validada com -race (scripts/test.sh e go run -race).
# Uso: ./scripts/bench.sh [reps]     (padrão: 5)
set -eu
REPS="${1:-5}"
cd "$(dirname "$0")/.."

# Compila antes e mede com os binários (go run incluiria o tempo de compilação).
mkdir -p bin relatorio/dados
go build -o bin/filosofos ./filosofos
go build -o bin/pc ./produtor-consumidor

CSV_F=relatorio/dados/filosofos.csv
CSV_P=relatorio/dados/pc.csv
rm -f "$CSV_F" "$CSV_P" relatorio/dados/base-deadlock.txt

# ---------------------------------------------------------------- Problema 1
# Estratégias corretas: N em {5,10}, R=100, durações default, REPS repetições (seed = nº da repetição).
for est in hierarquia limite garcom; do
    for n in 5 10; do
        for rep in $(seq 1 "$REPS"); do
            echo "filosofos: $est n=$n rep=$rep"
            ./bin/filosofos "-estrategia=$est" -n "$n" -r 100 -seed "$rep" -csv "$CSV_F" >/dev/null \
                || { echo "filosofos falhou ($est n=$n rep=$rep)" >&2; exit 1; }
        done
    done
done
# Base: uma execução mostrando o deadlock (a saída vai para um arquivo).
# O "|| true" evita abortar caso o programa saia com código != 0 ao detectar o deadlock.
echo "filosofos: base (deadlock esperado)"
./bin/filosofos -estrategia=base -n 5 -r 100 > relatorio/dados/base-deadlock.txt || true

# ---------------------------------------------------------------- Problema 2
# P=4, C=4. Cenários (atrasos por item), formato "nome prod cons itens":
#   A: produtores mais rápidos (prod 1ms, cons 3ms)   -> buffer tende a encher
#   B: consumidores mais rápidos (prod 3ms, cons 1ms) -> buffer tende a esvaziar
#   C: sem atraso nenhum                              -> mede só o custo da sincronização
while read -r nome prod cons itens; do
    for versao in channel sem; do
        for k in 1 10 100; do
            for rep in $(seq 1 "$REPS"); do
                echo "pc: cenario=$nome $versao k=$k rep=$rep"
                ./bin/pc "-versao=$versao" -p 4 -c 4 -k "$k" -itens "$itens" \
                    "-prod-delay=$prod" "-cons-delay=$cons" -seed "$rep" -csv "$CSV_P" >/dev/null \
                    || { echo "pc falhou (cenario $nome $versao k=$k rep=$rep)" >&2; exit 1; }
            done
        done
    done
done <<'CENARIOS'
A 1ms 3ms 300
B 3ms 1ms 300
C 0ms 0ms 50000
CENARIOS
echo "Pronto. CSVs em relatorio/dados/. Rode ./scripts/agrega.sh para gerar as tabelas."
