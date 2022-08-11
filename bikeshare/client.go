package bikeshare

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	spark "bitbucket.org/dtolpin/gosparkline"
	gbfsspec "github.com/Eraac/gbfs/spec/v2.0"
	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
	"golang.org/x/exp/slices"
	"jonwillia.ms/biketray/systems"
)

const pollInterval = 60 * time.Second
const sparklineLen = 60

type Client struct {
	nearbyResult systems.NearbyResult
	cancel       func()
	mgr          *Manager
}

func newClient(ctx context.Context, m *Manager, nearbyResult systems.NearbyResult) *Client {
	ctx, cancel := context.WithCancel(ctx)
	c := &Client{
		nearbyResult: nearbyResult,
		cancel:       cancel,
		mgr:          m,
	}
	go c.run(ctx)
	return c
}
func (c *Client) Close() {
	log.Println("close", c.nearbyResult.System.Name)
	c.cancel()
}

var geo1 = ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Meter, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingIsSymmetric)

func (c *Client) run(ctx context.Context) {
	locationC := c.mgr.geoMgr.Subscribe()
	defer c.mgr.geoMgr.Unsubscribe(locationC)
	log.Println("run")

	sparklines := make(map[string][]float64)

	location := <-locationC
	for ctx.Err() == nil {

		if ctx.Err() != nil {
			return
		}

		var ss gbfsspec.FeedStationStatus
		if err := c.nearbyResult.Client.Get(gbfsspec.FeedKeyStationStatus, &ss); err != nil {
			log.Println("FeedKeyStationStatus", err)
			continue
		}
		stationMap := make(map[string]gbfsspec.StationStatus, len(ss.Data.Stations))
		for _, st := range ss.Data.Stations {
			stationMap[st.StationID] = st
			frac := float64(st.NumBikesAvailable)
			sparklines[st.StationID] = append([]float64{frac}, sparklines[st.StationID]...)
			if len(sparklines[st.StationID]) > sparklineLen {
				sparklines[st.StationID] = sparklines[st.StationID][:sparklineLen]
			}
		}
		nextUpdate := time.Now().Add(pollInterval)

	NEXT_LOC:
		dist := func(s gbfsspec.StationInformation) float64 {
			lat, lon := location.Lat, location.Lon
			distance, _ := geo1.To(lat, lon, s.Latitude, s.Longitude)
			return distance
		}
		slices.SortFunc(c.nearbyResult.FeedStationInformation.Data.Stations, func(a, b gbfsspec.StationInformation) bool {
			return dist(a) < dist(b)
		})

		var output []string = make([]string, 0, len(c.nearbyResult.FeedStationInformation.Data.Stations))

		for _, s := range c.nearbyResult.FeedStationInformation.Data.Stations {
			lat, lon := location.Lat, location.Lon
			distance, bearing := geo1.To(lat, lon, s.Latitude, s.Longitude)

			statusStr := "?????"
			st, ok := stationMap[s.StationID]
			if ok {
				sl := string([]rune(spark.Line(append([]float64{0}, sparklines[s.StationID]...)))[1:])
				statusStr = fmt.Sprintf("%2.1d/%2.1d %s", st.NumBikesAvailable, s.Capacity, sl)
			}

			unit := "m"
			if distance > 10000 {
				unit = "km"
				distance /= 1000
			}

			str := fmt.Sprintf("%s (%4.1f%s %2s)\n%s", s.Name, distance, unit, direction(bearing), statusStr)
			output = append(output, str)
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		log.Println("client out", len(c.nearbyResult.FeedStationInformation.Data.Stations))
		c.mgr.clientResult(c.nearbyResult.System, output)

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(nextUpdate)):
		case location = <-locationC:
			fmt.Println("client loc")
			goto NEXT_LOC
		}

	}

}

func direction(bearing float64) string {
	const degrees = 360
	if bearing < 0 {
		bearing += degrees
	}
	dirs := []string{
		"N",
		"NE",
		"E",
		"SE",
		"S",
		"SW",
		"W",
		"NW",
	}
	dirSize := degrees / len(dirs)
	bearing -= float64(dirSize) / 2
	if bearing < 0 {
		bearing += degrees
	}
	idx := int(math.Round(bearing/float64(dirSize))) % len(dirs)
	return dirs[idx]
}
