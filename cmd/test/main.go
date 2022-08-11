package main

import (
	"fmt"
	"log"

	"github.com/petoc/gbfs"
)

func main() {
	c, err := gbfs.NewClient(gbfs.ClientOptions{
		AutoDiscoveryURL: "https://gbfs.citibikenyc.com/gbfs/gbfs.json",
		DefaultLanguage:  "en",
	})
	if err != nil {
		log.Fatal(err)
	}
	_ = gbfs.FeedFreeBikeStatusBike{}
	si := &gbfs.FeedStationStatus{}
	err = c.Get(si)
	if err != nil {
		log.Fatal(err)
	}
	// st := si.Data.Stations[600]
	fmt.Println(si.Data)
	for _, st := range si.Data.Stations {
		fmt.Println(*st.StationID, *st.NumEBikesAvailable, *st.NumEBikesAvailable)
	}
}
