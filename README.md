# Golang WebSocket Library

Basic implementation of a substatial portion of the WebSocket Protocol as specified in IETF RFC 6455.  Specific requirements not implemented here include

* TLS; Running the protocol using the wss scheme is not supported

* Subprotocols; Tthe client doesn't provide an interface to request
  them and the server will not include the Sec-WebSocket-Protocol
  header in its handshake response.

* Extensions; Neither the client nor server support any protocol
  extensions.

An online demonstration of an echo service is hosted at [http://golang-ws.inyotech.com](http://golang-ws.inyotech.com).

## Building

1. Make sure the go language compiler and build tools are installed on
your system and your $GOPATH environment variable is correctly set.

```
$ which go
/usr/local/go/bin/go
$ echo $GOPATH
/home/ubuntu/go
```

Depending on your specific set up the output of the above commands may
vary.  For help with install and setup of the language compiler or
tools visit [this link](https://golang.org/doc/install).

2. Download, build and install this package with a single `go get` command.

```
$ go get github.com/inyotech/golang-ws/...
```

Assuming the above build steps were executed correctly, both 
echo_client and echo_server executable images will be installed under
`$GOPATH/bin`.

## Echo Client

A command line client demonstration that accepts websocket url and
message arguments.  It will attempt to open a websocket connection to
the endpoint in the given url.  On success it will send the message
through the connection then wait for a response.

```
$ $GOPATH/bin/echo_client --url ws://echo.websocket.org --message "websocket demonstration"
2017/05/03 17:56:44.530881 sending: websocket demonstration
2017/05/03 17:56:44.622566 received: websocket demonstration

```

## Echo Server

A more complex but still relatively simple websocket server
demonstration.  This executable image will listen for http requests
and response with a simple page that opens a web socket connection
back to itself.  Once the connection is open the client will begin to
continually send text messages through the websocket connection and
wait for the responses.  The client will display both the sent and
received messages along with timestamps.

`go get` will install the echo_server image under `$GOPATH/bin` and
the location of the required html template file will need to be
provided on the command line. By default the server will listen for
http requests at http://localhost:9000.

```
$ $GOPATH/bin/echo_server --WorkingDir=$GOPATH/src/github.com/inyotech/golang-ws/echo_server
2017/05/03 18:32:34 main.go:54: Starting /home/ubuntu/go/bin/echo_server
...
```

