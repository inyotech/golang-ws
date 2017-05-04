package ws

import (
	"fmt"
	"testing"
	"net/http"
)

func TestFrame(t * testing.T) {

	message := "test message"

	frame := NewTextFrame(message)

	if frame.Type != TextFrame {
		t.Error("message payload mismatch")
	}

	if string(frame.Payload) != message {
		t.Error("message payload mismatch")
	}

	frame = newCloseFrame(1001, "channel closed")

	if frame.Type != CloseFrame {
		t.Error("incorrect type in close frame")
	}

	code, message, err := ParseCloseFrame(frame)
	if err != nil {
		t.Error(err)
	}

	if code != 1001 {
		t.Error("incorrect code in close frame")
	}

	if message != "channel closed" {
		t.Error("incorrect messsage in close frame")
	}

}

func wsHandler(ch chan *Frame) {

	defer close(ch)

	for {
		frame, ws_ok := <-ch
		if !ws_ok {
			return
		}
		ch<-frame
	}
}

func startHttpServer() *http.Server {

	server := &http.Server{Addr: ":8080"}

	http.Handle("/", WsHandlerFunc(wsHandler))

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()

	return server
}

func TestWebSocket(t * testing.T) {

	server := startHttpServer()

	channel, err := Dial("ws://localhost:8080/", "")
	if err != nil {
		t.Error(err)
	}

	message := "test message"

	sentFrame := NewTextFrame(message)

	sentFrame.Mask = true

	channel <- sentFrame

	receivedFrame := <- channel

	if string(receivedFrame.Payload) != message {
		t.Error("sent and received frames don't match")
	}

	close(channel)

	err = server.Shutdown(nil)
	if err != nil {
		t.Error(err)
	}
}
