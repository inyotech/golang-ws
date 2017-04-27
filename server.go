package main

import (
	"fmt"
        "bufio"
	"net"
	"net/http"
	"html/template"
	"inyotech/ws/wsio"

)

func httpHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("www/index.html")
	if err != nil {
		panic(err)
	}

	t.Execute(w, nil)
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

func wsHandler(conn net.Conn) {

	defer conn.Close()

	readerWriter := bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)

	frameReader := wsio.NewFrameReader(readerWriter.Reader)
	frameWriter := wsio.NewFrameWriter(readerWriter.Writer)

	request, err := http.ReadRequest(readerWriter.Reader)
	if err != nil {
		panic(err)
	}

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

	go func() {
		listener, err := net.Listen("tcp", ":8081")
		if err != nil {
			panic(err)
		}
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			go wsHandler(conn)
		}

	}()

	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)

}
