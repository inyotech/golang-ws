package main

import (
	"fmt"

	"github.com/inyotech/golang-ws/ws"
)

func main() {

	channel, err := ws.Dial("ws://echo.websocket.org:80")
	if err != nil {
		panic(err)
	}

	frame := ws.NewTextFrame("Test message")

	fmt.Println("sending frame", frame, string(frame.Payload))

	channel <-frame

	frame = <-channel

	fmt.Println("recevied frame", frame, string(frame.Payload))

	close(channel)
}
