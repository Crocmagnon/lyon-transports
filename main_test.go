package main

import (
	"encoding/json"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/jarcoal/httpmock"
	"gotest.tools/v3/assert"
	"net/http"
	"testing"
	"time"
)

func TestGetStatus(t *testing.T) {
	_, api := humatest.New(t)

	addRoutes(api, GrandLyonConfig{}, nil)

	resp := api.Get("/")
	assert.Equal(t, resp.Code, http.StatusOK)
}

func TestGetStop(t *testing.T) {
	_, api := humatest.New(t)

	transport := httpmock.NewMockTransport()
	client := &http.Client{
		Transport: transport,
	}
	config := GrandLyonConfig{
		Client: client,
	}
	now := func() time.Time {
		location, err := time.LoadLocation("Europe/Paris")
		if err != nil {
			t.Errorf("Could not load Europe/Paris")
		}

		return time.Date(2022, 8, 25, 8, 23, 10, 0, location)
	}

	transport.RegisterResponder(http.MethodGet,
		"https://download.data.grandlyon.com/ws/rdata/tcl_sytral.tclarret/all.json?maxfeatures=-1",
		httpmock.NewBytesResponder(http.StatusOK, httpmock.File("./testdata/stops.json").Bytes()))
	transport.RegisterResponder(http.MethodGet,
		"https://download.data.grandlyon.com/ws/rdata/tcl_sytral.tclpassagearret/all.json?maxfeatures=-1",
		httpmock.NewBytesResponder(http.StatusOK, httpmock.File("./testdata/passages.json").Bytes()))

	addRoutes(api, config, now)

	t.Run("stop exists", func(t *testing.T) {
		resp := api.Get("/tcl/stop/290")
		assert.Equal(t, resp.Code, http.StatusOK)

		var passages Passages
		err := json.Unmarshal(resp.Body.Bytes(), &passages)
		assert.NilError(t, err)

		assert.DeepEqual(t, passages, Passages{
			Passages: []Passage{
				{
					Ligne:  "37",
					Delays: []string{"Passé", "Proche"},
					Destination: Stop{
						ID:   46642,
						Name: "Charpennes",
					},
				},
				{
					Ligne:  "C17",
					Delays: []string{"Proche", "10 min"},
					Destination: Stop{
						ID:   46644,
						Name: "Charpennes",
					},
				},
			},
			Stop: Stop{
				ID:   290,
				Name: "Buers - Salengro",
			},
		})
	})
}

func TestGetVelovStation(t *testing.T) {
	_, api := humatest.New(t)

	transport := httpmock.NewMockTransport()
	client := &http.Client{
		Transport: transport,
	}
	config := GrandLyonConfig{
		Client: client,
	}

	transport.RegisterResponder(http.MethodGet,
		"https://data.grandlyon.com/fr/datapusher/ws/rdata/jcd_jcdecaux.jcdvelov/all.json?maxfeatures=-1&start=1",
		httpmock.NewBytesResponder(http.StatusOK, httpmock.File("./testdata/station_info.json").Bytes()))

	addRoutes(api, config, time.Now)

	t.Run("station not found", func(t *testing.T) {
		resp := api.Get("/velov/station/0")
		assert.Equal(t, resp.Code, http.StatusNotFound)
	})

	t.Run("station exists", func(t *testing.T) {
		resp := api.Get("/velov/station/10039")
		assert.Equal(t, resp.Code, http.StatusOK)

		var station Station
		err := json.Unmarshal(resp.Body.Bytes(), &station)
		assert.NilError(t, err)

		assert.DeepEqual(t, station, Station{
			Name:             "Bouvier",
			BikesAvailable:   9,
			DocksAvailable:   7,
			AvailabilityCode: 1,
		})
	})
}
