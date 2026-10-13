package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Resultado é o que cada consumidor devolve à main ao terminar (via channel).
type Resultado struct {
	ID      int
	Itens   []Item
	Ociosos int       // vezes em que o Get estourou o timeout (só o consumidor 0)
	Ultimo  time.Time // instante do último item consumido
}

// Resumo reúne as métricas de uma execução.
type Resumo struct {
	Total      int
	Duracao    time.Duration
	Throughput float64 // itens/s
	PorConsumo []int
	OcupMedia  float64
	Timeouts   int
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func imprimirResumo(r Resumo) {
	fmt.Printf("itens consumidos: %d em %.2f ms\n", r.Total, ms(r.Duracao))
	fmt.Printf("throughput: %.0f itens/s\n", r.Throughput)
	fmt.Printf("itens por consumidor: %v\n", r.PorConsumo)
	fmt.Printf("ocupação média do buffer: %.2f\n", r.OcupMedia)
	fmt.Printf("timeouts do consumidor 0 (ocioso): %d\n", r.Timeouts)
}

// anexarCSV acrescenta uma linha por execução (cria o cabeçalho se o arquivo é novo).
func anexarCSV(caminho string, cfg Config, r Resumo) error {
	f, err := os.OpenFile(caminho, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if info, _ := f.Stat(); info.Size() == 0 {
		fmt.Fprintln(f, "versao,p,c,k,itens,prod_delay_ms,cons_delay_ms,seed,total,duracao_ms,throughput,ocupacao_media,timeouts,itens_por_consumidor")
	}
	porC := make([]string, len(r.PorConsumo))
	for i, n := range r.PorConsumo {
		porC[i] = fmt.Sprint(n)
	}
	fmt.Fprintf(f, "%s,%d,%d,%d,%d,%g,%g,%d,%d,%.4f,%.2f,%.4f,%d,%s\n",
		cfg.Versao, cfg.P, cfg.C, cfg.K, cfg.Itens, ms(cfg.ProdDelay), ms(cfg.ConsDelay), cfg.Seed,
		r.Total, ms(r.Duracao), r.Throughput, r.OcupMedia, r.Timeouts, strings.Join(porC, ";"))
	return nil
}
