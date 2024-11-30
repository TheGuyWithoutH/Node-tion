package unit

import (
	"Node-tion/backend/peer"
	"Node-tion/backend/peer/impl"
	"Node-tion/backend/transport"
	"Node-tion/backend/transport/channel"
	"Node-tion/backend/transport/udp"
)

var peerFac peer.Factory = impl.NewPeer

var channelFac transport.Factory = channel.NewTransport
var udpFac transport.Factory = udp.NewUDP
