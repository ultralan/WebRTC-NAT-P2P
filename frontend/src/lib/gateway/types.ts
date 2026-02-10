import type { PeerId } from '@libp2p/interface'
import type { P2PNode } from '../libp2p/node'

export type RouteType = 'direct' | 'relay'

export interface GatewayConfig {
  targetPeerId: string
  relayAddr: string
  rttThreshold?: number
  rttCheckInterval?: number
}

export interface P2PRequestInit {
  method?: string
  headers?: Record<string, string>
  body?: string | Uint8Array
}

export interface P2PResponse {
  ok: boolean
  status: number
  statusText: string
  headers: Headers
  body: ReadableStream<Uint8Array> | null
  text(): Promise<string>
  json<T = unknown>(): Promise<T>
  arrayBuffer(): Promise<ArrayBuffer>
}

export interface RouteStatus {
  available: boolean
  rtt: number
  lastCheck: number
}

export interface TransportContext {
  node: P2PNode
  targetPeerId: PeerId
  relayAddr: string
}

export interface Transport {
  readonly type: RouteType
  send(request: Uint8Array): Promise<AsyncIterable<Uint8Array>>
  ping(): Promise<number>
  isAvailable(): boolean
}
