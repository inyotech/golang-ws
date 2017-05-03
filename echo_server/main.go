package main

import (
	"os"
	"log"
	"flag"
	"net/url"
	"net/http"
	"html/template"

	"github.com/inyotech/golang-ws/ws"
)

func httpHandler(response http.ResponseWriter, request *http.Request) {

	log.Printf("http request: %s from %s", request.RequestURI, request.RemoteAddr)

	t, err := template.ParseFiles("template/index.html")
	if err != nil {
		log.Panic(err)
	}

	// Url that the client should use to connect to the websocket endpoint.
	websocketUrl := &url.URL{
		Scheme: "ws",
		Host: request.Host,
		Path: "/echo_service",
	}

	data := struct {
		WebSocketUrl string
	}{
		websocketUrl.String(),
	}

	t.Execute(response, data)
}

func wsHandler(ch chan *ws.Frame) {

	defer close(ch)

	log.Printf("handling websocket connection")

	for {
		frame, ws_ok := <-ch
		if !ws_ok {
			return
		}
		ch<-frame
	}


}


type options struct {
	WorkingDir string
	BindAddr string
}

func main() {


	log.SetFlags(log.Ldate|log.Ltime|log.Lshortfile)

	log.Printf("Starting %s", os.Args[0])

	var options options

	flag.StringVar(&options.WorkingDir, "WorkingDir", ".", "Working directory, file search path is relative to this (default '.')")
	flag.StringVar(&options.BindAddr, "BindAddr", "localhost:9000", "Ip:Port to listen on (default 'localhost:9000')")

	flag.Parse()

	err := os.Chdir(options.WorkingDir)
	if err != nil {
		panic(err)
	}

	http.Handle("/echo_service", ws.WsHandlerFunc(wsHandler))
	http.HandleFunc("/", httpHandler)
	err = http.ListenAndServe(options.BindAddr, nil)
	if err != nil {
		log.Print(err)
	}

}
