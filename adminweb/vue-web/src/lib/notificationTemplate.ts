type RecordValue = Record<string, unknown>

export function objectValue(value: unknown): value is RecordValue {
  return !!value && typeof value === "object" && !Array.isArray(value)
}

export function templateVariables(schemaText: string, texts: string[]) {
  const variables = new Map<string, string>()
  for (const text of texts) {
    for (const match of text.matchAll(/\{\{\s*\.?([A-Za-z_][A-Za-z0-9_]*)\s*\}\}/g)) {
      if (["else", "end", "break", "continue", "nil", "true", "false"].includes(match[1]!) && !match[0].includes(".")) continue
      if (!variables.has(match[1]!)) variables.set(match[1]!, match[0])
    }
  }
  try {
    const schema: unknown = JSON.parse(schemaText || "{}")
    if (objectValue(schema) && objectValue(schema.properties)) {
      for (const name of Object.keys(schema.properties)) {
        if (/^[A-Za-z_][A-Za-z0-9_]*$/.test(name) && !variables.has(name)) variables.set(name, `{{${name}}}`)
      }
    }
  } catch {
    // A malformed schema must not prevent editing the original template text.
  }
  return [...variables].map(([name, token]) => ({ name, token }))
}

export function parameterFields(schemaText: string, texts: string[]) {
  const schema: unknown = JSON.parse(schemaText || "{}")
  if (!objectValue(schema)) throw new Error("schema")
  const properties = objectValue(schema.properties) ? schema.properties : {}
  const required = Array.isArray(schema.required) ? schema.required : []
  const names = new Set([...Object.keys(properties), ...required.filter((name): name is string => typeof name === "string"), ...templateVariables("{}", texts).map(item => item.name)])
  return [...names].map(name => {
    const rule = objectValue(properties[name]) ? properties[name] : {}
    const type = typeof rule.type === "string" ? rule.type : "string"
    // Complex schemas retain the existing JSON editor, without coercing values.
    const supported = ["string", "number", "integer", "boolean"].includes(type)
      && (rule.type === undefined || typeof rule.type === "string")
      && (!Array.isArray(rule.enum) || rule.enum.every(value => typeof value === (type === "integer" ? "number" : type)))
      && Object.keys(rule).every(key => ["type", "title", "description", "enum", "format", "minLength", "maxLength", "pattern", "minimum", "maximum", "default", "examples"].includes(key))
      && (!rule.format || ["email", "uri", "date", "date-time"].includes(String(rule.format)))
    return { name, rule, type, required: required.includes(name), supported }
  })
}

export function initialTemplatePayload(schema: string, texts: string[]) {
  // Do not manufacture numeric/boolean values or populate schema defaults.
  return JSON.stringify(Object.fromEntries(templateVariables(schema, texts).map(item => [item.name, ""])), null, 2)
}
