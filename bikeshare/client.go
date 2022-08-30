package bikeshare

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	spark "bitbucket.org/dtolpin/gosparkline"
	"github.com/petoc/gbfs"
	"golang.org/x/exp/slices"
	"jonwillia.ms/biketray/geo"
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

func (c *Client) run(ctx context.Context) {
	locationC := c.mgr.geoMgr.Subscribe(false)
	defer c.mgr.geoMgr.Unsubscribe(locationC)
	log.Println("run")

	sparklines := make(map[gbfs.ID][]float64)

	hasStations := c.nearbyResult.FeedStationInformation != nil && c.nearbyResult.FeedStationInformation.Data != nil &&
		len(c.nearbyResult.FeedStationInformation.Data.Stations) > 0
	hasBikes := c.nearbyResult.FeedFreeBikeStatus != nil && c.nearbyResult.FeedFreeBikeStatus.Data != nil &&
		len(c.nearbyResult.FeedFreeBikeStatus.Data.Bikes) > 0

	location := <-locationC
	const limitNearbyStation = 20
	const limitNearbyBike = 100

LOOP:
	for ctx.Err() == nil {

		var ss gbfs.FeedStationStatus
		var stationMap map[gbfs.ID]*gbfs.FeedStationStatusStation
		var wantedLen int

		if hasStations {
			if err := c.nearbyResult.Client.Get(&ss); err != nil {
				log.Println("FeedStationStatus", err)
				continue LOOP
			}
			stationMap = make(map[gbfs.ID]*gbfs.FeedStationStatusStation, len(ss.Data.Stations))
			for _, st := range ss.Data.Stations {
				stationMap[*st.StationID] = st
				frac := float64(*st.NumBikesAvailable)
				sparklines[*st.StationID] = append([]float64{frac}, sparklines[*st.StationID]...)
				if len(sparklines[*st.StationID]) > sparklineLen {
					sparklines[*st.StationID] = sparklines[*st.StationID][:sparklineLen]
				}
			}
			wantedLen += len(c.nearbyResult.FeedStationInformation.Data.Stations)
		}
		var fbs gbfs.FeedFreeBikeStatus
		if hasBikes {
			if err := c.nearbyResult.Client.Get(&fbs); err != nil {
				log.Println("FeedFreeBikeStatus", err)
				continue LOOP
			}
			wantedLen += len(fbs.Data.Bikes)
		}

		nextUpdate := time.Now().Add(pollInterval)

	NEXT_LOC:
		dist := func(lat, lon *gbfs.Coordinate) float64 {
			d, _ := geo.Distance(location, lat, lon)
			return d
		}

		type outputTemp struct {
			distance float64
			datum    Datum
		}

		var output = make([]outputTemp, 0, wantedLen)

		if hasStations {
			slices.SortFunc(c.nearbyResult.FeedStationInformation.Data.Stations, func(a, b *gbfs.FeedStationInformationStation) bool {
				return dist(a.Lat, a.Lon) < dist(b.Lat, b.Lon)
			})

			for _, s := range constrain(c.nearbyResult.FeedStationInformation.Data.Stations, limitNearbyStation) {
				distance, bearing := geo.Distance(location, s.Lat, s.Lon)

				statusStr := "?????"
				st, ok := stationMap[*s.StationID]
				var sl string
				if ok && st.NumBikesAvailable != nil && s.Capacity != nil {

					supStrs := []string{}
					for _, bike := range []struct {
						Label     string
						Available *int64
					}{
						// {"🚲", st.NumBikesAvailable},
						{"🛵", st.NumScootersAvailable},
						{"⚡", st.NumEBikesAvailable},
					} {
						if bike.Available == nil {
							continue
						}

						supStrs = append(supStrs, fmt.Sprintf("%d%s", *bike.Available, bike.Label))
					}
					var supStr string
					if len(supStrs) > 0 {
						supStr = fmt.Sprintf(" (%s)", strings.Join(supStrs, " "))
					}

					sl = string([]rune(spark.Line(append([]float64{0}, sparklines[*s.StationID]...)))[1:])
					statusStr = fmt.Sprintf("%2.1d/%2.1d%s", *st.NumBikesAvailable, *s.Capacity, supStr)
				}

				unit := "m"
				if distance > 10000 {
					unit = "km"
					distance /= 1000
				}

				// TODO: Darwin doesn't like newlines
				str := fmt.Sprintf("%s (%4.1f%s %2s)\r\n%s", *s.Name, distance, unit, direction(bearing), statusStr)
				output = append(output, outputTemp{
					distance: distance,
					datum: Datum{
						Label:     str,
						Sparkline: sl,
						LocationInfo: geo.LocationInfo{
							Lat: s.Lat.Float64,
							Lon: s.Lon.Float64,
						},
					},
				})
			}
		}

		if hasBikes {
			slices.SortFunc(fbs.Data.Bikes, func(a, b *gbfs.FeedFreeBikeStatusBike) bool {
				return (dist(a.Lat, a.Lon) < dist(b.Lat, b.Lon)) &&
					!(a.IsReserved != nil && bool(*a.IsReserved)) &&
					!(a.IsDisabled != nil && bool(*a.IsDisabled))
			})
			for _, b := range constrain(fbs.Data.Bikes, limitNearbyBike) {
				distance, bearing := geo.Distance(location, b.Lat, b.Lon)
				unit := "m"
				if distance > 10000 {
					unit = "km"
					distance /= 1000
				}

				name := "🚲"
				if b.VehicleTypeID != nil {
					name = string(*b.VehicleTypeID)
				}

				str := fmt.Sprintf("%s (%4.1f%s %2s)", name, distance, unit, direction(bearing))
				output = append(output, outputTemp{
					distance: distance,
					datum: Datum{
						Label: str,
						LocationInfo: geo.LocationInfo{
							Lat: b.Lat.Float64,
							Lon: b.Lon.Float64,
						},
					},
				})
			}
		}

		slices.SortFunc(output, func(a, b outputTemp) bool { return a.distance < b.distance })
		lines := make([]Datum, 0, len(output))
		for _, ot := range output {
			lines = append(lines, ot.datum)
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		log.Println("client out", len(lines))
		c.mgr.clientResult(c.nearbyResult.System, lines)

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

func constrain[E any](x []E, limit int) []E {
	if len(x) < limit {
		limit = len(x)
	}
	return x[:limit]
}
