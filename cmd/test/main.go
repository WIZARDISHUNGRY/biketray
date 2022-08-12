package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/petoc/gbfs"
)

type FeedStationInformationStation struct {
	VehicleTypeCapacity map[*ID]int64 `json:"vehicle_type_capacity,omitempty,string"`
}

type ID string

func main() {

	const str1 = `
	{	
		"vehicle_type_capacity":{"1234":"1234}
	}
	`
	var fsi FeedStationInformationStation
	err := json.Unmarshal([]byte(str1), &fsi)

	c, err := gbfs.NewClient(gbfs.ClientOptions{
		AutoDiscoveryURL: "https://stables.donkey.bike/api/public/gbfs/2/donkey_barcelona/gbfs.json",
		DefaultLanguage:  "en",
	})
	if err != nil {
		log.Fatal(err)
	}
	_ = gbfs.FeedFreeBikeStatusBike{}
	si := &gbfs.FeedStationInformation{}
	err = c.Get(si)
	if err != nil {
		log.Fatal(err)
	}
	// st := si.Data.Stations[600]
	fmt.Println(si.Data)
	for _, st := range si.Data.Stations {
		fmt.Println(*st.StationID)
	}
}
