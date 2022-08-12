package systems

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
	"github.com/dnaeon/go-vcr/v2/recorder"
	petoc "github.com/petoc/gbfs"
	"golang.org/x/sync/errgroup"
	"jonwillia.ms/biketray/geo"
)

const CSV = "https://raw.githubusercontent.com/NABSA/gbfs/master/systems.csv"
const systemDist = 60000 // meters
var httpUA string = func() string {
	return "biktray"
}()

type System struct {
	CountryCode      string
	Name             string
	Location         string
	SystemID         string
	URL              string
	AutoDiscoveryURL string
}

func Load() []System {
	r, err := http.NewRequest(http.MethodGet, CSV, nil)
	if err != nil {
		panic(err)
	}
	resp, err := httpClient.Do(r)
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}
	if len(records) < 2 {
		panic("len records wrong")
	}
	if len(records[0]) != 6 {
		panic("num calls wrong")
	}

	systems := make([]System, 0)

	for _, systemRow := range records[1:] {
		system := System{
			CountryCode:      systemRow[0],
			Name:             systemRow[1],
			Location:         systemRow[2],
			SystemID:         systemRow[3],
			URL:              systemRow[4],
			AutoDiscoveryURL: systemRow[5],
		}
		systems = append(systems, system)
	}
	for _, system := range systems {
		fmt.Println(system.AutoDiscoveryURL)
	}
	return systems
}

func Test(systems []System) map[System]*petoc.Client {

	// system := systems["bird-new-york"]

	ctx := context.TODO()
	g, ctx := errgroup.WithContext(ctx)

	var (
		mutex         sync.Mutex
		systemClients = make(map[System]*petoc.Client)
	)
	g.SetLimit(16)
	for _, system := range systems {
		system := system
		g.Go(func() error {
			// if system.AutoDiscoveryURL != "https://data.lime.bike/api/partners/v2/gbfs/new_york/gbfs.json" {
			// 	return nil
			// }
			pc, err := petoc.NewClient(petoc.ClientOptions{
				AutoDiscoveryURL: system.AutoDiscoveryURL,
				UserAgent:        httpUA,
				HTTPClient:       &httpClient,
				DefaultLanguage:  "en",
			})
			if err != nil {
				panic(err)
			}
			si := &petoc.FeedSystemInformation{}
			err = pc.Get(si)

			if err == nil {

				mutex.Lock()
				systemClients[system] = pc
				mutex.Unlock()
			} else {
				log.Println("can't load", system.AutoDiscoveryURL)
			}
			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		panic(err)
	}

	fmt.Println("Test done", len(systems), len(systemClients))
	return systemClients
}

var httpClient, StopRecorder = func() (http.Client, func() error) {
	r, err := recorder.New("http-cache")
	if err != nil {
		log.Fatalf("recorder.New: %v", err)
	}
	r.SkipRequestLatency = true

	var pass atomic.Uint32

	r.Passthroughs = append(r.Passthroughs,
		func(req *http.Request) bool {
			return pass.Load() != 0
		},
	)

	return http.Client{
			Timeout:   10 * time.Second,
			Transport: r,
		}, func() error {
			pass.Store(1)
			return r.Stop()
		}
}()

type AutoDiscovery struct {
	Data struct {
		En struct {
			Feeds []struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"feeds"`
		} `json:"en"`
	} `json:"data"`
	LastUpdated int64 `json:"last_updated"`
	TTL         int64 `json:"ttl"`
}

type NearbyResult struct {
	System                 System
	FeedStationInformation petoc.FeedStationInformation
	Client                 *petoc.Client
}

var geo1 = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)

func Nearby(ctx context.Context, clientsC <-chan map[System]*petoc.Client, mgr *geo.Manager) <-chan map[System]NearbyResult {
	c := make(chan map[System]NearbyResult, 1)

	go func() {
		locC := mgr.Subscribe()
		defer mgr.Unsubscribe(locC)

		var location geo.LocationInfo

		select {
		case location = <-locC:
		case <-ctx.Done():
			return
		}
		var clients map[System]*petoc.Client
	MAIN_LOOP:
		for {
			select {
			case location = <-locC:
				continue MAIN_LOOP
			case <-ctx.Done():
				return
			case clients = <-clientsC:
			}

			g, _ := errgroup.WithContext(ctx)
			var (
				mutex       sync.Mutex
				initResults map[System]NearbyResult = make(map[System]NearbyResult)
			)

			for system, client := range clients {
				system, client := system, client
				g.Go(func() error {
					var si petoc.FeedStationInformation
					if err := client.Get(&si); err != nil {
						fmt.Println("station info", system.Name, system.AutoDiscoveryURL, err)
						return nil
					}

					mutex.Lock()
					initResults[system] = NearbyResult{
						System:                 system,
						FeedStationInformation: si,
						// StationInformation:     station,
						Client: client,
					}
					mutex.Unlock()

					return nil
				})
			}

			if err := g.Wait(); err != nil {
				panic(err)
			}

			for ctx.Err() == nil {

				dist := func(s *petoc.FeedStationInformationStation) float64 {
					distance, _ := geo1.To(location.Lat, location.Lon, s.Lat.Float64, s.Lon.Float64)
					return distance
				}

				var results map[System]NearbyResult = make(map[System]NearbyResult)

			NEXT_SYSTEM:
				for k, v := range initResults {
					if v.FeedStationInformation.Data == nil {
						log.Println("Stationless system", v.System.AutoDiscoveryURL)
						continue NEXT_SYSTEM // Stationless system
						panic(v.System.AutoDiscoveryURL + " v.FeedStationInformation.Data")
					}
				NEXT_STATION:
					for _, station := range v.FeedStationInformation.Data.Stations {
						if station.Lat == nil || station.Lon == nil {
							// log.Println("Placeless station", *station.Name)
							continue NEXT_STATION
						}

						d := dist(station)
						if d < systemDist {
							fmt.Println("station in range", v.System.Name, *station.Name, d)
							results[k] = v
							continue NEXT_SYSTEM
						}
					}
				}

				fmt.Println("Nearby", len(results))
				c <- results
				select {
				case <-ctx.Done():
					return
				case location = <-locC:
				}
			}
		}
	}()
	return c
}
