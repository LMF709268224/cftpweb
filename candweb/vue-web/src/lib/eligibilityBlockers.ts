export type EligibilityBlockerLike = {
  blocker_type?: string
  description?: string
  details?: unknown[]
}

const credentialDefinitionIDPattern = /^[0-9A-HJKMNP-TV-Z]{26}$/i
const membershipPrerequisiteDescriptionPattern = /\bprerequisite qualification\s+(\/\S+)\s+for membership\b/i

function qualificationReferenceLabel(reference: unknown) {
  const value = String(reference || "").trim()
  if (!value || credentialDefinitionIDPattern.test(value)) return ""
  if (!value.startsWith("/")) return value

  const segments = value.split("/").filter(Boolean)
  const code = segments.at(-1) || ""
  return code.replace(/[-_]+/g, " ").toUpperCase()
}

export function membershipPrerequisiteQualificationLabels(blocker?: EligibilityBlockerLike) {
  if (!blocker || blocker.blocker_type !== "MISSING_PREREQUISITE_QUALIFICATION") return []

  const structuredLabels = (Array.isArray(blocker.details) ? blocker.details : [])
    .map(qualificationReferenceLabel)
    .filter(Boolean)
  if (structuredLabels.length > 0) return Array.from(new Set(structuredLabels))

  const descriptionMatch = String(blocker.description || "").match(membershipPrerequisiteDescriptionPattern)
  const fallbackLabel = qualificationReferenceLabel(descriptionMatch?.[1])
  return fallbackLabel ? [fallbackLabel] : []
}
