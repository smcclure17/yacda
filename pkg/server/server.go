package server

import (
	"context"
	"fmt"
	"time"

	"github.com/smcclure17/writr/pkg/cache"
	"github.com/smcclure17/writr/pkg/models"

	"golang.org/x/net/websocket"
)

// Server is the main server instance.
type Server struct {
	clients map[*websocket.Conn]string // Mapping of websocket connections to document names
	cache   cache.Cache                // In-memory cache of messages
}

// NewServer creates a new server instance
func NewServer() *Server {
	return &Server{
		clients: make(map[*websocket.Conn]string),
		cache:   *cache.NewCache(),
	}
}

// HandleMessages reads messages from clients and broadcast to all appropriate clients
func (s *Server) HandleMessages(ws *websocket.Conn) {
	documentName := s.clients[ws]
	buf := make([]byte, 1024)
	for {
		n, err := ws.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			fmt.Println("Error handling message: ", err)
			break
		}

		msg := models.CreateMessage(string(buf[:n]), documentName)
		go s.BroadcastMessage(msg)
	}
}

// BroadcastMessage sends a message to all clients connected to the same document
func (s *Server) BroadcastMessage(msg models.Message) {
	for client := range s.clients {
		if s.clients[client] == msg.Document {
			client.Write([]byte(msg.Message))
		}
	}

	// Save document to cache for 1 minute
	s.cache.Client.Set(context.Background(), msg.Document, msg.Message, time.Minute)
}

// HandleConnections handles new websocket connections
func (s *Server) HandleConnections(ws *websocket.Conn) {
	params := ws.Request().URL.Query()
	documentName := params.Get("document")
	s.clients[ws] = documentName

	// Load document from cache and send to client on connection
	redisMessage := s.cache.GetCacheMessage(documentName)
	msg := models.CreateMessage(redisMessage, documentName)
	ws.Write([]byte(msg.Message))

	s.HandleMessages(ws)
}