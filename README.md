# T1 — FPPD (PUCRS): dois problemas clássicos de concorrência em Go

1. **Jantar dos Filósofos** (`filosofos/`) — 4 estratégias: `base` (deadlock proposital), `hierarquia`, `limite`, `garcom`.
2. **Produtor/Consumidor** (`produtor-consumidor/`) — buffer limitado em duas versões: `channel` e `sem` (semáforos próprios).

**Regra central:** nenhum `sync`, `sync/atomic` ou `context`. `Mutex`, `Semaforo` e `Latch` (no lugar do `WaitGroup`)
foram implementados pelo grupo em cima de **channels** (`mutex.go`, `semaforo.go`, `latch.go`, copiados em cada pasta,
pois cada problema é um programa independente).

## Pré-requisitos (Windows + PowerShell, ou Linux)

- **Go** (o `go.mod` pede `go 1.27`): `winget install GoLang.Go`
- **gcc** — obrigatório para `go run -race` no Windows (o detector de race usa cgo):
  `winget install BrechtSanders.WinLibs.POSIX.UCRT`
- Abra um terminal **novo** depois de instalar (para o PATH atualizar). Confira com `go version`, `go env CGO_ENABLED` (deve ser `1`) e `gcc --version`.
- **Linux:** instale Go (`go 1.27`) e gcc (`sudo apt install gcc` ou equivalente; o `-race` precisa de cgo). Os scripts `.sh` só precisam de `bash` e `awk`.
- Não é preciso `make`; os scripts são PowerShell. Há uma versão `.ps1` (Windows) e uma `.sh` (Linux) de cada script.

## Scripts

| Comando | O que faz |
|---|---|
| `.\scripts\check.ps1` | falha se algum `.go` importar `sync`, `sync/atomic` ou `context`; roda `go vet ./...` |
| `.\scripts\test.ps1` | `go test -race ./...` |
| `.\scripts\bench.ps1` | compila os binários em `bin\` e roda os experimentos **sem** `-race`; gera `relatorio\dados\filosofos.csv` e `pc.csv` (~4 min) |
| `.\scripts\agrega.ps1` | agrega os CSVs em `relatorio\dados\tabelas.md` (mediana e [mín–máx]) |

Se o PowerShell bloquear scripts: `powershell -ExecutionPolicy Bypass -File .\scripts\check.ps1`.

### Linux

| Comando | O que faz |
|---|---|
| `./scripts/check.sh` | igual ao `check.ps1` |
| `./scripts/test.sh` | `go test -race ./...` |
| `./scripts/bench.sh [reps]` | compila em `bin/` e roda os experimentos (padrão 5 repetições; ~4 min) |
| `./scripts/agrega.sh` | gera `relatorio/dados/tabelas.md` (mesma saída do `agrega.ps1`) |

Se faltar permissão de execução: `chmod +x scripts/*.sh` (ou `bash scripts/check.sh`).

## Problema 1 — Filósofos

```powershell
go run -race ./filosofos -estrategia=base       -n 5 -r 20   # trava: imprime DEADLOCK DETECTADO
go run -race ./filosofos -estrategia=hierarquia -n 5 -r 20
go run -race ./filosofos -estrategia=limite     -n 5 -r 20
go run -race ./filosofos -estrategia=garcom     -n 5 -r 20
go run ./filosofos -estrategia=garcom -n 10 -r 100 -csv resultados.csv
```

| Flag | Padrão | Significado |
|---|---|---|
| `-n` | 5 | número de filósofos (≥ 2) |
| `-r` | 10 | ciclos **por filósofo** (pensar → fome → pegar → comer → soltar) |
| `-estrategia` | hierarquia | `base`, `hierarquia`, `limite` ou `garcom` |
| `-pensar`, `-comer` | 10ms | duração média, com jitter aleatório de ±50% |
| `-pausa` | 50ms | espera entre o 1º e o 2º garfo na `base` |
| `-timeout` | 2s | watchdog: tempo sem nenhum evento para declarar deadlock |
| `-seed` | 1 | semente (reprodutibilidade) |
| `-csv` | — | arquivo CSV onde anexar uma linha por filósofo |

Saída: tabela por filósofo (refeições, espera média/máx/p95/total, instante de término) e agregados
(desvio-padrão das esperas médias, índice de Jain, ordem de término) e `runtime.NumGoroutine() = 1`.
Na `base` o programa termina com código 0 após imprimir o estado de cada filósofo. Um deadlock em qualquer
outra estratégia seria bug e termina com código 1. Cuidado: `-timeout` menor que `-pensar`/`-comer`/`-pausa`
causa falso positivo.

## Problema 2 — Produtor/Consumidor

```powershell
go run -race ./produtor-consumidor -versao=channel
go run -race ./produtor-consumidor -versao=sem -k 1
# demonstra o timeout do consumidor 0 (comportamento "ocioso"):
go run -race ./produtor-consumidor -versao=sem -prod-delay=50ms -timeout=20ms -p 1 -c 2 -itens 20
```

| Flag | Padrão | Significado |
|---|---|---|
| `-p`, `-c` | 4, 4 | produtores, consumidores |
| `-k` | 10 | capacidade do buffer |
| `-itens` | 1000 | itens **por produtor** |
| `-versao` | channel | `channel` ou `sem` |
| `-timeout` | 200ms | timeout do consumidor 0 (os outros bloqueiam sem limite) |
| `-prod-delay`, `-cons-delay` | 0 | tempo médio simulado por item (jitter ±50%) |
| `-seed`, `-csv` | 1, — | semente; CSV onde anexar uma linha por execução |

O programa termina com `VERIFICAÇÃO FINAL: OK` (nada perdido, nada repetido, `NumGoroutine == 1`) ou, se falhar,
`FALHOU` e código 1.

## Testes e race detector

```powershell
.\scripts\test.ps1          # go test -race ./...  (Mutex, Semáforo e Latch)
.\scripts\check.ps1         # nada de sync/atomic/context + go vet
```

## Estrutura

```
filosofos/            main.go (laço do filósofo, monitor/watchdog)  estrategias.go  stats.go  mutex.go semaforo.go latch.go
produtor-consumidor/  main.go  buffer.go  buffer_channel.go  buffer_sem.go  stats.go  mutex.go semaforo.go latch.go
scripts/              check, test, bench, agrega (.ps1 para Windows, .sh para Linux)
relatorio/            relatorio.md (rascunho)  dados/ (CSVs, base-deadlock.txt, tabelas.md)
AI-USO.md             registro do uso de IA
```
