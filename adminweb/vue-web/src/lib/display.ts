export type JsonRecord = Record<string, unknown>

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
