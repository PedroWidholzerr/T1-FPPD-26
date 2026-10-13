# T1 — Programação Concorrente: Jantar dos Filósofos e Produtor/Consumidor em Go

**Disciplina:** FPPD — PUCRS  **Grupo:** Pedro Widholzer e Felipe Flores  **Repositório:** github.com/PedroWidholzerr/T1-FPPD-26

Todos os números vêm de `relatorio/dados/*.csv` (tabelas completas em `relatorio/dados/tabelas.md`), medidos em Windows 11 com Go 1.27.0, **sem** `-race`, com 5 repetições por configuração; cada célula é *mediana [mín–máx]*. A corretude foi validada separadamente com `go run -race` e `go test -race`.

## 1. Primitivos de sincronização implementados

Não usamos `sync`, `sync/atomic` nem `context` (verificado por `scripts/check.ps1`); toda a sincronização é feita com channels.

- **`Mutex`** — `chan struct{}` de capacidade 1. *Invariante:* cheio = travado, vazio = livre. `Lock` envia (bloqueia se cheio); `Unlock` recebe com `select/default` e dá `panic` se o channel estava vazio; `LockTimeout` faz `select` com `time.After`.
- **`Semaforo`** — `chan struct{}` de capacidade `max`; cada elemento é uma permissão. *Invariante:* `0 ≤ len(ch) ≤ max`. `Acquire` recebe; `Release` envia com `select/default` e dá `panic` se excederia `max`; também `AcquireTimeout` e `Valor() = len(ch)`.
- **`Latch`** (no lugar do `WaitGroup`) — `chan struct{}` com buffer `n`. *Invariante:* `Done` nunca bloqueia (buffer `n`) e `Wait` recebe exatamente `n` vezes, logo só retorna quando todos sinalizaram.

Testes (`go test -race`): 8 goroutines × 100 mil incrementos protegidos pelo `Mutex` dão resultado exato; `Unlock` sem `Lock` e `Release` acima de `max` dão `panic`; o semáforo nunca admite mais de `max` simultâneos; `Wait` só retorna após `n` `Done`.

## 2. Problema 1 — Jantar dos Filósofos

### 2.1 Solução e decisões de projeto

Cada garfo é um `Mutex`; o filósofo `i` usa o garfo `i` (esquerdo) e `(i+1)%N` (direito). O laço do filósofo é **único**: executa exatamente **R ciclos** (pensar → fome → pegar → comer → soltar) e só a interface `Estrategia{Pegar, Soltar}` muda:

- **base:** pega o esquerdo, dorme `-pausa` (50 ms), pega o direito.
- **hierarquia:** pega primeiro o garfo de menor índice.
- **limite:** `Semaforo` com N−1 permissões ("lugares à mesa") antes de pegar os garfos.
- **garçom:** uma goroutine é a única dona do estado dos garfos. Os filósofos pedem por um channel (cada pedido traz o channel de resposta do filósofo) e avisam a liberação por outro. O garçom mantém uma **fila FIFO estrita**: só atende a cabeça da fila e só quando os dois garfos estão livres. Ninguém fura a fila (sem starvation), ao custo de paralelismo.

**Monitor/watchdog sem `sync`:** os filósofos enviam eventos (`pensando`, `fome`, `pegou-esquerdo`, `comendo`, `terminou`) à **main**, única dona do estado. A main faz `select` entre receber um evento e `time.After(timeout)`; se o tempo estoura com filósofos ativos, imprime `DEADLOCK DETECTADO` com o estado de cada um e termina com código 0 (resultado esperado da `base`). As métricas ficam numa struct local de cada filósofo e seguem para a main no evento `terminou`, sem estado compartilhado.

### 2.2 Deadlock e starvation (condições de Coffman)

