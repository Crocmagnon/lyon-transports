package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"net/http"
)

type Station struct {
	Name             string `json:"name"`
	BikesAvailable   int    `json:"bikes_available"`
	DocksAvailable   int    `json:"docks_available"`
	AvailabilityCode int    `json:"availability_code"`
}

type stationInfo struct {
	Values []struct {
		AvailabilityCode    int    `json:"availabilitycode"`
		AvailableBikeStands int    `json:"available_bike_stands"`
		AvailableBikes      int    `json:"available_bikes"`
		Name                string `json:"name"`
		Number              int    `json:"number"`
	} `json:"values"`
}

var errStationNotFound = errors.New("station not found")

func getStation(ctx context.Context, client *http.Client, stationID int) (*Station, error) {
	var info stationInfo
	err := requests.URL("https://data.grandlyon.com/fr/datapusher/ws/rdata/jcd_jcdecaux.jcdvelov/all.json?maxfeatures=-1&start=1").
		Client(client).
		ToJSON(&info).
		Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying station info: %w", err)
	}

	station := Station{}

	for _, sInfo := range info.Values {
		if sInfo.Number == stationID {
			station.Name = sInfo.Name
			station.BikesAvailable = sInfo.AvailableBikes
			station.DocksAvailable = sInfo.AvailableBikeStands
			station.AvailabilityCode = sInfo.AvailabilityCode
			break
		}
	}

	if station.Name == "" {
		return nil, errStationNotFound
	}

	return &station, nil
}
