package main

import (
	"fmt"
	"bufio"
	"time"
	"net"
	"net/http"
	"strconv"
	"math/rand"
	"encoding/base64"
	"crypto/sha1"
	
	"github.com/inyotech/golang-ws/ws"
)

func main() {

	const websocketGuid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	rand.Seed(time.Now().UnixNano())

	conn, err := net.Dial("tcp", "echo.websocket.org:80")
	if err != nil {
		panic(err)
	}

	readWriter := bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)

	headers := http.Header{}
	headers.Add("Host", "echo.websocket.org")
	headers.Add("Upgrade", "websocket")
	headers.Add("Connection", "Upgrade")
	headers.Add("Origin", "http://www.websocket.org")
	
	data := make([]byte, 16)
	rand.Read(data)
	requestKey := base64.StdEncoding.EncodeToString(data)
	
	headers.Add("Sec-Websocket-Key", requestKey)
	headers.Add("Sec-Websocket-Version", strconv.Itoa(13))

	readWriter.Write([]byte("GET ws://echo.websocket.org/?encoding=text HTTP/1.1\r\n"))
	headers.Write(readWriter.Writer)
	readWriter.Write([]byte("\r\n\r\n"))
	readWriter.Flush()

	response, err := http.ReadResponse(readWriter.Reader, nil)
	if err != nil {
		panic(err)
	}

	if response.StatusCode != 101 {
		panic(fmt.Sprintf("unexpected response code ", response.StatusCode))
	}

	if response.Header.Get("Upgrade") != "websocket" {
		panic("No Upgrade: websocket header")
	}

	if response.Header.Get("Connection") != "Upgrade" {
		panic("No Upgrade: websocket header")
	}

	acceptKey := response.Header.Get("Sec-Websocket-Accept")
	
	checksum := sha1.Sum([]byte(requestKey + websocketGuid))
	expectedKey := base64.StdEncoding.EncodeToString(checksum[:])

	if acceptKey != expectedKey {
		panic("request, accept key mismatch")
	}

	if response.Header.Get("Sec-Websocket-Extensions") != "" {
		panic("unexpected extension(s)")
	}

	if response.Header.Get("Sec-Websocket-Protocol") != "" {
		panic("unexpected subprotocol")
	}

	fmt.Println("starting websocket")

	frame := ws.NewTextFrame("Test message")
	frame.Mask = true
	
	fmt.Println("writing frame", frame, string(frame.Payload))
	
	frameReadWriter := ws.NewFrameReadWriter(readWriter)

	err = frameReadWriter.WriteFrame(frame)
	if err != nil {
		panic(err)
	}

	fmt.Println("after write", frame)
	
	frame, err = frameReadWriter.ReadFrame()
	if err != nil {
		panic(err)
	}

	if frame.Type == ws.CloseFrame {
		fmt.Println(ws.ParseCloseFrame(frame))
	}

	fmt.Println("recevied frame", frame, string(frame.Payload))

}
