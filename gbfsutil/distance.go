package gbfsutil

import (
	"math"

	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
	"github.com/petoc/gbfs"
	"jonwillia.ms/biketray/geo"
)

var geo1 = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)

func Distance(location geo.LocationInfo, lat, lon *gbfs.Coordinate) (float64, float64) {
	if lat == nil || lon == nil {
		return math.Inf(1), math.NaN()
	}
	return geo1.To(location.Lat, location.Lon, lat.Float64, lon.Float64)
}
