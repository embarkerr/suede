package kansowebsockets

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"sync"
)

type WSServerError struct {
	message string
}

func (err *WSServerError) Error() string {
	return err.message
}

type wsserver struct {
	Host         uint16
	Path         string
	OnConnect    func()
	OnDisconnect func()
	OnMessage    func([]byte)
	active       bool
	clients      []*net.Conn
}

func WebSocketServer(port uint16, path string) (*wsserver, error) {
	wsServer := &wsserver{
		Host:   port,
		Path:   path,
		active: false,
	}

	return wsServer, nil
}

// Start spins up the WebSocket server entry point in a new goroutine, returning control to the
// caller. A sync.WaitGroup is required and must be handled by the caller in the calling function.
// If wg.Wait() is not called in the calling function, the WebSocket server will exit immediately.
//
// If the caller does not need to regain control, consider calling Run or RunCallback instead.
func (wsServer *wsserver) Start(wg *sync.WaitGroup) {
	http.HandleFunc(wsServer.Path, wsServer.runServer)
	wg.Add(1)
	go http.ListenAndServe(":"+fmt.Sprintf("%d", wsServer.Host), nil)
	wsServer.active = true
}

// RunCallback spins up the WebSocket server and runs the handler function passed as an argument.
// RunCallback does not return control to the caller until the WebSocket server is shutdown.
//
// If the caller needs to regain control while the WebSocket server is active, consider calling
// Start instead, and managing a sync.WaitGroup manually.
func (wsServer *wsserver) RunCallback(handler func()) {
	var wg sync.WaitGroup
	wsServer.Start(&wg)

	if handler != nil {
		handler()
	}

	wg.Wait()
}

// Run is an alias for RunCallback(nil). It is used to spin up the WebSocket Server which does not
// have any handler behaviour. Run does not return control to the caller until the WebSocket server
// is shutdown.
//
// If the caller needs to regain control while the WebSocket server is active, consider calling
// Start instead, and managing a sync.WaitGroup manually.
func (wsServer *wsserver) Run() {
	wsServer.RunCallback(nil)
}

func (wsServer *wsserver) IsActive() bool {
	return wsServer.active
}

func (wsServer *wsserver) Clients() []*net.Conn {
	return wsServer.clients
}

func (wsServer *wsserver) runServer(res http.ResponseWriter, req *http.Request) {
	connection, connectionErr := wsServer.handleConnection(res, req)
	if connectionErr != nil {
		panic("Connection failed")
	}

	wsServer.readFromConnection(connection)

	closeErr := (*connection).Close()
	if closeErr != nil {
		panic("Failed to close connection")
	}

	for i := range wsServer.clients {
		if wsServer.clients[i] == connection {
			wsServer.clients = append(wsServer.clients[:i], wsServer.clients[i+1:]...)
			break
		}
	}
	connection = nil

	if wsServer.OnDisconnect != nil {
		wsServer.OnDisconnect()
	}
}

func (wsServer *wsserver) handleConnection(res http.ResponseWriter, req *http.Request) (*net.Conn, error) {
	if req.Header.Get("Upgrade") != "websocket" {
		return nil, &WSServerError{message: "Request header not requesting websocket upgrade"}
	}

	wsKey := req.Header.Get("Sec-WebSocket-Key")
	wsAccept := GenerateWSAccept(wsKey)

	hijacker, ok := res.(http.Hijacker)
	if !ok {
		return nil, &WSServerError{message: "Failed to hijack the connection"}
	}

	connection, _, _ := hijacker.Hijack()
	wsServer.clients = append(wsServer.clients, &connection)

	var content []byte
	content = append(content, "HTTP/1.1 101 Switching Protocols\r\n"...)
	content = append(content, "Upgrade: websocket\r\n"...)
	content = append(content, "Connection: Upgrade\r\n"...)
	content = append(content, fmt.Sprintf("Sec-WebSocket-Accept: %s", wsAccept)...)
	content = append(content, "\r\n\r\n"...)
	connection.Write(content)

	if wsServer.OnConnect != nil {
		wsServer.OnConnect()
	}

	return &connection, nil
}

