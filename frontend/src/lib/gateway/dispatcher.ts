import type { Transport, P2PRequestInit, P2PResponse } from './types'
import type { RouteSelector } from './route-selector'
import { encodeRequest, decodeResponse } from './protocol'

export class Dispatcher {
  private directTransport: Transport
  private relayTransport: Transport
  private routeSelector: RouteSelector

  constructor(
    directTransport: Transport,
    relayTransport: Transport,
    routeSelector: RouteSelector
  ) {
    this.directTransport = directTransport
    this.relayTransport = relayTransport
    this.routeSelector = routeSelector
  }

  async dispatch(
    path: string,
    init: P2PRequestInit = {}
  ): Promise<P2PResponse> {
    const route = this.routeSelector.selectRoute()
    const transport = route === 'direct'
      ? this.directTransport
      : this.relayTransport

    const request = encodeRequest(path, init)

    try {
      const source = await transport.send(request)
      return decodeResponse(source)
    } catch (err) {
      if (route === 'direct') {
        return this.fallbackToRelay(path, init)
      }
      throw err
    }
  }

  private async fallbackToRelay(
    path: string,
    init: P2PRequestInit
  ): Promise<P2PResponse> {
    const request = encodeRequest(path, init)
    const source = await this.relayTransport.send(request)
    return decodeResponse(source)
  }
}
