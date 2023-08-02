package kansowebsockets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"strings"
	"sync"
)

type wsclient struct {
	host         string
	path         string
	onConnect    func()
	onDisconnect func()
	onMessage    func([]byte)
	connection   *net.Conn
}

func WebSocket(rawURL string, onConnect, onDisconnect func(), onMessage func([]byte)) (*wsclient, error) {
	urlObject, urlErr := url.Parse(rawURL)
	if urlErr != nil {
		fmt.Printf("Error creating URL object: %s\n", urlErr.Error())
		return nil, urlErr
	}

	wsClient := &wsclient{
		host:         urlObject.Host,
		path:         urlObject.Path,
		onConnect:    onConnect,
		onDisconnect: onDisconnect,
		onMessage:    onMessage,
	}

	return wsClient, nil
}

func (wsClient *wsclient) OnConnect(callback func()) {
	wsClient.onConnect = callback
}

func (wsClient *wsclient) OnDisconnect(callback func()) {
	wsClient.onDisconnect = callback
}

func (wsClient *wsclient) OnMessage(callback func([]byte)) {
	wsClient.onMessage = callback
}

func (wsClient *wsclient) Connect() error {
	conn, connErr := net.Dial("tcp", wsClient.host)
	if connErr != nil {
		fmt.Printf("Error connecting to %s, terminating connection.\n", wsClient.host)
		conn.Close()
		return connErr
	}

	wsClient.connection = &conn

	wsKey := GenerateWSKey()
	wsAccept := GenerateWSAccept(wsKey)

	var content []byte
	content = append(content, fmt.Sprintf("GET %s HTTP/1.1\r\n", wsClient.path)...)
	content = append(content, fmt.Sprintf("Host: %s\r\n", wsClient.host)...)
	content = append(content, "Upgrade: websocket\r\n"...)
	content = append(content, "Connection: Upgrade\r\n"...)
	content = append(content, "Sec-WebSocket-Version: 13\r\n"...)
	content = append(content, fmt.Sprintf("Sec-WebSocket-Key: %s", wsKey)...)
	content = append(content, "\r\n\r\n"...)
	conn.Write(content)

	ackBuffer := make([]byte, 256)
	_, readErr := conn.Read(ackBuffer)
	if readErr != nil {
		fmt.Println(readErr.Error())
		conn.Close()
		return readErr
	}

	responseReader := bytes.NewBuffer(ackBuffer)
	for true {
		line, readStrError := responseReader.ReadString('\n')
		if readStrError != nil {
			if readStrError == io.EOF {
				break
			}

			fmt.Printf("Read Error: %s\n", readStrError.Error())
			conn.Close()
			return readStrError
		}

		switch {
		case strings.HasPrefix(line, "Upgrade"):
			if !strings.HasSuffix(line, "websocket\r\n") {
				fmt.Println("Response not a WebSocket upgrade")
				return nil
			}

		case strings.HasPrefix(line, "Sec-WebSocket-Accept"):
			headerValue := strings.Split(line, ": ")[1]
			if strings.TrimSpace(headerValue) != string(wsAccept) {
				fmt.Printf("Invalid WS Key.\nExpected: %s\nReceived: %s\n",
					wsAccept, headerValue)
				return nil
			}
		}
	}

	if wsClient.onConnect != nil {
		wsClient.onConnect()
	}

	return nil
}

func (wsClient *wsclient) Read(wg *sync.WaitGroup) {
	wg.Add(1)
	go wsClient.readFromConnection(wg)
}

func (wsClient *wsclient) readFromConnection(wg *sync.WaitGroup) {
	defer wg.Done()
	defer (*wsClient.connection).Close()

	readBuffer := make([]byte, 256)

ReadForever:
	for true {
		bytesRead, readErr := (*wsClient.connection).Read(readBuffer)
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
		if mask > 0 {
			fmt.Println("Server should not set mask bit")
			break
		}

		payloadLength := payloadInfoByte & 0b01111111

		data := make([]byte, payloadLength)
		switch {
		case payloadLength < 126:
			data = wsClient.readFrameData(readBuffer[2:], uint64(payloadLength))

		case payloadLength == 126:
			sizeBytes := []byte{readBuffer[2], readBuffer[3]}
			payloadLength16 := binary.BigEndian.Uint16(sizeBytes)
			data = wsClient.readFrameData(readBuffer[4:], uint64(payloadLength16))

		case payloadLength == 127:
			sizeBytes := []byte{
				readBuffer[2], readBuffer[3], readBuffer[4], readBuffer[5],
				readBuffer[6], readBuffer[7], readBuffer[8], readBuffer[9],
			}
			payloadLength64 := binary.BigEndian.Uint64(sizeBytes)
			data = wsClient.readFrameData(readBuffer[10:], uint64(payloadLength64))
		}

		if wsClient.onMessage != nil {
			wsClient.onMessage(data)
		}
	}
}

func (wsClient *wsclient) readFrameData(readBuffer []byte, length uint64) []byte {
	data := make([]byte, 0, length)
	for i := 0; i < len(readBuffer); i++ {
		data = append(data, readBuffer[i])
	}

	if length <= uint64(len(data)) {
		return data
	}

	bytesRemaining := length - uint64(len(data))
	frameBuffer := make([]byte, bytesRemaining)
	bytesRead, err := (*wsClient.connection).Read(frameBuffer)
	if err != nil {
		fmt.Println("Continutation read err")
		fmt.Println(err.Error())
	}

	for i := 0; i < bytesRead; i++ {
		data = append(data, frameBuffer[i])
	}

	return data
}

func (wsClient *wsclient) Send(data []byte) {
	mask := make([]byte, 4)
	rand.Read(mask)

	maskedData := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		maskedByte := data[i] ^ mask[i%4]
		maskedData = append(maskedData, maskedByte)
	}

	frameLength := len(maskedData) + 2
	// TODO: Handle frames when data length > 125

	frame := make([]byte, 0, frameLength)
	frame = append(frame, 0x81)
	frame = append(frame, 0b10000000|byte(len(maskedData)))
	frame = append(frame, mask...)
	frame = append(frame, maskedData...)

	n, err := (*wsClient.connection).Write(frame)
	if err != nil {
		fmt.Printf("Send error: %s\n", err.Error())
	} else {
		fmt.Printf("Bytes written: %d\n", n)
	}

	fmt.Println("sent")
}
