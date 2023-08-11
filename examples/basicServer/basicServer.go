package main

import (
	"fmt"
	ws "github.com/matt-bourke/kanso-websockets"
	"sync"
)

func main() {
	wsServer, wsErr := ws.WebSocketServer(8080, "/chat")
	if wsErr != nil {
		panic("Could not create WebSocket server")
	}

	wsServer.OnConnect = func() {
		fmt.Println("Client connected")
	}

	wsServer.OnDisconnect = func() {
		fmt.Println("Client disconnected")
	}

	wsServer.OnMessage = func(data []byte) {
		fmt.Printf("Message = %s\n", data)
		wsServer.Broadcast([]byte("broadcasting..."))
	}

	// Once created, the server can be started in 3 different ways:
	// Run - simplest and least flexible
	wsServer.Run()

	// RunCallback - same as run, but executes a callback while the server is active
	wsServer.RunCallback(func() {
		fmt.Println("Kanso WebSocket server running")
		fmt.Printf("Port: %s\tPath: %s\n", fmt.Sprintf("%d", wsServer.Host), wsServer.Path)
		// add any additional logic here
	})

	// Start - most flexible as it returns control to caller.
	// However, it requires an externally handled WaitGroup
	var wg sync.WaitGroup
	wsServer.Start(&wg)

	// add any additional logic here, which will be executed as normal program
	// for example:
	fmt.Println("Kanso WebSocket server running")
	fmt.Printf("Port: %s\tPath: %s\n", fmt.Sprintf("%d", wsServer.Host), wsServer.Path)

	wg.Wait()
	// End of `Start` example
}
