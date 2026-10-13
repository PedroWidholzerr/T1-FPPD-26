package main

import "time"

// Mutex implementado com um channel de capacidade 1.
//
// Invariante: o channel cheio (len == 1) significa "travado"; vazio significa "livre".
// Lock envia um valor (bloqueia se já estiver cheio) e Unlock retira o valor.
type Mutex struct {
	ch chan struct{}
}

func NovoMutex() *Mutex {
	return &Mutex{ch: make(chan struct{}, 1)}
}

// Lock trava o mutex. Se já estiver travado, bloqueia até alguém chamar Unlock.
func (m *Mutex) Lock() {
	m.ch <- struct{}{}
}

// Unlock destrava. Usamos select/default para não bloquear: se o channel
// estiver vazio, o mutex não estava travado, o que é um erro de programação.
func (m *Mutex) Unlock() {
	select {
	case <-m.ch:
	default:
		panic("unlock de mutex destravado")
	}
}

// LockTimeout tenta travar por no máximo d. Retorna true se conseguiu travar.
func (m *Mutex) LockTimeout(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop() // libera o timer se o Lock vencer a corrida
	select {
	case m.ch <- struct{}{}:
		return true
	case <-t.C:
		return false
	}
}
