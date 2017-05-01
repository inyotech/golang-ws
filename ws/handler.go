package ws

import (
	"fmt"
	"strings"
	"strconv"
	"bufio"
	"net"
	"net/http"
	"net/url"
	"crypto/sha1"
	"encoding/base64"
	"math/rand"
	"errors"
)

const websocketGuid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func Dial(serverUrl string) (channel chan *Frame, err error) {

	parsedUrl, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}

	if parsedUrl.Scheme != "ws" {
		return nil, errors.New(fmt.Sprintf("Scheme %s not allowed", parsedUrl.Scheme))
	}

	conn, err := net.Dial("tcp", parsedUrl.Host)
	if err != nil {
		return nil, err
	}

	frameReadWriter, err := setupClientConnection(conn)
	if err != nil {
		panic(err)
	}

	clientChannel := make(chan *Frame)

	go dispatchFrames(clientChannel, frameReadWriter, false)

	return clientChannel, nil
}

type WsHandlerFunc func(chan *Frame)

func (clientFunc WsHandlerFunc) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {

	conn, responseReadWriter, err := responseWriter.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}

	frameReadWriter, err := setupServerConnection(responseReadWriter, request)
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	clientChannel := make(chan *Frame)

	go clientFunc(clientChannel)

	dispatchFrames(clientChannel, frameReadWriter, true)

}

func setupClientConnection(conn net.Conn) (*frameReadWriter, error) {

	readWriter := bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)

	headers := http.Header{}
	headers.Add("Host", "echo.websocket.org")
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

	expectedKey := generateAcceptKey(requestKey)
	if acceptKey != expectedKey {
		panic("request, accept key mismatch")
	}

	if response.Header.Get("Sec-Websocket-Extensions") != "" {
		panic("unexpected extension(s)")
	}

	if response.Header.Get("Sec-Websocket-Protocol") != "" {
		panic("unexpected subprotocol")
	}

	frameReadWriter := newFrameReadWriter(readWriter)

	return frameReadWriter, nil
}

func setupServerConnection(responseReadWriter *bufio.ReadWriter, r *http.Request) (*frameReadWriter, error) {

	if r.Method != "GET" {
		return nil, errors.New("connection requires GET method")
	}

	if !r.ProtoAtLeast(1, 1) {
		return nil, errors.New("connection requires at least HTTP/1.1")
	}

	if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
		return nil, errors.New("No Upgrade: websocket header found")
	}

	if strings.ToLower(r.Header.Get("Connection")) != "upgrade" {
		return nil, errors.New("No Connection: Upgrade header found")
	}

	var websocketKey string
	if websocketKey = r.Header.Get("Sec-Websocket-Key"); len(websocketKey) == 0 {
		return nil, errors.New("No Sec-Websocket-Key found")
	}

	decodedKey, err := base64.StdEncoding.DecodeString(websocketKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to base64 decode Sec-Websocket-Key %s", websocketKey))
	}

	if len(decodedKey) != 16 {
		return nil, errors.New(fmt.Sprintf("Invalid decoded Sec-Websocket-Key length %d", len(decodedKey)))
	}

	var websocketVersionString string
	if websocketVersionString = r.Header.Get("Sec-Websocket-Version"); len(websocketVersionString) == 0 {
		return nil, errors.New("No Sec-Websocket-Version found")
	}

	supportedWebsocketVersions := []int{13}

	websocketVersion, err := strconv.Atoi(websocketVersionString)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to parse Sec-Websocket-Version value %s", websocketVersionString))
	}

	if !intContainedInList(websocketVersion, supportedWebsocketVersions) {
		return nil, errors.New(fmt.Sprintf("Unsupported websocket version %d", websocketVersion))
	}

	acceptHeaders := http.Header{}

	acceptHeaders.Add("Upgrade", "websocket")
	acceptHeaders.Add("Connection", "Upgrade")
	acceptHeaders.Add("Sec-Websocket-Accept", generateAcceptKey(websocketKey))
	acceptHeaders.Add("Sec-Websocket-Version", fmt.Sprintf("%d", 13))

	response := &http.Response{
		Status: "Switching Protocols",
		StatusCode: 101,
		Proto: "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: acceptHeaders,
	}

	err = response.Write(responseReadWriter)
	if err != nil {
		return nil, err
	}

	responseReadWriter.Writer.Flush()

	frameReadWriter := newFrameReadWriter(responseReadWriter)

	return frameReadWriter, nil

}

func dispatchFrames(clientChannel chan *Frame, readWriter *frameReadWriter, isServer bool) {

	frameReaderChannel := make(chan *Frame)
	frameWriterChannel := make(chan *Frame)

	go readHandler(readWriter.frameReader, frameReaderChannel)
	go writeHandler(readWriter.frameWriter, frameWriterChannel)

dispatchLoop:

	for {
		select {
		case frame, ws_ok := <-frameReaderChannel:
			if !ws_ok {
				closeFrame := newCloseFrame(1001, "")
				if !isServer {
					closeFrame.Mask = true
				}
				frameWriterChannel <- closeFrame
				break dispatchLoop
			}

			switch(frame.Type) {

			case TextFrame, BinaryFrame:
				clientChannel <- frame

			case CloseFrame:
				if isServer {
					frame.Mask = false
				} else {
					frame.Mask = true
				}
				frameWriterChannel<-frame
				break dispatchLoop

			case PingFrame:
				if isServer {
					frame.Mask = false
				} else {
					frame.Mask = true
				}
				frameWriterChannel<-newPongFrame(frame.Payload)

			case PongFrame:

			default:
				fmt.Println("unhandled frame type", frame.Type)
			}

		case frame, client_ok := <-clientChannel:
			if !client_ok {
				closeFrame := newCloseFrame(1001, "")
				if !isServer {
					closeFrame.Mask = true
				}
				frameWriterChannel <- closeFrame
				break dispatchLoop
			}

			if isServer {
				frame.Mask = false
			} else {
				frame.Mask = true
			}

			frameWriterChannel<-frame
		}
	}

}

func intContainedInList(i int, l []int) bool {
	for _, value := range l {
		if value == i {
			return true
		}
	}
	return false
}

func generateAcceptKey(key string) string {

	keySuffix := "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	checksum := sha1.Sum([]byte(key + keySuffix))

	return base64.StdEncoding.EncodeToString(checksum[:])
}

func readHandler(frameReader *frameReader, ch chan<- *Frame) {


	for {
		frame, err := frameReader.ReadFrame()
		if err != nil {
			close(ch)
			return
		}
		ch<-frame
	}
}

func writeHandler(frameWriter *frameWriter, ch <-chan *Frame) {

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
