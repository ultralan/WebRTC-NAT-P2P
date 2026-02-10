package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

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

// HTTP 转发请求结构
type ProxyRequest struct {
	RequestID string            `json:"requestId"`
	DeviceID  string            `json:"deviceId"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
}

// HTTP 转发响应结构
type ProxyResponse struct {
	RequestID  string            `json:"requestId"`
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
}

// 待处理的代理请求
type PendingProxyRequest struct {
	ResponseChan chan ProxyResponse
	Timeout      time.Time
}

type SignalingServer struct {
	clients       map[string]*Client
	pendingProxy  map[string]*PendingProxyRequest
	mu            sync.RWMutex
}

func NewSignalingServer() *SignalingServer {
	return &SignalingServer{
		clients:      make(map[string]*Client),
		pendingProxy: make(map[string]*PendingProxyRequest),
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
	// 检查是否是 HTTP 代理响应
	if msg.Type == "proxy_response" {
		var proxyResp ProxyResponse
		err := json.Unmarshal(msg.Data, &proxyResp)
		if err != nil {
			log.Printf("Failed to parse proxy response: %s\n", err)
			return
		}

		// 查找待处理的请求
		s.mu.Lock()
		pending, exists := s.pendingProxy[proxyResp.RequestID]
		if exists {
			delete(s.pendingProxy, proxyResp.RequestID)
		}
		s.mu.Unlock()

		if exists {
			// 发送响应到等待的 HTTP 处理器
			select {
			case pending.ResponseChan <- proxyResp:
				log.Printf("Proxy response sent for request: %s\n", proxyResp.RequestID)
			default:
				log.Printf("Failed to send proxy response (channel closed): %s\n", proxyResp.RequestID)
			}
		} else {
			log.Printf("No pending request found for: %s\n", proxyResp.RequestID)
		}
		return
	}

	// 普通消息路由
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

// 处理 HTTP 代理请求
func (s *SignalingServer) handleProxy(w http.ResponseWriter, r *http.Request) {
	// 设置 CORS 头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var proxyReq ProxyRequest
	err := json.NewDecoder(r.Body).Decode(&proxyReq)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 检查设备是否在线
	s.mu.RLock()
	deviceClient, exists := s.clients[proxyReq.DeviceID]
	s.mu.RUnlock()

	if !exists {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// 创建响应通道
	responseChan := make(chan ProxyResponse, 1)
	pending := &PendingProxyRequest{
		ResponseChan: responseChan,
		Timeout:      time.Now().Add(30 * time.Second),
	}

	// 存储待处理请求
	s.mu.Lock()
	s.pendingProxy[proxyReq.RequestID] = pending
	s.mu.Unlock()

	// 构建转发消息
	proxyReqData, err := json.Marshal(proxyReq)
	if err != nil {
		s.mu.Lock()
		delete(s.pendingProxy, proxyReq.RequestID)
		s.mu.Unlock()
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	msg := Message{
		Type: "proxy_request",
		To:   proxyReq.DeviceID,
		Data: proxyReqData,
	}

	// 发送请求到设备
	err = deviceClient.conn.WriteJSON(msg)
	if err != nil {
		s.mu.Lock()
		delete(s.pendingProxy, proxyReq.RequestID)
		s.mu.Unlock()
		http.Error(w, "Failed to send request to device", http.StatusBadGateway)
		return
	}

	log.Printf("Proxy request sent to device %s: %s %s\n", proxyReq.DeviceID, proxyReq.Method, proxyReq.URL)

	// 等待响应（带超时）
	select {
	case proxyResp := <-responseChan:
		// 返回响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(proxyResp)
		log.Printf("Proxy response returned for request: %s\n", proxyReq.RequestID)

	case <-time.After(30 * time.Second):
		// 超时
		s.mu.Lock()
		delete(s.pendingProxy, proxyReq.RequestID)
		s.mu.Unlock()
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		log.Printf("Proxy request timeout: %s\n", proxyReq.RequestID)
	}
}

func main() {
	server := NewSignalingServer()

	http.HandleFunc("/ws", server.handleWebSocket)
	http.HandleFunc("/clients", server.handleListClients)
	http.HandleFunc("/proxy", server.handleProxy)

	log.Println("Signaling server started on :8888")
	log.Fatal(http.ListenAndServe(":8888", nil))
}
