package main

import "time"

// BufferChan usa o channel com buffer da própria linguagem como fila limitada.
// O channel já faz tudo: bloqueia o Put quando cheio, bloqueia o Get quando vazio.
// Não precisa de Mutex nem de Semaforo.
type BufferChan struct {
	ch chan Item
}

func NovoBufferChan(k int) *BufferChan {
	return &BufferChan{ch: make(chan Item, k)}
}

func (b *BufferChan) Put(it Item) {
	b.ch <- it
}

func (b *BufferChan) Get(timeout time.Duration) (Item, Status) {
	if timeout == 0 {
		it, ok := <-b.ch
		if !ok {
			return Item{}, Closed // channel fechado e vazio
		}
		return it, OK
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case it, ok := <-b.ch:
		if !ok {
			return Item{}, Closed
		}
		return it, OK
	case <-t.C:
		return Item{}, Timeout
	}
}

func (b *BufferChan) Len() int {
	return len(b.ch)
}

// Encerrar fecha o channel. Só a main pode chamar (depois que todos os
// produtores terminaram): fechar enquanto alguém ainda envia causaria panic.
// Os consumidores continuam recebendo o que sobrou e só depois veem ok == false.
func (b *BufferChan) Encerrar(nConsumidores int) {
	close(b.ch)
}
