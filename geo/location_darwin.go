//go:build darwin

package geo

import (
	"context"
	"log"

	"jonwillia.ms/biketray/geo/internal/darwin"
)

func Location(ctx context.Context) (<-chan LocationInfo, error) {
	output := make(chan LocationInfo, 1)
	s := darwin.Service{}
	err := s.Run(ctx) // TODO we don't retry if location services are disabled
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				// No cleanup of location services
				return
			case loc := <-s.Locations():
				if loc.Error != nil {
					log.Println("Darwin Location Error", loc.Error)
				}
			ANOTHER:
				select {
				case <-ctx.Done():
					// No cleanup of location services
					return
				case output <- LocationInfo{Lat: loc.Lat, Lon: loc.Lon, Error: loc.Error}:
				case loc = <-s.Locations():
					goto ANOTHER
				}
			}
		}
	}()
	return output, nil
}
