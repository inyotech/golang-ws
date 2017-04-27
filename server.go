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

func wsHandler(conn net.Conn) {
	
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
	
	frame, err := frameReader.ReadFrame()
	if err != nil {
		panic(err)
	}

	fmt.Println(string(frame.Payload))

	frame = &wsio.Frame{
		Fin: true,
		Opcode: 1,
		Payload: []byte("This is a response message."),
	}

	err = frameWriter.WriteFrame(frame)
	if err != nil {
		
	}
	
//	conn.Close()
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
