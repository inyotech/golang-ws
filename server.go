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

	for {
		frame, err := frameReader.ReadFrame()
		if err != nil {
			close(ch)
			return
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

	readerChannel := make(chan *wsio.Frame, 1)
	writerChannel := make(chan *wsio.Frame, 1)

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

	go readHandler(frameReader, readerChannel)

	go writeHandler(frameWriter, writerChannel)

	for {
		frame, more := <-readerChannel
		if !more {
			close(writerChannel)
			return
		}

		switch(frame.Opcode) {
		case 1, 2:
			fmt.Println("echoing frame")
			frame.Mask = []byte{}
			writerChannel<-frame
		case 8:
			fmt.Println("got connection close", frame.Payload)
			code, message, err := wsio.ParseCloseFrame(frame)
			if err != nil {
				panic(err)
			}
			fmt.Println("close frame", code, message)
			frame.Mask = []byte{}
			writerChannel<-frame
			conn.Close()
		case 9:
			fmt.Println("got ping")
		case 10:
			fmt.Println("got pong")
		default:
			fmt.Println("unhandled frame type", frame.Opcode)
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
