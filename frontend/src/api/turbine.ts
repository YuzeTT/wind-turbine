import request from './request'

export function listTurbines(status?: string) {
  return request.get('/turbines', { params: status ? { status } : {} })
}

export function getTurbine(id: number | string) {
  return request.get(`/turbines/${id}`)
}

export function listModels() {
  return request.get('/turbines/models')
}

export function updateTurbineStatus(id: number | string, data: {
  status: string
  operator?: string
  reason?: string
}) {
  return request.put(`/turbines/${id}/status`, data)
}
