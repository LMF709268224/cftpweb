export type EligibilityBlockerLike = {
  blocker_type?: string
  description?: string
  details?: unknown[]
}

function qualificationReferenceLabel(reference: string) {
  const segments = reference.split("/").filter(Boolean)
  const code = segments.at(-1) || reference
  return code.replace(/[-_]+/g, " ").toUpperCase()
}

export function membershipPrerequisiteQualificationLabels(
  blocker: EligibilityBlockerLike,
  configuredReferences: unknown,
) {
  if (blocker.blocker_type !== "MISSING_PREREQUISITE_QUALIFICATION" || !Array.isArray(configuredReferences)) return []

  const configured = Array.from(new Set(
    configuredReferences.map((reference) => String(reference || "").trim()).filter(Boolean),
  ))
  const details = new Set(
    (Array.isArray(blocker.details) ? blocker.details : [])
      .map((reference) => String(reference || "").trim())
      .filter(Boolean),
  )

  let missingReferences = configured.filter((reference) => details.has(reference))
  if (missingReferences.length === 0) {
    const description = String(blocker.description || "")
    missingReferences = configured.filter((reference) => description.includes(reference))
  }
  if (missingReferences.length === 0 && configured.length === 1) missingReferences = configured

  return missingReferences.map(qualificationReferenceLabel)
}
