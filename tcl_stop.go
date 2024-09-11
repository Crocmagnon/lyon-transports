package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/carlmjohnson/requests"
	"net/http"
	"regexp"
	"slices"
	"time"
)

type GrandLyonConfig struct {
	Username string
	Password string
	Client   *http.Client
}

type Stop struct {
	ID   int    `json:"id" example:"290"`
	Name string `json:"name" example:"Grange Blanche"`
}

func NewStop(tclStop TCLStop) Stop {
	return Stop{
		ID:   tclStop.Id,
		Name: tclStop.Nom,
	}
}

type Passage struct {
	Ligne       string   `json:"ligne" example:"49A"`
	Delays      []string `json:"delais" example:"53 min"`
	Destination Stop     `json:"destination"`
}

type Passages struct {
	Passages []Passage `json:"passages"`
	Stop     Stop      `json:"stop"`
}

type delay int

func (d delay) String() string {
	switch d {
	case -2:
		return "Passé"
	case -1:
		return "Proche"
	default:
		return fmt.Sprintf("%d min", d)
	}
}

var errNoPassageFound = errors.New("no passage found")

func getPassages(ctx context.Context, config GrandLyonConfig, now func() time.Time, stopID int) (*Passages, error) {
	client := config.Client
	if client == nil {
		client = &http.Client{}
	}

	var tclPassages TCLPassages

	err := requests.URL("https://download.data.grandlyon.com/ws/rdata/tcl_sytral.tclpassagearret/all.json?maxfeatures=-1").
		Client(client).
		BasicAuth(config.Username, config.Password).
		ToJSON(&tclPassages).
		Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching passages: %w", err)
	}

	type passageKey struct {
		line        string
		destination int
	}

	stops := map[int]TCLStop{stopID: {}}
	passages := make(map[passageKey][]delay)
	for _, passage := range tclPassages.Values {
		if passage.Id != stopID || passage.Type != "E" {
			continue
		}
		// Remove letter suffix to group by commercial line name
		line := regexp.MustCompile("[A-Z]$").ReplaceAllString(passage.Ligne, "")
		destination := passage.Idtarretdestination
		stops[destination] = TCLStop{}
		key := passageKey{line: line, destination: destination}
		delays := passages[key]
		delays = append(delays, getDelay(passage.Heurepassage, now()))
		passages[key] = delays
	}

	if len(passages) == 0 {
		return nil, errNoPassageFound
	}

	var tclStops TCLStops

	err = requests.URL("https://download.data.grandlyon.com/ws/rdata/tcl_sytral.tclarret/all.json?maxfeatures=-1").
		Client(client).
		BasicAuth(config.Username, config.Password).
		ToJSON(&tclStops).
		Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching stops: %w", err)
	}

	updated := 0

	for _, stop := range tclStops.Values {
		if _, stopToUpdate := stops[stop.Id]; stopToUpdate {
			stops[stop.Id] = stop
			updated++
		}

		if updated == len(stops) {
			break
		}
	}

	resPassages := make([]Passage, 0, len(passages))

	for key, delays := range passages {
		slices.Sort(delays)
		delaysStr := make([]string, len(delays))
		for i, delay := range delays {
			delaysStr[i] = delay.String()
		}
		resPassages = append(resPassages, Passage{
			Ligne:       key.line,
			Delays:      delaysStr,
			Destination: NewStop(stops[key.destination]),
		})
	}

	slices.SortFunc(resPassages, func(a, b Passage) int {
		if a.Ligne < b.Ligne {
			return -1
		}
		return 1
	})

	return &Passages{
		Passages: resPassages,
		Stop:     NewStop(stops[stopID]),
	}, nil
}

func getDelay(heurepassage string, now time.Time) delay {
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		location = time.UTC
	}

	passage, err := time.ParseInLocation("2006-01-02 15:04:05", heurepassage, location)
	if err != nil {
		return delay(-1)
	}

	if passage.Before(now) {
		return delay(-2)
	}

	dur := passage.Sub(now)
	minutes := int(dur.Minutes())

	if minutes <= 0 {
		return delay(-1)
	}

	return delay(minutes)
}

type TCLPassage struct {
	Coursetheorique     string `json:"coursetheorique"`
	Delaipassage        string `json:"delaipassage"`
	Direction           string `json:"direction"`
	Gid                 int    `json:"gid"`
	Heurepassage        string `json:"heurepassage"`
	Id                  int    `json:"id"`
	Idtarretdestination int    `json:"idtarretdestination"`
	LastUpdateFme       string `json:"last_update_fme"`
	Ligne               string `json:"ligne"`
	Type                string `json:"type"`
}

type TCLPassages struct {
	Fields     []string     `json:"fields"`
	LayerName  string       `json:"layer_name"`
	NbResults  int          `json:"nb_results"`
	TableAlias interface{}  `json:"table_alias"`
	TableHref  string       `json:"table_href"`
	Values     []TCLPassage `json:"values"`
}

type TCLStop struct {
	Adresse              string  `json:"adresse"`
	Ascenseur            bool    `json:"ascenseur"`
	Commune              *string `json:"commune"`
	Desserte             string  `json:"desserte"`
	Escalator            bool    `json:"escalator"`
	Gid                  int     `json:"gid"`
	Id                   int     `json:"id"`
	Insee                *string `json:"insee"`
	LastUpdate           string  `json:"last_update"`
	LastUpdateFme        string  `json:"last_update_fme"`
	Lat                  float64 `json:"lat"`
	LocaliseFaceAAdresse bool    `json:"localise_face_a_adresse"`
	Lon                  float64 `json:"lon"`
	Nom                  string  `json:"nom"`
	Pmr                  bool    `json:"pmr"`
}

type TCLStops struct {
	Fields     []string    `json:"fields"`
	LayerName  string      `json:"layer_name"`
	NbResults  int         `json:"nb_results"`
	TableAlias interface{} `json:"table_alias"`
	TableHref  string      `json:"table_href"`
	Values     []TCLStop   `json:"values"`
}
