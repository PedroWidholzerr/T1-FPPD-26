#!/usr/bin/env bash
# Falha (exit 1) se algum .go importar sync, sync/atomic ou context; depois roda go vet.
set -u
cd "$(dirname "$0")/.."

if proibidos=$(grep -rnE --include='*.go' '"(sync|sync/atomic|context)"' .); then
    echo "ERRO: imports proibidos encontrados:" >&2
    echo "$proibidos" | sed 's/^/  /' >&2
    exit 1
fi
echo "OK: nenhum import de sync, sync/atomic ou context."

go vet ./... || exit $?
echo "OK: go vet limpo."
