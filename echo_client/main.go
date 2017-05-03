package main

import (
	"log"
	"flag"

	"github.com/inyotech/golang-ws/ws"
)

func main() {

	log.SetFlags(log.Ldate|log.Ltime|log.Lmicroseconds)

	url := flag.String("url", "ws://echo.websocket.org/", "Web socket url (default 'ws://echo.websocket.org/')")
	message := flag.String("message", "Test message", "Message to send (default 'Test message')")

	flag.Parse()

	channel, err := ws.Dial(*url, "")
	if err != nil {
		panic(err)
	}

	frame := ws.NewTextFrame(*message)

	log.Printf("sending: %s", string(frame.Payload))

	channel <-frame

	frame = <-channel

	log.Printf("received: %s", string(frame.Payload))

	close(channel)
}