| Estratégia | Exclusão mútua | Posse e espera | Não-preempção | Espera circular | Resultado |
|---|---|---|---|---|---|
| base | vale | vale | vale | **vale** | deadlock |
| hierarquia | vale | vale | vale | **quebrada** (garfos numerados) | sem deadlock |
| limite | vale | vale | vale | **quebrada** (≤ N−1 sentados ⇒ algum pega os 2 garfos) | sem deadlock |
| garçom | vale | **quebrada** (recebe os 2 juntos ou nenhum) | vale | não se forma | sem deadlock |

**Evidência da `base`** (`relatorio/dados/base-deadlock.txt`, N=5, R=100): o watchdog disparou após 2 s sem eventos; os 5 filósofos seguravam o garfo esquerdo esperando o direito, com **0 refeições concluídas** cada.

**Starvation:** só o garçom o impede por construção (FIFO estrito). Em `hierarquia` e `limite`, os filósofos bloqueados num `Lock`/`Acquire` esperam num channel, e o runtime do Go mantém essas filas em ordem de chegada, o que na prática evita starvation; porém a especificação da linguagem não garante essa ordem, então não a tratamos como garantia. Nos experimentos, os 225 registros (um por filósofo por execução) completaram as R=100 refeições. Como R é fixo por filósofo, **a fairness é medida pelo tempo de espera** (fome → dois garfos), não pelo número de refeições.

### 2.3 Fairness e custo

R=100, pensar=comer=10 ms.

| estratégia | N | duração (ms) | espera média (ms) | espera máx (ms) | desvio das médias (ms) | Jain |
|---|---|---|---|---|---|---|
| hierarquia | 5 | 2924 [2851–2942] | 6.71 [6.52–6.96] | 38.62 [29.11–49.14] | 1.182 [0.912–1.514] | 0.9682 [0.9527–0.9831] |
| limite | 5 | 3196 [2842–4488] | 10.85 [7.25–23.38] | 43.69 [26.57–50.70] | 0.446 [0.242–0.484] | 0.9980 [0.9959–0.9999] |
| garçom | 5 | 3434 [3226–4663] | 12.46 [10.73–24.57] | 44.14 [27.59–51.34] | 0.439 [0.291–0.470] | 0.9993 [0.9981–0.9998] |
| hierarquia | 10 | 2942 [2941–3036] | 6.79 [6.55–7.05] | 38.75 [36.45–40.21] | 1.346 [1.062–1.519] | 0.9629 [0.9541–0.9757] |
| limite | 10 | 3844 [3517–4394] | 16.42 [13.53–22.17] | 80.04 [52.65–85.88] | 0.504 [0.436–0.567] | 0.9991 [0.9985–0.9994] |
| garçom | 10 | 4298 [3867–4715] | 20.12 [16.98–23.43] | 49.65 [47.01–81.03] | 0.447 [0.300–0.614] | 0.9996 [0.9990–0.9998] |

- A **hierarquia é a mais rápida e tem a menor espera média, mas é a menos justa**: Jain ≈ 0,96–0,97 e desvio das médias de 2,6 a 3× o das outras duas. Para N=5, a espera total média por filósofo (5 execuções) foi 813 ms (filósofo 0), 666, 592, 516 (filósofo 3) e 774 ms (filósofo 4). A assimetria tem explicação direta no algoritmo: o filósofo 4 (garfos 4 e 0) e o filósofo 0 pegam ambos o garfo 0 como primeira escolha, enquanto cada um dos demais disputa como primeira escolha apenas o garfo do próprio índice. Verificamos esse perfil apenas para N=5.
- **Limite e garçom são praticamente igualitários** (Jain ≥ 0,998), mas custam tempo: com N=10, a duração sobe de 2942 ms (hierarquia) para 3844 ms (limite, +31%) e 4298 ms (garçom, +46%); a espera média vai de 6,79 para 16,42 e 20,12 ms.
- O custo maior do garçom é coerente com seu desenho: as decisões passam por uma única goroutine e só a cabeça da fila é atendida, o que reduz o paralelismo (não medimos isoladamente cada efeito).
- Em N=5 as faixas [mín–máx] de `limite` e `garçom` são largas (duração até 4488 e 4663 ms) e se sobrepõem; com 5 repetições não distinguimos essas duas estratégias entre si em duração.

