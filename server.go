package main

import (
	"fmt"
	"net/http"
	"html/template"
	"inyotech/ws/wsio"

)

func httpHandler(response http.ResponseWriter, request *http.Request) {
	t, err := template.ParseFiles("www/index.html")
	if err != nil {
		panic(err)
	}

	t.Execute(response, nil)
}

func readHandler(frameReader *wsio.FrameReader, ch chan<- *wsio.Frame) {

	defer close(ch)

	for {
		frame, err := frameReader.ReadFrame()
		if err != nil {
			break
		}
		ch<-frame
	}
}

func writeHandler(frameWriter *wsio.FrameWriter, ch <-chan *wsio.Frame) {

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

func wsHandler(w http.ResponseWriter, request *http.Request) {

	fmt.Println("wsHandler")

	conn, readerWriter, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	frameReader := wsio.NewFrameReader(readerWriter.Reader)
	frameWriter := wsio.NewFrameWriter(readerWriter.Writer)

	handshakeData, err := wsio.DoHandshake(request)
	if err != nil {
		panic(err)
	}

	response := wsio.FormHandshakeResponse(handshakeData)

	err = response.Write(readerWriter.Writer)
	if err != nil {
		panic(err)
	}

	readerWriter.Writer.Flush()

	frameReaderChannel := make(chan *wsio.Frame)
	frameWriterChannel := make(chan *wsio.Frame)

	go readHandler(frameReader, frameReaderChannel)
	go writeHandler(frameWriter, frameWriterChannel)

	for {
		frame, more := <-frameReaderChannel
		if !more {
			return
		}

		switch(frame.Type) {
		case wsio.TextFrame, wsio.BinaryFrame:
			frame.Mask = false
			fmt.Println(frame)
			frameWriterChannel<-frame
		case wsio.CloseFrame:
			code, message, err := wsio.ParseCloseFrame(frame)
			if err != nil {
				panic(err)
			}
			fmt.Println("close frame", code, message)
			frame.Mask = false
			frameWriterChannel<-frame
			close(frameWriterChannel)
			return
		case wsio.PingFrame:
			fmt.Println("got ping")
		case wsio.PongFrame:
			fmt.Println("got pong")
		default:
			fmt.Println("unhandled frame type", frame.Type)
		}
	}

}

func main() {

	http.HandleFunc("/service", wsHandler)
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)

}
