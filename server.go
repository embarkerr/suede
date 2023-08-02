package kansowebsockets

import (
	"fmt"
	"net/http"
	"sync"
)

type wsserver struct {
	host         string
	path         string
	clients      []wsclient
	onConnect    func()
	onDisconnect func()
	onMessage    func([]byte)
}

func WebSocketServer(port, path string, onConnect, onDisconnect func(), onMessage func([]byte)) (*wsserver, error) {
	wsServer := &wsserver{
		host:         port,
		path:         path,
		onConnect:    onConnect,
		onDisconnect: onDisconnect,
		onMessage:    onMessage,
	}

	return wsServer, nil
}

func (wsServer *wsserver) Start() {
	var wg sync.WaitGroup
	http.HandleFunc(wsServer.path, runServer)

	wg.Add(1)

	go http.ListenAndServe(":"+wsServer.host, nil)

	wg.Wait()
}

func runServer(w http.ResponseWriter, request *http.Request) {
	if request.Header.Get("Upgrade") != "" {
		wsKey := request.Header.Get("Sec-WebSocket-Key")
		wsAccept := GenerateWSAccept(wsKey)

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}

		connection, readWriter, _ := hijacker.Hijack()
		defer connection.Close()

		var content []byte
		content = append(content, "HTTP/1.1 101 Switching Protocols\r\n"...)
		content = append(content, "Upgrade: websocket\r\n"...)
		content = append(content, "Connection: Upgrade\r\n"...)
		content = append(content, fmt.Sprintf("Sec-WebSocket-Accept: %s", wsAccept)...)
		content = append(content, "\r\n\r\n"...)
		connection.Write(content)

		for true {
			b, err := readWriter.Reader.ReadByte()
			if err != nil {
				// fmt.Printf("No byte error: %s\n", err)
				continue
			}

			opCode := b & 0b00001111
			if opCode == 0x8 {
				break
			}

			bs := readWriter.Reader.Buffered()

			mask := false
			mask_start := 2
			mask_value := []byte{}
			payload_len := 0
			data_index := 0
			message := []byte{}

			for i := 1; i < bs+1; i++ {
				data, err := readWriter.Reader.ReadByte()
				if err != nil {
					fmt.Printf("No byte error: %s", err)
					continue
				}

				if i == 1 {
					mask = (data & 0b10000000) > 0
					payload_len = int(data & 0b01111111)

					if mask {
						if payload_len == 126 {
							mask_start = 4
						} else if payload_len == 127 {
							mask_start = 8
						}
					}
				}

				if i >= mask_start && i < mask_start+4 {
					mask_value = append(mask_value, data)
				}

				if i >= mask_start+4 {
					data_byte := data ^ mask_value[data_index%4]
					message = append(message, data_byte)
					data_index++
				}
			}

			fmt.Printf("Message = %s\n", message)

			responseMessage := []byte("I hear you loud and clear")
			responsePayload := []byte{}
			responsePayload = append(responsePayload, 0x81)
			responsePayload = append(responsePayload, 0x19)
			responsePayload = append(responsePayload, responseMessage...)
			connection.Write(responsePayload)
		}
	}

	fmt.Println("Client disconnected")
}

func (wsServer *wsserver) Send() {

}

func (wsServer *wsserver) Broadcast() {

}

func (wsServer *wsserver) Close() {

}
