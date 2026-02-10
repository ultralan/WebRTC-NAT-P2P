package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Message struct {
	Type string          `json:"type"`
	From string          `json:"from"`
	To   string          `json:"to"`
	Data json.RawMessage `json:"data"`
}

// API请求和响应结构
type APIRequest struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Path   string                 `json:"path"`
	Params map[string]interface{} `json:"params"`
}

type APIResponse struct {
	ID     string      `json:"id"`
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error,omitempty"`
}

// HTTP 网关请求和响应结构
type HTTPGatewayRequest struct {
	RequestID string            `json:"requestId"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
}

type HTTPGatewayResponse struct {
	RequestID  string            `json:"requestId"`
	Status     int               `json:"status"`
	StatusText string            `json:"statusText"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
}

var (
	serverURL = flag.String("server", "ws://8.138.247.82:8888/ws", "Signaling server URL")
	deviceID  = flag.String("id", "device-1", "Device ID")
	wsMutex   sync.Mutex // 保护 WebSocket 写入操作
)

func main() {
	flag.Parse()

	// 连接到信令服务器
	wsURL := *serverURL + "?id=" + *deviceID + "&type=device"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer conn.Close()

	log.Printf("Connected to signaling server as %s\n", *deviceID)

	// 创建WebRTC配置
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// 在goroutine中处理信令消息
	go func() {
		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				log.Println("Signaling connection closed:", err)
				log.Println("WebRTC P2P connection will continue working...")
				break
			}

			switch msg.Type {
			case "offer":
				go handleOffer(conn, &msg, config)
	case "proxy_request":
			go handleProxyRequest(conn, &msg)
			}
		}
	}()

	// 保持程序运行，让WebRTC连接继续工作
	log.Println("Device is running. Press Ctrl+C to exit.")
	select {}
}

func handleOffer(conn *websocket.Conn, msg *Message, config webrtc.Configuration) {
	// 创建PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Println("Failed to create peer connection:", err)
		return
	}

	// 监听数据通道（由frontend创建）
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("Data channel '%s' received from frontend\n", d.Label())

		d.OnOpen(func() {
			log.Println("Data channel opened - Device backend ready to handle requests")
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			// 先尝试解析为通用的 JSON 对象，判断请求类型
			var rawRequest map[string]interface{}
			err := json.Unmarshal(msg.Data, &rawRequest)
			if err != nil {
				log.Printf("Failed to parse request: %s\n", err)
				return
			}

			// 判断是 HTTP 网关请求还是旧的 API 请求
			if requestID, ok := rawRequest["requestId"].(string); ok && requestID != "" {
				// HTTP 网关请求
				var httpReq HTTPGatewayRequest
				err := json.Unmarshal(msg.Data, &httpReq)
				if err != nil {
					log.Printf("Failed to parse HTTP gateway request: %s\n", err)
					return
				}

				log.Printf("Received HTTP gateway request: %s %s\n", httpReq.Method, httpReq.URL)

				// 处理 HTTP 网关请求
				resp := handleHTTPGatewayRequest(&httpReq)

				// 发送响应
				respJSON, err := json.Marshal(resp)
				if err != nil {
					log.Printf("Failed to marshal HTTP gateway response: %s\n", err)
					return
				}

				d.Send(respJSON)
			} else {
				// 旧的 API 请求（保持兼容）
				var req APIRequest
				err := json.Unmarshal(msg.Data, &req)
				if err != nil {
					log.Printf("Failed to parse API request: %s\n", err)
					return
				}

				log.Printf("Received API request: %s %s\n", req.Method, req.Path)

				// 处理请求
				resp := handleAPIRequest(&req)

				// 发送响应
				respJSON, err := json.Marshal(resp)
				if err != nil {
					log.Printf("Failed to marshal API response: %s\n", err)
					return
				}

				d.Send(respJSON)
			}
		})

		d.OnClose(func() {
			log.Println("Data channel closed")
		})
	})

	// 处理ICE候选
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}

		candidateJSON, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			log.Println("Failed to marshal candidate:", err)
			return
		}

		sendMessage(conn, Message{
			Type: "candidate",
			To:   msg.From,
			Data: candidateJSON,
		})
	})

	// 设置远程描述
	var offer webrtc.SessionDescription
	err = json.Unmarshal(msg.Data, &offer)
	if err != nil {
		log.Println("Failed to unmarshal offer:", err)
		return
	}

	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		log.Println("Failed to set remote description:", err)
		return
	}

	// 创建应答
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		log.Println("Failed to create answer:", err)
		return
	}

	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		log.Println("Failed to set local description:", err)
		return
	}

	// 发送应答
	answerJSON, err := json.Marshal(answer)
	if err != nil {
		log.Println("Failed to marshal answer:", err)
		return
	}

	sendMessage(conn, Message{
		Type: "answer",
		To:   msg.From,
		Data: answerJSON,
	})

	// 监听ICE候选消息
	go listenForCandidates(conn, peerConnection, msg.From)
}

