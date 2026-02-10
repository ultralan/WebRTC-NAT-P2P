import type { Transport, RouteType, TransportContext } from '../types'

export abstract class BaseTransport implements Transport {
  abstract readonly type: RouteType
  protected ctx: TransportContext
  protected available = false

  constructor(ctx: TransportContext) {
    this.ctx = ctx
  }

  abstract send(request: Uint8Array): Promise<AsyncIterable<Uint8Array>>
  abstract ping(): Promise<number>

  isAvailable(): boolean {
    return this.available
  }

  protected setAvailable(value: boolean): void {
    this.available = value
  }
}
