import { hasLocalizedMessageSections, messageMarkdownToHtml, selectLocalizedMessageText } from "../../../../candweb/vue-web/src/lib/messageContent"

export type MessagePreviewLanguage = "zh" | "en"

function formatPayloadValue(value: unknown): string {
  if (value === null || value === undefined) return "<nil>"
  if (Array.isArray(value)) return `[${value.map(formatPayloadValue).join(" ")}]`
  if (typeof value === "object") {
    const entries = Object.entries(value).sort(([left], [right]) => left.localeCompare(right))
    return `map[${entries.map(([key, item]) => `${key}:${formatPayloadValue(item)}`).join(" ")}]`
  }
  return String(value)
}

export function renderMessageTemplateText(template: string, payload: Record<string, unknown>): string {
  let rendered = template
  for (const [key, value] of Object.entries(payload)) {
    const replacement = formatPayloadValue(value)
    rendered = rendered.replaceAll(`{{${key}}}`, replacement).replaceAll(`{{ ${key} }}`, replacement)
  }
  return rendered
}

export function previewMessageTitle(title: string, language: MessagePreviewLanguage): string {
  const localized = selectLocalizedMessageText(title, language)
  const cleaned = localized
    .replace(/^#{1,6}\s*/gm, "")
    .replace(/\*\*(.*?)\*\*/g, "$1")
    .replace(/\[(.*?)\]\((.*?)\)/g, "$1")
    .replace(/\{\{.*?\}\}/g, "")
    .replace(/\s+/g, " ")
    .trim()
  if (!cleaned || hasLocalizedMessageSections(title)) return cleaned

  const parts = cleaned.replace(/\s*\/\s*/g, "\n").split("\n").map((part) => part.trim()).filter(Boolean)
  if (parts.length <= 1) return cleaned
  const score = language === "zh"
    ? (part: string) => (part.match(/[\u3400-\u9fff]/g)?.length || 0) * 2 - (part.match(/[A-Za-z]/g)?.length || 0) * 0.15
    : (part: string) => (part.match(/[A-Za-z]/g)?.length || 0) - (part.match(/[\u3400-\u9fff]/g)?.length || 0) * 1.5
  return [...parts].sort((left, right) => score(right) - score(left))[0] || cleaned
}

export function previewMessageContent(content: string, language: MessagePreviewLanguage): string {
  return messageMarkdownToHtml(content, language)
}
