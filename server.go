package main

import (
	"fmt"
	"net/http"
	"html/template"
	"inyotech/ws/wsio"

)

func httpHandler(response http.ResponseWriter, request *http.Request) {
	t, err := template.ParseFiles("www/index.html")
	if err != nil {
		panic(err)
	}

	t.Execute(response, nil)
}

type wshandlerfunc func(chan *wsio.Frame)

func handlews(c chan *wsio.Frame) {

	for {
		frame := <-c
		fmt.Println("handlews", string(frame.Payload))
	}
}

func (clientFunc wshandlerfunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	fmt.Println("wsHandler")

	conn, frameReadWriter, err := wsio.SetupConnection(w, r)
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	wsio.DispatchFrames(clientFunc, frameReadWriter)

}

func main() {

	http.Handle("/service", wshandlerfunc(handlews))
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)

}
