import dayjs from "dayjs"
export function formatBackendDate(dateStr?: string | null): string {
  if (!dateStr) return ""
  return dayjs(dateStr).format("YYYY-MM-DD HH:mm:ss")
}

export function formatBackendDateMinute(dateStr?: string | null): string {
  if (!dateStr) return ""
  return dayjs(dateStr).format("YYYY-MM-DD HH:mm")
}

export function formatBackendDateOnly(dateStr?: string | null): string {
  if (!dateStr) return ""
  return dayjs(dateStr).format("YYYY-MM-DD")
}

export function resolvePath(object: Record<string, any>, path: string) {
  return path.split(".").reduce<any>((current, key) => current?.[key], object)
}