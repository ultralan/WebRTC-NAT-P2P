import type { Transport } from './types'
import type { RouteSelector } from './route-selector'

export class RTTMonitor {
  private directTransport: Transport
  private relayTransport: Transport
  private routeSelector: RouteSelector
  private interval: number
  private timer: ReturnType<typeof setInterval> | null = null

  constructor(
    directTransport: Transport,
    relayTransport: Transport,
    routeSelector: RouteSelector,
    interval: number
  ) {
    this.directTransport = directTransport
    this.relayTransport = relayTransport
    this.routeSelector = routeSelector
    this.interval = interval
  }

  async start(): Promise<void> {
    await this.check()
    this.timer = setInterval(() => this.check(), this.interval)
  }

  stop(): void {
    if (this.timer) {
      clearInterval(this.timer)
      this.timer = null
    }
  }

  private async check(): Promise<void> {
    const [directRtt, relayRtt] = await Promise.all([
      this.pingDirect(),
      this.pingRelay()
    ])

    this.routeSelector.updateDirectStatus(
      directRtt !== Infinity,
      directRtt
    )
    this.routeSelector.updateRelayStatus(
      relayRtt !== Infinity,
      relayRtt
    )
  }

  private async pingDirect(): Promise<number> {
    try {
      return await this.directTransport.ping()
    } catch {
      return Infinity
    }
  }

  private async pingRelay(): Promise<number> {
    try {
      return await this.relayTransport.ping()
    } catch {
      return Infinity
    }
  }
}