func (wsServer *wsserver) readFromConnection(connection *net.Conn) {
	readBuffer := make([]byte, 256)

ReadForever:
	for true {
		bytesRead, readErr := (*connection).Read(readBuffer)
		if readErr != nil {
			fmt.Printf("Read Error: %s\n", readErr.Error())
		}

		if bytesRead < 2 {
			fmt.Println("Not enough bytes for a frame")
			continue
		}

		controlByte := readBuffer[0]
		opCode := controlByte & 0b00001111
		switch opCode {
		case 0x8:
			break ReadForever

		case 0x9:
			// ping

		case 0xA:
			// pong

		default:
		}

		payloadInfoByte := readBuffer[1]
		mask := payloadInfoByte & 0b10000000
		if mask == 0 {
			fmt.Println("Client should set mask bit")
			break
		}

		payloadLength := payloadInfoByte & 0b01111111

		fmt.Println(payloadLength)
		data := make([]byte, 0, payloadLength)
		switch {
		case payloadLength < 126:
			maskValue := readBuffer[2:6]
			data = wsServer.readFrameData(connection, maskValue, readBuffer[6:], uint64(payloadLength))

		case payloadLength == 126:
			sizeBytes := []byte{readBuffer[2], readBuffer[3]}
			maskValue := readBuffer[4:8]
			payloadLength16 := binary.BigEndian.Uint16(sizeBytes)
			data = wsServer.readFrameData(connection, maskValue, readBuffer[8:], uint64(payloadLength16))

		case payloadLength == 127:
			sizeBytes := []byte{
				readBuffer[2], readBuffer[3], readBuffer[4], readBuffer[5],
				readBuffer[6], readBuffer[7], readBuffer[8], readBuffer[9],
			}
			maskValue := readBuffer[10:14]
			payloadLength64 := binary.BigEndian.Uint64(sizeBytes)
			data = wsServer.readFrameData(connection, maskValue, readBuffer[14:], uint64(payloadLength64))
		}

		if wsServer.OnMessage != nil {
			wsServer.OnMessage(data)
		}
	}
}

func (wsServer *wsserver) readFrameData(connection *net.Conn, mask []byte, readBuffer []byte, length uint64) []byte {
	data := make([]byte, 0, length)
	for i := 0; i < len(readBuffer); i++ {
		data = append(data, readBuffer[i]^mask[i%4])
		if uint64(len(data)) == length {
			break
		}
	}

	bytesWritten := uint64(len(data))
	if length <= bytesWritten {
		return data
	}

	bytesRemaining := length - bytesWritten
	frameBuffer := make([]byte, bytesRemaining)
	bytesRead, err := (*connection).Read(frameBuffer)
	if err != nil {
		fmt.Println("Continutation read err")
		fmt.Println(err.Error())
	}

	offset := len(data) % 4
	for i := 0; i < bytesRead; i++ {
		unmaskedData := frameBuffer[i] ^ mask[(i+offset)%4]
		data = append(data, unmaskedData)
	}

	return data
}

func (wsServer *wsserver) Send(connection *net.Conn, data []byte) {
	payloadLength := len(data)
	frameLength := payloadLength + 2
	responsePayload := make([]byte, 0, frameLength)

	// TODO: Handle frames when data length > 125
	responsePayload = append(responsePayload, 0x81)
	responsePayload = append(responsePayload, byte(payloadLength))
	responsePayload = append(responsePayload, data...)
	(*connection).Write(responsePayload)
}

func (wsServer *wsserver) Broadcast(data []byte) {
	for _, client := range wsServer.clients {
		wsServer.Send(client, data)
	}
}

func (wsServer *wsserver) Close() {
	// TODO: this
}
