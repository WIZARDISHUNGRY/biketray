package links

import (
	"fmt"
	"runtime"

	"github.com/skratchdot/open-golang/open"
	"jonwillia.ms/biketray/geo"
)

// https://www.google.com/maps/@40.6962621,-73.9189533
// http://maps.google.co.uk/maps?q=loc:52.03877,-2.3416&z=15
// https://www.google.com/maps?saddr=My+Location&daddr=43.12345,-76.12345

func OpenLocation(startLoc *geo.LocationInfo, loc geo.LocationInfo) error {
	host := "www.google.com"
	var isApple = false
	if runtime.GOOS == "darwin" {
		host = "maps.apple.com"
		isApple = true
	}
	startStr := "My+Location"
	if startLoc != nil {
		startStr = fmt.Sprintf("%f,%f", startLoc.Lat, startLoc.Lon)
	}
	s := fmt.Sprintf("https://%s/maps?saddr=%s&daddr=%f,%f", host, startStr, loc.Lat, loc.Lon)
	if isApple {
		return open.StartWith(s, "Maps.App")
	}
	return open.Start(s)
}
