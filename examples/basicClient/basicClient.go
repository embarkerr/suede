package main

import (
	"fmt"
	ws "kanso-websockets"
	"sync"
)

func main() {
	wsClient, wsErr := ws.WebSocket("http://localhost:8080/chat")
	if wsErr != nil {
		fmt.Println("Unable to connect WebSocket")
		return
	}

	wsClient.OnConnect = func() {
		fmt.Println("Kanso WebSocket client connected")
		wsClient.Send([]byte("Hello I am a new client"))
	}

	wsClient.OnDisconnect = func() {
		fmt.Println("Disconnect callback")
	}

	wsClient.OnMessage = func(data []byte) {
		fmt.Printf("Data = %s\n", data)
	}

	// Once created, the client can be started in 3 different ways:
	// Run - simplest and least flexible
	wsClient.Run()

	// RunCallback - same as run, but executes a callback while the client is connected
	wsClient.RunCallback(func() {
		fmt.Println("Kanso WebSocket client connected")
		fmt.Printf("Connected to %s\n", "/chat")
	})

	// Connect - most flexible as it returns control to the caller.
	// However, it requires an externally handled WaitGroup
	var wg sync.WaitGroup
	connectErr := wsClient.Connect(&wg)
	if connectErr != nil {
		panic("WebSocket client failed to connect")
	}

	// add any additional logic here (before the wg.Wait() call), which will be executed as normal
	// for example:
	fmt.Println("Kanso WebSocket client connected")
	wsClient.Send([]byte("Hello from Kanso WebSocket client!"))

	wg.Wait()
	// End of 'Connect' example
}
