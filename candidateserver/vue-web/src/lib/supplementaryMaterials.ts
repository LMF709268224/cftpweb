export type SupplementaryMaterial = {
  material_id?: string
  materialId?: string
  course_id?: string
  courseId?: string
  kind?: string
  data_json?: string | unknown[] | Record<string, unknown>
  dataJson?: string | unknown[] | Record<string, unknown>
  version?: number
  created_at?: string
  updated_at?: string
}

export type SupplementaryMaterialItem = {
  key: string
  chapter: string
  type: string
  title: string
  description: string
  url: string
  sourceKind: string
}

export function normalizeSupplementaryMaterials(raw: SupplementaryMaterial | SupplementaryMaterial[] | unknown): SupplementaryMaterial[] {
  if (!raw) return []
  if (Array.isArray(raw)) return raw.filter(isRecord) as SupplementaryMaterial[]
  if (isRecord(raw)) return [raw as SupplementaryMaterial]
  return []
}

export function parseSupplementaryMaterialItems(materials: SupplementaryMaterial[], fallbackChapter = "Chapter"): SupplementaryMaterialItem[] {
  return materials.flatMap((material, materialIndex) => {
    const data = parseSupplementaryJson(material.data_json ?? material.dataJson)
    const records = supplementaryRecordsFromData(data)

    return records.map((record, recordIndex) => {
      const title = stringFromRecord(record, ["title", "name", "label", "heading"])
      const description = stringFromRecord(record, ["description", "desc", "summary", "detail", "content"])
      const type = stringFromRecord(record, ["type", "material_type", "resource_type", "kind"]) || "Material"
      const chapter = stringFromRecord(record, ["chapter", "chapter_title", "chapterTitle", "section"]) || fallbackChapter
      const url = stringFromRecord(record, ["url", "link", "href", "external_url", "externalUrl"])
      const materialId = material.material_id || material.materialId || "supplementary"
      const fallbackKey = `${materialId}-${materialIndex}-${recordIndex}`

      return {
        key: stringFromRecord(record, ["id", "material_id", "materialId", "resource_id", "resourceId", "key"]) || fallbackKey,
        chapter,
        type,
        title: title || "Untitled material",
        description,
        url,
        sourceKind: material.kind || "",
      }
    })
  })
}

export function parseSupplementaryJson(dataJson: SupplementaryMaterial["data_json"]) {
  if (!dataJson) return null
  if (typeof dataJson !== "string") return dataJson

  const trimmed = dataJson.trim()
  if (!trimmed) return null

  try {
    const parsed = JSON.parse(trimmed)
    if (typeof parsed === "string" && parsed.trim()) return parseSupplementaryJson(parsed)
    return parsed
  } catch {
    return null
  }
}

export function supplementaryRecordsFromData(data: unknown): Record<string, unknown>[] {
  if (Array.isArray(data)) return data.filter(isRecord)
  if (!isRecord(data)) return []

  for (const key of ["items", "resources", "materials", "data", "data_json", "dataJson", "list"]) {
    const value = data[key]
    if (Array.isArray(value)) return value.filter(isRecord)
    const parsed = parseSupplementaryJson(value as SupplementaryMaterial["data_json"])
    if (Array.isArray(parsed)) return parsed.filter(isRecord)
  }

  return [data]
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

function stringFromRecord(record: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = record[key]
    if (typeof value === "string" && value.trim()) return value.trim()
    if (typeof value === "number" && Number.isFinite(value)) return String(value)
  }
  return ""
}