### 2.4 Encerramento ordenado

Cada filósofo sai do laço após R ciclos, solta os garfos, envia `terminou` e chama `Latch.Done`. A main aguarda o `Latch`; no garçom, **a main fecha o channel de pedidos** e espera a goroutine do garçom sair (channel `fim`). Ao final imprime `runtime.NumGoroutine() = 1`. Exceção: na `base` o watchdog detecta o deadlock e o processo termina com `os.Exit(0)`.

## 3. Problema 2 — Produtor/Consumidor

### 3.1 Solução e decisões de projeto

Produtores e consumidores são idênticos nas duas versões e usam a interface `Buffer{Put, Get(timeout), Len, Encerrar}`.

- **channel:** o buffer é `make(chan Item, K)`; `Get` com timeout usa `select` com `time.After`; `Encerrar` é o `close(ch)`, feito pela main.
- **sem:** anel `[K]Item` com índices `in`/`out`, `notFull` (começa em K) e `notEmpty` (começa em 0). `Put`: `notFull.Acquire → muIn.Lock → escreve → muIn.Unlock → notEmpty.Release`; `Get` é o espelho com `muOut`. Os semáforos garantem que há vaga/item; os **dois mutexes** impedem dois produtores no mesmo slot e dois consumidores no mesmo item, e, por serem separados, permitem que um produtor e um consumidor trabalhem ao mesmo tempo. `Encerrar(c)` insere `c` *poison pills*; como o anel é FIFO, todos os itens reais saem antes.
- **Timeout:** só o consumidor 0 usa `Get(timeout)`; ao estourar, conta um "ocioso", imprime uma linha e volta ao laço (com `AcquireTimeout` nada é consumido). Demonstração: `-prod-delay=50ms -timeout=20ms` (README). Nos benchmarks a coluna `timeouts` é 0 em todas as execuções, pois o timeout padrão (200 ms) nunca estourou.
- **Verificação final:** total consumido = total produzido, nenhum `(ProdID, Seq)` repetido ou inesperado (mapa na main) e `NumGoroutine == 1`; se falhar, `os.Exit(1)`.
- **Métricas:** throughput, itens por consumidor, ocupação média (amostrador a cada 1 ms, parado pelo channel `quit` antes do encerramento) e timeouts.

### 3.2 Efeito de K e comparação channel × semáforos

P=4, C=4. **A:** produtores mais rápidos (prod 1 ms, cons 3 ms, 300 itens/prod). **B:** consumidores mais rápidos (prod 3 ms, cons 1 ms, 300 itens/prod). **C:** sem atraso, 50 000 itens/prod, para isolar o custo da sincronização. Throughput em itens/s:

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

- **Ocupação média:** no cenário A o buffer fica praticamente cheio (0,97–0,98 para K=1; 9,9 para K=10; 95,6–96,3 para K=100, nas duas versões); no B fica praticamente vazio (≈ 0 em todos os K). Isso confirma qual lado é o lento em cada cenário.
- **Cenários A e B (dominados pelo `Sleep`):** o throughput fica entre 979 e 1151 itens/s, cerca de 73% a 86% do teto teórico do lado lento (4 goroutines / 3 ms ≈ 1333 itens/s); atribuímos a diferença ao jitter e ao excesso dos `Sleep` (hipótese, não medida). **K quase não altera o resultado.** Entre as versões, `sem` tem a mediana maior no A (9–14% acima do `channel`) e `channel` a tem maior no B para K=1 e 10 (~10%), ou seja, o sentido se inverte entre cenários; por isso tratamos como ruído de agendamento/timers e não como custo de sincronização.
- **Cenário C (só sincronização):** o `channel` escala com K (1,11 M → 1,88 M → 3,11 M itens/s) e o `sem` fica praticamente plano (708 k → 705 k → 756 k). Com K=100, o `channel` faz ≈ 4,1× o throughput do `sem`. Explicação mais provável (não medida isoladamente): cada `Put`/`Get` do `sem` custa 4 operações em channel (Acquire, Lock, Unlock, Release) contra 1 do `channel`, e `muIn`/`muOut` serializam produtores entre si e consumidores entre si, algo que aumentar K não alivia.
- **Limitação de medição:** em execuções muito curtas a duração é limitada pela resolução do relógio do Windows (um teste inicial com 1000 itens/prod mediu duração 0); por isso o cenário C usa 50 000 itens/prod.

