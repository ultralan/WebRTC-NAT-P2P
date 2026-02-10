/**
 * DataChannel Fetch - 通过 WebRTC DataChannel 发送 HTTP 请求
 * 提供类似原生 fetch API 的接口
 */

class DataChannelFetch {
  constructor() {
    this.dataChannel = null
    this.pendingRequests = new Map()
    this.requestTimeout = 30000 // 30秒超时
    this.backendURL = 'http://8.138.247.82:8888' // 信令服务器地址
    this.deviceId = 'device-1' // 目标设备ID
  }

  /**
   * 设置 DataChannel
   * @param {RTCDataChannel} dataChannel
   */
  setDataChannel(dataChannel) {
    this.dataChannel = dataChannel

    // 监听消息，处理响应
    if (this.dataChannel) {
      this.dataChannel.addEventListener('message', (event) => {
        this.handleResponse(event.data)
      })
    }
  }

  /**
   * 处理从 DataChannel 接收到的响应
   * @param {string|ArrayBuffer} data
   */
  handleResponse(data) {
    try {
      // 如果是 ArrayBuffer，先转换为字符串
      let jsonString = data
      if (data instanceof ArrayBuffer) {
        const decoder = new TextDecoder('utf-8')
        jsonString = decoder.decode(data)
      }

      const response = JSON.parse(jsonString)

      // 检查是否是 HTTP 响应
      if (response.requestId && this.pendingRequests.has(response.requestId)) {
        const { resolve, reject, timeoutId } = this.pendingRequests.get(response.requestId)

        // 清除超时定时器
        clearTimeout(timeoutId)

        // 删除待处理请求
        this.pendingRequests.delete(response.requestId)

        // 解析响应
        resolve(new DataChannelResponse(response))
      }
    } catch (error) {
      console.error('Failed to parse response:', error)
    }
  }

  /**
   * 发送 HTTP 请求（类似 fetch API）
   * @param {string} url 请求 URL
   * @param {Object} options 请求选项
   * @returns {Promise<DataChannelResponse>}
   */
  fetch(url, options = {}) {
    // 检查 DataChannel 是否可用
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      // DataChannel 不可用，使用 backend 转发
      console.log('DataChannel not available, using backend proxy')
      return this.fetchViaBackend(url, options)
    }

    // DataChannel 可用，使用 P2P 直连
    console.log('Using P2P DataChannel')
    return this.fetchViaDataChannel(url, options)
  }

  /**
   * 通过 DataChannel 发送请求
   */
  fetchViaDataChannel(url, options = {}) {
    return new Promise((resolve, reject) => {

      // 生成唯一请求 ID
      const requestId = this.generateRequestId()

      // 解析 URL
      const urlObj = new URL(url, window.location.origin)

      // 构建请求对象
      const request = {
        requestId,
        method: options.method || 'GET',
        url: urlObj.href,
        path: urlObj.pathname + urlObj.search,
        headers: options.headers || {},
        body: options.body || null
      }

      // 设置超时
      const timeoutId = setTimeout(() => {
        this.pendingRequests.delete(requestId)
        reject(new Error(`Request timeout after ${this.requestTimeout}ms`))
      }, this.requestTimeout)

      // 保存待处理请求
      this.pendingRequests.set(requestId, { resolve, reject, timeoutId })

      // 发送请求
      try {
        this.dataChannel.send(JSON.stringify(request))
      } catch (error) {
        clearTimeout(timeoutId)
        this.pendingRequests.delete(requestId)
        reject(error)
      }
    })
  }

  /**
   * 通过 backend 转发请求
   */
  async fetchViaBackend(url, options = {}) {
    const requestId = this.generateRequestId()

    // 构建代理请求对象
    const proxyRequest = {
      requestId,
      deviceId: this.deviceId,
      method: options.method || 'GET',
      url: url,
      headers: options.headers || {},
      body: options.body || null
    }

    try {
      // 发送请求到 backend 的 /proxy 端点
      const response = await fetch(`${this.backendURL}/proxy`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(proxyRequest)
      })

      if (!response.ok) {
        throw new Error(`Backend proxy error: ${response.status} ${response.statusText}`)
      }

      // 解析响应
      const proxyResponse = await response.json()

      // 返回 DataChannelResponse 对象
      return new DataChannelResponse(proxyResponse)
    } catch (error) {
      throw new Error(`Failed to fetch via backend: ${error.message}`)
    }
  }

  /**
   * 生成唯一请求 ID
   * @returns {string}
   */
  generateRequestId() {
    return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  /**
   * GET 请求快捷方法
   */
  get(url, options = {}) {
    return this.fetch(url, { ...options, method: 'GET' })
  }

  /**
   * POST 请求快捷方法
   */
  post(url, body, options = {}) {
    return this.fetch(url, {
      ...options,
      method: 'POST',
      body: typeof body === 'string' ? body : JSON.stringify(body),
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      }
    })
  }

  /**
   * PUT 请求快捷方法
   */
  put(url, body, options = {}) {
    return this.fetch(url, {
      ...options,
      method: 'PUT',
      body: typeof body === 'string' ? body : JSON.stringify(body),
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      }
    })
  }

  /**
   * DELETE 请求快捷方法
   */
  delete(url, options = {}) {
    return this.fetch(url, { ...options, method: 'DELETE' })
  }
}

/**
 * DataChannel 响应对象（类似 fetch Response）
 */
class DataChannelResponse {
  constructor(responseData) {
    this.requestId = responseData.requestId
    this.status = responseData.status || 200
    this.statusText = responseData.statusText || 'OK'
    this.headers = responseData.headers || {}
    this._body = responseData.body
    this.ok = this.status >= 200 && this.status < 300
  }

  /**
   * 解析 JSON 响应
   */
  async json() {
    if (typeof this._body === 'string') {
      return JSON.parse(this._body)
    }
    return this._body
  }

  /**
   * 获取文本响应
   */
  async text() {
    if (typeof this._body === 'string') {
      return this._body
    }
    return JSON.stringify(this._body)
  }

  /**
   * 克隆响应
   */
  clone() {
    return new DataChannelResponse({
      requestId: this.requestId,
      status: this.status,
      statusText: this.statusText,
      headers: { ...this.headers },
      body: this._body
    })
  }
}

// 创建全局实例
const dataChannelFetch = new DataChannelFetch()

export default dataChannelFetch
export { DataChannelFetch, DataChannelResponse }
