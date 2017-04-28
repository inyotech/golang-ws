package main

import (
	"fmt"
	"net/http"
	"html/template"
	"time"
	"github.com/inyotech/golang-ws/wsio"

)

func httpHandler(response http.ResponseWriter, request *http.Request) {
	t, err := template.ParseFiles("www/index.html")
	if err != nil {
		panic(err)
	}

	t.Execute(response, nil)
}

func handlews1(ch chan *wsio.Frame) {

	for {
		frame, ws_ok := <-ch
		if !ws_ok {
			fmt.Println("handlews1 channel closed")
			return
		}
		fmt.Println(string(frame.Payload))
		frame.Payload = append([]byte("handlews1 "), frame.Payload...)
		ch<-frame
	}
}

func handlews2(ch chan *wsio.Frame) {

	for i:=0;i<10;i++ {
		select {
		case frame, ws_ok := <-ch:
			if !ws_ok {
				fmt.Println("handlews2 channel closed")
				return
			}
			fmt.Println(string(frame.Payload))
		case <-time.After(time.Second):
			frame := wsio.NewTextFrame("message from handlerws2")
			fmt.Println(string(frame.Payload))
			ch<-frame
		}
	}
	close(ch)
}


func main() {

	http.Handle("/service1", wsio.WsHandlerFunc(handlews1))
	http.Handle("/service2", wsio.WsHandlerFunc(handlews2))
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)

}
