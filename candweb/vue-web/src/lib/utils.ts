import dayjs from "dayjs"

type DateValue = string | number | Date | null | undefined

function parseDateValue(value: DateValue) {
  if (value === null || value === undefined || value === "") return null

  if (typeof value === "string" && /^\d{10,13}$/.test(value.trim())) {
    const numericValue = Number(value)
    return dayjs(numericValue < 10000000000 ? numericValue * 1000 : numericValue)
  }

  return dayjs(value)
}

function formatDateValue(value: DateValue, pattern: string): string {
  const date = parseDateValue(value)
  if (!date) return ""
  return date.isValid() ? date.format(pattern) : String(value)
}

export function formatBackendDate(value?: DateValue): string {
  return formatDateValue(value, "DD MMM YYYY, HH:mm:ss")
}

export function formatBackendDateMinute(value?: DateValue): string {
  return formatDateValue(value, "DD MMM YYYY, HH:mm")
}

export function formatBackendDateOnly(value?: DateValue): string {
  return formatDateValue(value, "DD MMM YYYY")
}

export function resolvePath(object: Record<string, any>, path: string) {
  return path.split(".").reduce<any>((current, key) => current?.[key], object)
}
