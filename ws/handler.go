package ws

import (
	"fmt"
	"log"
	"strings"
	"strconv"
	"bufio"
	"net"
	"net/http"
	"net/url"
	"crypto/sha1"
	"encoding/base64"
	"math/rand"
)

const websocketGuid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func Dial(serverUrl string, origin string) (channel chan *Frame, err error) {

	parsedUrl, err := url.Parse(serverUrl)
	if err != nil {
		return
	}

	if parsedUrl.Scheme != "ws" {
		err = fmt.Errorf("Scheme %s not allowed", parsedUrl.Scheme)
		return
	}

	if len(parsedUrl.Port()) == 0 {
		parsedUrl.Host += ":80"
	}

	conn, err := net.Dial("tcp", parsedUrl.Host)
	if err != nil {
                return
	}

	frameReadWriter, err := setupClientConnection(conn, parsedUrl, origin)
	if err != nil {
		return
	}

	channel = make(chan *Frame)

	go dispatchFrames(channel, frameReadWriter, false)

	return
}

type WsHandlerFunc func(chan *Frame)

func (clientFunc WsHandlerFunc) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {

	conn, responseReadWriter, err := responseWriter.(http.Hijacker).Hijack()
	if err != nil {
		log.Print(err)
		return
	}

	frameReadWriter, err := setupServerConnection(responseReadWriter, request)
	if err != nil {
		log.Print(err)
		return
	}

	defer conn.Close()

	clientChannel := make(chan *Frame)

	go clientFunc(clientChannel)

	dispatchFrames(clientChannel, frameReadWriter, true)

}

func setupClientConnection(conn net.Conn, url *url.URL, origin string) (frameReadWriter *frameReadWriter, err error) {

	readWriter := bufio.NewReadWriter(
		bufio.NewReader(conn),
		bufio.NewWriter(conn),
	)

	headers := http.Header{}
	headers.Add("Host", url.Host)
	headers.Add("Upgrade", "websocket")
	headers.Add("Connection", "Upgrade")
	if len(origin) != 0 {
		headers.Add("Origin", origin)
	}

	data := make([]byte, 16)
	rand.Read(data)
	requestKey := base64.StdEncoding.EncodeToString(data)

	headers.Add("Sec-Websocket-Key", requestKey)
	headers.Add("Sec-Websocket-Version", strconv.Itoa(13))

	readWriter.Write([]byte("GET "+url.RequestURI()+" HTTP/1.1\r\n"))
	headers.Write(readWriter.Writer)
	readWriter.Write([]byte("\r\n"))
	readWriter.Writer.Flush()

	response, err := http.ReadResponse(readWriter.Reader, &http.Request{Method: "GET"})
	if err != nil {
		return
	}

	if response.StatusCode != 101 {
		err = fmt.Errorf("Unexpected response code %d", response.StatusCode)
		return
	}

	if strings.ToLower(response.Header.Get("Upgrade")) != "websocket" {
		err = fmt.Errorf("No Upgrade: websocket header")
		return
	}

	if strings.ToLower(response.Header.Get("Connection")) != "upgrade" {
		err = fmt.Errorf("No Upgrade: websocket header")
		return
	}

	acceptKey := response.Header.Get("Sec-Websocket-Accept")

	expectedKey := generateAcceptKey(requestKey)
	if acceptKey != expectedKey {
		err = fmt.Errorf("request, accept key mismatch")
		return
	}

	if response.Header.Get("Sec-Websocket-Extensions") != "" {
		err = fmt.Errorf("unexpected extension(s)")
		return
	}

	if response.Header.Get("Sec-Websocket-Protocol") != "" {
		err = fmt.Errorf("unexpected subprotocol")
		return
	}

	frameReadWriter = newFrameReadWriter(readWriter)

	return
}

func setupServerConnection(responseReadWriter *bufio.ReadWriter, r *http.Request) (frameReadWriter *frameReadWriter, err error) {

	if r.Method != "GET" {
		fmt.Errorf("connection requires GET method")
		return
	}

	if !r.ProtoAtLeast(1, 1) {
		err = fmt.Errorf("connection requires at least HTTP/1.1")
		return
	}

	if strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
		err = fmt.Errorf("No Upgrade: websocket header found")
		return
	}

	if strings.ToLower(r.Header.Get("Connection")) != "upgrade" {
		err = fmt.Errorf("No Connection: Upgrade header found")
		return
	}

	var websocketKey string
	if websocketKey = r.Header.Get("Sec-Websocket-Key"); len(websocketKey) == 0 {
		err = fmt.Errorf("No Sec-Websocket-Key found")
		return
	}

	decodedKey, err := base64.StdEncoding.DecodeString(websocketKey)
	if err != nil {
		err = fmt.Errorf("Unable to base64 decode Sec-Websocket-Key %s", websocketKey)
		return
	}

	if len(decodedKey) != 16 {
		err = fmt.Errorf("Invalid decoded Sec-Websocket-Key length %d", len(decodedKey))
		return
	}

	var websocketVersionString string
	if websocketVersionString = r.Header.Get("Sec-Websocket-Version"); len(websocketVersionString) == 0 {
		err = fmt.Errorf("No Sec-Websocket-Version found")
		return
	}

	supportedWebsocketVersions := []int{13}

	websocketVersion, err := strconv.Atoi(websocketVersionString)
	if err != nil {
		err = fmt.Errorf("Failed to parse Sec-Websocket-Version value %s", websocketVersionString)
		return
	}

	if !intContainedInList(websocketVersion, supportedWebsocketVersions) {
		err = fmt.Errorf("Unsupported websocket version %d", websocketVersion)
		return
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
		return
	}

	responseReadWriter.Writer.Flush()

	frameReadWriter = newFrameReadWriter(responseReadWriter)

	return

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
				log.Printf("unhandled frame type", frame.Type)
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
