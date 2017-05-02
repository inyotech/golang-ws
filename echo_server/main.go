package main

import (
	"os"
	"log"
	"flag"
	"net/http"
	"html/template"

	"github.com/inyotech/golang-ws/ws"
)

func (options options) httpHandler(response http.ResponseWriter, request *http.Request) {

	log.Printf("http request: %s from %s", request.RequestURI, request.RemoteAddr)

	t, err := template.ParseFiles("template/index.html")
	if err != nil {
		log.Panic(err)
	}

	t.Execute(response, options)
}

func (options options) wsHandler(ch chan *ws.Frame) {

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
	WsUrl string
}

func main() {


	log.SetFlags(log.Ldate|log.Ltime|log.Lshortfile)

	log.Printf("Starting %s", os.Args[0])

	var options options

	flag.StringVar(&options.WorkingDir, "WorkingDir", ".", "Working directory, file search path is relative to this (default '.')")
	flag.StringVar(&options.BindAddr, "BindAddr", "localhost:9000", "Ip:Port to listen on (default 'localhost:9000')")
	flag.StringVar(&options.WsUrl, "WsUrl", "ws://localhost:9000/", "Web socket base url (default 'ws://localhost:9000/')")

	flag.Parse()

	err := os.Chdir(options.WorkingDir)
	if err != nil {
		panic(err)
	}

	http.Handle("/echo_service", ws.WsHandlerFunc(options.wsHandler))
	http.HandleFunc("/", options.httpHandler)
	err = http.ListenAndServe(options.BindAddr, nil)
	if err != nil {
		log.Print(err)
	}

}
