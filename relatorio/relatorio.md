# T1 — Programação Concorrente: Jantar dos Filósofos e Produtor/Consumidor em Go

**Disciplina:** FPPD — PUCRS  **Grupo:** [CONFIRMAR nomes]  **Repositório:** github.com/PedroWidholzerr/T1-FPPD-26

> Rascunho. Os números vêm de `relatorio/dados/*.csv` (tabelas completas em `relatorio/dados/tabelas.md`), medidos em Windows 11, Go 1.27.0, **sem** `-race`, 5 repetições por configuração (célula = mediana [mín–máx]). Trechos `[CONFIRMAR]` são interpretações que o grupo deve validar antes de entregar. Alvo: 3–5 páginas.

## 1. Primitivos de sincronização implementados

Nenhum pacote `sync`, `sync/atomic` ou `context` é usado (verificado por `scripts/check.ps1`). Tudo é feito com channels.

| Primitivo | Implementação | Invariante |
|---|---|---|
| `Mutex` | `chan struct{}` com capacidade 1 | channel cheio = travado, vazio = livre. `Lock` envia (bloqueia se cheio); `Unlock` recebe com `select/default` e dá `panic` se estava vazio; `LockTimeout` faz `select` com `time.After`. |
| `Semaforo` | `chan struct{}` com capacidade `max`, cada elemento é uma permissão | `0 ≤ len(ch) ≤ max`. `Acquire` recebe; `Release` envia com `select/default` e dá `panic` se excederia `max`; `AcquireTimeout` e `Valor()=len(ch)`. |
| `Latch` (substitui `WaitGroup`) | `chan struct{}` com buffer `n` | `Done` envia sem bloquear (buffer `n`); `Wait` recebe exatamente `n` vezes, logo só retorna quando todos sinalizaram. |

Testes (`go test -race`): 8 goroutines × 100 mil incrementos protegidos pelo `Mutex` dão resultado exato; `Unlock` sem `Lock` e `Release` acima de `max` dão `panic`; o semáforo nunca admite mais de `max` simultâneos; `Wait` só retorna após `n` `Done`.

## 2. Problema 1 — Jantar dos Filósofos

### 2.1 Solução e decisões de projeto

Cada garfo é um `Mutex`. O filósofo `i` usa o garfo `i` (esquerdo) e `(i+1)%N` (direito). O laço do filósofo é **único** (`main.go`): executa exatamente **R ciclos** (pensar → fome → pegar os garfos → comer → soltar) e a variação está na interface `Estrategia{Pegar, Soltar}`:

- **base:** pega o esquerdo, dorme `-pausa` (50 ms), pega o direito.
- **hierarquia:** pega primeiro o garfo de menor índice.
- **limite:** `Semaforo` com N−1 permissões ("lugares à mesa") antes de pegar os garfos.
- **garçom:** uma goroutine é a única dona do estado dos garfos. Os filósofos pedem por um channel (cada pedido traz o channel de resposta do filósofo) e avisam a liberação por outro. O garçom mantém uma **fila FIFO estrita**: só atende a cabeça da fila, e só quando os dois garfos estão livres. Isso impede que alguém "fure a fila" (sem starvation), ao custo de paralelismo.

**Monitor/watchdog sem `sync`:** os filósofos enviam eventos (`pensando`, `fome`, `pegou-esquerdo`, `comendo`, `terminou`) para a **main**, única dona do estado. A main faz `select` entre receber um evento e `time.After(timeout)`; se o tempo estoura com filósofos ativos, imprime `DEADLOCK DETECTADO` com o estado de cada um e termina com código 0 (resultado esperado da `base`). As métricas ficam em uma struct local de cada filósofo e são entregues à main no evento `terminou` (sem estado compartilhado).

### 2.2 Deadlock e starvation (condições de Coffman)

