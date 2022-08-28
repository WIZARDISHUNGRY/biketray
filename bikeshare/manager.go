package bikeshare

import (
	"context"
	"fmt"
	"sync"

	"jonwillia.ms/biketray/geo"
	"jonwillia.ms/biketray/systems"
)

type Manager struct {
	nearbySystemsC      <-chan map[systems.System]systems.NearbyResult
	nearbySystemsMirror chan map[systems.System]systems.NearbyResult
	nearbySystems       map[systems.System]*Client
	clientResults       chan ClientResult
	geoMgr              *geo.Manager
	mutex               sync.Mutex
}

type ClientResult struct {
	System systems.System
	Data   []Datum
}

type Datum struct {
	Label        string
	LocationInfo geo.LocationInfo
}

func NewManager(ctx context.Context, geoMgr *geo.Manager, nearbySystemsC <-chan map[systems.System]systems.NearbyResult) *Manager {
	m := &Manager{
		nearbySystemsC:      nearbySystemsC,
		geoMgr:              geoMgr,
		nearbySystems:       make(map[systems.System]*Client, 1),
		nearbySystemsMirror: make(chan map[systems.System]systems.NearbyResult, 1),
		clientResults:       make(chan ClientResult, 1),
	}
	go m.run(ctx)
	return m
}

func (m *Manager) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case incoming := <-m.nearbySystemsC:
			// This needs a restart if it gets an existing system that suddently adds stations or bikes as a top level feature
			m.nearbySystemsMirror <- incoming
			for k, nearbySystem := range incoming {
				client, ok := m.nearbySystems[k]
				if !ok {
					fmt.Println("nearby system", k.Name)
					client = newClient(ctx, m, nearbySystem)
					m.nearbySystems[k] = client
				}
			}
			for k, client := range m.nearbySystems {
				if _, ok := incoming[k]; !ok {
					client.Close()
					delete(m.nearbySystems, k)
				}
			}
		}
	}
}
func (m *Manager) clientResult(k systems.System, data []Datum) {
	m.clientResults <- ClientResult{System: k, Data: data}
}
func (m *Manager) ClientResults() <-chan ClientResult {
	return m.clientResults
}

func (m *Manager) NearbyResults() <-chan map[systems.System]systems.NearbyResult {
	return m.nearbySystemsMirror
}
