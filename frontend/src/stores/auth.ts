import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, getProfile as getProfileApi } from '@/api/auth'

interface UserInfo {
  id: number
  username: string
  nickname: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<UserInfo | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(username: string, password: string) {
    const res = await loginApi({ username, password })
    token.value = res.data.token
    user.value = res.data.user
    localStorage.setItem('token', token.value)
    return res.data
  }

  async function fetchProfile() {
    if (!token.value) return null
    try {
      const res = await getProfileApi()
      user.value = res.data as UserInfo
      return user.value
    } catch {
      // token 失效
      logout()
      return null
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  return { token, user, isLoggedIn, isAdmin, login, fetchProfile, logout }
})
