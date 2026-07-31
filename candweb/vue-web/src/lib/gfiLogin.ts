import { apiClient } from "@/lib/apiClient"
import { rememberPostLoginRedirect } from "@/lib/authStorage"

export async function startGfiLogin(postLoginRedirect?: string) {
  if (postLoginRedirect) rememberPostLoginRedirect(postLoginRedirect)

  const callbackUrl = encodeURIComponent(`${window.location.origin}/callback`)
  const response = await apiClient(`/api/auth/login-url?callback=${callbackUrl}`, {
    timeoutMs: 10000,
    suppressErrorToast: true,
    suppressAuthRedirect: true,
  })

  if (!response?.url) throw new Error("AUTH_LOGIN_URL_MISSING")
  window.location.assign(response.url)
}
