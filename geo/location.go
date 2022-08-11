package geo

import (
	"context"
	"fmt"
	"log"

	"github.com/maltegrosse/go-geoclue2"
)

type LocationInfo struct {
	Description string
	Lat, Lon    float64
}

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
	location, err := client.GetLocation()
	if err != nil {
		return nil, fmt.Errorf("client.GetLocation: %w", err) // TODO might this fail
	}
	updates := client.SubscribeLocationUpdated()
	go func() {
		for {
			output <- newLocationInfo(location)
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

func newLocationInfo(loc geoclue2.GeoclueLocation) LocationInfo {
	lat, err := loc.GetLatitude()
	if err != nil {
		log.Println("GetLatitude", err)
	}
	lon, err := loc.GetLongitude()
	if err != nil {
		log.Println("GetLongitude", err)
	}
	desc, err := loc.GetDescription()
	if err != nil {
		log.Println("GetDescription", err)
	}
	return LocationInfo{Lat: lat, Lon: lon, Description: desc}
}
