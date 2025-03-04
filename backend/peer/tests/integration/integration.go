package integration

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"Node-tion/backend/internal/binnode"
	"Node-tion/backend/peer"
	"Node-tion/backend/peer/impl"
	"Node-tion/backend/transport"
	"Node-tion/backend/transport/proxy"
	"Node-tion/backend/transport/udp"
)

var studentFac peer.Factory = impl.NewPeer
var referenceFac peer.Factory

func init() {
	path := getPath()
	referenceFac = binnode.GetBinnodeFac(path)
}

// getPath returns the path in the PEER_BIN_PATH variable if set, otherwise a
// path of form ./node.<OS>.<ARCH>. For example "./node.darwin.amd64". It panics
// in case of an unsupported OS/ARCH.
func getPath() string {
	path := os.Getenv("PEER_BIN_PATH")
	if path != "" {
		return path
	}

	bin := fmt.Sprintf("./node.%s.%s", runtime.GOOS, runtime.GOARCH)

	// check if the binary exists
	_, err := os.Stat(bin)
	if err != nil {
		panic(fmt.Sprintf("unsupported OS/architecture combination: %v/%v",
			runtime.GOOS, runtime.GOARCH))
	}

	return bin
}

// skipIfWIndows will skip a test if the detected Operating System is Windows
func skipIfWIndows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("The Windows network stack makes this test fail erratically - please run this on a Linux system")
	}
}

var udpFac transport.Factory = udp.NewUDP
var proxyFac transport.Factory = proxy.NewProxy
