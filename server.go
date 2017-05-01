package main

import (
	"net/http"
	"html/template"
	"github.com/inyotech/golang-ws/ws"

)

func httpHandler(response http.ResponseWriter, request *http.Request) {
	t, err := template.ParseFiles("www/index.html")
	if err != nil {
		panic(err)
	}

	t.Execute(response, nil)
}

func handlews(ch chan *ws.Frame) {

	for {
		frame, ws_ok := <-ch
		if !ws_ok {
			return
		}
		frame.Payload = append([]byte("received: "), frame.Payload...)
		ch<-frame
	}
}

func main() {

	http.Handle("/service", ws.WsHandlerFunc(handlews))
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)

}
