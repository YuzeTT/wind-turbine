import axios from 'axios'
import router from '@/router'

const request = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  timeout: 15000,
})

// 请求拦截器：自动带 token
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error),
)

// 响应拦截器：统一处理
request.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      // 401 → 跳登录
      if (res.code === 401) {
        localStorage.removeItem('token')
        router.push('/login')
      }
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      router.push('/login')
    }
    return Promise.reject(error)
  },
)

export default request
