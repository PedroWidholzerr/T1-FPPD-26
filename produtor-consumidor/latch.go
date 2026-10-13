package main

// Latch substitui o WaitGroup: espera n eventos "Done".
//
// Invariante: o buffer tem tamanho n, então cada Done envia sem bloquear;
// Wait recebe exatamente n vezes, só retornando quando todos terminaram.
type Latch struct {
	n  int
	ch chan struct{}
}

func NovoLatch(n int) *Latch {
	return &Latch{n: n, ch: make(chan struct{}, n)}
}

// Done sinaliza que uma tarefa terminou (chamar exatamente uma vez por tarefa).
func (l *Latch) Done() {
	l.ch <- struct{}{}
}

// Wait bloqueia até receber os n sinais de Done.
func (l *Latch) Wait() {
	for i := 0; i < l.n; i++ {
		<-l.ch
	}
}
