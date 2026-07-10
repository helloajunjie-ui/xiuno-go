import axios from 'axios'

const instance = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  withCredentials: true,
})

instance.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code === 0) {
      return res.data
    }
    console.error('业务拦截:', res.message)
    return Promise.reject(new Error(res.message || 'Error'))
  },
  (error) => {
    if (error.response) {
      const status = error.response.status
      if (status === 401) {
        console.warn('登录态失效，请重新登录')
      } else if (status === 404) {
        console.warn('资源不存在 (404)')
      } else {
        console.error('服务器开小差了 (500)')
      }
    } else {
      console.error('网络请求失败')
    }
    return Promise.reject(error)
  }
)

// 封装泛型请求方法，让 TypeScript 能正确推断解包后的返回类型
const request = {
  get<T = unknown>(url: string, config?: Record<string, unknown>): Promise<T> {
    return instance.get(url, config) as Promise<T>
  },
  post<T = unknown>(url: string, data?: unknown, config?: Record<string, unknown>): Promise<T> {
    return instance.post(url, data, config) as Promise<T>
  },
  put<T = unknown>(url: string, data?: unknown, config?: Record<string, unknown>): Promise<T> {
    return instance.put(url, data, config) as Promise<T>
  },
  delete<T = unknown>(url: string, config?: Record<string, unknown>): Promise<T> {
    return instance.delete(url, config) as Promise<T>
  },
}

export default request