| Estratégia | Exclusão mútua | Posse e espera | Não-preempção | Espera circular | Resultado |
|---|---|---|---|---|---|
| base | vale | vale | vale | **vale** | deadlock |
| hierarquia | vale | vale | vale | **quebrada** (garfos numerados) | livre de deadlock |
| limite | vale | vale | vale | **quebrada** (≤ N−1 sentados ⇒ algum consegue os 2 garfos) | livre de deadlock |
| garçom | vale | **quebrada** (recebe os 2 juntos ou nenhum) | vale | não se forma | livre de deadlock |

**Evidência da `base`** (`relatorio/dados/base-deadlock.txt`, N=5, R=100): o watchdog disparou após 2 s sem eventos; os 5 filósofos seguravam o garfo esquerdo esperando o direito, com **0 refeições concluídas** cada.

**Starvation:** apenas o garçom o impede por construção (FIFO estrito). Em `hierarquia` e `limite` não há garantia formal de ordem [CONFIRMAR: o grupo deve decidir se argumenta com a fila interna dos channels do Go]. Nos experimentos nenhum filósofo ficou sem comer: todos completaram as R refeições em todas as execuções corretas. Como R é fixo por filósofo, **a fairness é medida pelo tempo de espera**, não pelo número de refeições.

### 2.3 Fairness e custo (dados reais)

R=100, pensar=comer=10 ms, 5 repetições.

| estratégia | N | duração (ms) | espera média (ms) | espera máx (ms) | desvio das médias (ms) | Jain |
|---|---|---|---|---|---|---|
| hierarquia | 5 | 2924 [2851–2942] | 6.71 [6.52–6.96] | 38.62 [29.11–49.14] | 1.182 [0.912–1.514] | 0.9682 [0.9527–0.9831] |
| limite | 5 | 3196 [2842–4488] | 10.85 [7.25–23.38] | 43.69 [26.57–50.70] | 0.446 [0.242–0.484] | 0.9980 [0.9959–0.9999] |
| garçom | 5 | 3434 [3226–4663] | 12.46 [10.73–24.57] | 44.14 [27.59–51.34] | 0.439 [0.291–0.470] | 0.9993 [0.9981–0.9998] |
| hierarquia | 10 | 2942 [2941–3036] | 6.79 [6.55–7.05] | 38.75 [36.45–40.21] | 1.346 [1.062–1.519] | 0.9629 [0.9541–0.9757] |
| limite | 10 | 3844 [3517–4394] | 16.42 [13.53–22.17] | 80.04 [52.65–85.88] | 0.504 [0.436–0.567] | 0.9991 [0.9985–0.9994] |
| garçom | 10 | 4298 [3867–4715] | 20.12 [16.98–23.43] | 49.65 [47.01–81.03] | 0.447 [0.300–0.614] | 0.9996 [0.9990–0.9998] |

Leitura honesta dos números:

- A **hierarquia é a mais rápida e a de menor espera média, mas a menos justa** (Jain ≈ 0,96–0,97, desvio das médias 2–3× maior). Na soma da espera por filósofo (N=5, média das 5 execuções, ms), os filósofos 0 e 4 esperaram mais (813 e 774) que o 3 (516). Os dois competem pelo garfo 0, que ambos tentam pegar primeiro [CONFIRMAR: explicação da assimetria].
- **Limite e garçom são praticamente igualitários** (Jain ≥ 0,998), mas custam tempo: com N=10, a duração total sobe de 2942 ms (hierarquia) para 3844 ms (limite, +31%) e 4298 ms (garçom, +46%), e a espera média de 6,79 para 16,42 e 20,12 ms.
- O garçom serializa as decisões numa só goroutine e atende só a cabeça da fila, o que reduz o paralelismo; isso é coerente com seu maior tempo [CONFIRMAR].
- As faixas [mín–máx] de `limite` e `garçom` em N=5 são largas (duração até 4488 e 4663 ms); com 5 repetições, diferenças pequenas entre elas não são conclusivas.

### 2.4 Encerramento ordenado

