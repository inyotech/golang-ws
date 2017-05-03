# Web Socket Library

Basic implementation of a substatial portion of the WebSocket Protocol as specified in IETF RFC 6455.  Specific requirements not implemented here include

* TLS; Running the protocol using the wss scheme is not supported

* Subprotocols; Tthe client doesn't provide an interface to request
  them and the server will not include the Sec-WebSocket-Protocol
  header in its handshake response.

* Extensions; Neither the client nor server support any protocol
  extensions.

## Building

1. Make sure the go language compiler and build tools are installed on
your system and your $GOPATH environment variable is correctly set.

```
$ which go
/usr/local/go/bin/go
$ echo $GOPATH
/home/ubuntu/go
```

## Echo Client

Command line client demonstration that accepts websocket url and message
arguments.  It will attempt to open a websocket connection to the
endpoint in the given url.  On success it will send the message
through the connection then wait for a response.

```
echo_client$ ./echo_client --url ws://echo.websocket.org --message "websocket client demonstration"
2017/05/03 17:36:33.428695 sending: websocket client demonstration
2017/05/03 17:36:33.518832 received: websocket client demonstration

```

## Echo Server


## Quick Start

## Apache Web Proxy

## Systemd daemonization

