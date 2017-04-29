package main

import (
	"fmt"
	"bufio"
	"strings"
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

	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		panic(err)
	}

	readWriter := bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)

	headers := http.Header{}
	headers.Add("Host", "127.0.0.1:9000")
	headers.Add("Upgrade", "websocket")
	headers.Add("Connection", "Upgrade")
	headers.Add("Origin", "http://localhost")

	data := make([]byte, 16)
	rand.Read(data)
	requestKey := base64.StdEncoding.EncodeToString(data)

	headers.Add("Sec-Websocket-Key", requestKey)
	headers.Add("Sec-Websocket-Version", strconv.Itoa(13))

	readWriter.Write([]byte("GET / HTTP/1.1\r\n"))
	headers.Write(readWriter.Writer)
	readWriter.Write([]byte("\r\n"))
	readWriter.Writer.Flush()

	response, err := http.ReadResponse(readWriter.Reader, &http.Request{Method: "GET"})
	if err != nil {
		panic(err)
	}

	if response.StatusCode != 101 {
		panic(fmt.Sprintf("unexpected response code ", response.StatusCode))
	}

	if strings.ToLower(response.Header.Get("Upgrade")) != "websocket" {
		panic("No Upgrade: websocket header")
	}

	if strings.ToLower(response.Header.Get("Connection")) != "upgrade" {
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

	frame := ws.NewTextFrame("Test message")
	frame.Mask = true

	frameReadWriter := ws.NewFrameReadWriter(readWriter)

	fmt.Println("sending frame", frame, string(frame.Payload))

	err = frameReadWriter.WriteFrame(frame)
	if err != nil {
		panic(err)
	}

	frame, err = frameReadWriter.ReadFrame()
	if err != nil {
		panic(err)
	}

	fmt.Println("recevied frame", frame, string(frame.Payload))

	frame = ws.NewCloseFrame(1001, "closing")
	frame.Mask = true
	err = frameReadWriter.WriteFrame(frame)
	if err != nil {
		panic(err)
	}

	frame, err = frameReadWriter.ReadFrame()
	if err != nil {
		panic(err)
	}

	if frame.Type != ws.CloseFrame {
		panic("expected close frame")
	}

	fmt.Println(ws.ParseCloseFrame(frame))
}
