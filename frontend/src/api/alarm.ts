import request from './request'

export function listAlarms(params: {
  status?: string
  severity?: string
  turbine_id?: number | string
  page?: number
  page_size?: number
}) {
  return request.get('/alarms', { params })
}

export function alarmStats() {
  return request.get('/alarms/stats')
}

export function createAlarm(data: {
  turbine_id: number
  type: string
  severity: string
  title: string
  description?: string
  operator?: string
  stop_turbine?: boolean
}) {
  return request.post('/alarms', data)
}

export function resolveAlarm(id: number, data: {
  resolved_by?: string
  comment?: string
  restart?: boolean
}) {
  return request.put(`/alarms/${id}/resolve`, data)
}
