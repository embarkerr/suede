package main

import (
	"fmt"

	"github.com/embarkerr/suede"
)

func main() {
	server, wsErr := suede.WebSocketServer(8080, "/chat")
	if wsErr != nil {
		panic("could not start server")
	}

	server.OnConnect = func() {
		server.Broadcast([]byte("New user joined the chat!"))
	}

	server.OnDisconnect = func() {
		server.Broadcast([]byte("User has left the chat"))
	}

	server.OnMessage = func(data []byte) {
		server.Broadcast(data)
	}

	fmt.Println("Server starting")
	server.Run()
}
