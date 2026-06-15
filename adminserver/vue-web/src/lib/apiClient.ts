import { toast } from "vue-sonner"

export async function apiClient(endpoint: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers)
  const token = localStorage.getItem("access_token")

  if (token) headers.set("Authorization", `Bearer ${token}`)
  if (options.body && !(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const response = await fetch(endpoint, {
    credentials: "include",
    ...options,
    headers,
  })

  if (response.status === 401) {
    localStorage.removeItem("access_token")
    localStorage.removeItem("admin_name")
    toast.error("登录已过期，请重新登录")
    setTimeout(() => {
      window.location.href = "/login"
    }, 1200)
    throw new Error("401 Unauthorized")
  }

  let payload: any
  try {
    payload = await response.json()
  } catch {
    if (!response.ok) {
      toast.error("请求失败")
      throw new Error("Request failed")
    }
    return response
  }

  if (!response.ok || (payload.code !== 200 && payload.code !== 201)) {
    const message = payload.message || payload.error || "请求失败"
    toast.error(message)
    throw new Error(message)
  }

  return payload.data
}

export function toQuery(params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && String(value).trim() !== "") {
      query.set(key, String(value))
    }
  })
  const text = query.toString()
  return text ? `?${text}` : ""
}
