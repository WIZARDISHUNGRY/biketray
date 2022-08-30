//go:build !windows && !darwin

package geo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/maltegrosse/go-geoclue2"
)

func Location(ctx context.Context) (<-chan LocationInfo, error) {

	output := make(chan LocationInfo, 1)

	// create new Instance of Geoclue Manager
	gcm, err := geoclue2.NewGeoclueManager()
	if err != nil {
		return nil, fmt.Errorf("geoclue2.NewGeoclueManager: %w", err)
	}

	// create new Instance of Geoclue Client
	client, err := gcm.GetClient()
	if err != nil {
		return nil, fmt.Errorf("gcm.GetClient: %w", err)

	}

	// desktop id is required to start the client
	// (double check your geoclue.conf file)
	err = client.SetDesktopId("firefox")
	if err != nil {
		return nil, fmt.Errorf("client.SetDesktopId: %w", err)
	}

	// Set RequestedAccuracyLevel
	err = client.SetRequestedAccuracyLevel(geoclue2.GClueAccuracyLevelExact)
	if err != nil {
		return nil, fmt.Errorf("client.SetRequestedAccuracyLevel: %w", err)
	}

	// client must be started before requesting the location
	err = client.Start()
	if err != nil {
		return nil, fmt.Errorf("client.Start: %w", err)
	}

	go func() {
	LOC_AGAIN:
		var location geoclue2.GeoclueLocation
		location, err = client.GetLocation()
		if err != nil {
			log.Println("GetLocation", err)
			select {
			case <-ctx.Done():
				return
			case output <- LocationInfo{Error: err}:
			}
			time.Sleep(100 * time.Millisecond)
			goto LOC_AGAIN
		}
		output <- newLocationInfo(location)

		updates := client.SubscribeLocationUpdated()
		for {
			select {
			case <-ctx.Done():
				return

			case v, ok := <-updates:
				if !ok {
					return // TODO crash?
				}
				_, location, err = client.ParseLocationUpdated(v)
				if err != nil {
					log.Println("client.ParseLocationUpdated", err)
				}
			}
		}
	}()
	return output, nil
}

func newLocationInfo(loc geoclue2.GeoclueLocation) (li LocationInfo) {
	lat, err := loc.GetLatitude()
	if err != nil {
		log.Println("GetLatitude", err)
		li.Error = err
		return
	}
	li.Lat = lat

	lon, err := loc.GetLongitude()
	if err != nil {
		log.Println("GetLongitude", err)
		li.Error = err
		return
	}
	li.Lon = lon

	desc, err := loc.GetDescription()
	if err != nil {
		log.Println("GetDescription", err)
		li.Error = err
		return
	}
	li.Description = desc
	return
}
