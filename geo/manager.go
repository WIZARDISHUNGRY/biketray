package geo

import (
	"context"
	"log"
	"sync"
)

type Manager struct {
	c       <-chan LocationInfo
	hasLoc  bool
	lastLoc LocationInfo
	mutex   sync.Mutex
	dests   map[<-chan LocationInfo]struct {
		c             chan<- LocationInfo
		includeErrors bool
	}
}

func NewManager(ctx context.Context, c <-chan LocationInfo) *Manager {
	m := &Manager{
		c: c,
		dests: make(map[<-chan LocationInfo]struct {
			c             chan<- LocationInfo
			includeErrors bool
		}),
	}
	go m.run(ctx)
	return m
}

func (m *Manager) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case loc, ok := <-m.c:
			if !ok {
				log.Println("geoMgr read on closed channel, exiting")
				return
			}
			m.mutex.Lock()
			if loc.Error == nil {
				m.lastLoc = loc
				m.hasLoc = true
			}
			i := 0
			for drain, sub := range m.dests {
				i++

				if loc.Error != nil && !sub.includeErrors {
					continue
				}

			OUTPUT_AGAIN:
				select {
				case <-ctx.Done():
					m.mutex.Unlock()
					return
				case sub.c <- loc:
				default:
					log.Println("geoMgr clearing blocked channel", i)
					select {
					case <-drain:
					default:
					}
					goto OUTPUT_AGAIN
				}
			}
			m.mutex.Unlock()
		}
	}
}

func (m *Manager) Subscribe(includeErrors bool) <-chan LocationInfo {
	c := make(chan LocationInfo, 1)
	m.mutex.Lock()
	m.dests[c] = struct {
		c             chan<- LocationInfo
		includeErrors bool
	}{
		c:             c,
		includeErrors: includeErrors,
	}
	if m.hasLoc && (includeErrors || m.lastLoc.Error == nil) {
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

func (m *Manager) CurrentLocation() (LocationInfo, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.lastLoc, m.hasLoc
}
