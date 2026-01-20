package main

import (
	"encoding/json"
	"flag"
	"log"
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

var (
	serverURL = flag.String("server", "ws://localhost:8080/ws", "Signaling server URL")
	deviceID  = flag.String("id", "device-1", "Device ID")
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

	// 处理消息
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		switch msg.Type {
		case "offer":
			go handleOffer(conn, &msg, config)
		}
	}
}

func handleOffer(conn *websocket.Conn, msg *Message, config webrtc.Configuration) {
	// 创建PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Println("Failed to create peer connection:", err)
		return
	}

	// 创建数据通道
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		log.Println("Failed to create data channel:", err)
		return
	}

	dataChannel.OnOpen(func() {
		log.Println("Data channel opened - Device backend ready to handle requests")
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		// 解析API请求
		var req APIRequest
		err := json.Unmarshal(msg.Data, &req)
		if err != nil {
			log.Printf("Failed to parse request: %s\n", err)
			return
		}

		log.Printf("Received API request: %s %s\n", req.Method, req.Path)

		// 处理请求
		resp := handleAPIRequest(&req)

		// 发送响应
		respJSON, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Failed to marshal response: %s\n", err)
			return
		}

		dataChannel.Send(respJSON)
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
