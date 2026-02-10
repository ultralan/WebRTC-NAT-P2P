import { peerIdFromString } from '@libp2p/peer-id'
import type { P2PNode } from '../libp2p/node'
import type {
  GatewayConfig,
  P2PRequestInit,
  P2PResponse,
  TransportContext
} from './types'
import { DirectTransport, RelayTransport } from './transports'
import { RouteSelector } from './route-selector'
import { RTTMonitor } from './rtt-monitor'
import { Dispatcher } from './dispatcher'

const DEFAULT_RTT_THRESHOLD = 200
const DEFAULT_RTT_CHECK_INTERVAL = 5000

export class P2PGateway {
  private node: P2PNode
  private config: Required<GatewayConfig>
  private directTransport: DirectTransport
  private relayTransport: RelayTransport
  private routeSelector: RouteSelector
  private rttMonitor: RTTMonitor
  private dispatcher: Dispatcher
  private started = false

  constructor(node: P2PNode, config: GatewayConfig) {
    this.node = node
    this.config = {
      targetPeerId: config.targetPeerId,
      relayAddr: config.relayAddr,
      rttThreshold: config.rttThreshold ?? DEFAULT_RTT_THRESHOLD,
      rttCheckInterval: config.rttCheckInterval ?? DEFAULT_RTT_CHECK_INTERVAL
    }

    const ctx: TransportContext = {
      node: this.node,
      targetPeerId: peerIdFromString(this.config.targetPeerId),
      relayAddr: this.config.relayAddr
    }

    this.directTransport = new DirectTransport(ctx)
    this.relayTransport = new RelayTransport(ctx)
    this.routeSelector = new RouteSelector(this.config.rttThreshold)

    this.rttMonitor = new RTTMonitor(
      this.directTransport,
      this.relayTransport,
      this.routeSelector,
      this.config.rttCheckInterval
    )

    this.dispatcher = new Dispatcher(
      this.directTransport,
      this.relayTransport,
      this.routeSelector
    )
  }

  async start(): Promise<void> {
    if (this.started) return
    await this.rttMonitor.start()
    this.started = true
  }

  stop(): void {
    if (!this.started) return
    this.rttMonitor.stop()
    this.started = false
  }

  async fetch(
    path: string,
    init?: P2PRequestInit
  ): Promise<P2PResponse> {
    if (!this.started) {
      throw new Error('Gateway not started')
    }
    return this.dispatcher.dispatch(path, init)
  }

  getStatus() {
    return this.routeSelector.getStatus()
  }
}
