package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（生产环境需要限制）
	},
}

type Client struct {
	conn     *websocket.Conn
	clientID string
	clientType string // "device" or "frontend"
}

type Message struct {
	Type     string          `json:"type"`
	From     string          `json:"from"`
	To       string          `json:"to"`
	Data     json.RawMessage `json:"data"`
}

type SignalingServer struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

func NewSignalingServer() *SignalingServer {
	return &SignalingServer{
		clients: make(map[string]*Client),
	}
}

func (s *SignalingServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	clientID := r.URL.Query().Get("id")
	clientType := r.URL.Query().Get("type")

	if clientID == "" || clientType == "" {
		log.Println("Missing id or type parameter")
		return
	}

	client := &Client{
		conn:       conn,
		clientID:   clientID,
		clientType: clientType,
	}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	log.Printf("Client connected: %s (type: %s)\n", clientID, clientType)

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
		log.Printf("Client disconnected: %s\n", clientID)
	}()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		msg.From = clientID
		s.routeMessage(&msg)
	}
}

func (s *SignalingServer) routeMessage(msg *Message) {
	s.mu.RLock()
	targetClient, exists := s.clients[msg.To]
	s.mu.RUnlock()

	if !exists {
		log.Printf("Target client not found: %s\n", msg.To)
		return
	}

	err := targetClient.conn.WriteJSON(msg)
	if err != nil {
		log.Println("Write error:", err)
	}
}

func (s *SignalingServer) handleListClients(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clientList := make([]map[string]string, 0)
	for id, client := range s.clients {
		clientList = append(clientList, map[string]string{
			"id":   id,
			"type": client.clientType,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(clientList)
}

func main() {
	server := NewSignalingServer()

	http.HandleFunc("/ws", server.handleWebSocket)
	http.HandleFunc("/clients", server.handleListClients)

	log.Println("Signaling server started on :8888")
	log.Fatal(http.ListenAndServe(":8888", nil))
}
