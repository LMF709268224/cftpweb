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
  const hasLeadingPlus = trimmed.startsWith("+")
  const digits = trimmed.replace(/\D/g, "").slice(0, 15)
  return `${hasLeadingPlus ? "+" : ""}${digits}`
}

export function isValidInternationalPhone(value: unknown, required = false) {
  const normalized = normalizeInternationalPhone(value)
  if (!normalized) return !required
  const digits = normalized.replace(/\D/g, "")
  return digits.length >= 7 && digits.length <= 15 && /^\+?\d+$/.test(normalized)
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
