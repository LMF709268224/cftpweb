import { getErrorMessage } from "./errorCodes"
import { toast } from "sonner"

// 统一封装的 API 请求工具，自动处理 401 拦截和业务错误弹窗
export async function apiClient(endpoint: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers)
  
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("access_token")
    if (token) {
      headers.set("Authorization", `Bearer ${token}`)
    }
  }
  
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
    if (!res.ok) {
      const errorMsg = `请求异常 (${res.status})`
      toast.error(errorMsg)
      throw new Error(errorMsg)
    }
    // 如果不是 JSON，但状态正常，直接返回原始 response
    return res
  }

  // 3. 统一拦截业务错误 (状态码非 200，或者业务 code 非 200)
  if (!res.ok || data.code !== 200) {
    // 自动通过 errorCodes 字典获取本地化提示
    const currentLang = typeof window !== "undefined" ? (localStorage.getItem("app_lang") || "zh") : "zh";
    const baseMsg = getErrorMessage(data.error_code, currentLang as "zh" | "en");
    
    // 如果是请求参数错误，且后端有提供具体的报错字段信息，则拼接起来
    const errorMsg = (data.error_code === "INVALID_REQUEST" && data.message)
      ? `${baseMsg}: ${data.message}`
      : (baseMsg !== "发生未知错误，请联系客服" && baseMsg !== "An unknown error occurred. Please contact support." ? baseMsg : (data.message || "请求失败"));

    toast.error(errorMsg)
    throw new Error(errorMsg)
  }

  // 4. 请求成功，直接把真实的 data 剥离出来返回给组件！组件再也不用判断 code === 200 了
  return data.data
}
