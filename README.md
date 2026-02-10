# WebRTC 打洞系统 MVP

这是一个基于WebRTC的NAT穿透系统，包含三个组件：
- **Backend**: Go + WebSocket信令服务器（部署在公网）
- **Device**: Go + Pion WebRTC后端服务（内网设备，提供API服务）
- **Frontend**: React + WebRTC浏览器客户端

## 架构说明

```
Frontend (浏览器) <---> Backend (信令服务器) <---> Device (后端服务)
                              |
                         WebSocket信令
                         (交换SDP/ICE)
                              |
                    Frontend <==WebRTC P2P==> Device
                         (直接请求Device API)
```

**核心流程：**
1. Backend作为信令服务器，帮助Frontend和Device交换SDP和ICE候选
2. 通过STUN服务器实现NAT穿透，建立P2P连接
3. **打洞成功后，Frontend通过WebRTC数据通道直接请求Device的API服务**
4. Device作为后端服务，处理API请求并返回响应（无需经过Backend中转）

## 快速开始

### 1. 启动Backend信令服务器

```bash
cd backend
go mod download
go run main.go
```

服务器将在 `http://8.138.247.82:8888` 启动

### 2. 启动Device客户端

```bash
cd device
go mod download
go run main.go -id device-1 -server ws://8.138.247.82:8888/ws
```

参数说明：
- `-id`: 设备ID（默认: device-1）
- `-server`: 信令服务器地址（默认: ws://8.138.247.82:8888/ws）

### 3. 启动Frontend

```bash
cd frontend
npm install
npm run dev
```

浏览器访问 `http://localhost:3000`

## 使用步骤

1. 确保Backend和Device都在运行
2. 在浏览器中打开Frontend
3. 点击"连接信令服务器"按钮
4. 输入设备ID（默认是 device-1）
5. 点击"连接设备"按钮
6. 等待WebRTC连接建立
7. **连接成功后，可以直接调用Device的API服务：**
   - 点击"获取设备信息"查看设备状态
   - 点击"获取时间"获取设备时间
   - 点击"获取传感器数据"获取模拟的传感器数据
   - 点击"Echo测试"测试请求-响应机制
8. 所有API请求都通过WebRTC数据通道直接发送到Device，无需经过Backend中转

## 公网部署

### Backend部署到公网服务器

1. 在公网服务器上运行Backend：
```bash
go run main.go
```

2. 确保防火墙开放8888端口

3. Device连接时使用公网地址：
```bash
go run main.go -server ws://your-server-ip:8888/ws
```

4. Frontend修改WebSocket地址（在 `src/App.jsx` 中）：
```javascript
const ws = new WebSocket(`ws://your-server-ip:8888/ws?id=${clientId.current}&type=frontend`)
```

## API接口

### Device API端点（通过WebRTC数据通道）

打洞成功后，Frontend可以直接调用Device的以下API：

**请求格式：**
```json
{
  "id": "request-id",
  "method": "GET",
  "path": "/info|/time|/data|/echo",
  "params": {}
}
```

**响应格式：**
```json
{
  "id": "request-id",
  "status": 200,
  "data": {},
  "error": ""
}
```

**可用端点：**

1. **GET /info** - 获取设备信息
   ```json
   {
     "device_id": "device-1",
     "status": "online",
     "type": "IoT Device",
     "version": "1.0.0"
   }
   ```

2. **GET /time** - 获取设备时间
   ```json
   {
     "timestamp": 1234567890,
     "datetime": "2024-01-01T00:00:00Z"
   }
   ```

3. **GET /data** - 获取传感器数据
   ```json
   {
     "temperature": 23.5,
     "humidity": 65.2,
     "pressure": 1013.25
   }
   ```

4. **GET /echo** - 回显测试
   - 请求参数会被原样返回

### WebSocket信令协议

连接: `ws://8.138.247.82:8888/ws?id=<client-id>&type=<device|frontend>`

消息格式：
```json
{
  "type": "offer|answer|candidate",
  "from": "sender-id",
  "to": "receiver-id",
  "data": "JSON字符串"
}
```

### HTTP接口

- `GET /clients`: 获取当前连接的客户端列表

## 技术栈

- **Backend**: Go 1.21, gorilla/websocket, pion/webrtc
- **Device**: Go 1.21, pion/webrtc
- **Frontend**: React 18, Vite, WebRTC API

## 注意事项

1. 本项目是MVP版本，生产环境需要添加：
   - 身份验证和授权
   - HTTPS/WSS加密传输
   - TURN服务器（用于对称NAT穿透）
   - 错误处理和重连机制
   - 日志和监控

2. STUN服务器使用的是Google公共服务器，生产环境建议自建

3. 如果NAT类型为对称NAT，可能需要TURN服务器中转

## 故障排查

1. **连接失败**: 检查Backend是否运行，防火墙是否开放
2. **WebRTC连接超时**: 检查STUN服务器是否可达，考虑添加TURN服务器
3. **消息无法发送**: 检查数据通道是否已打开（状态显示"已连接到设备"）

## 许可证

MIT