### 3.3 Deadlock e starvation

Não há espera circular: o produtor só espera por vaga, o consumidor só por item, e cada operação libera o mutex antes do `Release` do semáforo, então a espera termina assim que o outro lado progride. O algoritmo não impede formalmente que um produtor/consumidor específico seja preterido. Nos dados (`itens_por_consumidor`): nos cenários A e B todo consumidor processou pelo menos 289 (`channel`) e 292 (`sem`) dos 300 itens da média, ou seja, nenhum ficou parado. No cenário C a distribuição é bem mais desigual no `channel`: o consumidor 0 foi o que menos consumiu em 15 de 15 execuções (média de 23 447 itens contra 58 851 dos demais); no `sem`, em 11 de 15 (47 338 contra 50 887). Causa provável: só o consumidor 0 cria um timer (`time.NewTimer`) a cada `Get`, o que o torna mais lento por item; o efeito é maior no `channel`, onde o resto da operação é mais barato (não testamos isoladamente).

### 3.4 Encerramento ordenado

Produtores chamam `Latch.Done` ao terminar. A main espera todos, para o amostrador (`quit`) e só então chama `Encerrar(c)` — **a main, nunca um produtor**: no `channel` fecha o channel (consumidores drenam o resto e recebem `Closed`); no `sem` insere `c` pills (cada consumidor consome uma e sai). A main espera o `Latch` dos consumidores, recebe de cada um seu slice de itens por um channel com buffer e faz a verificação final.

## 4. Uso de IA

A arquitetura das soluções e o raciocínio de como resolver cada problema (estratégias contra deadlock, monitor/watchdog por channels, primitivos sobre channels, encerramento ordenado, desenho dos experimentos) foram definidos e desenvolvidos pelo grupo, que os descreveu em um roteiro detalhado. A ferramenta de IA (Claude Code, modelo Sonnet 5.5) foi usada apenas para **gerar o código** e os **testes** a partir desse roteiro, além de scripts de apoio (PowerShell) e de ajustes de ambiente e medição (instalação de Go/gcc, SSH, PowerShell 5.1, duração 0 no Windows). Os resultados dos experimentos foram **conferidos pelo grupo**, que executou e revisou o código. Registro completo em `AI-USO.md`.

## 5. Referências

- Dijkstra, E. W. Hierarchical ordering of sequential processes. *Acta Informatica*, 1971.
- Coffman, E. G.; Elphick, M. J.; Shoshani, A. System Deadlocks. *ACM Computing Surveys*, 3(2), 1971.
- Jain, R.; Chiu, D.; Hawe, W. *A Quantitative Measure of Fairness and Discrimination for Resource Allocation in Shared Computer Systems*. DEC-TR-301, 1984.
- Tanenbaum, A. S.; Bos, H. *Modern Operating Systems*. Pearson (produtor/consumidor com semáforos; jantar dos filósofos).
- Documentação de Go: *The Go Memory Model*; pacotes `time`, `flag`, `runtime`.
- Enunciado do T1 — FPPD, PUCRS.
