export type JsonRecord = Record<string, unknown>

type MinorAmountFormatOptions = {
  fractionDigits?: number
  useGrouping?: boolean
}

function integerString(value: unknown) {
  if (typeof value === "bigint") return value.toString()
  if (typeof value === "number") return Number.isSafeInteger(value) ? String(value) : null
  if (typeof value !== "string") return null
  const normalized = value.trim()
  return /^[+-]?\d+$/.test(normalized) ? normalized : null
}

export function formatMinorAmount(value: unknown, options: MinorAmountFormatOptions = {}) {
  const fractionDigits = options.fractionDigits ?? 2
  if (!Number.isInteger(fractionDigits) || fractionDigits < 0 || fractionDigits > 20) return null

  const normalized = integerString(value)
  if (normalized === null) return null

  const amount = BigInt(normalized)
  const negative = amount < 0n
  const absolute = negative ? -amount : amount
  const divisor = 10n ** BigInt(fractionDigits)
  const whole = absolute / divisor
  const fraction = fractionDigits
    ? (absolute % divisor).toString().padStart(fractionDigits, "0").replace(/0+$/, "")
    : ""
  const wholeText = options.useGrouping ? whole.toLocaleString("zh-CN") : whole.toString()

  return `${negative ? "-" : ""}${wholeText}${fraction ? `.${fraction}` : ""}`
}

export function formatDecimalAmount(value: unknown) {
  if (typeof value === "number") {
    if (!Number.isFinite(value)) return null
    return Object.is(value, -0) ? "0" : String(value)
  }
  if (typeof value === "bigint") return value.toString()
  if (typeof value !== "string") return null

  const normalized = value.trim()
  const match = /^([+-]?)(\d+)(?:\.(\d+))?$/.exec(normalized)
  if (!match) return null

  const whole = match[2].replace(/^0+(?=\d)/, "")
  const fraction = (match[3] || "").replace(/0+$/, "")
  const sign = match[1] === "-" && (whole !== "0" || fraction) ? "-" : ""
  return `${sign}${whole}${fraction ? `.${fraction}` : ""}`
}

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