Cada filósofo sai do laço após R ciclos, solta seus garfos, envia `terminou` e chama `Latch.Done`. A main aguarda o `Latch`; no garçom, **a main fecha o channel de pedidos** e espera a goroutine do garçom sair (channel `fim`). Ao final imprime `runtime.NumGoroutine() = 1`. Exceção: na `base` o deadlock é detectado pelo watchdog e o processo termina com `os.Exit(0)`.

## 3. Problema 2 — Produtor/Consumidor

### 3.1 Solução e decisões de projeto

Produtores e consumidores são idênticos nas duas versões; ambos usam a interface `Buffer{Put, Get(timeout), Len, Encerrar}`.

- **channel:** o buffer é `make(chan Item, K)`. `Get` com timeout usa `select` com `time.After`. Encerrar = `close(ch)` pela main.
- **sem:** anel `[K]Item` com `in`/`out`, `notFull` (começa em K) e `notEmpty` (começa em 0). `Put`: `notFull.Acquire → muIn.Lock → escreve → muIn.Unlock → notEmpty.Release`; `Get` é o espelho com `muOut`. Os semáforos garantem vaga/item; os **dois mutexes separados** impedem dois produtores no mesmo slot e dois consumidores no mesmo item, e deixam produtor e consumidor trabalharem em paralelo. `Encerrar(c)` coloca `c` *poison pills* (o anel é FIFO, então todos os itens reais saem antes).
- **Timeout:** só o consumidor 0 usa `Get(timeout)`; em `Timeout` conta um "ocioso", imprime uma linha e volta ao laço. No `AcquireTimeout` nada é consumido ao estourar. Demonstração: `-prod-delay=50ms -timeout=20ms` (comando no README); nos experimentos do benchmark (`pc.csv`) a coluna `timeouts` é 0 em todas as execuções, pois o timeout padrão (200 ms) nunca estourou.
- **Verificação final:** total consumido = total produzido; nenhum `(ProdID, Seq)` repetido ou fora do esperado (mapa na main); `NumGoroutine == 1`. Falha ⇒ `os.Exit(1)`.
- **Métricas:** throughput, itens por consumidor, ocupação média (amostrador a cada 1 ms, parado por channel `quit` antes do encerramento) e timeouts.

### 3.2 Efeito de K e comparação channel × semáforos

P=4, C=4. Cenário A: produtores mais rápidos (prod 1 ms, cons 3 ms, 300 itens/prod). Cenário B: consumidores mais rápidos (prod 3 ms, cons 1 ms, 300 itens/prod). Cenário C (extra): sem atraso, 50 000 itens/prod, para isolar o custo da sincronização. Throughput em itens/s:

| cenário | K | channel | sem |
|---|---|---|---|
| A | 1 | 1013 [812–1125] | 1151 [1131–1170] |
| A | 10 | 979 [782–1052] | 1101 [1018–1174] |
| A | 100 | 1040 [905–1061] | 1132 [1110–1162] |
| B | 1 | 1082 [978–1152] | 986 [946–1024] |
| B | 10 | 1116 [1102–1163] | 1014 [995–1097] |
| B | 100 | 1089 [967–1162] | 1111 [1072–1135] |
| C | 1 | 1110394 [999139–1115654] | 707916 [596604–741088] |
| C | 10 | 1878504 [1376471–2406255] | 704655 [655139–710047] |
| C | 100 | 3113548 [1937050–3573158] | 756470 [729487–994225] |

