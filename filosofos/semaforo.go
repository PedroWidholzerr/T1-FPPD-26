package main

import "time"

// Semaforo de contagem implementado com um channel de capacidade max.
//
// Modelo "permissões dentro do channel": cada elemento no channel é uma
// permissão disponível. Invariante: 0 <= len(ch) <= max.
// Acquire retira uma permissão; Release devolve uma.
type Semaforo struct {
	ch chan struct{}
}

// NovoSemaforo cria um semáforo com no máximo max permissões,
// já começando com inicial permissões disponíveis.
func NovoSemaforo(max, inicial int) *Semaforo {
	s := &Semaforo{ch: make(chan struct{}, max)}
	for i := 0; i < inicial; i++ {
		s.ch <- struct{}{}
	}
	return s
}

// Acquire pega uma permissão; bloqueia se não houver nenhuma.
func (s *Semaforo) Acquire() {
	<-s.ch
}

// Release devolve uma permissão. Se já estiver cheio (excederia max),
// é erro de programação: panic, em vez de bloquear para sempre.
func (s *Semaforo) Release() {
	select {
	case s.ch <- struct{}{}:
	default:
		panic("release acima do máximo do semáforo")
	}
}

// AcquireTimeout tenta pegar uma permissão por no máximo d.
// Retorna false (sem consumir nada) se o tempo estourou.
func (s *Semaforo) AcquireTimeout(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-s.ch:
		return true
	case <-t.C:
		return false
	}
}

// Valor devolve quantas permissões estão disponíveis agora.
func (s *Semaforo) Valor() int {
	return len(s.ch)
}
