package geo

import (
	"context"
	"time"
)

type LocationInfo struct {
	Description string
	Lat, Lon    float64
}
type LocationFunc func(ctx context.Context) (<-chan LocationInfo, error)

func RateLimit(f LocationFunc, thresholdMeters float64, minimumRate time.Duration) LocationFunc {
	return func(ctx context.Context) (<-chan LocationInfo, error) {
		c, err := f(ctx)
		if err != nil {
			return c, err
		}
		output := make(chan LocationInfo, 1)

		go func() {
			defer close(output)
			current := LocationInfo{
				Lat: 90,
				Lon: 0,
			}
			lastUpdate := time.Time{}
			for {
				select {
				case <-ctx.Done():
					return
				case nextLoc, ok := <-c:
				NEXT_LOC:
					if !ok {
						return
					}
					distanceFromLast, _ := distance(current.Lat, current.Lon, nextLoc.Lat, nextLoc.Lon)
					if distanceFromLast > thresholdMeters || time.Now().Sub(lastUpdate) > minimumRate {
						lastUpdate = time.Now()
						current = nextLoc
						select {
						case output <- nextLoc:
						case <-ctx.Done():
							return
						case nextLoc, ok = <-c:
							goto NEXT_LOC
						}
					}
				}
			}
		}()
		return output, nil
	}
}
