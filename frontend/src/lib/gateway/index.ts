export { P2PGateway } from './gateway'
export { RouteSelector } from './route-selector'
export { RTTMonitor } from './rtt-monitor'
export { Dispatcher } from './dispatcher'

export * from './types'
export * from './transports'
export * from './protocol'

import type { P2PNode } from '../libp2p/node'
import type { GatewayConfig } from './types'
import { P2PGateway } from './gateway'

export function createGateway(
  node: P2PNode,
  config: GatewayConfig
): P2PGateway {
  return new P2PGateway(node, config)
}
