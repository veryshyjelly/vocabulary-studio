package main

import (
	"log"
	"os"
	"strconv"

	"github.com/veryshyjelly/vocabulary-studio/frontend"
	"github.com/veryshyjelly/vocabulary-studio/internal/desktop"
	vocab "github.com/veryshyjelly/vocabulary-studio/sdk"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	_ = vocab.LoadDotEnvFile(".env.local")
	_ = vocab.LoadDotEnvFile(".env")

	var app *application.App

	hub, err := desktop.NewHub(desktop.HubOptions{
		EmitEvent: func(name string, data any) {
			if app != nil && app.Event != nil {
				app.Event.Emit(name, data)
			}
		},
	})
	if err != nil {
		log.Fatalf("initialize desktop hub: %v", err)
	}

	app = application.New(application.Options{
		Name:        "Vocabulary Studio",
		Description: "Desktop learning companion for Vocabulary.com",
		Services: []application.Service{
			application.NewService(hub.Auth),
			application.NewService(hub.Trainer),
			application.NewService(hub.Dictionary),
			application.NewService(hub.Lists),
			application.NewService(hub.Discovery),
			application.NewService(hub.Progress),
			application.NewService(hub.Settings),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(frontend.DistFS()),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			hub.Shutdown()
		},
		// Only used when built with `-tags server` (headless browser preview of the real app).
		Server: application.ServerOptions{
			Host: "localhost",
			Port: serverPort(),
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Vocabulary Studio",
		Width:            1320,
		Height:           860,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(250, 249, 245),
		URL:              "/",
	})
	window.Center()
	window.Show()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func serverPort() int {
	if p, err := strconv.Atoi(os.Getenv("VOCAB_SERVER_PORT")); err == nil && p > 0 {
		return p
	}
	return 8765
}
