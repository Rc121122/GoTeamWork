package main

import (
	"embed"
	"flag"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	mode := flag.String("mode", "client", "Mode: 'host' for central-server host or 'client' for central-server client")
	flag.Parse()

	wapp := application.New(application.Options{
		Name:        "GOproject",
		Description: "Collaboration helper",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	service := NewApp(*mode)
	service.wailsApp = wapp
	wapp.RegisterService(application.NewService(service))

	wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "GOproject",
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	if err := wapp.Run(); err != nil {
		log.Fatalf("failed to run Wails app: %v", err)
	}
}
