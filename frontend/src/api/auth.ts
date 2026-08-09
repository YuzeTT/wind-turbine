import request from './request'

export function login(data: { username: string; password: string }) {
  return request.post('/auth/login', data)
}

export function getProfile() {
  return request.get('/auth/profile')
}

export function changePassword(data: { old_password: string; new_password: string }) {
  return request.put('/auth/password', data)
}

export function register(data: {
  username: string
  password: string
  nickname?: string
  role?: string
}) {
  return request.post('/auth/register', data)
}

export function listUsers() {
  return request.get('/auth/users')
}

export function updateUserStatus(id: number, status: string) {
  return request.put(`/auth/users/${id}/status`, { status })
}
