
export function formatBackendDate(dateStr?: string | null): string {
  if (!dateStr) return ""
  const safeStr = dateStr.endsWith("Z") ? dateStr.slice(0, -1) : dateStr
  const d = new Date(safeStr)
  if (Number.isNaN(d.getTime())) return dateStr

  const pad = (n: number) => n.toString().padStart(2, "0")
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export function resolvePath(object: Record<string, any>, path: string) {
  return path.split(".").reduce<any>((current, key) => current?.[key], object)
}
