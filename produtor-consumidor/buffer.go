package main

import "time"

// Item é o que os produtores colocam no buffer. (ProdID, Seq) identifica cada
// item de forma única, o que permite conferir no final se nada se perdeu
// nem se duplicou. Pill é a "poison pill" usada na versão com semáforos para
// avisar um consumidor de que acabou.
type Item struct {
	ProdID int
	Seq    int
	Pill   bool
}

type Status int

const (
	OK      Status = iota // item entregue
	Timeout               // o tempo acabou e nada foi consumido
	Closed                // não há mais itens e nunca haverá: consumidor deve sair
)

// Buffer é a interface comum: produtores e consumidores são idênticos nas
// duas versões (channel e semáforos), só a implementação muda.
type Buffer interface {
	Put(it Item)
	Get(timeout time.Duration) (Item, Status) // timeout 0 = bloqueia sem limite
	Len() int                                 // ocupação atual (para a amostragem)
	Encerrar(nConsumidores int)               // chamado pela main quando os produtores terminaram
}
