//go:build darwin

package darwin

//#cgo CFLAGS: -x objective-c
//#cgo LDFLAGS: -framework cocoa -framework Foundation -framework CoreLocation
//#include "location.h"
import "C"
import (
	"context"
	"errors"
	"log"
	"sync"
)

// https://github.com/WIZARDISHUNGRY/osx-location
// https://coderwall.com/p/l9jr5a/accessing-cocoa-objective-c-from-go-with-cgo

// NSNumber -> Go int
func goint(i *C.NSNumber) int { return int(C.nsnumber2int(i)) }

// NSString -> C string
func cstring(s *C.NSString) *C.char { return C.nsstring2cstring(s) }

// NSString -> Go string
func gostring(s *C.NSString) string { return C.GoString(cstring(s)) }

//export goWithError
func goWithError(h C.int, str *C.char) {
	s := byHandle(int(h))
	errStr := C.GoString(str)
	s.cgoErrors <- errors.New(errStr)
}

//export goWithCoords
func goWithCoords(h C.int, coords *C.Coords) {
	s := byHandle(int(h))
	s.locations <- Location{Lat: float64(coords.lat), Lon: float64(coords.lon)}
}

type Service struct {
	cgoErrors chan error
	locations chan Location
}

type Location struct {
	Lat, Lon float64
}

var (
	mutex   sync.Mutex
	handles []*Service
)

func byHandle(h int) *Service {
	mutex.Lock()
	defer mutex.Unlock()
	return handles[h]
}

func getHandle(s *Service) int {
	mutex.Lock()
	defer mutex.Unlock()
	l := len(handles)
	handles = append(handles, s)
	return l
}

func (s *Service) Run(ctx context.Context) error {
	s.cgoErrors = make(chan error, 1)
	s.locations = make(chan Location, 1)
	h := getHandle(s)

	go func() {
		_ = C.run(C.int(h))
		log.Println("darwin location services returned")
		// defer close(s.cgoErrors)
		// defer close(s.locations)
		// TODO need way to signal location services to shutdown

		// TODO does this not work in goroutine
	}() //  TODO no way to cancel

	return nil
}

func (s *Service) Errors() <-chan error {
	return s.cgoErrors
}
func (s *Service) Locations() <-chan Location {
	return s.locations
}
