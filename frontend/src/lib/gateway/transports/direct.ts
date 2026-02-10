import { BaseTransport } from './base'
import type { RouteType, TransportContext } from '../types'

const HTTP_PROTOCOL = '/http/1.1'
const PING_PROTOCOL = '/ping/1.0.0'

export class DirectTransport extends BaseTransport {
  readonly type: RouteType = 'direct'

  constructor(ctx: TransportContext) {
    super(ctx)
  }

  async send(request: Uint8Array): Promise<AsyncIterable<Uint8Array>> {
    const stream = await this.ctx.node.dialProtocol(
      this.ctx.targetPeerId,
      HTTP_PROTOCOL
    )

    await stream.sink([request])

    this.setAvailable(true)
    return this.wrapSource(stream.source)
  }

  private async *wrapSource(
    source: AsyncIterable<{ subarray(): Uint8Array }>
  ): AsyncIterable<Uint8Array> {
    for await (const chunk of source) {
      yield chunk.subarray()
    }
  }

  async ping(): Promise<number> {
    const start = Date.now()
    try {
      const stream = await this.ctx.node.dialProtocol(
        this.ctx.targetPeerId,
        PING_PROTOCOL
      )

      const encoder = new TextEncoder()
      await stream.sink([encoder.encode('ping')])

      for await (const _ of stream.source) {
        break
      }

      this.setAvailable(true)
      return Date.now() - start
    } catch {
      this.setAvailable(false)
      return Infinity
    }
  }
}
