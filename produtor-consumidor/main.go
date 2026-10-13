// Produtor/Consumidor com buffer limitado em duas versões:
// channel nativo e semáforos próprios. Sem pacote sync: só channels.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"
)

// Config reúne os parâmetros da execução (vêm das flags).
type Config struct {
	P, C, K, Itens int
	Versao         string
	Timeout        time.Duration
	ProdDelay      time.Duration
	ConsDelay      time.Duration
	Seed           int64
	CSV            string
}

func main() {
	var cfg Config
	flag.IntVar(&cfg.P, "p", 4, "número de produtores")
	flag.IntVar(&cfg.C, "c", 4, "número de consumidores")
	flag.IntVar(&cfg.K, "k", 10, "capacidade do buffer")
	flag.IntVar(&cfg.Itens, "itens", 1000, "itens por produtor")
	flag.StringVar(&cfg.Versao, "versao", "channel", "channel|sem")
	flag.DurationVar(&cfg.Timeout, "timeout", 200*time.Millisecond, "timeout do consumidor 0")
	flag.DurationVar(&cfg.ProdDelay, "prod-delay", 0, "tempo médio simulado por item no produtor (jitter ±50%)")
	flag.DurationVar(&cfg.ConsDelay, "cons-delay", 0, "tempo médio simulado por item no consumidor (jitter ±50%)")
	flag.Int64Var(&cfg.Seed, "seed", 1, "semente do gerador aleatório")
	flag.StringVar(&cfg.CSV, "csv", "", "arquivo CSV para anexar resultados (opcional)")
	flag.Parse()

	if cfg.P < 1 || cfg.C < 1 || cfg.K < 1 || cfg.Itens < 0 {
		fmt.Println("uso inválido: precisa de -p >= 1, -c >= 1, -k >= 1 e -itens >= 0")
		os.Exit(2)
	}
	var buf Buffer
	switch cfg.Versao {
	case "channel":
		buf = NovoBufferChan(cfg.K)
	case "sem":
		buf = NovoBufferSem(cfg.K)
	default:
		fmt.Println("versão desconhecida:", cfg.Versao)
		os.Exit(2)
	}

	prods := NovoLatch(cfg.P)
	cons := NovoLatch(cfg.C)
	resultados := make(chan Resultado, cfg.C) // com buffer: o consumidor nunca bloqueia ao entregar

	// Amostrador da ocupação do buffer (1 ms); `quit` o para e ele devolve a média.
	quit := make(chan struct{})
	media := make(chan float64, 1)
	go amostrar(buf, quit, media)

	inicio := time.Now()
	for id := 0; id < cfg.C; id++ {
		go consumidor(id, cfg, buf, cons, resultados)
	}
	for id := 0; id < cfg.P; id++ {
		go produtor(id, cfg, buf, prods)
	}

	// ---- Encerramento ordenado.
	prods.Wait() // 1. todos os produtores terminaram de produzir
	close(quit)  // 2. para o amostrador (antes de encerrar)
	ocupMedia := <-media
	buf.Encerrar(cfg.C) // 3. a MAIN avisa o fim (close ou poison pills)
	cons.Wait()         // 4. consumidores drenam o resto e saem

	// ---- Coleta e verificação.
	rs := make([]Resultado, cfg.C)
	for i := 0; i < cfg.C; i++ {
		r := <-resultados // já estão no buffer do channel
		rs[r.ID] = r
	}
	resumo, ok := verificar(cfg, rs, inicio, ocupMedia)
	fmt.Printf("versão=%s p=%d c=%d k=%d itens/produtor=%d\n", cfg.Versao, cfg.P, cfg.C, cfg.K, cfg.Itens)
	imprimirResumo(resumo)

	// Os consumidores acabaram de dar Done; esperamos eles saírem de fato.
	for i := 0; i < 200 && runtime.NumGoroutine() > 1; i++ {
		time.Sleep(time.Millisecond)
	}
	if runtime.NumGoroutine() != 1 {
		fmt.Printf("goroutines restantes: %d (esperado 1)\n", runtime.NumGoroutine())
		ok = false
	}
	if cfg.CSV != "" {
		if err := anexarCSV(cfg.CSV, cfg, resumo); err != nil {
			fmt.Println("erro ao gravar CSV:", err)
			os.Exit(1)
		}
	}
	if !ok {
		fmt.Println("VERIFICAÇÃO FINAL: FALHOU")
		os.Exit(1)
	}
	fmt.Println("VERIFICAÇÃO FINAL: OK (nada perdido, nada repetido, NumGoroutine == 1)")
}

