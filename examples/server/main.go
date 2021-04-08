package main

import (
	"fmt"
	"log"
	"net/http"

	"nhooyr.io/websocket"
)

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home Page")
}

func wsEndpoint(w http.ResponseWriter, r *http.Request, room *Room) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	websocketConnection, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		log.Println(err)
		return
	}

	client, err := newClient(NewConnection(websocketConnection, r))
	if err != nil {
		log.Println(err)
		return
	}
	client.assignToRoom(room)
	room.registerChannel <- client

	go client.runReadMessages()
	go client.runWriteMessages()

	err = client.conn.WriteMessage([]byte("hello"))
	if err != nil {
		log.Println(err)
	}

	// wait until client disconnects
	<-r.Context().Done()
}

func setupRoutes() {
	room := newRoom()
	room.Deploy()

	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { wsEndpoint(w, r, room) })
}

func main() {
	log.Println("Hello World")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
