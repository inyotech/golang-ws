package main

import (
	"fmt"
	"flag"

	"github.com/inyotech/golang-ws/ws"
)

func main() {

	url := flag.String("url", "ws://echo.websocket.org/", "Web socket url (default 'ws://echo.websocket.org/')")
	message := flag.String("message", "Test message", "Message to send (default 'Test message')")

	flag.Parse()

	channel, err := ws.Dial(*url, "")
	if err != nil {
		panic(err)
	}

	frame := ws.NewTextFrame(*message)

	fmt.Println("sending message: ", string(frame.Payload))

	channel <-frame

	frame = <-channel

	fmt.Println("recevied message: ", string(frame.Payload))

	close(channel)
}
