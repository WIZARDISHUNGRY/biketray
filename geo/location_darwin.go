//go:build darwin

package geo

import (
	"context"
	"time"
)

func Location(ctx context.Context) (<-chan LocationInfo, error) {
	output := make(chan LocationInfo, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-time.After(time.Second):
				output <- LocationInfo{
					Description: "WTF",
				}
			}
		}
	}()
	return output, nil
}
