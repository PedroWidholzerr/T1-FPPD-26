package main

import "time"

// BufferSem é o algoritmo clássico: anel de K posições + dois semáforos + dois mutexes.
//
//	notFull  conta VAGAS livres  (começa em K): o produtor espera aqui se o anel está cheio.
//	notEmpty conta ITENS prontos (começa em 0): o consumidor espera aqui se está vazio.
//
// Os semáforos garantem que há vaga/item, mas NÃO impedem que dois produtores
// escolham a mesma vaga ao mesmo tempo (nem dois consumidores o mesmo item).
// Por isso existem muIn e muOut: protegem os índices in e out.
// São dois mutexes separados para um produtor e um consumidor poderem
// trabalhar ao mesmo tempo (eles mexem em posições diferentes do anel).
type BufferSem struct {
	buf      []Item
	in, out  int
	notFull  *Semaforo
	notEmpty *Semaforo
	muIn     *Mutex
	muOut    *Mutex
}

func NovoBufferSem(k int) *BufferSem {
	return &BufferSem{
		buf:      make([]Item, k),
		notFull:  NovoSemaforo(k, k),
		notEmpty: NovoSemaforo(k, 0),
		muIn:     NovoMutex(),
		muOut:    NovoMutex(),
	}
}

func (b *BufferSem) Put(it Item) {
	b.notFull.Acquire() // reserva uma vaga (bloqueia se cheio)
	b.muIn.Lock()
	b.buf[b.in] = it
	b.in = (b.in + 1) % len(b.buf)
	b.muIn.Unlock()
	b.notEmpty.Release() // avisa: há um item a mais
}

func (b *BufferSem) Get(timeout time.Duration) (Item, Status) {
	if timeout == 0 {
		b.notEmpty.Acquire()
	} else if !b.notEmpty.AcquireTimeout(timeout) {
		return Item{}, Timeout // nada foi consumido: nenhum estado mudou
	}
	b.muOut.Lock()
	it := b.buf[b.out]
	b.out = (b.out + 1) % len(b.buf)
	b.muOut.Unlock()
	b.notFull.Release() // devolve a vaga
	if it.Pill {
		return Item{}, Closed
	}
	return it, OK
}

// Len é o número de itens prontos, que é exatamente o valor do semáforo notEmpty.
func (b *BufferSem) Len() int {
	return b.notEmpty.Valor()
}

// Encerrar coloca uma poison pill para cada consumidor. Como o anel é FIFO,
// todos os itens reais saem antes das pills; cada consumidor pega uma pill e sai.
func (b *BufferSem) Encerrar(nConsumidores int) {
	for i := 0; i < nConsumidores; i++ {
		b.Put(Item{Pill: true})
	}
}
