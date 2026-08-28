export function isSystemCredentialDefinition(definition: any) {
  const resourcePath = String(definition?.respath || definition?.res_path || "").trim().toLowerCase()
  if (!resourcePath) return false
  return resourcePath.split(/[\\/]+/).filter(Boolean).includes("system")
}

export function candidateVisibleCredentialDefinitions(definitions: unknown) {
  return (Array.isArray(definitions) ? definitions : []).filter((definition) => !isSystemCredentialDefinition(definition))
}
