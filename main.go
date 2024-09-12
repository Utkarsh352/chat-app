package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	clients map[*Client]bool
	lock    sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
	}
}

func (h *Hub) register(client *Client) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.clients[client] = true
}

func (h *Hub) unregister(client *Client) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

func (h *Hub) broadcast(message []byte) {
	h.lock.Lock()
	defer h.lock.Unlock()
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			h.unregister(client)
		}
	}
}

func handleConnection(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error while upgrading connection:", err)
		return
	}
	client := &Client{
		conn: conn,
		send: make(chan []byte),
	}
	hub.register(client)

	go client.readMessages(hub)
	client.writeMessages()
}

func (c *Client) readMessages(hub *Hub) {
	defer func() {
		hub.unregister(c)
		c.conn.Close()
	}()
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		hub.broadcast(msg)
	}
}

func (c *Client) writeMessages() {
	for msg := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			break
		}
	}
}

func main() {
	hub := newHub()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleConnection(hub, w, r)
	})
	http.Handle("/", http.FileServer(http.Dir("./static")))

	port := "8080"
	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
