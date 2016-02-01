package main

import (
	"fmt"
	// "github.com/gopherjs/gopherjs/js"
	"github.com/gopherjs/websocket"
	"honnef.co/go/js/dom"
	"net/rpc"
)

var ws *websocket.Conn
var rpcClient *rpc.Client

func getWSBaseURL() string {
	document := dom.GetWindow().Document().(dom.HTMLDocument)
	location := document.Location()

	wsProtocol := "ws"
	if location.Protocol == "https:" {
		wsProtocol = "wss"
	}
	return fmt.Sprintf("%s://%s:%s/ws", wsProtocol, location.Hostname, location.Port)
}

func websocketInit() *websocket.Conn {
	wsBaseURL := getWSBaseURL()
	wss, err := websocket.Dial(wsBaseURL)
	if err != nil {
		print("failed to open websocket")
	}
	ws = wss
	rpcClient = rpc.NewClient(ws)

	// Now we can spawn a pinger against the backetd
	go sendPings(55000)

	// And run a server at this end that accepts pings
	//go PingServer(ws)

	return wss
}
