package main

import (
	"fmt"
	ws "kanso-websockets"
	"time"
)

func main() {
	wsClient, wsErr := ws.WebSocket("http://localhost:8080/ping")
	if wsErr != nil {
		panic("ws client failed to create")
	}

	wsClient.RunCallback(func() {
		fmt.Println("WS Client connected, pinging...")
		tick := time.Tick(1000 * time.Millisecond)
		for range tick {
			fmt.Println("Pinging...")
			wsClient.Ping()
		}
	})
}
