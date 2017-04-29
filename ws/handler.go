package ws

import (
	"fmt"
	"net"
	"net/http"
)

type WsHandlerFunc func(chan *Frame)

func (clientFunc WsHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	conn, frameReadWriter, err := setupConnection(w, r)
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	dispatchFrames(clientFunc, frameReadWriter)

}

func setupConnection(w http.ResponseWriter, r *http.Request) (net.Conn, *FrameReadWriter, error) {

	conn, readerWriter, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}

	handshakeData, err := DoHandshake(r)
	if err != nil {
		return nil, nil, err
	}

	response := FormHandshakeResponse(handshakeData)

	err = response.Write(readerWriter.Writer)
	if err != nil {
		return nil, nil, err
	}

	readerWriter.Writer.Flush()

	frameReadWriter := NewFrameReadWriter(readerWriter)

	return conn, frameReadWriter, nil

}

func dispatchFrames(clientFunc func(chan *Frame), readWriter *FrameReadWriter) {

	clientChannel := make(chan *Frame)
	frameReaderChannel := make(chan *Frame)
	frameWriterChannel := make(chan *Frame)

	go readHandler(readWriter.FrameReader, frameReaderChannel)
	go writeHandler(readWriter.FrameWriter, frameWriterChannel)
	go clientFunc(clientChannel)


	var closedSent bool = false

dispatchLoop:

	for {
		select {
		case frame, ws_ok := <-frameReaderChannel:
			if !ws_ok {
				frameWriterChannel<-NewCloseFrame(1001, "")
				break dispatchLoop
			}

			switch(frame.Type) {

			case TextFrame, BinaryFrame:
				clientChannel<-frame

			case CloseFrame:
				if !closedSent {
					frame.Mask = false
					frameWriterChannel<-frame
				}
				break dispatchLoop

			case PingFrame:
				frameWriterChannel<-newPongFrame(frame.Payload)

			case PongFrame:

			default:
				fmt.Println("unhandled frame type", frame.Type)
			}

		case frame, client_ok := <-clientChannel:
			if !client_ok {
				frameWriterChannel<-NewCloseFrame(1001, "")
				break dispatchLoop
			}

			if frame.Mask {
				frame.Mask = false
			}

			frameWriterChannel<-frame
		}
	}

}

func readHandler(frameReader *FrameReader, ch chan<- *Frame) {


	for {
		frame, err := frameReader.ReadFrame()
		if err != nil {
			close(ch)
			return
		}
		ch<-frame
	}
}

func writeHandler(frameWriter *FrameWriter, ch <-chan *Frame) {

	for {
		frame, more := <-ch
		if !more {
			return
		}

		err := frameWriter.WriteFrame(frame)
		if err != nil {
			return
		}
	}
}
