export interface TurbineData {
  id: number
  name: string
  model: string
  rated_power: number
  status: string
  power: number
  rotor_speed: number
  wind_speed: number
  wind_direction: number
  temperature: number
  latitude: number
  longitude: number
  last_update: string
  today_power: number
  total_power: number
  availability: number
  farm: string
}

export interface WSCallbacks {
  onOpen?: () => void
  onClose?: () => void
  onError?: (error: unknown) => void
  /** 单台风机数据更新（增量推送） */
  onTurbineUpdate?: (turbine: TurbineData, time: string) => void
  /** 心跳 */
  onHeartbeat?: (time: string) => void
}

export function connectWebSocket(token: string, callbacks: WSCallbacks = {}) {
  const ws = new WebSocket(`ws://localhost:8080/ws?token=${encodeURIComponent(token)}`)

  ws.onopen = () => callbacks.onOpen?.()
  ws.onclose = () => callbacks.onClose?.()
  ws.onerror = (error) => callbacks.onError?.(error)

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      switch (msg.type) {
        case 'welcome':
          callbacks.onOpen?.() // 也视为已连接
          break
        case 'turbine_update':
          callbacks.onTurbineUpdate?.(msg.turbine as TurbineData, msg.time)
          break
        case 'heartbeat':
          callbacks.onHeartbeat?.(msg.time)
          break
      }
    } catch {
      // 忽略非 JSON 消息
    }
  }

  return ws
}
