#!/usr/bin/env bash
# Roda todos os testes com o detector de data race (exige gcc/cgo no Linux).
cd "$(dirname "$0")/.."
go test -race ./...
