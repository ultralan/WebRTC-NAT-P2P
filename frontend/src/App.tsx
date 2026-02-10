import { useEffect, useRef } from 'react'
import { useP2P } from './hooks/useP2P'
import { BookManager } from './components/BookManager'

const DEFAULT_RELAY = '/ip4/127.0.0.1/udp/4002/quic-v1/p2p/12D3KooWG53rJbdyC1yqdNuMgVTeE1s4QdFZbtCbkHRraNPbCWLh'

function App() {
  const { status, error, connect } = useP2P()
  const retryRef = useRef<number | null>(null)

  useEffect(() => {
    const tryConnect = () => {
      if (status === 'disconnected') {
        connect(DEFAULT_RELAY)
      }
    }

    tryConnect()

    retryRef.current = window.setInterval(() => {
      if (status === 'disconnected') {
        tryConnect()
      }
    }, 5000)

    return () => {
      if (retryRef.current) clearInterval(retryRef.current)
    }
  }, [status, connect])

  return (
    <div style={{ padding: '20px' }}>
      <h1>P2P Book Manager</h1>

      <div style={{ marginBottom: '20px', padding: '10px', background: '#f5f5f5' }}>
        <p>P2P Status: {status}</p>
        {error && <p style={{ color: 'red' }}>{error}</p>}
      </div>

      <BookManager />
    </div>
  )
}

export default App
