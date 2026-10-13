# Registro de uso de IA

Ferramenta: **Claude Code** (modelo Claude Sonnet 5.5, Anthropic), executando no terminal/VS Code do grupo,
a partir de um prompt detalhado escrito pelo grupo (fases, regras, estrutura de arquivos). Este arquivo alimenta a
seção obrigatória de uso de IA do relatório. O grupo deve revisar e ajustar o texto para refletir o que de fato fez.

## O que a IA fez, por fase

| Fase | Atividade | O que foi gerado / decidido pela IA |
|---|---|---|
| 0 — Ambiente e repositório | Preparação | Diagnosticou que Go e gcc não estavam instalados e os instalou via `winget` (após confirmação). Inicializou o repositório, criou `.gitignore`, `.gitattributes`, `go.mod`, `CLAUDE.md`. Ao falhar a autenticação SSH, gerou uma chave nova dedicada (`id_ed25519_github_t1`) e configurou `core.sshCommand` só neste repositório; o cadastro da chave pública no GitHub foi feito pelo usuário. |
| 1 — Primitivos | Código + testes | `Mutex`, `Semaforo`, `Latch` sobre channels; testes (contador 8×100 mil, panic em `Unlock`/`Release`, timeouts, limite de simultaneidade, `Wait` do `Latch`); `check.ps1` e `test.ps1`. |
| 2 — Filósofos | Projeto e código | Interface `Estrategia`, as 4 estratégias, garçom com fila FIFO, monitor/watchdog por channel, métricas (p95, Jain, desvio-padrão), saída CSV. |
| 3 — Produtor/Consumidor | Projeto e código | Interface `Buffer`, versões `channel` e `sem` (anel + 2 semáforos + 2 mutexes + poison pills), consumidor 0 com timeout, verificação final por mapa, amostrador de ocupação. |
| 4 — Experimentos | Scripts e análise | `bench.ps1` e `agrega.ps1`; execução dos experimentos; geração de `tabelas.md`. |
| 5 — Documentação | Texto | `README.md`, este arquivo e o rascunho de `relatorio/relatorio.md` (números copiados das tabelas geradas). |

## Depuração feita durante o trabalho (para o grupo conhecer)

- Os parâmetros de tempo foram primeiro `float` em ms e foram trocados para `time.Duration` (`-prod-delay=50ms`), como no enunciado.
- No PowerShell 5.1, `-estrategia=$est` não expande a variável (precisa de aspas) e `.ps1` com acentos precisa de BOM UTF-8; ambos corrigidos.
- No Windows, a duração medida de uma execução muito curta (4000 itens em poucos ms) saiu como **0**, gerando `throughput = +Inf`. O cenário sem atraso passou a usar 50 000 itens por produtor e o cálculo ficou protegido contra divisão por zero.

## Limites e responsabilidade do grupo

- A IA não substitui o entendimento: **cada integrante precisa saber explicar qualquer linha** (ver as perguntas prováveis no `README.md`).
- Interpretações de causa (por que um resultado saiu assim) estão marcadas `[CONFIRMAR]` no relatório e devem ser conferidas pelo grupo.
- Todos os números do relatório vêm dos CSVs em `relatorio/dados/`, gerados por `scripts/bench.ps1`; nenhum foi inventado.
- A IA rodou os testes e as execuções com `-race` na máquina do grupo (Windows 11, Go 1.27.0).
