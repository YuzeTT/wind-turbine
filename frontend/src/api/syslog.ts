import request from './request'

export function listSystemLogs(params: {
  level?: string
  module?: string
  page?: number
  page_size?: number
}) {
  return request.get('/system-logs', { params })
}
