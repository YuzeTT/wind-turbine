import request from './request'

export function listOperationLogs(params: {
  turbine_id?: number | string
  action?: string
  page?: number
  page_size?: number
}) {
  return request.get('/operation-logs', { params })
}

export function createOperationLog(data: {
  turbine_id: number
  operator?: string
  action: string
  reason?: string
}) {
  return request.post('/operation-logs', data)
}
