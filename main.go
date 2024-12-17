package main

import (
	"embed"
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"path/filepath"

	"github.com/jackpal/gateway"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"Node-tion/backend/peer"
	"Node-tion/backend/peer/impl"
	"Node-tion/backend/registry/standard"
	"Node-tion/backend/storage/inmemory"
	"Node-tion/backend/transport/udp"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

// findInterfaceByIP returns the network interface and the specific IP address
// associated with the given IP.
func findInterfaceByIP(ip net.IP) (*net.Interface, net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}

	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, nil, err
		}

		for _, addr := range addrs {
			var ipNet *net.IPNet
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
			case *net.IPAddr:
				ipNet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
			}

			if ipNet != nil && ipNet.Contains(ip) {
				return &iface, ipNet.IP, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no interface found for IP: %s", ip.String())
}

func main() {

	var peerFactory = impl.NewPeer

	trans := udp.NewUDP()

	ip, err := gateway.DiscoverGateway()
	if err != nil {
		log.Fatalf("Error discovering gateway: %v", err)
	}

	_, ip, err = findInterfaceByIP(ip)
	if err != nil {
		log.Fatalf("Error finding interface: %v", err)
	}

	sock, err := trans.CreateSocket(ip.String() + ":0")
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
		AntiEntropyInterval: time.Second * 5,
		HeartbeatInterval:   time.Second * 1,
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
