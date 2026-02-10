import type { P2PResponse } from '../types'

function findHeaderEnd(buffer: Uint8Array): number {
  const pattern = [13, 10, 13, 10] // \r\n\r\n
  for (let i = 0; i <= buffer.length - 4; i++) {
    if (buffer[i] === pattern[0] &&
        buffer[i + 1] === pattern[1] &&
        buffer[i + 2] === pattern[2] &&
        buffer[i + 3] === pattern[3]) {
      return i
    }
  }
  return -1
}

function parseHeaders(headerText: string) {
  const lines = headerText.split('\r\n')
  const statusLine = lines[0]
  const match = statusLine.match(/HTTP\/\d\.\d\s+(\d+)\s*(.*)/)

  const status = match ? parseInt(match[1], 10) : 0
  const statusText = match ? match[2] : ''
  const headers = new Headers()

  for (let i = 1; i < lines.length; i++) {
    const colonIdx = lines[i].indexOf(':')
    if (colonIdx > 0) {
      const key = lines[i].slice(0, colonIdx).trim()
      const value = lines[i].slice(colonIdx + 1).trim()
      headers.append(key, value)
    }
  }

  return { status, statusText, headers }
}

function concat(chunks: Uint8Array[]): Uint8Array {
  const len = chunks.reduce((a, c) => a + c.length, 0)
  const result = new Uint8Array(len)
  let offset = 0
  for (const chunk of chunks) {
    result.set(chunk, offset)
    offset += chunk.length
  }
  return result
}

export function decodeResponse(source: AsyncIterable<Uint8Array>): P2PResponse {
  let headersParsed = false
  let status = 0
  let statusText = ''
  let headers = new Headers()
  let buffer = new Uint8Array(0)
  let bodyStartIndex = 0

  const bodyStream = new ReadableStream<Uint8Array>({
    async start(controller) {
      try {
        for await (const chunk of source) {
          const newBuffer = new Uint8Array(buffer.length + chunk.length)
          newBuffer.set(buffer, 0)
          newBuffer.set(chunk, buffer.length)
          buffer = newBuffer

          if (!headersParsed) {
            const headerEnd = findHeaderEnd(buffer)
            if (headerEnd !== -1) {
              const headerBytes = buffer.slice(0, headerEnd)
              const headerText = new TextDecoder().decode(headerBytes)
              const parsed = parseHeaders(headerText)
              status = parsed.status
              statusText = parsed.statusText
              headers = parsed.headers
              headersParsed = true
              bodyStartIndex = headerEnd + 4

              if (bodyStartIndex < buffer.length) {
                controller.enqueue(buffer.slice(bodyStartIndex))
              }
              buffer = new Uint8Array(0)
            }
          } else {
            controller.enqueue(chunk)
          }
        }
        controller.close()
      } catch (err) {
        controller.error(err)
      }
    }
  })

  const response: P2PResponse = {
    get ok() { return status >= 200 && status < 300 },
    get status() { return status },
    get statusText() { return statusText },
    get headers() { return headers },
    body: bodyStream,

    async text() {
      const reader = bodyStream.getReader()
      const chunks: Uint8Array[] = []
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        if (value) chunks.push(value)
      }
      return new TextDecoder().decode(concat(chunks))
    },

    async json<T = unknown>(): Promise<T> {
      const text = await this.text()
      return JSON.parse(text)
    },

    async arrayBuffer(): Promise<ArrayBuffer> {
      const reader = bodyStream.getReader()
      const chunks: Uint8Array[] = []
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        if (value) chunks.push(value)
      }
      return concat(chunks).buffer as ArrayBuffer
    }
  }

  return response
}
