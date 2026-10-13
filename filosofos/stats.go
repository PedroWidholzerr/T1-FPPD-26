package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"
)

// Metricas de um filósofo. Cada filósofo mexe só na sua e a entrega à main
// no fim (via channel), então não há estado compartilhado.
type Metricas struct {
	Refeicoes int
	Esperas   []time.Duration // fome -> dois garfos, uma por refeição
	Total     time.Duration
	Max       time.Duration
	Termino   time.Duration // instante de término desde o início da simulação
}

func (m *Metricas) Registrar(espera time.Duration) {
	m.Refeicoes++
	m.Esperas = append(m.Esperas, espera)
	m.Total += espera
	if espera > m.Max {
		m.Max = espera
	}
}

func (m Metricas) Media() time.Duration {
	if m.Refeicoes == 0 {
		return 0
	}
	return m.Total / time.Duration(m.Refeicoes)
}

// P95: valor abaixo do qual ficam 95% das esperas.
func (m Metricas) P95() time.Duration {
	if len(m.Esperas) == 0 {
		return 0
	}
	o := append([]time.Duration(nil), m.Esperas...)
	sort.Slice(o, func(i, j int) bool { return o[i] < o[j] })
	idx := int(math.Ceil(0.95*float64(len(o)))) - 1
	return o[idx]
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

func desvioPadrao(xs []float64) float64 {
	soma := 0.0
	for _, x := range xs {
		soma += x
	}
	media := soma / float64(len(xs))
	var q float64
	for _, x := range xs {
		q += (x - media) * (x - media)
	}
	return math.Sqrt(q / float64(len(xs)))
}

// jain: índice de Jain = (Σx)² / (N·Σx²). Vale 1 se todos esperam igual
// e tende a 1/N quando um só concentra a espera.
func jain(xs []float64) float64 {
	var s, s2 float64
	for _, x := range xs {
		s += x
		s2 += x * x
	}
	if s2 == 0 {
		return 1
	}
	return s * s / (float64(len(xs)) * s2)
}

// imprimirResumo mostra a tabela por filósofo e os agregados.
func imprimirResumo(mets []Metricas, ordem []int, duracao time.Duration) {
	fmt.Println("\nfil  refeições  espera_média  espera_máx    p95    espera_total  término (ms)")
	medias := make([]float64, len(mets))
	totais := make([]float64, len(mets))
	for i, m := range mets {
		medias[i], totais[i] = ms(m.Media()), ms(m.Total)
		fmt.Printf("%3d  %9d  %12.2f  %10.2f  %6.2f  %12.2f  %12.2f\n",
			i, m.Refeicoes, ms(m.Media()), ms(m.Max), ms(m.P95()), ms(m.Total), ms(m.Termino))
	}
	fmt.Printf("\nduração total: %.2f ms\n", ms(duracao))
	fmt.Printf("desvio-padrão das esperas médias: %.3f ms\n", desvioPadrao(medias))
	fmt.Printf("índice de Jain (espera total): %.4f\n", jain(totais))
	fmt.Printf("ordem de término: %v\n", ordem)
}

// anexarCSV acrescenta uma linha por filósofo (cria o cabeçalho se o arquivo é novo).
func anexarCSV(caminho string, cfg Config, mets []Metricas, duracao time.Duration) error {
	f, err := os.OpenFile(caminho, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if info, _ := f.Stat(); info.Size() == 0 {
		fmt.Fprintln(f, "estrategia,n,r,pensar_ms,comer_ms,seed,filosofo,refeicoes,espera_media_ms,espera_max_ms,p95_ms,espera_total_ms,termino_ms,duracao_ms,jain,desvio_medias_ms")
	}
	totais := make([]float64, len(mets))
	medias := make([]float64, len(mets))
	for i, m := range mets {
		totais[i], medias[i] = ms(m.Total), ms(m.Media())
	}
	j, dp := jain(totais), desvioPadrao(medias)
	for i, m := range mets {
		fmt.Fprintf(f, "%s,%d,%d,%g,%g,%d,%d,%d,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f,%.4f\n",
			cfg.Estrategia, cfg.N, cfg.R, ms(cfg.Pensar), ms(cfg.Comer), cfg.Seed,
			i, m.Refeicoes, ms(m.Media()), ms(m.Max), ms(m.P95()), ms(m.Total), ms(m.Termino),
			ms(duracao), j, dp)
	}
	return nil
}
