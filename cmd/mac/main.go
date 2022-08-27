package main

import (
	"context"
	"fmt"

	"jonwillia.ms/biketray/geo"
)

// https://github.com/WIZARDISHUNGRY/osx-location
// https://coderwall.com/p/l9jr5a/accessing-cocoa-objective-c-from-go-with-cgo
func main() {
	l, _ := geo.Location(context.Background())
	for loc := range l {
		fmt.Println(loc)
	}
}
