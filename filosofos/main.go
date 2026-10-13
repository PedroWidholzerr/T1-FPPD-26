// Jantar dos Filósofos com 4 estratégias. Sem pacote sync: só channels.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"
)

// Config reúne os parâmetros da simulação (vêm das flags).
type Config struct {
	N, R       int
	Estrategia string
	Pensar     time.Duration
	Comer      time.Duration
	Pausa      time.Duration
	Timeout    time.Duration
	Seed       int64
	CSV        string
}

type TipoEvento int

const (
	Pensando TipoEvento = iota
	Fome
	PegouEsq // só na estratégia base
	Comendo
	Terminou
)

var nomes = []string{"pensando", "fome", "pegou-esquerdo", "comendo", "terminou"}

// Evento é o que o filósofo conta à main. A main é a ÚNICA dona do estado
// de cada filósofo, por isso não precisamos de mutex nem atomic.
type Evento struct {
	ID    int
	Tipo  TipoEvento
	Stats Metricas // só preenchido em Terminou
}

func main() {
	var cfg Config
	var pensarMs, comerMs, pausaMs, timeoutMs float64
	flag.IntVar(&cfg.N, "n", 5, "número de filósofos (>= 2)")
	flag.IntVar(&cfg.R, "r", 10, "ciclos por filósofo")
	flag.StringVar(&cfg.Estrategia, "estrategia", "hierarquia", "base|hierarquia|limite|garcom")
	flag.Float64Var(&pensarMs, "pensar", 10, "duração média de pensar (ms, jitter ±50%)")
	flag.Float64Var(&comerMs, "comer", 10, "duração média de comer (ms, jitter ±50%)")
	flag.Float64Var(&pausaMs, "pausa", 50, "espera entre o 1º e o 2º garfo na base (ms)")
	flag.Float64Var(&timeoutMs, "timeout", 2000, "watchdog: ms sem eventos para declarar deadlock")
	flag.Int64Var(&cfg.Seed, "seed", 1, "semente do gerador aleatório")
	flag.StringVar(&cfg.CSV, "csv", "", "arquivo CSV para anexar resultados (opcional)")
	flag.Parse()
	cfg.Pensar, cfg.Comer = msParaDur(pensarMs), msParaDur(comerMs)
	cfg.Pausa, cfg.Timeout = msParaDur(pausaMs), msParaDur(timeoutMs)

	if cfg.N < 2 || cfg.R < 0 {
		fmt.Println("uso inválido: precisa de -n >= 2 e -r >= 0")
		os.Exit(2)
	}

	// Buffer no channel de eventos só para a main não atrasar os filósofos.
	eventos := make(chan Evento, cfg.N*4)
	garfos := novosGarfos(cfg.N)
	var est Estrategia
	switch cfg.Estrategia {
	case "base":
		est = &Base{garfos: garfos, pausa: cfg.Pausa, eventos: eventos}
	case "hierarquia":
		est = &Hierarquia{garfos: garfos}
	case "limite":
		est = NovoLimite(garfos)
	case "garcom":
		est = NovoGarcom(cfg.N)
	default:
		fmt.Println("estratégia desconhecida:", cfg.Estrategia)
		os.Exit(2)
	}

	inicio := time.Now()
	latch := NovoLatch(cfg.N)
	for id := 0; id < cfg.N; id++ {
		go filosofo(id, cfg, est, eventos, latch, inicio)
	}

	// ---- Monitor / watchdog: a main recebe eventos ou estoura o timeout.
	estado := make([]Evento, cfg.N) // último evento de cada um
	temEsq := make([]bool, cfg.N)   // base: segura o garfo esquerdo?
	refeicoes := make([]int, cfg.N)
	mets := make([]Metricas, cfg.N)
	var ordem []int
	ativos := cfg.N
	for ativos > 0 {
		select {
		case ev := <-eventos:
			estado[ev.ID] = ev
			switch ev.Tipo {
			case PegouEsq:
				temEsq[ev.ID] = true
			case Comendo:
				refeicoes[ev.ID]++
				temEsq[ev.ID] = false
			case Terminou:
				mets[ev.ID] = ev.Stats
				ordem = append(ordem, ev.ID)
				ativos--
			}
		case <-time.After(cfg.Timeout):
			relatarDeadlock(cfg, estado, temEsq, refeicoes)
			if cfg.Estrategia == "base" {
				os.Exit(0) // resultado esperado da base
			}
			os.Exit(1) // deadlock em estratégia que deveria ser livre dele = bug
		}
	}

	// ---- Encerramento ordenado.
	latch.Wait() // todos os filósofos sinalizaram Done
	duracao := time.Since(inicio)
	if g, ok := est.(*Garcom); ok {
		g.Encerrar() // fecha o channel de pedidos; o garçom sai
	}

	fmt.Printf("estratégia=%s n=%d r=%d seed=%d\n", cfg.Estrategia, cfg.N, cfg.R, cfg.Seed)
	imprimirResumo(mets, ordem, duracao)
	if cfg.CSV != "" {
		if err := anexarCSV(cfg.CSV, cfg, mets, duracao); err != nil {
			fmt.Println("erro ao gravar CSV:", err)
			os.Exit(1)
		}
	}

	// Verificação: só a main deve restar. Os filósofos acabaram de dar Done e
	// podem levar um instante para sair de fato, então esperamos um pouco.
	for i := 0; i < 200 && runtime.NumGoroutine() > 1; i++ {
		time.Sleep(time.Millisecond)
	}
	fmt.Printf("runtime.NumGoroutine() = %d (esperado 1)\n", runtime.NumGoroutine())
	if runtime.NumGoroutine() != 1 {
		os.Exit(1)
	}
}

