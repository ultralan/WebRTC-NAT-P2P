import { useState, useCallback, useRef } from 'react'
import type { P2PNode } from '../lib/libp2p/node'
import type { GatewayConfig, P2PResponse, P2PRequestInit } from '../lib/gateway'
import { createGateway, P2PGateway } from '../lib/gateway'

export type GatewayStatus = 'idle' | 'starting' | 'running' | 'error'

export function useGateway() {
  const [status, setStatus] = useState<GatewayStatus>('idle')
  const [error, setError] = useState<string | null>(null)
  const gatewayRef = useRef<P2PGateway | null>(null)

  const start = useCallback(async (
    node: P2PNode,
    config: GatewayConfig
  ) => {
    setStatus('starting')
    setError(null)

    try {
      const gw = createGateway(node, config)
      await gw.start()
      gatewayRef.current = gw
      setStatus('running')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      setStatus('error')
    }
  }, [])

  const stop = useCallback(() => {
    if (gatewayRef.current) {
      gatewayRef.current.stop()
      gatewayRef.current = null
      setStatus('idle')
    }
  }, [])

  const fetch = useCallback(async (
    path: string,
    init?: P2PRequestInit
  ): Promise<P2PResponse> => {
    if (!gatewayRef.current) {
      throw new Error('Gateway not running')
    }
    return gatewayRef.current.fetch(path, init)
  }, [])

  const getStatus = useCallback(() => {
    return gatewayRef.current?.getStatus() ?? null
  }, [])

  return {
    status,
    error,
    start,
    stop,
    fetch,
    getStatus
  }
}
