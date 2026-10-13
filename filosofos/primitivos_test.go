package main

import (
	"testing"
	"time"
)

// 8 goroutines x 100 mil incrementos num contador comum: sem exclusão mútua
// o resultado seria menor e o -race acusaria data race.
func TestMutexContador(t *testing.T) {
	const gor, inc = 8, 100000
	m := NovoMutex()
	l := NovoLatch(gor)
	contador := 0
	for i := 0; i < gor; i++ {
		go func() {
			for j := 0; j < inc; j++ {
				m.Lock()
				contador++
				m.Unlock()
			}
			l.Done()
		}()
	}
	l.Wait()
	if contador != gor*inc {
		t.Fatalf("contador = %d, esperado %d", contador, gor*inc)
	}
}

func TestMutexUnlockSemLockPanica(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("esperava panic")
		}
	}()
	NovoMutex().Unlock()
}

func TestMutexLockTimeout(t *testing.T) {
	m := NovoMutex()
	if !m.LockTimeout(10 * time.Millisecond) {
		t.Fatal("mutex livre deveria travar")
	}
	if m.LockTimeout(10 * time.Millisecond) {
		t.Fatal("mutex travado deveria expirar")
	}
	m.Unlock()
	if !m.LockTimeout(10 * time.Millisecond) {
		t.Fatal("após Unlock deveria travar de novo")
	}
}

// Nunca mais que max goroutines dentro da região; o contador de "dentro"
// é protegido por um Mutex próprio.
func TestSemaforoLimite(t *testing.T) {
	const max, gor = 3, 20
	s := NovoSemaforo(max, max)
	mu := NovoMutex()
	l := NovoLatch(gor)
	dentro, maior := 0, 0
	for i := 0; i < gor; i++ {
		go func() {
			s.Acquire()
			mu.Lock()
			dentro++
			if dentro > maior {
				maior = dentro
			}
			mu.Unlock()
			time.Sleep(time.Millisecond)
			mu.Lock()
			dentro--
			mu.Unlock()
			s.Release()
			l.Done()
		}()
	}
	l.Wait()
	if maior > max {
		t.Fatalf("chegaram %d goroutines simultâneas, máximo %d", maior, max)
	}
	if s.Valor() != max {
		t.Fatalf("valor final = %d, esperado %d", s.Valor(), max)
	}
}

func TestSemaforoReleaseAcimaDoMaxPanica(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("esperava panic")
		}
	}()
	NovoSemaforo(2, 2).Release()
}

func TestSemaforoAcquireTimeout(t *testing.T) {
	s := NovoSemaforo(1, 0)
	if s.AcquireTimeout(10 * time.Millisecond) {
		t.Fatal("sem permissões deveria expirar")
	}
	s.Release()
	if !s.AcquireTimeout(10 * time.Millisecond) {
		t.Fatal("com permissão deveria conseguir")
	}
}

func TestLatchWaitEsperaNDones(t *testing.T) {
	const n = 5
	l := NovoLatch(n)
	feitos := make(chan struct{})
	go func() { l.Wait(); close(feitos) }()
	for i := 0; i < n-1; i++ {
		l.Done()
	}
	select {
	case <-feitos:
		t.Fatal("Wait retornou antes do último Done")
	case <-time.After(20 * time.Millisecond):
	}
	l.Done()
	select {
	case <-feitos:
	case <-time.After(time.Second):
		t.Fatal("Wait não retornou após n Done")
	}
}
