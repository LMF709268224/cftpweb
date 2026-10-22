export const PROFILE_TEXT_LIMITS = {
  name: 20,
  short: 64,
  address: 160,
  bio: 500,
  postalCode: 16,
}

export const GENDER_OPTIONS = ["Male", "Female"] as const

export type GenderOption = (typeof GENDER_OPTIONS)[number]

export function isValidEmail(value: unknown) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(String(value || "").trim())
}

export type BirthdateError = "invalid" | "future" | "underage" | ""

export function validateAdultBirthdate(value: unknown, today = new Date()): BirthdateError {
  const text = String(value || "").trim()
  if (!/^\d{4}-\d{2}-\d{2}$/.test(text)) return "invalid"
  const [year, month, day] = text.split("-").map(Number)
  const date = new Date(`${text}T00:00:00Z`)
  if (Number.isNaN(date.getTime()) || date.getUTCFullYear() !== year ||
      date.getUTCMonth() + 1 !== month || date.getUTCDate() !== day) return "invalid"
  const todayText = today.toISOString().slice(0, 10)
  if (text > todayText) return "future"
  const cutoff = new Date(Date.UTC(today.getUTCFullYear() - 18, today.getUTCMonth(), today.getUTCDate()))
  if (date > cutoff) return "underage"
  return ""
}

const genderAliases: Record<string, GenderOption> = {
  male: "Male",
  m: "Male",
  man: "Male",
  boy: "Male",
  "1": "Male",
  男: "Male",
  男性: "Male",
  female: "Female",
  f: "Female",
  woman: "Female",
  girl: "Female",
  "2": "Female",
  女: "Female",
  女性: "Female",
}

export function normalizeGender(value: unknown) {
  const text = String(value || "").trim()
  if (!text) return ""
  return genderAliases[text.toLowerCase()] || genderAliases[text] || ""
}

export function normalizeInternationalPhone(value: unknown) {
  const trimmed = String(value || "").trim()
  if (!/^\+?[\d\s()-]*$/.test(trimmed)) return trimmed
  const hasLeadingPlus = trimmed.startsWith("+")
  const digits = trimmed.replace(/\D/g, "")
  return `${hasLeadingPlus ? "+" : ""}${digits}`
}

export function isValidInternationalPhone(value: unknown, required = false) {
  const normalized = normalizeInternationalPhone(value)
  if (!normalized) return !required
  return /^\+?\d{7,15}$/.test(normalized)
}

export function normalizePostalCode(value: unknown) {
  return String(value || "").trim().replace(/\s+/g, " ").toUpperCase()
}

export function isValidPostalCode(value: unknown, required = false) {
  const normalized = normalizePostalCode(value)
  if (!normalized) return !required
  return normalized.length >= 2 && normalized.length <= PROFILE_TEXT_LIMITS.postalCode && /^[A-Z0-9][A-Z0-9 -]*[A-Z0-9]$/.test(normalized)
}

export function trimToMax(value: unknown, maxLength: number) {
  return String(value || "").trim().slice(0, maxLength)
}
