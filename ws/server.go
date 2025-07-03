package ws

import (
	"net/http"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin (for development)
		// In production, you should validate the origin
		return true
	},
}

type UserClient[T any] struct {
	conn *websocket.Conn
	send chan []byte
	data T
}

func (c *UserClient[T]) Send(message []byte) {
	c.send <- message
}

func (c *UserClient[T]) writePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Error().Err(err).Msg("WriteMessage error")
			return
		}
	}
}

func (c *UserClient[T]) readPump(hubManager HubManager[T]) {
	hub := hubManager.GetHub()
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if err := hubManager.OnReceiveMessage(c, messageBytes); err != nil {
			log.Warn().Err(err).Msg("HandleMessage error")
			continue
		}
	}
}

type HubManager[T any] interface {
	OnReceiveMessage(client *UserClient[T], message []byte) error
	OnRegister(client *UserClient[T]) error
	OnUnregister(client *UserClient[T]) error
	GetHub() *Hub[T]
}

func ConnectSocket[T any](hubManager HubManager[T], w http.ResponseWriter, r *http.Request, initialData T) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := NewUserClient(conn, initialData)

	hubManager.GetHub().register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump(hubManager)
}

type Hub[T any] struct {
	clients    mapset.Set[*UserClient[T]]
	broadcast  chan []byte
	register   chan *UserClient[T]
	unregister chan *UserClient[T]
}

func (h *Hub[T]) Broadcast(message []byte) {
	h.clients.Each(func(client *UserClient[T]) bool {
		client.Send(message)
		return true
	})
}

var ServerHub *Hub[any]

func NewUserClient[T any](conn *websocket.Conn, data T) *UserClient[T] {
	return &UserClient[T]{
		conn: conn,
		send: make(chan []byte, 256),
		data: data,
	}
}

func Run[T any](hubManager HubManager[T]) {
	c := hubManager.GetHub()
	for {
		select {
		case client := <-c.register:
			c.clients.Add(client)
			log.Info().Int("Client Count", c.clients.Cardinality()).Msg("Client connected")
			if err := hubManager.OnRegister(client); err != nil {
				log.Error().Err(err).Msg("OnRegister error")
			}
		case client := <-c.unregister:
			if c.clients.Contains(client) {
				c.clients.Remove(client)
				close(client.send)
				log.Info().Int("Client Count", c.clients.Cardinality()).Msg("Client disconnected")
				if err := hubManager.OnUnregister(client); err != nil {
					log.Error().Err(err).Msg("OnUnregister error")
				}
			}
		case message := <-c.broadcast:
			c.Broadcast(message)
		}
	}
}

func NewHub[T any]() *Hub[T] {
	return &Hub[T]{
		clients:    mapset.NewSet[*UserClient[T]](),
		broadcast:  make(chan []byte),
		register:   make(chan *UserClient[T]),
		unregister: make(chan *UserClient[T]),
	}
}

func Init() {
	ServerHub = &Hub[any]{
		clients:    mapset.NewSet[*UserClient[any]](),
		broadcast:  make(chan []byte),
		register:   make(chan *UserClient[any]),
		unregister: make(chan *UserClient[any]),
	}

}
