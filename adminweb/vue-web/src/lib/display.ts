export type JsonRecord = Record<string, unknown>

const idLikePattern = /(^id$|_id$|ulid|uuid|gpath|object_key|file_hash|json$)/i

export function formatDate(value: unknown) {
  if (typeof value !== "string" || !value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString("zh-CN", { hour12: false })
}

export function isPrimitive(value: unknown) {
  return value === null || ["string", "number", "boolean"].includes(typeof value)
}

export function humanizeKey(key: string) {
  return key
    .replace(/_/g, " ")
    .replace(/\b\w/g, (char) => char.toUpperCase())
    .replace(/\bUlid\b/g, "ULID")
}

export function getDisplayTitle(item: JsonRecord, fallback = "") {
  const keys = [
    "name",
    "title",
    "product_name",
    "subject",
    "email",
    "candidate_name",
    "course_title",
    "pipeline_name",
    "bundle_name",
    "template_name",
  ]

  for (const key of keys) {
    const value = item[key]
    if (typeof value === "string" && value.trim()) {
      return value
    }
  }

  return fallback
}

export function getDisplaySubtitle(item: JsonRecord, fallback = "") {
  const keys = ["description", "category_tips", "biz_type", "type", "status", "raw_status", "created_at"]
  for (const key of keys) {
    const value = item[key]
    if (typeof value === "string" && value.trim()) {
      return key.endsWith("_at") ? formatDate(value) : value
    }
  }
  return fallback
}

export function getListFields(item: JsonRecord) {
  return Object.entries(item)
    .filter(([key, value]) => !idLikePattern.test(key) && isPrimitive(value))
    .slice(0, 6)
    .map(([key, value]) => ({
      key,
      label: humanizeKey(key),
      value: key.endsWith("_at") ? formatDate(value) : String(value ?? "-"),
    }))
}

export function getStatusTone(value: unknown) {
  const text = String(value || "").toLowerCase()
  if (text.includes("completed") || text.includes("active") || text.includes("approved") || text.includes("paid")) {
    return "success"
  }
  if (text.includes("wait") || text.includes("pending") || text.includes("review") || text.includes("ready")) {
    return "warning"
  }
  if (text.includes("failed") || text.includes("cancel") || text.includes("reject") || text.includes("terminated")) {
    return "danger"
  }
  return "neutral"
}
