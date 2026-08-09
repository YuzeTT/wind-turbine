import request from './request'

export function overview() {
  return request.get('/dashboard/overview')
}

export function statusDistribution() {
  return request.get('/dashboard/status-distribution')
}

export function powerTrend() {
  return request.get('/dashboard/power-trend')
}

export function availability(days = 7) {
  return request.get('/dashboard/availability', { params: { days } })
}

export function powerRanking() {
  return request.get('/dashboard/power-ranking')
}

export function windRose() {
  return request.get('/dashboard/wind-rose')
}

export function dailyEnergy() {
  return request.get('/dashboard/daily-energy')
}

export function mapData() {
  return request.get('/dashboard/map')
}

export function systemInfo() {
  return request.get('/dashboard/system-info')
}
