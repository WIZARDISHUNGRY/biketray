package geo

import (
	"math"

	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
	"github.com/petoc/gbfs"
)

var geo1 = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)

func Distance(location LocationInfo, lat, lon *gbfs.Coordinate) (float64, float64) {
	if lat == nil || lon == nil {
		return math.Inf(1), math.NaN()
	}
	return distance(location.Lat, location.Lon, lat.Float64, lon.Float64)
}
func distance(lat1, lon1, lat2, lon2 float64) (float64, float64) {
	return geo1.To(lat1, lon1, lat2, lon2)
}
