package wsio

import (
	"fmt"
	"net"
	"net/http"
	"encoding/binary"
)

func SetupConnection(w http.ResponseWriter, r *http.Request) (net.Conn, *FrameReadWriter, error) {

	fmt.Println("setup")

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

func DispatchFrames(clientFunc func(chan *Frame), readWriter *FrameReadWriter) {

	clientChannel := make(chan *Frame)
	frameReaderChannel := make(chan *Frame)
	frameWriterChannel := make(chan *Frame)

	go readHandler(readWriter.FrameReader, frameReaderChannel)
	go writeHandler(readWriter.FrameWriter, frameWriterChannel)
	go clientFunc(clientChannel)

	for {
		select {
		case frame, ws_ok := <-frameReaderChannel:
			if !ws_ok {
				close(frameWriterChannel)
				close(clientChannel)
				return
			}

			switch(frame.Type) {
			case TextFrame, BinaryFrame:
				frame.Mask = false
				fmt.Println(frame)
				clientChannel<-frame
			case CloseFrame:
				code, message, err := ParseCloseFrame(frame)
				if err != nil {
					panic(err)
				}
				fmt.Println("close frame", code, message)
				frame.Mask = false
				frameWriterChannel<-frame
				close(frameReaderChannel)
				close(frameWriterChannel)
				close(clientChannel)
				return
			case PingFrame:
				fmt.Println("got ping")
			case PongFrame:
				fmt.Println("got pong")
			default:
				fmt.Println("unhandled frame type", frame.Type)
			}

		case frame, client_ok := <-clientChannel:
			if !client_ok {
				fmt.Println("client closed")
				code := make([]byte, 2)
				binary.BigEndian.PutUint16(code, 1001)
				closeFrame := &Frame{
					Type: CloseFrame,
					Payload: code,
					fin: true,
				}
				frameWriterChannel<-closeFrame
				close(frameReaderChannel)
				close(frameWriterChannel)
				return
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
			break
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
			panic(err)
		}
	}
}
