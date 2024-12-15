package main

import (
	"Node-tion/backend/peer"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"Node-tion/backend/peer/impl"
	"Node-tion/backend/types"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {

	var peerFactory = impl.NewPeer

	conf := peer.Configuration{
		Socket:              nil,
		MessageRegistry:     nil,
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
		Storage:    nil,
		TotalPeers: 0,
		PaxosThreshold: func(u uint) int {
			return int(u/2 + 1)
		},
		PaxosID:            0,
		PaxosProposerRetry: 0,
	}

	node := peerFactory(conf)

	app := &App{}

	// Create application with options
	err := wails.Run(&options.App{
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
		LogLevel:         logger.DEBUG,
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		WindowStartState: options.Normal,

		Bind: []interface{}{
			app,
			node,
			//These are redundant but necessary for the bindings for type infratsructure compatibility
			&types.CRDTOperationsMessage{},
			&types.CRDTOperation{},
			&types.CRDTRemoveBlock{},
			&types.CRDTAddBlock{},
			&types.CRDTUpdateBlock{},
			&types.CRDTInsertChar{},
			&types.CRDTDeleteChar{},
			&types.CRDTAddMark{},
			&types.CRDTRemoveMark{},
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
