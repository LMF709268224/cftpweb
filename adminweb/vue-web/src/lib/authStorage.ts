const LEGACY_TOKEN_KEY = "access_token"
const AUTH_KEY = "is_authenticated"
const USER_NAME_KEY = "user_name"

export function isAuthenticated() {
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

export function getUserName() {
  return localStorage.getItem(USER_NAME_KEY) || "Admin"
}
