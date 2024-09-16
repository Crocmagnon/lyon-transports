package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"net/http"
	"time"
)

type Options struct {
	Host              string `help:"Host to listen to" default:"127.0.0.1"`
	Port              int    `help:"Port to listen on" default:"8888"`
	GrandLyonUsername string `help:"Grand Lyon username" short:"u" required:"true"`
	GrandLyonPassword string `help:"Grand Lyon password" short:"p" required:"true"`
	CORSAllowedOrigin string `help:"CORS allowed origin"`
}

type statusOutput struct {
	Body struct {
		Status string `json:"status" example:"ok" doc:"API status"`
	}
}

type stopOutput struct {
	Body Passages
}

type velovOutput struct {
	Body Station
}

func addRoutes(api huma.API, glConfig GrandLyonConfig, now func() time.Time) {
	huma.Register(api, huma.Operation{
		OperationID: "healthcheck",
		Method:      http.MethodGet,
		Path:        "/",
		Summary:     "Get API status",
		Description: "Get the status of the API.",
	}, func(ctx context.Context, input *struct{}) (*statusOutput, error) {
		resp := &statusOutput{}
		resp.Body.Status = "ok"
		return resp, nil
	})

	huma.Get(api, "/tcl/stop/{stopID}", func(ctx context.Context, input *struct {
		StopID int `path:"stopID" doc:"Stop id to monitor. Can be obtained using https://data.grandlyon.com/portail/fr/jeux-de-donnees/points-arret-reseau-transports-commun-lyonnais/donnees"`
	}) (*stopOutput, error) {
		passages, err := getPassages(ctx, glConfig, now, input.StopID)
		if errors.Is(err, errNoPassageFound) {
			return nil, huma.NewError(http.StatusNotFound, "no passage found")
		}

		if err != nil {
			return nil, err
		}

		return &stopOutput{Body: *passages}, nil
	})

	huma.Get(api, "/velov/station/{stationID}", func(ctx context.Context, input *struct {
		StationID int `path:"stationID" doc:"Station id to monitor. Can be obtained using https://data.grandlyon.com/portail/fr/jeux-de-donnees/stations-velo-v-metropole-lyon/donnees"`
	}) (*velovOutput, error) {
		station, err := getStation(ctx, glConfig.Client, input.StationID)
		if errors.Is(err, errStationNotFound) {
			return nil, huma.NewError(http.StatusNotFound, "station not found")
		}

		if err != nil {
			return nil, err
		}

		return &velovOutput{Body: *station}, nil
	})
}

func main() {
	// Create a CLI app which takes a port option.
	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		// Create a new router & API
		router := chi.NewRouter()
		router.Use(cors.Handler(cors.Options{
			AllowedOrigins: []string{options.CORSAllowedOrigin},
		}))

		api := humachi.New(router, huma.DefaultConfig("My API", "1.0.0"))
		server := http.Server{
			Addr:    fmt.Sprintf("%s:%d", options.Host, options.Port),
			Handler: router,
		}

		glConfig := GrandLyonConfig{
			Username: options.GrandLyonUsername,
			Password: options.GrandLyonPassword,
		}

		addRoutes(api, glConfig, time.Now)

		hooks.OnStart(func() {
			fmt.Printf("Starting server on %s...\n", server.Addr)
			if err := server.ListenAndServe(); err != nil {
				fmt.Printf("Error running server: %s\n", err)
			}
		})

		hooks.OnStop(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				fmt.Printf("Error shutting down server: %s\n", err)
			}
		})
	})

	cli.Run()
}