func msParaDur(v float64) time.Duration { return time.Duration(v * float64(time.Millisecond)) }

// filosofo executa exatamente R ciclos: pensar -> fome -> pegar -> comer -> soltar.
// O laço é o mesmo para qualquer estratégia.
func filosofo(id int, cfg Config, est Estrategia, eventos chan<- Evento, l *Latch, inicio time.Time) {
	rng := rand.New(rand.NewSource(cfg.Seed + int64(id))) // um gerador por goroutine
	var m Metricas
	for i := 0; i < cfg.R; i++ {
		eventos <- Evento{ID: id, Tipo: Pensando}
		dormir(rng, cfg.Pensar)

		eventos <- Evento{ID: id, Tipo: Fome}
		t0 := time.Now()
		est.Pegar(id)
		espera := time.Since(t0) // fome -> dois garfos em mãos
		m.Registrar(espera)

		eventos <- Evento{ID: id, Tipo: Comendo}
		dormir(rng, cfg.Comer)
		est.Soltar(id)
	}
	m.Termino = time.Since(inicio)
	eventos <- Evento{ID: id, Tipo: Terminou, Stats: m}
	l.Done()
}

// dormir espera d com jitter aleatório de ±50%.
func dormir(rng *rand.Rand, d time.Duration) {
	if d > 0 {
		time.Sleep(time.Duration(float64(d) * (0.5 + rng.Float64())))
	}
}

func relatarDeadlock(cfg Config, estado []Evento, temEsq []bool, refeicoes []int) {
	fmt.Printf("\nDEADLOCK DETECTADO (nenhum evento em %v, estratégia=%s)\n", cfg.Timeout, cfg.Estrategia)
	for id, ev := range estado {
		e, d := esquerdo(id), direito(id, cfg.N)
		quem := ""
		switch {
		case ev.Tipo == Comendo:
			quem = fmt.Sprintf("segura garfos %d e %d", e, d)
		case temEsq[id]:
			quem = fmt.Sprintf("segura garfo %d e espera o garfo %d", e, d)
		case ev.Tipo == Fome:
			quem = "com fome, sem garfos"
		}
		fmt.Printf("  filósofo %d: %-14s %s (refeições concluídas: %d)\n", id, nomes[ev.Tipo], quem, refeicoes[id])
	}
}
