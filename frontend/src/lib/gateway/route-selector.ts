import type { RouteType, RouteStatus } from './types'

export class RouteSelector {
  private rttThreshold: number
  private directStatus: RouteStatus
  private relayStatus: RouteStatus

  constructor(rttThreshold: number) {
    this.rttThreshold = rttThreshold
    this.directStatus = {
      available: false,
      rtt: Infinity,
      lastCheck: 0
    }
    this.relayStatus = {
      available: false,
      rtt: Infinity,
      lastCheck: 0
    }
  }

  updateDirectStatus(available: boolean, rtt: number): void {
    this.directStatus = {
      available,
      rtt,
      lastCheck: Date.now()
    }
  }

  updateRelayStatus(available: boolean, rtt: number): void {
    this.relayStatus = {
      available,
      rtt,
      lastCheck: Date.now()
    }
  }

  selectRoute(): RouteType {
    const { directStatus, relayStatus, rttThreshold } = this

    if (!directStatus.available) {
      return 'relay'
    }

    if (!relayStatus.available) {
      return 'direct'
    }

    if (directStatus.rtt <= rttThreshold) {
      return 'direct'
    }

    if (relayStatus.rtt < directStatus.rtt) {
      return 'relay'
    }

    return 'direct'
  }

  getStatus() {
    return {
      direct: { ...this.directStatus },
      relay: { ...this.relayStatus }
    }
  }
}
