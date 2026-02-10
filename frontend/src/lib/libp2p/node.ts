import { createLibp2p } from 'libp2p'
import { webRTC } from '@libp2p/webrtc'
import { webSockets } from '@libp2p/websockets'
import { noise } from '@chainsafe/libp2p-noise'
import { yamux } from '@chainsafe/libp2p-yamux'
import { circuitRelayTransport } from '@libp2p/circuit-relay-v2'
import { dcutr } from '@libp2p/dcutr'
import { identify } from '@libp2p/identify'
import type { Libp2p } from 'libp2p'

export type P2PNode = Libp2p

export interface NodeConfig {
  relayAddr: string
}

export async function createNode(config: NodeConfig): Promise<P2PNode> {
  const node = await createLibp2p({
    transports: [
      webSockets(),
      webRTC(),
      circuitRelayTransport(),
    ],
    connectionEncrypters: [noise()],
    streamMuxers: [yamux()],
    services: {
      identify: identify(),
      dcutr: dcutr(),
    },
  })

  await node.start()
  console.log('P2P Node started, PeerID:', node.peerId.toString())

  return node
}
