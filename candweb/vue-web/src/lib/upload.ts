export const UPLOAD_TIMEOUT_MS = 30000

export async function sha256Hex(file: File) {
  const buffer = await file.arrayBuffer()
  const hash = await crypto.subtle.digest("SHA-256", buffer)
  return Array.from(new Uint8Array(hash)).map((byte) => byte.toString(16).padStart(2, "0")).join("")
}

export async function uploadWithTimeout(url: string, init: RequestInit) {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), UPLOAD_TIMEOUT_MS)
  try {
    return await fetch(url, { ...init, signal: controller.signal })
  } finally {
    window.clearTimeout(timeoutId)
  }
}
