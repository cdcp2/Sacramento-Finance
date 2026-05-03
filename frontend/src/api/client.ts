import axios from 'axios'
import type { TokenPair } from '@/types'

const client = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
})

// Adjunta el access token en cada request
client.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// Si recibe 401 intenta refrescar el token una vez
let isRefreshing = false
let waitingQueue: Array<(token: string) => void> = []

client.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config
    if (error.response?.status !== 401 || original._retry) {
      return Promise.reject(error)
    }

    const refreshToken = localStorage.getItem('refresh_token')
    if (!refreshToken) {
      clearSession()
      return Promise.reject(error)
    }

    if (isRefreshing) {
      return new Promise((resolve) => {
        waitingQueue.push((token) => {
          original.headers.Authorization = `Bearer ${token}`
          resolve(client(original))
        })
      })
    }

    original._retry = true
    isRefreshing = true

    try {
      const { data } = await axios.post<{ data: TokenPair }>('/api/v1/auth/refresh', {
        refresh_token: refreshToken,
      })
      const { access_token, refresh_token } = data.data
      localStorage.setItem('access_token', access_token)
      localStorage.setItem('refresh_token', refresh_token)
      waitingQueue.forEach((cb) => cb(access_token))
      waitingQueue = []
      original.headers.Authorization = `Bearer ${access_token}`
      return client(original)
    } catch {
      clearSession()
      return Promise.reject(error)
    } finally {
      isRefreshing = false
    }
  },
)

function clearSession() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  window.location.href = '/login'
}

export default client
