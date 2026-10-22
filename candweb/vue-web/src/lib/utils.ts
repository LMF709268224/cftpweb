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

export function formatNotificationDate(value?: DateValue, lang: Lang = "en", now = Date.now()): string {
  const date = parseDateValue(value)
  if (!date?.isValid()) return formatBackendDate(value, lang)

  const elapsed = now - date.valueOf()
  if (elapsed < 0 || elapsed >= 7 * 24 * 60 * 60 * 1000) return formatBackendDate(value, lang)
  if (elapsed < 60 * 1000) return lang === "zh" ? "刚刚" : "Just now"

  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  const unit = elapsed < hour ? "minute" : elapsed < day ? "hour" : "day"
  const divisor = unit === "minute" ? minute : unit === "hour" ? hour : day
  return new Intl.RelativeTimeFormat(lang === "zh" ? "zh" : "en", { numeric: "always" })
    .format(-Math.floor(elapsed / divisor), unit)
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
