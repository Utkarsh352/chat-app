package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

var upgrader = websocket.New(upgraderConfig)

var hub = newHub()

func upgraderConfig(c *websocket.Conn) bool {
	return true
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

func handleConnection(c *websocket.Conn) {
	client := &Client{
		conn: c,
		send: make(chan []byte),
	}
	hub.register(client)

	go client.readMessages()
	client.writeMessages()
}

func (c *Client) readMessages() {
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
	app := fiber.New()

	app.Static("/", "./static")

	app.Get("/ws", websocket.New(handleConnection))

	app.Get("/", func(c *fiber.Ctx) error {
		url := os.Getenv("RAILWAY_URL")
		if url == "" {
			url = "http://localhost:8080"
		}
		return c.SendString(fmt.Sprintf("Chat app is running at %s", url))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server started on :%s\n", port)
	log.Fatal(app.Listen(":" + port))
}
