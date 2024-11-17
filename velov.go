package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"net/http"
	"strings"
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
	var (
		info        stationInfo
		errResponse string
	)

	err := requests.URL("https://data.grandlyon.com/fr/datapusher/ws/rdata/jcd_jcdecaux.jcdvelov/all.json?maxfeatures=-1&start=1").
		AddValidator(requests.ValidatorHandler(requests.DefaultValidator, requests.ToString(&errResponse))).
		Client(client).
		ToJSON(&info).
		Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying station info: %w (%v)", err, errResponse)
	}

	station := Station{}

	for _, sInfo := range info.Values {
		if sInfo.Number == stationID {
			station.Name = formatName(sInfo.Name)
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

func formatName(name string) string {
	nameParts := strings.SplitN(name, " - ", 2)
	if len(nameParts) >= 2 {
		name = nameParts[1]
	}
	return cases.Title(language.French).String(name)
}
