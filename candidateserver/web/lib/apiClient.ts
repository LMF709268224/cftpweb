import { getErrorMessage } from "./errorCodes"
import { toast } from "sonner"

// 统一封装的 API 请求工具，自动处理 401 拦截和业务错误弹窗
export async function apiClient(endpoint: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers)
  
  const res = await fetch(endpoint, {
    credentials: "include",
    ...options,
    headers,
  })

  // 1. 拦截 401
  if (res.status === 401) {
    if (typeof window !== "undefined") {
      localStorage.removeItem("is_authenticated")
      localStorage.removeItem("user_name")
      const currentLang = (localStorage.getItem("app_lang") || "zh") as "zh" | "en"
      toast.error(getErrorMessage("UNAUTHORIZED", currentLang))
    } else {
      toast.error("401 Unauthorized")
    }
    setTimeout(() => window.location.href = "/login", 1500)
    throw new Error("401 Unauthorized")
  }

  // 2. 尝试解析 JSON
  let data;
  try {
    data = await res.json()
  } catch (e) {
    // 如果不是 JSON，直接返回原始 response
    return res
  }

  // 3. 统一拦截业务错误 (状态码非 200，或者业务 code 非 200)
  if (!res.ok || data.code !== 200) {
    // 自动通过 errorCodes 字典获取本地化提示
    const currentLang = typeof window !== "undefined" ? (localStorage.getItem("app_lang") || "zh") : "zh";
    const errorMsg = getErrorMessage(data.error_code, currentLang as "zh" | "en") || data.message || "请求失败"
    toast.error(errorMsg)
    throw new Error(errorMsg)
  }

  // 4. 请求成功，直接把真实的 data 剥离出来返回给组件！组件再也不用判断 code === 200 了
  return data.data
}
