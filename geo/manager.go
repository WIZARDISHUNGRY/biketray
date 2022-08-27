package geo

import (
	"context"
	"fmt"
	"log"
	"sync"
)

type Manager struct {
	c       <-chan LocationInfo
	hasLoc  bool
	lastLoc LocationInfo
	mutex   sync.Mutex
	dests   map[<-chan LocationInfo]chan<- LocationInfo
}

func NewManager(ctx context.Context, c <-chan LocationInfo) *Manager {
	m := &Manager{
		c:     c,
		dests: make(map[<-chan LocationInfo]chan<- LocationInfo),
	}
	go m.run(ctx)
	return m
}

func (m *Manager) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case loc := <-m.c:
			panic("ok")
			m.mutex.Lock()
			fmt.Println("geo mgr", loc)
			m.lastLoc = loc
			m.hasLoc = true
			for _, c := range m.dests {
				select {
				case <-ctx.Done():
					m.mutex.Unlock()
					return
				case c <- loc:
				}
			}
			m.mutex.Unlock()
		}
	}
}

func (m *Manager) Subscribe() <-chan LocationInfo {
	c := make(chan LocationInfo, 1)
	m.mutex.Lock()
	m.dests[c] = c
	if m.hasLoc {
		log.Println("hasLoc")
		c <- m.lastLoc
	}
	m.mutex.Unlock()
	return c
}

func (m *Manager) Unsubscribe(c <-chan LocationInfo) {
	m.mutex.Lock()
	delete(m.dests, c)
	m.mutex.Unlock()
}