- **Ocupação média:** no cenário A o buffer fica praticamente cheio (≈ 0,98 para K=1; 9,9 para K=10; ≈ 96 para K=100 em ambas as versões); no B fica praticamente vazio (≈ 0 em todos os K). Consistente com quem é o lado lento.
- **Cenários A e B:** o throughput fica em ~1000–1150 itens/s em todas as configurações, ordem de grandeza do limite imposto pelo lado lento (4 goroutines / 3 ms ≈ 1333 itens/s) [CONFIRMAR]. **K quase não altera** o resultado e não há vencedor consistente: `sem` tem a mediana maior no A, `channel` no B (K=1 e 10), com faixas que em parte se sobrepõem. Não afirmamos diferença; o tempo é dominado pelo `Sleep` (e pela resolução de timers do Windows [CONFIRMAR]).
- **Cenário C (só sincronização):** o `channel` escala com K (1,11 M → 1,88 M → 3,11 M itens/s) e a versão `sem` fica praticamente plana (708 k → 705 k → 756 k). Com K=100 o `channel` faz ≈ 4,1× o throughput de `sem`. Explicação plausível [CONFIRMAR]: cada `Put`/`Get` da versão `sem` custa 4 operações em channel (Acquire, Lock, Unlock, Release) contra 1 no `channel`, e `muIn`/`muOut` serializam os produtores entre si e os consumidores entre si, o que um K maior não alivia.
- **Limitação:** a duração de execuções muito curtas é afetada pela resolução do relógio do Windows (um teste inicial com 1000 itens/prod mediu duração 0); por isso o cenário C usa 50 000 itens/prod.

### 3.3 Deadlock e starvation

Não há ciclo de espera: produtor só espera por vaga e consumidor só por item, e cada `Put`/`Get` libera o que adquiriu antes de terminar (o mutex é solto antes do `Release` do semáforo). A espera só termina quando há progresso do outro lado. Starvation formal de um produtor/consumidor específico não é impedida pelo algoritmo [CONFIRMAR]. Nos dados (`pc.csv`, coluna `itens_por_consumidor`): nos cenários A e B todos os consumidores processaram pelo menos 289 (`channel`) e 292 (`sem`) dos 300 itens que lhes cabiam em média, ou seja, ninguém ficou parado. No cenário C (sem atraso, 200 000 itens), a distribuição é mais desigual, e mais no `channel` (K=100, seed 1: `15533; 67968; 65431; 51068`) do que no `sem` (K=100, seed 1: `42763; 47483; 49494; 60260`); o consumidor 0, que usa `Get` com timeout, é o que menos recebe em todos os pares mostrados [CONFIRMAR: causa].

### 3.4 Encerramento ordenado

Produtores chamam `Latch.Done` ao terminar seus itens. A main espera todos, para o amostrador (`quit`) e só então chama `Encerrar(c)` — **a main, nunca um produtor**: no `channel` fecha o channel (consumidores drenam o que restou e recebem `Closed`); no `sem` insere `c` pills (cada consumidor consome uma e sai). A main espera o `Latch` dos consumidores, recebe de cada um seu slice de itens (via channel com buffer) e faz a verificação final.

## 4. Uso de IA

Ver `AI-USO.md`. Resumo: o Claude Code (Sonnet 5.5) projetou e gerou o código, os testes, os scripts e os experimentos a partir de um roteiro escrito pelo grupo; corrigiu problemas de ambiente (instalação de Go/gcc, SSH, PowerShell 5.1) e de medição (duração 0 no Windows). O grupo revisou e [CONFIRMAR] se responsabiliza por cada afirmação.

## 5. Referências

- Dijkstra, E. W. "Hierarchical ordering of sequential processes", 1971 (jantar dos filósofos, hierarquia de recursos).
- Coffman, E. G.; Elphick, M.; Shoshani, A. "System Deadlocks", ACM Computing Surveys, 1971 (condições de Coffman).
- Jain, R.; Chiu, D.; Hawe, W. "A Quantitative Measure of Fairness...", DEC-TR-301, 1984 (índice de Jain).
- Tanenbaum, A. S.; Bos, H. *Modern Operating Systems* (produtor/consumidor com semáforos, jantar dos filósofos).
- Documentação oficial de Go: *The Go Memory Model*; pacotes `time`, `flag`, `runtime`.
- [CONFIRMAR] material da disciplina/enunciado do T1.
