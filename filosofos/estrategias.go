package main

import "time"

// Estrategia decide COMO um filósofo consegue seus dois garfos.
// O laço do filósofo (main.go) é o mesmo para todas; só isto muda.
type Estrategia interface {
	Pegar(id int)  // só retorna com os dois garfos em mãos
	Soltar(id int) // devolve os dois garfos
}

// Convenção: o filósofo i tem o garfo i à esquerda e o garfo (i+1)%n à direita.
func esquerdo(id int) int   { return id }
func direito(id, n int) int { return (id + 1) % n }

// novosGarfos cria n garfos; cada garfo é um Mutex próprio (cheio = em uso).
func novosGarfos(n int) []*Mutex {
	g := make([]*Mutex, n)
	for i := range g {
		g[i] = NovoMutex()
	}
	return g
}

// ---------------------------------------------------------------- base --

// Base: pega o esquerdo, espera um pouco, pega o direito.
// Se todos pegarem o esquerdo antes de qualquer um pegar o direito, forma-se
// um ciclo de espera e ninguém avança: DEADLOCK (as 4 condições de Coffman valem).
type Base struct {
	garfos  []*Mutex
	pausa   time.Duration
	eventos chan<- Evento // para avisar a main que o esquerdo já foi pego
}

func (b *Base) Pegar(id int) {
	b.garfos[esquerdo(id)].Lock()
	b.eventos <- Evento{ID: id, Tipo: PegouEsq}
	time.Sleep(b.pausa) // a pausa aumenta a chance de todos pegarem o esquerdo
	b.garfos[direito(id, len(b.garfos))].Lock()
}

func (b *Base) Soltar(id int) {
	b.garfos[esquerdo(id)].Unlock()
	b.garfos[direito(id, len(b.garfos))].Unlock()
}

// ----------------------------------------------------------- hierarquia --

// Hierarquia: todos pegam primeiro o garfo de MENOR índice e depois o maior.
// Isso numera os recursos e quebra a espera circular: nunca há ciclo.
type Hierarquia struct {
	garfos []*Mutex
}

func (h *Hierarquia) Pegar(id int) {
	a, b := esquerdo(id), direito(id, len(h.garfos))
	if a > b {
		a, b = b, a
	}
	h.garfos[a].Lock()
	h.garfos[b].Lock()
}

func (h *Hierarquia) Soltar(id int) {
	h.garfos[esquerdo(id)].Unlock()
	h.garfos[direito(id, len(h.garfos))].Unlock()
}

// --------------------------------------------------------------- limite --

// Limite: no máximo n-1 filósofos "sentados" ao mesmo tempo (semáforo com n-1
// permissões). Com n-1 sentados, pelo menos um consegue os dois garfos, então
// o ciclo completo de espera é impossível.
type Limite struct {
	garfos []*Mutex
	mesa   *Semaforo
}

func NovoLimite(garfos []*Mutex) *Limite {
	n := len(garfos)
	return &Limite{garfos: garfos, mesa: NovoSemaforo(n-1, n-1)}
}

func (l *Limite) Pegar(id int) {
	l.mesa.Acquire() // senta à mesa
	l.garfos[esquerdo(id)].Lock()
	l.garfos[direito(id, len(l.garfos))].Lock()
}

func (l *Limite) Soltar(id int) {
	l.garfos[esquerdo(id)].Unlock()
	l.garfos[direito(id, len(l.garfos))].Unlock()
	l.mesa.Release() // levanta da mesa
}

// --------------------------------------------------------------- garçom --

// pedido: um filósofo pede os garfos e espera a resposta no seu próprio channel.
type pedido struct {
	id   int
	resp chan struct{}
}

// Garcom: uma goroutine árbitro dona do estado dos garfos (ninguém mais mexe
// neles, então não há Mutex de garfo). Atende os pedidos em fila FIFO estrita:
// só olha a CABEÇA da fila e só a atende quando os dois garfos estão livres.
// Isso evita starvation (ninguém fura a fila), mas reduz o paralelismo: se a
// cabeça está bloqueada, quem vem atrás espera mesmo com garfos livres.
// Quebra "posse e espera": o filósofo recebe os dois garfos juntos ou nenhum.
type Garcom struct {
	n          int
	pedidos    chan pedido
	liberacoes chan int
	respostas  []chan struct{} // um channel de resposta por filósofo
	fim        chan struct{}   // fechado quando a goroutine do garçom sai
}

func NovoGarcom(n int) *Garcom {
	g := &Garcom{
		n:          n,
		pedidos:    make(chan pedido),
		liberacoes: make(chan int),
		respostas:  make([]chan struct{}, n),
		fim:        make(chan struct{}),
	}
	for i := range g.respostas {
		g.respostas[i] = make(chan struct{}, 1) // buffer 1: o garçom nunca bloqueia ao responder
	}
	go g.atender()
	return g
}

func (g *Garcom) Pegar(id int) {
	g.pedidos <- pedido{id: id, resp: g.respostas[id]}
	<-g.respostas[id] // bloqueia até o garçom autorizar
}

func (g *Garcom) Soltar(id int) {
	g.liberacoes <- id
}

// Encerrar é chamado pela main depois que todos os filósofos terminaram:
// fechar o channel de pedidos é o sinal de "fim" para o garçom.
func (g *Garcom) Encerrar() {
	close(g.pedidos)
	<-g.fim // espera o garçom realmente sair
}

func (g *Garcom) atender() {
	defer close(g.fim)
	ocupado := make([]bool, g.n) // estado dos garfos: só esta goroutine lê/escreve
	var fila []pedido
	for {
		select {
		case p, ok := <-g.pedidos:
			if !ok {
				return // main fechou o channel: encerramento
			}
			fila = append(fila, p)
		case id := <-g.liberacoes:
			ocupado[esquerdo(id)] = false
			ocupado[direito(id, g.n)] = false
		}
		// Atende só a cabeça da fila, enquanto os dois garfos dela estiverem livres.
		for len(fila) > 0 {
			id := fila[0].id
			e, d := esquerdo(id), direito(id, g.n)
			if ocupado[e] || ocupado[d] {
				break
			}
			ocupado[e], ocupado[d] = true, true
			fila[0].resp <- struct{}{}
			fila = fila[1:]
		}
	}
}
