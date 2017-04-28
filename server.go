package main

import (
	"fmt"
	"net/http"
	"html/template"
	"time"
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

func handlews1(c chan *wsio.Frame) {

	for {
		frame, ws_ok := <-c
		if !ws_ok {
			fmt.Println("handlews1 closed")
			return
		}
		fmt.Println("handlews1", string(frame.Payload))
		frame.Payload = append([]byte("handlews1 "), frame.Payload...)
		c<-frame
	}
}

func handlews2(c chan *wsio.Frame) {

	for i:=0;i<3;i++ {
		select {
		case <-c:
		default:
		}
		frame := wsio.NewTextFrame("message from handlerws2")
		c<-frame
		time.Sleep(time.Second)
		fmt.Println("after sleep")
	}
	fmt.Println("closing ws2")
	close(c)
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

	http.Handle("/service1", wshandlerfunc(handlews1))
	http.Handle("/service2", wshandlerfunc(handlews2))
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)

}
