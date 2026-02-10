import { useState, useCallback } from 'react'
import type { P2PNode } from '../lib/libp2p/node'
import { createNode } from '../lib/libp2p/node'
import { HttpClient } from '../lib/libp2p/http-client'

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected'

export function useP2P() {
  const [node, setNode] = useState<P2PNode | null>(null)
  const [status, setStatus] = useState<ConnectionStatus>('disconnected')
  const [error, setError] = useState<string | null>(null)
  const [client, setClient] = useState<HttpClient | null>(null)

  const connect = useCallback(async (relayAddr: string) => {
    setStatus('connecting')
    setError(null)

    try {
      const n = await createNode({ relayAddr })
      setNode(n)
      setClient(new HttpClient(n))
      setStatus('connected')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      setStatus('disconnected')
    }
  }, [])

  return { node, status, error, client, connect }
}
