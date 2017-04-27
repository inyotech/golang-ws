package wsio

import (
	"fmt"
	"strings"
	"strconv"
	"net/http"
	"encoding/base64"
	"crypto/sha1"
	"errors"
)

type HandshakeData struct {
	origin string
	resource string
	key string
	acceptKey string
	version int
	acceptVersion int
	requestSubprotocols []string
	acceptSubprotocol string
	requestExtensions []string
	acceptExtensions []string
	acceptExtensionsHeader string
}

func DoHandshake(request *http.Request) (*HandshakeData, error) {

	if request.Method != "GET" {
		return nil, errors.New("connection requires GET method")
	}

	if !request.ProtoAtLeast(1, 1) {
		return nil, errors.New("connection requires at least HTTP/1.1")
	}

	if header, ok := request.Header["Upgrade"]; !(ok && stringContainedInList("websocket", header)) {
		return nil, errors.New("No Upgrade: websocket header found")
	}

	request.Header.Del("Upgrade")

	if header, ok := request.Header["Connection"]; !(ok && stringContainedInList("Upgrade", header)) {
		return nil, errors.New("No Connection: Upgrade header found")
	}

	request.Header.Del("Connection")

	var websocketKey string
	if websocketKey = request.Header.Get("Sec-Websocket-Key"); len(websocketKey) == 0 {
		return nil, errors.New("No Sec-Websocket-Key found")
	}

	decodedKey, err := base64.StdEncoding.DecodeString(websocketKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to base64 decode Sec-Websocket-Key %s", websocketKey))
	}

	if len(decodedKey) != 16 {
		return nil, errors.New(fmt.Sprintf("Invalid decoded Sec-Websocket-Key length %d", len(decodedKey)))
	}

	request.Header.Del("Sec-Websocket-Key")

	var websocketVersionString string
	if websocketVersionString = request.Header.Get("Sec-Websocket-Version"); len(websocketVersionString) == 0 {
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

	request.Header.Del("Sec-Websocket-Version")

	originHeader := request.Header.Get("Origin")

	request.Header.Del("Origin")

	var requestedProtocols []string
	if websocketProtocol, ok := request.Header["Sec-Websocket-Protocol"]; ok {
		requestedProtocols = parseSubprotocolHeaders(websocketProtocol)
	}

	request.Header.Del("Sec-Websocket-Protocol")

	var requestedExtensions []string
	if websocketExtension, ok := request.Header["Sec-Websocket-Extensions"]; ok {
		requestedExtensions = parseExtensionHeaders(websocketExtension)
	}

        request.Header.Del("Sec-Websocket-Extensions")

	handshakeData := &HandshakeData{
		origin: originHeader,
		key: websocketKey,
		acceptKey: generateAcceptKey(websocketKey),
		version: websocketVersion,
		acceptVersion: 13,
		resource: request.RequestURI,
		requestSubprotocols: requestedProtocols,
		requestExtensions: requestedExtensions,
	}

	return handshakeData, nil
}

func FormHandshakeResponse(data *HandshakeData) *http.Response {

	acceptHeaders := http.Header{}

	acceptHeaders.Add("Upgrade", "websocket")
	acceptHeaders.Add("Connection", "Upgrade")
	acceptHeaders.Add("Sec-Websocket-Accept", data.acceptKey)
	acceptHeaders.Add("Sec-Websocket-Version", fmt.Sprintf("%d", data.acceptVersion))

	if data.acceptSubprotocol != "" {
		acceptHeaders.Add("Sec-Websocket-Protocol", data.acceptSubprotocol)
	}

	if data.acceptExtensionsHeader != "" {
		acceptHeaders.Add("Sec-Websocket-Protocol", data.acceptExtensionsHeader)
	}

	response := &http.Response{
		Status: "Switching Protocols",
		StatusCode: 101,
		Proto: "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: acceptHeaders,
	}

	return response
}

func generateAcceptKey(key string) string {

	keySuffix := "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	checksum := sha1.Sum([]byte(key + keySuffix))

	return base64.StdEncoding.EncodeToString(checksum[:])
}

func stringContainedInList(s string, l []string) bool {
	for _, value := range l {
		if strings.Contains(value, s) {
			return true
		}
	}
	return false
}

func intContainedInList(i int, l []int) bool {
	for _, value := range l {
		if value == i {
			return true
		}
	}
	return false
}

func parseSubprotocolHeaders(subprotocolHeaders []string) []string {

	var subprotocols []string

	for _, protocolStrings := range subprotocolHeaders {
		for _, protocolString := range strings.Split(protocolStrings, ",") {
			subprotocols = append(subprotocols, strings.TrimSpace(protocolString))
		}
	}
	return subprotocols

}

func parseExtensionHeaders(extensionHeaders []string) []string {

	var extensions []string

	for _, extensionStrings := range extensionHeaders {
		for _, extensionString := range strings.Split(extensionStrings, ",") {
			value := strings.TrimSpace(extensionString)
			for _, extension := range strings.Split(value, ";") {
				extension = strings.TrimSpace(extension)
				extensions = append(extensions, extension)
			}
		}
	}
	return extensions
}