// simular espera d com jitter de ±50% (representa o trabalho de produzir/consumir).
func simular(rng *rand.Rand, d time.Duration) {
	if d > 0 {
		time.Sleep(time.Duration(float64(d) * (0.5 + rng.Float64())))
	}
}

func produtor(id int, cfg Config, buf Buffer, l *Latch) {
	rng := rand.New(rand.NewSource(cfg.Seed + int64(id)))
	for seq := 0; seq < cfg.Itens; seq++ {
		simular(rng, cfg.ProdDelay)
		buf.Put(Item{ProdID: id, Seq: seq})
	}
	l.Done()
}

// consumidor lê até receber Closed. Só o consumidor 0 usa timeout; quando
// estoura, executa o comportamento alternativo (conta e registra "ocioso") e continua.
func consumidor(id int, cfg Config, buf Buffer, l *Latch, saida chan<- Resultado) {
	rng := rand.New(rand.NewSource(cfg.Seed + 1000 + int64(id)))
	timeout := time.Duration(0) // 0 = bloqueia sem limite
	if id == 0 {
		timeout = cfg.Timeout
	}
	r := Resultado{ID: id}
	for {
		it, st := buf.Get(timeout)
		if st == Closed {
			break
		}
		if st == Timeout {
			r.Ociosos++
			if r.Ociosos <= 3 { // só as primeiras linhas, para não poluir a saída
				fmt.Printf("consumidor 0: ocioso (timeout de %v sem itens)\n", timeout)
			}
			continue
		}
		simular(rng, cfg.ConsDelay)
		r.Itens = append(r.Itens, it)
		r.Ultimo = time.Now()
	}
	saida <- r
	l.Done()
}

// amostrar lê a ocupação do buffer a cada 1 ms até receber quit; devolve a média.
func amostrar(buf Buffer, quit <-chan struct{}, media chan<- float64) {
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	soma, n := 0, 0
	for {
		select {
		case <-tick.C:
			soma += buf.Len()
			n++
		case <-quit:
			m := 0.0
			if n > 0 {
				m = float64(soma) / float64(n)
			}
			media <- m
			return
		}
	}
}

// verificar confere: total consumido == produzido, nenhum (ProdID, Seq)
// repetido nem fora do esperado. Também calcula as métricas.
func verificar(cfg Config, rs []Resultado, inicio time.Time, ocup float64) (Resumo, bool) {
	ok := true
	vistos := make(map[[2]int]bool)
	r := Resumo{OcupMedia: ocup}
	var ultimo time.Time
	for _, res := range rs {
		r.PorConsumo = append(r.PorConsumo, len(res.Itens))
		r.Total += len(res.Itens)
		if res.ID == 0 {
			r.Timeouts = res.Ociosos
		}
		if res.Ultimo.After(ultimo) {
			ultimo = res.Ultimo
		}
		for _, it := range res.Itens {
			chave := [2]int{it.ProdID, it.Seq}
			if vistos[chave] {
				fmt.Printf("ERRO: item repetido %v\n", chave)
				ok = false
			}
			if it.ProdID < 0 || it.ProdID >= cfg.P || it.Seq < 0 || it.Seq >= cfg.Itens {
				fmt.Printf("ERRO: item inesperado %v\n", chave)
				ok = false
			}
			vistos[chave] = true
		}
	}
	esperado := cfg.P * cfg.Itens
	if r.Total != esperado {
		fmt.Printf("ERRO: consumidos %d, produzidos %d\n", r.Total, esperado)
		ok = false
	}
	if !ultimo.IsZero() {
		r.Duracao = ultimo.Sub(inicio)
		r.Throughput = float64(r.Total) / r.Duracao.Seconds()
	}
	return r, ok
}
