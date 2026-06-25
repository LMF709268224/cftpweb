const TOKEN_KEY = "access_token"
const AUTH_KEY = "is_authenticated"
const USER_NAME_KEY = "user_name"

export function getAccessToken() {
  return localStorage.getItem(TOKEN_KEY) || ""
}

export function setAuthSession(token: string, userName?: string) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token)
    localStorage.setItem(AUTH_KEY, "true")
  }

  if (userName) {
    localStorage.setItem(USER_NAME_KEY, userName)
  }
}

export function clearAuthSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(AUTH_KEY)
  localStorage.removeItem(USER_NAME_KEY)
}

export function getUserName() {
  return localStorage.getItem(USER_NAME_KEY) || "Admin"
}
