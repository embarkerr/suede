package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/embarkerr/suede"
)

func main() {
	client, wsErr := suede.WebSocket("http://localhost:8080/chat")
	if wsErr != nil {
		panic("could not connect")
	}

	fmt.Print("Enter your name > ")
	reader := bufio.NewReader(os.Stdin)
	name, _ := reader.ReadString('\n')
	name = strings.Replace(name, "\n", "", -1)

	client.OnConnect = func() {
		client.Send([]byte(name + " has joined the chat"))
	}

	client.OnMessage = func(data []byte) {
		fmt.Printf("%s\n", data)
	}

	client.RunCallback(func() {
		fmt.Println("Connected. Welcome to the chatroom.")
		reader := bufio.NewReader(os.Stdin)
		for true {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Read error")
				continue
			}

			fmt.Print("\u001b[1A\u001b[2K")
			message = strings.Replace(message, "\n", "", -1)
			client.Send([]byte("[" + name + "] " + message))
		}
	})
}
