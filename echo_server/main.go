package main

import (
	"fmt"
	"os"
	"flag"
	"net/http"
	"html/template"

	"github.com/inyotech/golang-ws/ws"
)

func (options options) httpHandler(response http.ResponseWriter, request *http.Request) {

	t, err := template.ParseFiles("template/index.html")
	if err != nil {
		panic(err)
	}

	t.Execute(response, options)
}

func (options options) wsHandler(ch chan *ws.Frame) {

	fmt.Println(options)

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

	var options options

	flag.StringVar(&options.WorkingDir, "WorkingDir", ".", "Working directory, file search path is relative to this (default '.')")
	flag.StringVar(&options.BindAddr, "BindAddr", "localhost:9000", "Ip:Port to listen on (default 'localhost:9000')")
	flag.StringVar(&options.WsUrl, "WsUrl", "ws://localhost:9000/", "Web socket base url (default 'ws://localhost:9000/')")

	flag.Parse()

	fmt.Println(options)

	err := os.Chdir(options.WorkingDir)
	if err != nil {
		panic(err)
	}

	http.Handle("/echo_service", ws.WsHandlerFunc(options.wsHandler))
	http.HandleFunc("/", options.httpHandler)
	http.ListenAndServe(options.BindAddr, nil)

}
