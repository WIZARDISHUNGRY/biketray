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
			case err := <-s.Errors():
				log.Println("Darwin Location Error", err)
				// TODO prompt to /usr/bin/open "x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices"
			case loc := <-s.Locations():
			ANOTHER:
				select {
				case <-ctx.Done():
					// No cleanup of location services
					return
				case output <- LocationInfo{Lat: loc.Lat, Lon: loc.Lon}:
				case loc = <-s.Locations():
					goto ANOTHER
				}
			}
		}
	}()
	return output, nil
}
