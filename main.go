package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"golang.org/x/exp/maps"
	"jonwillia.ms/biketray/bikeshare"
	"jonwillia.ms/biketray/geo"
	"jonwillia.ms/biketray/systems"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	systray.Run(func() { onReady(ctx) }, func() { onExit(ctx, cancel) })
}

const timeFmt = time.RFC822

func onReady(ctx context.Context) {

	lat := flag.Float64("lat", math.NaN(), "lat")
	lon := flag.Float64("lon", math.NaN(), "lat")
	flagReadCache := flag.Bool("readCache", true, "read systems cache")
	flagWriteCache := flag.Bool("writeCache", true, "write systems cache")

	flag.Parse()

	systray.SetIcon(icon.Data)
	systray.SetTitle("BikeTray")
	systray.SetTooltip("Pretty awesome超级棒")

	var (
		locChan <-chan geo.LocationInfo
		err     error
	)

	if math.IsNaN(*lat) && math.IsNaN(*lon) {
		locChan, err = geo.Location(ctx)
		if err != nil {
			log.Fatalf("geo.Location: %v", err)
		}
	} else {
		c := make(chan geo.LocationInfo, 1)
		locChan = c
		wakeFakeGeo := make(chan geo.LocationInfo)
		go func() {
			g := geo.LocationInfo{Lat: *lat, Lon: *lon}
			for {
				fmt.Println("fake geo", g)
				c <- g
				select {
				case <-time.After(time.Minute):
				case g = <-wakeFakeGeo:
					fmt.Println("wake fake geo")
				}
			}
		}()
		mi := systray.AddMenuItem("Teleport to", "")

		for city, geo := range map[string]geo.LocationInfo{
			"Central Park":            {40.785091, -73.968285},
			"Spanish Steps, Rome":     {41.905991, 12.482775},
			"Corona Heights Park, SF": {37.765678, -122.438713},
			"Montreal":                {45.508888, -73.561668},
			"Buckingham Palace":       {51.501476, -0.140634},
			"Soldier Field, Chicago":  {41.862366, -87.617256},
			//
			"omphalos": {},
		} {
			city, geo := city, geo
			si := mi.AddSubMenuItem(city, "")
			go func() {
				for {
					<-si.ClickedCh
					fmt.Println("teleport to ", city, geo)
					wakeFakeGeo <- geo
				}
			}()
		}
	}

	geoMgr := geo.NewManager(ctx, locChan)

	start := time.Now()
	var csvSystems []systems.System
	var ok bool
	if *flagReadCache {
		csvSystems, ok = systems.LoadCache()
	}
	if !ok {
		csvSystems = systems.Load() // slow!
	}
	clients := systems.Test(csvSystems) // slow!
	if *flagWriteCache {
		systems.WriteCache(maps.Keys(clients))
	}
	dur := time.Since(start)
	log.Println("boot duration", dur)

	topMenus := make(map[systems.System]*systray.MenuItem)
	subMenus := make(map[systems.System][]*systray.MenuItem)
	for system := range clients {
		name := fmt.Sprintf("%s (%s)", system.Name, system.Location)
		fmt.Println(name)
		mi := systray.AddMenuItem(name, system.SystemID)
		mi.Hide()
		topMenus[system] = mi
	}

	initSubMenus := func(mi *systray.MenuItem, system systems.System) {
		for i := 0; i < 10; i++ {
			sub := mi.AddSubMenuItem("", "")
			sub.Hide()
			sub.SetIcon(icon.Data)
			subMenus[system] = append(subMenus[system], sub)
		}
	}

	systemsNearbyC := systems.Nearby(ctx, clients, geoMgr)
	bsMgr := bikeshare.NewManager(ctx, geoMgr, systemsNearbyC)
	// Sets the icon of a menu item. Only available on Mac and Windows.

	// mCiti := systray.AddMenuItem("CitiBike", "")
	// mStations := make([]*systray.MenuItem, 0)
	// for i := 0; i < 10; i++ {
	// 	mi := mCiti.AddSubMenuItem("", "")
	// 	mi.Hide()
	// 	mi.SetIcon(icon.Data)
	// 	mStations = append(mStations, mi)
	// }

	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")
	mQuit.SetIcon(icon.Data)

	type activeSystem struct {
		NearbyResult systems.NearbyResult
		Cancel       func()
	}

	for {
		select {
		case <-mQuit.ClickedCh: // TODO all click handlers should be in a tight loop because there is a default
			systray.Quit()
		case nr := <-bsMgr.NearbyResults():
			log.Println("visible systems update", len(nr))
			for system, mi := range topMenus {

				if _, ok := nr[system]; ok {
					fmt.Println("show", system.Name)
					mi.Show()
				} else {
					mi.Hide()
				}
			}
		case cr := <-bsMgr.ClientResults():
			mCiti, ok := topMenus[cr.System]
			if !ok {
				log.Println("mCiti, ok := topMenus[cr.System]")
				continue
			}
			mCiti.Show()
			mStations, ok := subMenus[cr.System]
			if !ok {
				initSubMenus(mCiti, cr.System)
				mStations, _ = subMenus[cr.System]
			}

			mCiti.SetTooltip(time.Now().Format(timeFmt))
			for i, mi := range mStations {
				if i >= len(cr.Data) {
					mi.Hide()
					continue
				}
				mi.Show()
				mi.SetTitle(cr.Data[i])
			}
		}
	}
}

func onExit(ctx context.Context, cancel func()) {
	defer cancel()
}
