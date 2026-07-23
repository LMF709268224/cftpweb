const LEGACY_TOKEN_KEY = "access_token"
const AUTH_KEY = "is_authenticated"
const USER_NAME_KEY = "user_name"
const POST_LOGIN_REDIRECT_KEY = "post_login_redirect"

export function isAuthenticated() {
  const legacyToken = localStorage.getItem(LEGACY_TOKEN_KEY)
  if (legacyToken && localStorage.getItem(AUTH_KEY) !== "true") {
    localStorage.setItem(AUTH_KEY, "true")
  }
  localStorage.removeItem(LEGACY_TOKEN_KEY)
  return localStorage.getItem(AUTH_KEY) === "true"
}

export function setAuthSession(userName?: string) {
  localStorage.removeItem(LEGACY_TOKEN_KEY)
  localStorage.setItem(AUTH_KEY, "true")
  if (userName) {
    localStorage.setItem(USER_NAME_KEY, userName)
  }
}

export function clearAuthSession() {
  localStorage.removeItem(LEGACY_TOKEN_KEY)
  localStorage.removeItem(AUTH_KEY)
  localStorage.removeItem(USER_NAME_KEY)
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
