import { multiaddr } from '@multiformats/multiaddr'
import { BaseTransport } from './base'
import type { RouteType, TransportContext } from '../types'

const HTTP_PROTOCOL = '/http/1.1'
const PING_PROTOCOL = '/ping/1.0.0'

export class RelayTransport extends BaseTransport {
  readonly type: RouteType = 'relay'

  constructor(ctx: TransportContext) {
    super(ctx)
  }

  private getRelayedAddr(): ReturnType<typeof multiaddr> {
    const targetId = this.ctx.targetPeerId.toString()
    return multiaddr(`${this.ctx.relayAddr}/p2p-circuit/p2p/${targetId}`)
  }

  async send(request: Uint8Array): Promise<AsyncIterable<Uint8Array>> {
    const relayedAddr = this.getRelayedAddr()

    const stream = await this.ctx.node.dialProtocol(
      relayedAddr,
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
      const relayedAddr = this.getRelayedAddr()

      const stream = await this.ctx.node.dialProtocol(
        relayedAddr,
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
