type LearningUnitNameMessages = {
  l0Foundation: string
  l1aFinance: string
  l1bFintech: string
}

export function learningUnitDisplayName(value: unknown, messages: LearningUnitNameMessages) {
  const name = String(value || "").trim()
  const normalized = name.toLowerCase().replace(/\s+/g, " ")

  if (["l0 course", "l0 foundation", "cftp foundation course"].includes(normalized)) {
    return messages.l0Foundation
  }
  if (normalized === "l1a finance") return messages.l1aFinance
  if (normalized === "l1b fintech") return messages.l1bFintech
  return name
}
