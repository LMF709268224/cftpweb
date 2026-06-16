const ACCESS_TOKEN_KEY = "access_token"

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

function syncAccessTokenCookie(token: string) {
  document.cookie = `${ACCESS_TOKEN_KEY}=${encodeURIComponent(token)}; Path=/; SameSite=Lax`
}
