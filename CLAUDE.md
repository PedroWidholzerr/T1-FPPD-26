# T1 FPPD (PUCRS) — Jantar dos Filósofos e Produtor/Consumidor em Go

Trabalho em grupo (até 4 alunos) apresentado ao vivo: **todo o código precisa ser simples, curto,
comentado em português e fácil de explicar**. Clareza antes de esperteza; nada de abstração desnecessária.

## Regra central: nada de `sync`
Toda a sincronização é feita pelo grupo em cima de **channels** (`chan`, `select`, `close`, `len(chan)`).

| Pode | Não pode |
|---|---|
| `go`, `chan`, `select`, `close`, `len(chan)` | pacote `sync` (Mutex, RWMutex, WaitGroup, Cond, Once, Pool, Map) |
| `time`, `fmt`, `flag`, `os`, `sort`, `math`, `math/rand`, `runtime` | `sync/atomic` |
| `testing` (só em `_test.go`) | `context`, libs externas que abstraiam concorrência |

## Outras regras
- Tudo deve rodar limpo com `go run -race`.
- Cada problema (`filosofos/`, `produtor-consumidor/`) é um programa independente: os primitivos
  `mutex.go`, `semaforo.go`, `latch.go` ficam **copiados** (idênticos) nas duas pastas.
- Parâmetros por `flag`. Encerramento ordenado, sem goroutines bloqueadas
  (exceção: `-estrategia=base`, que trava de propósito e é detectada por watchdog).
- Ambiente: Windows + PowerShell. Scripts em `scripts/*.ps1`. Código Go portátil.
- Commits pequenos, em português: `tipo: descrição` (`feat:`, `test:`, `docs:`, `chore:`, `fix:`).
- Relatório: só usar números que vieram dos CSVs em `relatorio/dados/`; interpretações do grupo marcadas `[CONFIRMAR]`.

## Comandos
- `.\scripts\check.ps1`  — falha se algum `.go` importar `sync`, `sync/atomic` ou `context`; roda `go vet`.
- `.\scripts\test.ps1`   — `go test -race ./...`
- `.\scripts\bench.ps1`  — compila e roda os experimentos (sem `-race`), gera CSVs em `relatorio/dados/`.
- `.\scripts\agrega.ps1` — agrega os CSVs em `relatorio/dados/tabelas.md`.
