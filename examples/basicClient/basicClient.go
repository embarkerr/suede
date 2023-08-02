package main

import (
	"fmt"
	ws "kanso-websockets"
	"sync"
)

func main() {
	wsClient, err := ws.WebSocket("http://172.29.128.1:8080/chat",
		func() { fmt.Println("Connect callback") },
		func() { fmt.Println("Disconnect callback") },
		func(data []byte) { fmt.Printf("Data: %s\n", data) },
	)

	if err != nil {
		panic("Could not create WebSocket client")
	}

	fmt.Println("Kanso WebSocket client connected")
	wsClient.Connect()

	var wg sync.WaitGroup
	wsClient.Read(&wg)

	wsClient.Send([]byte("Hello from Kanso WebSocket client!"))

	wg.Wait()
}
