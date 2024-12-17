package main

import (
	"embed"
	"log"
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	
	"Node-tion/backend/peer"
	"Node-tion/backend/peer/impl"
	"Node-tion/backend/transport/udp"
	"Node-tion/backend/registry/standard"
	"Node-tion/backend/storage/inmemory"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {

	var peerFactory = impl.NewPeer

	trans := udp.NewUDP()

	sock, err := trans.CreateSocket("127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
		return
	}

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("socketaddress_%d", os.Getpid()))

	err = os.WriteFile(socketPath, []byte(sock.GetAddress()), os.ModePerm)
	if err != nil {
		log.Fatal(err)
		return
	}

	storage := inmemory.NewPersistency()

	conf := peer.Configuration{
		Socket:              sock,
		MessageRegistry:     standard.NewRegistry(),
		AntiEntropyInterval: 0,
		HeartbeatInterval:   0,
		AckTimeout:          3,
		ContinueMongering:   0.5,
		ChunkSize:           8192,
		BackoffDataRequest: peer.Backoff{
			Initial: 2,
			Factor:  2,
			Retry:   5,
		},
		Storage:    storage,
		TotalPeers: 0,
		PaxosThreshold: func(u uint) int {
			return int(u/2 + 1)
		},
		PaxosID:            0,
		PaxosProposerRetry: 0,
	}

	node := peerFactory(conf)

	app := &App{
		node: node,
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:             "Node-tion",
		Width:             1024,
		Height:            768,
		MinWidth:          1024,
		MinHeight:         768,
		MaxWidth:          1280,
		MaxHeight:         800,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:             nil,
		Logger:           nil,
		LogLevel:         logger.ERROR,
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		WindowStartState: options.Normal,

		Bind: []interface{}{
			app,
			node,
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			// DisableFramelessWindowDecorations: false,
			WebviewUserDataPath: "",
			ZoomFactor:          1.0,
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Node-tion",
				Message: "",
				Icon:    icon,
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