func listenForCandidates(conn *websocket.Conn, pc *webrtc.PeerConnection, fromID string) {
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			return
		}

		if msg.Type == "candidate" && msg.From == fromID {
			var candidate webrtc.ICECandidateInit
			err = json.Unmarshal(msg.Data, &candidate)
			if err != nil {
				log.Println("Failed to unmarshal candidate:", err)
				continue
			}

			err = pc.AddICECandidate(candidate)
			if err != nil {
				log.Println("Failed to add ICE candidate:", err)
			}
		}
	}
}

func sendMessage(conn *websocket.Conn, msg Message) {
	wsMutex.Lock()
	defer wsMutex.Unlock()

	err := conn.WriteJSON(msg)
	if err != nil {
		log.Println("Write error:", err)
	}
	time.Sleep(100 * time.Millisecond)
}

// 处理API请求
func handleAPIRequest(req *APIRequest) APIResponse {
	resp := APIResponse{
		ID:     req.ID,
		Status: 200,
	}

	switch req.Path {
	case "/info":
		resp.Data = map[string]interface{}{
			"device_id": *deviceID,
			"status":    "online",
			"type":      "IoT Device",
			"version":   "1.0.0",
		}

	case "/time":
		resp.Data = map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"datetime":  time.Now().Format(time.RFC3339),
		}

	case "/data":
		// 模拟设备数据
		resp.Data = map[string]interface{}{
			"temperature": 23.5,
			"humidity":    65.2,
			"pressure":    1013.25,
		}

	case "/echo":
		// 回显请求参数
		resp.Data = req.Params

	default:
		resp.Status = 404
		resp.Error = "Endpoint not found"
	}

	return resp
}

// 处理 HTTP 网关请求
func handleHTTPGatewayRequest(req *HTTPGatewayRequest) HTTPGatewayResponse {
	resp := HTTPGatewayResponse{
		RequestID:  req.RequestID,
		Status:     200,
		StatusText: "OK",
		Headers:    make(map[string]string),
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 构建 HTTP 请求
	var httpReq *http.Request
	var err error

	if req.Body != "" {
		httpReq, err = http.NewRequest(req.Method, req.URL, bytes.NewBufferString(req.Body))
	} else {
		httpReq, err = http.NewRequest(req.Method, req.URL, nil)
	}

	if err != nil {
		resp.Status = 500
		resp.StatusText = "Internal Server Error"
		resp.Body = map[string]string{"error": "Failed to create HTTP request: " + err.Error()}
		return resp
	}

	// 设置请求头
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// 发送 HTTP 请求
	log.Printf("Sending HTTP request: %s %s\n", req.Method, req.URL)
	httpResp, err := client.Do(httpReq)
	if err != nil {
		resp.Status = 502
		resp.StatusText = "Bad Gateway"
		resp.Body = map[string]string{"error": "Failed to send HTTP request: " + err.Error()}
		return resp
	}
	defer httpResp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Status = 500
		resp.StatusText = "Internal Server Error"
		resp.Body = map[string]string{"error": "Failed to read response body: " + err.Error()}
		return resp
	}

	// 设置响应状态
	resp.Status = httpResp.StatusCode
	resp.StatusText = httpResp.Status

	// 复制响应头
	for key := range httpResp.Header {
		resp.Headers[key] = httpResp.Header.Get(key)
	}

	// 尝试解析 JSON 响应
	var jsonBody interface{}
	if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
		resp.Body = jsonBody
	} else {
		// 如果不是 JSON，返回字符串
		resp.Body = string(bodyBytes)
	}

	log.Printf("HTTP response: %d %s\n", resp.Status, resp.StatusText)
	return resp
}

// 处理通过 WebSocket 接收的 HTTP 代理请求
func handleProxyRequest(conn *websocket.Conn, msg *Message) {
	// 解析代理请求
	var proxyReq HTTPGatewayRequest
	err := json.Unmarshal(msg.Data, &proxyReq)
	if err != nil {
		log.Printf("Failed to parse proxy request: %s\n", err)
		return
	}

	log.Printf("Received proxy request via WebSocket: %s %s\n", proxyReq.Method, proxyReq.URL)

	// 使用现有的 HTTP 网关处理函数
	httpResp := handleHTTPGatewayRequest(&proxyReq)

	// 构建响应消息
	respData, err := json.Marshal(httpResp)
	if err != nil {
		log.Printf("Failed to marshal proxy response: %s\n", err)
		return
	}

	// 发送响应回 backend
	responseMsg := Message{
		Type: "proxy_response",
		From: *deviceID,
		To:   msg.From,
		Data: respData,
	}

	err = conn.WriteJSON(responseMsg)
	if err != nil {
		log.Printf("Failed to send proxy response: %s\n", err)
		return
	}

	log.Printf("Proxy response sent back to backend for request: %s\n", proxyReq.RequestID)
}


