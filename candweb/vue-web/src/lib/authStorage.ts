const ACCESS_TOKEN_KEY = "access_token"
const POST_LOGIN_REDIRECT_KEY = "post_login_redirect"

export function getAccessToken() {
  const token = localStorage.getItem(ACCESS_TOKEN_KEY)
  if (token) syncAccessTokenCookie(token)
  return token
}

export function setAccessToken(token: string) {
  localStorage.setItem(ACCESS_TOKEN_KEY, token)
  syncAccessTokenCookie(token)
}

export function clearAccessToken() {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  document.cookie = `${ACCESS_TOKEN_KEY}=; Path=/; Max-Age=0; SameSite=Lax`
}

export function rememberPostLoginRedirect(path: string) {
  const normalized = normalizePostLoginRedirect(path)
  if (!normalized) return
  sessionStorage.setItem(POST_LOGIN_REDIRECT_KEY, normalized)
}

export function consumePostLoginRedirect() {
  const normalized = normalizePostLoginRedirect(sessionStorage.getItem(POST_LOGIN_REDIRECT_KEY) || "")
  sessionStorage.removeItem(POST_LOGIN_REDIRECT_KEY)
  return normalized
}

function normalizePostLoginRedirect(path: string) {
  const normalized = String(path || "").trim()
  if (!normalized || !normalized.startsWith("/") || normalized.startsWith("//")) return ""
  if (normalized === "/login" || normalized.startsWith("/login?") || normalized === "/callback" || normalized.startsWith("/callback?")) return ""
  return normalized
}

function syncAccessTokenCookie(token: string) {
  document.cookie = `${ACCESS_TOKEN_KEY}=${encodeURIComponent(token)}; Path=/; SameSite=Lax`
}
