import type { P2PNode } from './node'
import type { PeerId } from '@libp2p/interface'

const HTTP_PROTOCOL = '/http/1.1'

export interface HttpResponse {
  status: number
  body: string
}

export class HttpClient {
  private node: P2PNode

  constructor(node: P2PNode) {
    this.node = node
  }

  async request(
    peerId: PeerId,
    method: string,
    path: string
  ): Promise<HttpResponse> {
    const stream = await this.node.dialProtocol(peerId, HTTP_PROTOCOL)

    const request = `${method} ${path} HTTP/1.1\r\n\r\n`
    const encoder = new TextEncoder()
    const writer = stream.sink

    await writer([encoder.encode(request)])

    const chunks: Uint8Array[] = []
    for await (const chunk of stream.source) {
      chunks.push(chunk.subarray())
    }

    const decoder = new TextDecoder()
    const response = decoder.decode(this.concat(chunks))

    return this.parseResponse(response)
  }

  private concat(chunks: Uint8Array[]): Uint8Array {
    const len = chunks.reduce((a, c) => a + c.length, 0)
    const result = new Uint8Array(len)
    let offset = 0
    for (const chunk of chunks) {
      result.set(chunk, offset)
      offset += chunk.length
    }
    return result
  }

  private parseResponse(raw: string): HttpResponse {
    const idx = raw.indexOf('\r\n\r\n')
    const header = raw.slice(0, idx)
    const body = raw.slice(idx + 4)

    const statusLine = header.split('\r\n')[0]
    const status = parseInt(statusLine.split(' ')[1], 10)

    return { status, body }
  }
}
