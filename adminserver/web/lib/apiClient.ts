import { toast } from "sonner"
import { getErrorMessage, localizeApiErrorMessage } from "./errorCodes"

type ApiClientOptions = RequestInit & {
  timeoutMs?: number
}

const DEFAULT_API_TIMEOUT_MS = 60000

export async function apiClient(endpoint: string, options: ApiClientOptions = {}) {
  const { timeoutMs = DEFAULT_API_TIMEOUT_MS, signal, ...fetchOptions } = options
  const headers = new Headers(options.headers)
  const currentLang = (typeof window !== "undefined"
    ? localStorage.getItem("app_lang") || "zh"
    : "zh") as "zh" | "en"
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs)

  if (signal) {
    if (signal.aborted) {
      controller.abort()
    } else {
      signal.addEventListener("abort", () => controller.abort(), { once: true })
    }
  }

  if (typeof window !== "undefined") {
    const token = localStorage.getItem("access_token")
    if (token) headers.set("Authorization", `Bearer ${token}`)
  }

  let res: Response
  try {
    res = await fetch(endpoint, {
      credentials: "include",
      ...fetchOptions,
      headers,
      signal: controller.signal,
    })
  } catch (err) {
    const isAbort = err instanceof DOMException && err.name === "AbortError"
    const errorMsg = isAbort
      ? currentLang === "zh"
        ? "请求超时，请稍后重试"
        : "Request timed out. Please try again."
      : currentLang === "zh"
        ? "网络请求失败，请检查网络后重试"
        : "Network request failed. Please check your connection and try again."
    toast.error(errorMsg)
    throw new Error(errorMsg)
  } finally {
    clearTimeout(timeoutId)
  }

  if (res.status === 401) {
    if (typeof window !== "undefined") {
      localStorage.removeItem("is_authenticated")
      localStorage.removeItem("user_name")
      toast.error(getErrorMessage("UNAUTHORIZED", currentLang))
      setTimeout(() => {
        window.location.href = "/login"
      }, 1500)
    } else {
      toast.error("401 Unauthorized")
    }
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

  if (!res.ok || data.code !== 200) {
    const errorMsg = data.message
      ? localizeApiErrorMessage(data.error_code, data.message, currentLang)
      : getErrorMessage(data.error_code, currentLang)

    toast.error(errorMsg)
    throw new Error(errorMsg)
  }

  return data.data
}
