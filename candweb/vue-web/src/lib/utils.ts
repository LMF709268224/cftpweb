import dayjs from "dayjs"
import type { Lang } from "@/lib/language"

type DateValue = string | number | Date | null | undefined

function parseDateValue(value: DateValue) {
  if (value === null || value === undefined || value === "") return null

  if (typeof value === "string" && /^\d{10,13}$/.test(value.trim())) {
    const numericValue = Number(value)
    return dayjs(numericValue < 10000000000 ? numericValue * 1000 : numericValue)
  }

  return dayjs(value)
}

function formatDateValue(value: DateValue, lang: Lang, englishPattern: string, chinesePattern: string): string {
  const date = parseDateValue(value)
  if (!date) return ""
  return date.isValid() ? date.format(lang === "zh" ? chinesePattern : englishPattern) : String(value)
}

export function formatBackendDate(value?: DateValue, lang: Lang = "en"): string {
  return formatDateValue(value, lang, "DD MMM YYYY, HH:mm:ss", "YYYY年M月D日 HH:mm:ss")
}

export function formatBackendDateMinute(value?: DateValue, lang: Lang = "en"): string {
  return formatDateValue(value, lang, "DD MMM YYYY, HH:mm", "YYYY年M月D日 HH:mm")
}

export function formatBackendDateOnly(value?: DateValue, lang: Lang = "en"): string {
  return formatDateValue(value, lang, "DD MMM YYYY", "YYYY年M月D日")
}

export function resolvePath(object: Record<string, any>, path: string) {
  return path.split(".").reduce<any>((current, key) => current?.[key], object)
}
