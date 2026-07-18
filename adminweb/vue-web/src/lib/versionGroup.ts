export type LineageGroup<T> = {
  stem: string
  primary: T
  history: T[]
}

const COPY_SUFFIXES = [/\s*\(Copy\)\s*$/i, /\s*（副本）\s*$/, /\s*\(副本\)\s*$/]

/**
 * 去掉名称尾部所有 " (Copy)" / "（副本）" / " (副本)" 后缀（可叠加，如 "Foo (Copy)(Copy)"），返回名称主干。
 */
export function stripCopySuffix(name: string): string {
  let result = (name || "").trim()
  let changed = true
  while (changed && result) {
    changed = false
    for (const pattern of COPY_SUFFIXES) {
      const next = result.replace(pattern, "")
      if (next !== result) {
        result = next.trim()
        changed = true
      }
    }
  }
  return result
}

/**
 * 按名称主干把版本族聚为一组（纯函数，不修改入参）。
 *
 * - `getName`：取元素名称；名称去掉尾部所有 Copy/副本 后缀后作为 stem。
 * - `getKey`（可选）：附加分组维度（如分类），同 stem 且同 key 才归为一族。
 * - `items` 假定已按更新时间倒序排列：每组 `primary` = 组内第一个出现的元素，其余进 `history`。
 * - 返回顺序 = 各组首个元素在原列表中的出现顺序。
 */
export function groupByLineage<T>(
  items: T[],
  getName: (item: T) => string,
  getKey?: (item: T) => string,
): Array<LineageGroup<T>> {
  const groups = new Map<string, LineageGroup<T>>()
  const ordered: Array<LineageGroup<T>> = []

  for (const item of items) {
    const stem = stripCopySuffix(getName(item))
    const extraKey = getKey ? getKey(item) : ""
    const groupKey = `${extraKey}\u0000${stem}`
    const existing = groups.get(groupKey)
    if (existing) {
      existing.history.push(item)
    } else {
      const group: LineageGroup<T> = { stem, primary: item, history: [] }
      groups.set(groupKey, group)
      ordered.push(group)
    }
  }

  return ordered
}
