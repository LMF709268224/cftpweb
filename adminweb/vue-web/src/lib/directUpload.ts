import type { JsonRecord } from "./display"

export const MAX_BASIC_VIDEO_UPLOAD_BYTES = 200 * 1024 * 1024

export function isVideoFile(file: File) {
  return file.type.toLowerCase().startsWith("video/")
}

export async function sha256Hex(file: File) {
  const hashBuffer = await crypto.subtle.digest("SHA-256", await file.arrayBuffer())
  return Array.from(new Uint8Array(hashBuffer))
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("")
}

function signedHeaders(value: unknown) {
  if (!value || typeof value !== "object" || Array.isArray(value)) return {}
  return Object.fromEntries(
    Object.entries(value)
      .filter((entry): entry is [string, string] => typeof entry[1] === "string"),
  )
}

async function uploadError(response: Response) {
  const detail = (await response.text()).trim().slice(0, 500)
  return detail ? `Upload failed: ${response.status} ${detail}` : `Upload failed: ${response.status}`
}

export async function uploadToDirectURL(file: File, upload: JsonRecord) {
  const uploadURL = String(upload.upload_url || "").trim()
  if (!uploadURL) throw new Error("Missing upload URL")

  const provider = String(upload.bucket_name || "").trim().toLowerCase()
  let response: Response

  if (provider === "cloudflare") {
    if (!isVideoFile(file)) throw new Error("Cloudflare Stream only accepts video files")
    if (file.size > MAX_BASIC_VIDEO_UPLOAD_BYTES) {
      throw new Error("Video files must not exceed 200MB")
    }
    const formData = new FormData()
    formData.append("file", file, file.name)
    response = await fetch(uploadURL, {
      method: "POST",
      body: formData,
    })
  } else {
    response = await fetch(uploadURL, {
      method: "PUT",
      body: file,
      headers: signedHeaders(upload.signed_headers),
    })
  }

  if (!response.ok) throw new Error(await uploadError(response))
}
