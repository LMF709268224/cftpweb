import { toast } from "vue-sonner"
import { getErrorMessage, localizeApiErrorMessage } from "./errorCodes"

export async function apiClient(endpoint: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers)
  const token = localStorage.getItem("access_token")

  if (token) headers.set("Authorization", `Bearer ${token}`)
  if (options.body && !(options.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const res = await fetch(endpoint, {
    credentials: "include",
    ...options,
    headers,
  })

  const currentLang = (localStorage.getItem("app_lang") || "zh") as "zh" | "en"

  if (res.status === 401) {
    localStorage.removeItem("is_authenticated")
    localStorage.removeItem("user_name")
    toast.error(getErrorMessage("UNAUTHORIZED", currentLang))
    setTimeout(() => {
      window.location.href = "/login"
    }, 1500)
    throw new Error("401 Unauthorized")
  }

  let data: any
  try {
    data = await res.json()
  } catch {
    if (!res.ok) {
      const errorMsg = getErrorMessage("UNKNOWN_ERROR", currentLang)
      toast.error(errorMsg)
      throw new Error(errorMsg)
    }
    return res
  }

  if (!res.ok || (data.code !== 200 && data.code !== 201)) {
    const errorMsg = data.message
      ? localizeApiErrorMessage(data.error_code, data.message, currentLang)
      : getErrorMessage(data.error_code, currentLang)
    toast.error(errorMsg)
    throw new Error(errorMsg)
  }

  return data.data
}
