import { useState, useEffect, useRef } from 'react'
import dataChannelFetch from './dataChannelFetch'

function App() {
  const [connected, setConnected] = useState(false)
  const [deviceId, setDeviceId] = useState('device-1')
  const [messages, setMessages] = useState([])
  const [inputMessage, setInputMessage] = useState('')
  const [status, setStatus] = useState('未连接')
  const [apiResponse, setApiResponse] = useState(null)
  const [loading, setLoading] = useState(false)

  const wsRef = useRef(null)
  const pcRef = useRef(null)
  const dataChannelRef = useRef(null)
  const clientId = useRef('frontend-' + Math.random().toString(36).substr(2, 9))

  useEffect(() => {
    // 自动连接信令服务器和设备
    connectToSignaling()

    return () => {
      if (wsRef.current) {
        wsRef.current.close()
      }
      if (pcRef.current) {
        pcRef.current.close()
      }
    }
  }, [])

  const connectToSignaling = () => {
    // 避免重复连接
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      console.log('Already connected to signaling server')
      return
    }

    const ws = new WebSocket(`ws://8.138.247.82:8888/ws?id=${clientId.current}&type=frontend`)

    ws.onopen = () => {
      setStatus('已连接到信令服务器')
      wsRef.current = ws

      // 自动连接设备
      setTimeout(() => {
        connectToDevice()
      }, 500)
    }

    ws.onmessage = async (event) => {
      const msg = JSON.parse(event.data)

      if (msg.type === 'answer') {
        const answer = msg.data
        // 检查 PeerConnection 状态，避免重复设置
        if (pcRef.current && pcRef.current.signalingState === 'have-local-offer') {
          await pcRef.current.setRemoteDescription(answer)
          setStatus('正在建立数据通道...')
        } else {
          console.log('Ignoring answer, wrong state:', pcRef.current?.signalingState)
        }
      } else if (msg.type === 'candidate') {
        const candidate = msg.data
        if (pcRef.current) {
          await pcRef.current.addIceCandidate(candidate)
        }
      }
    }

    ws.onerror = (error) => {
      setStatus('连接错误: ' + error.message)
    }

    ws.onclose = () => {
      setStatus('信令服务器已断开 (WebRTC P2P连接仍然工作)')
    }
  }

  const connectToDevice = async () => {
    // 避免重复连接
    if (pcRef.current && pcRef.current.connectionState !== 'closed') {
      console.log('Already connecting/connected to device')
      return
    }

    setStatus('正在连接设备...')

    // 创建PeerConnection
    const config = {
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
    }
    const pc = new RTCPeerConnection(config)
    pcRef.current = pc

    // 创建数据通道
    const dataChannel = pc.createDataChannel('data')
    dataChannelRef.current = dataChannel

    console.log('DataChannel created, readyState:', dataChannel.readyState)

    dataChannel.onopen = () => {
      console.log('DataChannel opened!')
      setConnected(true)
      setStatus('已连接到设备')
      addMessage('系统', '数据通道已打开')

      // 设置 dataChannelFetch 的 dataChannel
      dataChannelFetch.setDataChannel(dataChannel)
      addMessage('系统', 'DataChannel Fetch 已就绪')
    }

    dataChannel.onmessage = (event) => {
      try {
        const response = JSON.parse(event.data)

        // 检查是否是 HTTP 网关响应（有 requestId 字段）
        if (response.requestId) {
          // HTTP 网关响应，由 dataChannelFetch 处理，这里不处理
          return
        }

        if (response.id) {
          // 旧的 API 响应
          setApiResponse(response)
          setLoading(false)
          try {
            addMessage('API响应', `${response.status} - ${JSON.stringify(response.data)}`)
          } catch (e) {
            addMessage('API响应', `${response.status} - [数据无法序列化]`)
          }
        } else {
          // 普通消息
          const message = event.data instanceof ArrayBuffer
            ? `[二进制数据: ${event.data.byteLength} 字节]`
            : String(event.data)
          addMessage('设备', message)
        }
      } catch (e) {
        const message = event.data instanceof ArrayBuffer
          ? `[二进制数据: ${event.data.byteLength} 字节]`
          : String(event.data)
        addMessage('设备', message)
      }
    }

    dataChannel.onerror = (error) => {
      console.error('DataChannel error:', error)
      setStatus('数据通道错误')
      addMessage('错误', '数据通道连接失败')
    }

    dataChannel.onclose = () => {
      setConnected(false)
      setStatus('数据通道已关闭')
    }

    // 处理ICE候选
    pc.onicecandidate = (event) => {
      if (event.candidate && wsRef.current) {
        wsRef.current.send(JSON.stringify({
          type: 'candidate',
          to: deviceId,
          data: event.candidate.toJSON()
        }))
      }
    }

    // 创建offer
    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)

    // 发送offer到设备
    wsRef.current.send(JSON.stringify({
      type: 'offer',
      to: deviceId,
      data: offer
    }))
  }

  const sendMessage = () => {
    if (dataChannelRef.current && inputMessage.trim()) {
      dataChannelRef.current.send(inputMessage)
      addMessage('我', inputMessage)
      setInputMessage('')
    }
  }

  const addMessage = (sender, text) => {
    setMessages(prev => [...prev, { sender, text, time: new Date().toLocaleTimeString() }])
  }

  const callAPI = (path, params = {}) => {
    if (!dataChannelRef.current) {
      alert('数据通道未连接')
      return
    }

    try {
      setLoading(true)
      const request = {
        id: Math.random().toString(36).substr(2, 9),
        method: 'GET',
        path: path,
        params: params
      }

      dataChannelRef.current.send(JSON.stringify(request))
      addMessage('API请求', `${request.method} ${path}`)
    } catch (error) {
      console.error('API调用失败:', error)
      setLoading(false)
      alert(`API调用失败: ${error.message}`)
    }
  }

  // 使用 dataChannelFetch 调用真实的 HTTP API
  const callHTTPAPI = async (url, method = 'GET', body = null) => {
    if (!connected) {
      alert('数据通道未连接')
      return
    }

    try {
      setLoading(true)
      addMessage('HTTP请求', `${method} ${url}`)

      let response
      if (method === 'GET') {
        response = await dataChannelFetch.get(url)
      } else if (method === 'POST') {
        response = await dataChannelFetch.post(url, body)
      } else if (method === 'PUT') {
        response = await dataChannelFetch.put(url, body)
      } else if (method === 'DELETE') {
        response = await dataChannelFetch.delete(url)
      }

      const data = await response.json()

      setApiResponse({
        id: response.requestId,
        status: response.status,
        data: data
      })

      addMessage('HTTP响应', `${response.status} - ${JSON.stringify(data).substring(0, 100)}`)
      setLoading(false)
    } catch (error) {
      console.error('HTTP API调用失败:', error)
      setLoading(false)
      addMessage('错误', error.message)
      alert(`HTTP API调用失败: ${error.message}`)
    }
  }

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>WebRTC Frontend</h1>

      <div style={{ marginBottom: '20px' }}>
        <p>状态: <strong style={{ color: connected ? 'green' : 'orange' }}>{status}</strong></p>
        <p>客户端ID: {clientId.current}</p>
        <p>设备ID: {deviceId}</p>
        <p>连接方式: {connected ? 'P2P DataChannel' : '未连接'}</p>
      </div>

      <div style={{
        border: '1px solid #ccc',
        padding: '10px',
        height: '300px',
        overflowY: 'scroll',
        marginBottom: '10px',
        backgroundColor: '#f9f9f9'
      }}>
        {messages.map((msg, idx) => (
          <div key={idx} style={{ marginBottom: '8px' }}>
            <strong>[{msg.time}] {msg.sender}:</strong> {msg.text}
          </div>
        ))}
      </div>

      <div>
        <input
          type="text"
          value={inputMessage}
          onChange={(e) => setInputMessage(e.target.value)}
          onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
          placeholder="输入消息..."
          disabled={!connected}
          style={{ padding: '8px', width: '300px', marginRight: '10px' }}
        />
        <button
          onClick={sendMessage}
          disabled={!connected}
          style={{ padding: '8px 16px' }}
        >
          发送
        </button>
      </div>

      <hr style={{ margin: '30px 0' }} />

      <h2>DataChannel Fetch 测试（真实HTTP请求）</h2>
      <div style={{ marginBottom: '20px' }}>
        <button
          onClick={() => callHTTPAPI('http://httpbin.org/get', 'GET')}
          disabled={!connected || loading}
          style={{ padding: '8px 16px', marginRight: '10px' }}
        >
          GET 请求测试
        </button>
        <button
          onClick={() => callHTTPAPI('http://httpbin.org/post', 'POST', { message: 'Hello from DataChannel!' })}
          disabled={!connected || loading}
          style={{ padding: '8px 16px', marginRight: '10px' }}
        >
          POST 请求测试
        </button>
        <button
          onClick={() => callHTTPAPI('http://httpbin.org/put', 'PUT', { data: 'Updated' })}
          disabled={!connected || loading}
          style={{ padding: '8px 16px', marginRight: '10px' }}
        >
          PUT 请求测试
        </button>
        <button
          onClick={() => callHTTPAPI('http://httpbin.org/delete', 'DELETE')}
          disabled={!connected || loading}
          style={{ padding: '8px 16px' }}
        >
          DELETE 请求测试
        </button>
      </div>

      <hr style={{ margin: '30px 0' }} />

      <h2>API调用测试（设备本地API）</h2>
      <div style={{ marginBottom: '20px' }}>
        <button
          onClick={() => callAPI('/info')}
          disabled={!connected || loading}
          style={{ padding: '8px 16px', marginRight: '10px' }}
        >
          获取设备信息
        </button>
        <button
          onClick={() => callAPI('/time')}
          disabled={!connected || loading}
          style={{ padding: '8px 16px', marginRight: '10px' }}
        >
          获取时间
        </button>
        <button
          onClick={() => callAPI('/data')}
          disabled={!connected || loading}
          style={{ padding: '8px 16px', marginRight: '10px' }}
        >
          获取传感器数据
        </button>
        <button
          onClick={() => callAPI('/echo', { message: 'Hello Device!' })}
          disabled={!connected || loading}
          style={{ padding: '8px 16px' }}
        >
          Echo测试
        </button>
      </div>

      {apiResponse && (
        <div style={{
          border: '1px solid #4CAF50',
          padding: '15px',
          backgroundColor: '#f1f8f4',
          borderRadius: '4px'
        }}>
          <h3>最新API响应:</h3>
          <p><strong>请求ID:</strong> {apiResponse.id}</p>
          <p><strong>状态码:</strong> {apiResponse.status}</p>
          {apiResponse.error && <p style={{ color: 'red' }}><strong>错误:</strong> {apiResponse.error}</p>}
          <p><strong>数据:</strong></p>
          <pre style={{ backgroundColor: '#fff', padding: '10px', overflow: 'auto' }}>
            {(() => {
              try {
                return JSON.stringify(apiResponse.data, null, 2)
              } catch (e) {
                return `Error: ${e.message}`
              }
            })()}
          </pre>
        </div>
      )}
    </div>
  )
}

export default App
