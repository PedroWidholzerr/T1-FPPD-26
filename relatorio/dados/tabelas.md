# Tabelas dos experimentos

Gerado por `scripts/agrega.ps1` a partir dos CSVs desta pasta. Cada célula é **mediana [mín–máx]** sobre as repetições.

## Problema 1 — Jantar dos Filósofos (R=100, pensar=comer=10ms)

Cada execução vira um ponto: espera média = média das esperas médias dos filósofos; espera máx = maior espera de qualquer filósofo.

| estratégia | N | reps | duração (ms) | espera média (ms) | espera máx (ms) | desvio das médias (ms) | Jain |
|---|---|---|---|---|---|---|---|
| garcom | 5 | 5 | 3434 [3226–4663] | 12.46 [10.73–24.57] | 44.14 [27.59–51.34] | 0.439 [0.291–0.470] | 0.9993 [0.9981–0.9998] |
| hierarquia | 5 | 5 | 2924 [2851–2942] | 6.71 [6.52–6.96] | 38.62 [29.11–49.14] | 1.182 [0.912–1.514] | 0.9682 [0.9527–0.9831] |
| limite | 5 | 5 | 3196 [2842–4488] | 10.85 [7.25–23.38] | 43.69 [26.57–50.70] | 0.446 [0.242–0.484] | 0.9980 [0.9959–0.9999] |
| garcom | 10 | 5 | 4298 [3867–4715] | 20.12 [16.98–23.43] | 49.65 [47.01–81.03] | 0.447 [0.300–0.614] | 0.9996 [0.9990–0.9998] |
| hierarquia | 10 | 5 | 2942 [2941–3036] | 6.79 [6.55–7.05] | 38.75 [36.45–40.21] | 1.346 [1.062–1.519] | 0.9629 [0.9541–0.9757] |
| limite | 10 | 5 | 3844 [3517–4394] | 16.42 [13.53–22.17] | 80.04 [52.65–85.88] | 0.504 [0.436–0.567] | 0.9991 [0.9985–0.9994] |

### Versão base (deadlock)

```

DEADLOCK DETECTADO (nenhum evento em 2s, estratégia=base)
  filósofo 0: pegou-esquerdo segura garfo 0 e espera o garfo 1 (refeições concluídas: 0)
  filósofo 1: pegou-esquerdo segura garfo 1 e espera o garfo 2 (refeições concluídas: 0)
  filósofo 2: pegou-esquerdo segura garfo 2 e espera o garfo 3 (refeições concluídas: 0)
  filósofo 3: pegou-esquerdo segura garfo 3 e espera o garfo 4 (refeições concluídas: 0)
  filósofo 4: pegou-esquerdo segura garfo 4 e espera o garfo 0 (refeições concluídas: 0)
```

## Problema 2 — Produtor/Consumidor (P=4, C=4)

### Cenário A (produtores rápidos: prod 1ms, cons 3ms) — itens por produtor: 300

| versão | K | reps | throughput (itens/s) | duração (ms) | ocupação média | timeouts do cons. 0 |
|---|---|---|---|---|---|---|
| channel | 1 | 5 | 1013 [812–1125] | 1184.3 [1066.8–1478.0] | 0.98 [0.95–0.98] | 0 [0–0] |
| sem | 1 | 5 | 1151 [1131–1170] | 1042.6 [1025.5–1060.9] | 0.97 [0.96–0.99] | 0 [0–0] |
| channel | 10 | 5 | 979 [782–1052] | 1225.4 [1140.2–1534.3] | 9.93 [9.87–9.93] | 0 [0–0] |
| sem | 10 | 5 | 1101 [1018–1174] | 1090.3 [1022.0–1178.9] | 9.94 [9.84–9.96] | 0 [0–0] |
| channel | 100 | 5 | 1040 [905–1061] | 1153.8 [1130.8–1325.4] | 95.62 [95.32–96.16] | 0 [0–0] |
| sem | 100 | 5 | 1132 [1110–1162] | 1060.4 [1033.1–1081.2] | 96.30 [95.63–96.54] | 0 [0–0] |

### Cenário B (consumidores rápidos: prod 3ms, cons 1ms) — itens por produtor: 300

| versão | K | reps | throughput (itens/s) | duração (ms) | ocupação média | timeouts do cons. 0 |
|---|---|---|---|---|---|---|
| channel | 1 | 5 | 1082 [978–1152] | 1108.7 [1041.5–1226.9] | 0.00 [0.00–0.01] | 0 [0–0] |
| sem | 1 | 5 | 986 [946–1024] | 1217.3 [1171.4–1267.9] | 0.01 [0.00–0.02] | 0 [0–0] |
| channel | 10 | 5 | 1116 [1102–1163] | 1074.8 [1031.6–1089.3] | 0.00 [0.00–0.00] | 0 [0–0] |
| sem | 10 | 5 | 1014 [995–1097] | 1183.2 [1094.3–1205.9] | 0.01 [0.00–0.02] | 0 [0–0] |
| channel | 100 | 5 | 1089 [967–1162] | 1101.6 [1032.4–1240.9] | 0.00 [0.00–0.00] | 0 [0–0] |
| sem | 100 | 5 | 1111 [1072–1135] | 1080.1 [1057.7–1119.9] | 0.00 [0.00–0.01] | 0 [0–0] |

### Cenário C (sem atraso) — itens por produtor: 50000

| versão | K | reps | throughput (itens/s) | duração (ms) | ocupação média | timeouts do cons. 0 |
|---|---|---|---|---|---|---|
| channel | 1 | 5 | 1110394 [999139–1115654] | 180.1 [179.3–200.2] | 0.53 [0.49–0.56] | 0 [0–0] |
| sem | 1 | 5 | 707916 [596604–741088] | 282.5 [269.9–335.2] | 0.00 [0.00–0.00] | 0 [0–0] |
| channel | 10 | 5 | 1878504 [1376471–2406255] | 106.5 [83.1–145.3] | 5.22 [4.89–5.36] | 0 [0–0] |
| sem | 10 | 5 | 704655 [655139–710047] | 283.8 [281.7–305.3] | 1.47 [0.98–4.62] | 0 [0–0] |
| channel | 100 | 5 | 3113548 [1937050–3573158] | 64.2 [56.0–103.2] | 56.00 [45.23–61.40] | 0 [0–0] |
| sem | 100 | 5 | 756470 [729487–994225] | 264.4 [201.2–274.2] | 64.11 [47.03–90.51] | 0 [0–0] |

