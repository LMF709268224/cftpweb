const DEFAULT_AUTH_REDIRECT = "/lms"
const AUTH_REDIRECT_KEY = "admin_auth_redirect"

export function resolveAuthRedirect(value: unknown) {
  if (typeof value !== "string") return DEFAULT_AUTH_REDIRECT

  const candidate = value.trim()
  if (
    !candidate.startsWith("/")
    || candidate.startsWith("//")
    || candidate.includes("\\")
    || /[\u0000-\u001f\u007f]/.test(candidate)
  ) return DEFAULT_AUTH_REDIRECT

  try {
    const url = new URL(candidate, window.location.origin)
    if (url.origin !== window.location.origin || url.pathname === "/login" || url.pathname === "/callback") {
      return DEFAULT_AUTH_REDIRECT
    }
    return `${url.pathname}${url.search}${url.hash}`
  } catch {
    return DEFAULT_AUTH_REDIRECT
  }
}

export function rememberAuthRedirect(value: unknown) {
  const redirect = resolveAuthRedirect(value)
  sessionStorage.setItem(AUTH_REDIRECT_KEY, redirect)
  return redirect
}

export function pendingAuthRedirect() {
  return resolveAuthRedirect(sessionStorage.getItem(AUTH_REDIRECT_KEY))
}

export function clearAuthRedirect() {
  sessionStorage.removeItem(AUTH_REDIRECT_KEY)
}

export function currentAuthRedirect() {
  return resolveAuthRedirect(`${window.location.pathname}${window.location.search}${window.location.hash}`)
}

export function loginPathWithRedirect(value: unknown) {
  return `/login?redirect=${encodeURIComponent(resolveAuthRedirect(value))}`
}
