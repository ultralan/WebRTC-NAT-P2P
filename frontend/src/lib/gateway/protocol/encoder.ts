import type { P2PRequestInit } from '../types'

export function encodeRequest(path: string, init: P2PRequestInit = {}): Uint8Array {
  const method = init.method || 'GET'
  const headers = init.headers || {}
  const body = init.body

  let bodyBytes: Uint8Array | null = null
  if (body) {
    bodyBytes = typeof body === 'string'
      ? new TextEncoder().encode(body)
      : body
  }

  const lines: string[] = []
  lines.push(`${method} ${path} HTTP/1.1`)
  lines.push('Host: p2p-backend')

  if (bodyBytes) {
    headers['Content-Length'] = String(bodyBytes.length)
  }

  for (const [key, value] of Object.entries(headers)) {
    lines.push(`${key}: ${value}`)
  }

  lines.push('')
  lines.push('')

  const headerBytes = new TextEncoder().encode(lines.join('\r\n'))

  if (!bodyBytes) {
    return headerBytes
  }

  const result = new Uint8Array(headerBytes.length + bodyBytes.length)
  result.set(headerBytes, 0)
  result.set(bodyBytes, headerBytes.length)
  return result
}
