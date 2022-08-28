package links

import (
	"fmt"

	"github.com/skratchdot/open-golang/open"
	"jonwillia.ms/biketray/geo"
)

// https://www.google.com/maps/@40.6962621,-73.9189533
// http://maps.google.co.uk/maps?q=loc:52.03877,-2.3416&z=15
// https://www.google.com/maps?saddr=My+Location&daddr=43.12345,-76.12345

func OpenLocation(startLoc *geo.LocationInfo, loc geo.LocationInfo) error {
	startStr := "My+Location"
	if startLoc != nil {
		startStr = fmt.Sprintf("%f,%f", startLoc.Lat, startLoc.Lon)
	}
	s := fmt.Sprintf("https://www.google.com/maps?saddr=%s&daddr=%f,%f", startStr, loc.Lat, loc.Lon)
	return open.Start(s)
}
