import { apiClient } from "./apiClient"
import type { JsonRecord } from "./display"

function recordList(value: unknown) {
  return Array.isArray(value)
    ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    : []
}

export async function fetchAllCursorRecords(endpoint: string, itemKey: string) {
  const records: JsonRecord[] = []
  const seenCursors = new Set<string>()
  let cursor = ""

  for (;;) {
    const url = new URL(endpoint, window.location.origin)
    url.searchParams.set("page_size", "100")
    if (cursor) url.searchParams.set("cursor", cursor)

    const data = await apiClient<JsonRecord>(`${url.pathname}${url.search}`)
    if (!Array.isArray(data[itemKey])) throw new Error(`Missing cursor list field: ${itemKey}`)
    records.push(...recordList(data[itemKey]))

    const nextCursor = String(data.next_cursor || "").trim()
    if (!nextCursor) return records
    if (nextCursor === cursor || seenCursors.has(nextCursor)) {
      throw new Error("Cursor pagination did not advance")
    }
    seenCursors.add(nextCursor)
    cursor = nextCursor
  }
}
