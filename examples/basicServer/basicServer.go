package main

import (
	"fmt"
	ws "kanso-websockets"
)

func main() {
	wsServer, err := ws.WebSocketServer("8080", "/chat",
		func() { fmt.Println("Connect callback") },
		func() { fmt.Println("Disconnect callback") },
		func(data []byte) { fmt.Printf("Data: %s\n", data) },
	)

	if err != nil {
		panic("Could not create WebSocket server")
	}

	fmt.Println("Kanso WebSocket Server running")
	fmt.Printf("Port: %s\tPath: %s", ":8080", "/chat\n")

	wsServer.Start()
}
