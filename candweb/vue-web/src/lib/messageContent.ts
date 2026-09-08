import type { Lang } from "./language"

const candidatePortalBaseURL = "{{CandidatePortalBaseURL}}"
const languageMarkerPattern = /<!--\s*lang\s*:\s*(zh(?:-cn)?|en(?:-us)?)\s*-->/gi

type MessageLocale = "zh-CN" | "en-US"

function normalizeMessageLocale(value: string): MessageLocale | null {
  const normalized = value.trim().toLowerCase()
  if (normalized === "zh" || normalized === "zh-cn") return "zh-CN"
  if (normalized === "en" || normalized === "en-us") return "en-US"
  return null
}

function messageLocale(lang: Lang): MessageLocale {
  return lang === "zh" ? "zh-CN" : "en-US"
}

export function hasLocalizedMessageSections(value: string) {
  languageMarkerPattern.lastIndex = 0
  return languageMarkerPattern.test(value)
}

export function selectLocalizedMessageText(value: string, lang: Lang) {
  const source = String(value || "").replace(/\r\n/g, "\n")
  languageMarkerPattern.lastIndex = 0
  const markers = [...source.matchAll(languageMarkerPattern)]
  if (markers.length === 0) return source.trim()

  const sections = new Map<MessageLocale, string[]>()
  for (let index = 0; index < markers.length; index += 1) {
    const marker = markers[index]
    const locale = normalizeMessageLocale(marker[1] || "")
    if (!locale || marker.index === undefined) continue

    const contentStart = marker.index + marker[0].length
    const contentEnd = markers[index + 1]?.index ?? source.length
    const content = source.slice(contentStart, contentEnd).trim()
    if (content) sections.set(locale, [...(sections.get(locale) || []), content])
  }

  const preferredLocale = messageLocale(lang)
  const preferred = sections.get(preferredLocale)
  if (preferred?.length) return preferred.join("\n\n")

  const englishFallback = sections.get("en-US")
  if (englishFallback?.length) return englishFallback.join("\n\n")

  return [...sections.values()][0]?.join("\n\n") || ""
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;")
}

function renderInlineText(value: string) {
  return escapeHtml(value)
    .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
    .replace(/\{\{.*?\}\}/g, "")
}

type SafeMessageLink = {
  href: string
  internal: boolean
}

export function safeMessageLink(target: string): SafeMessageLink | null {
  let href = target.trim()
  if (href.startsWith(candidatePortalBaseURL)) {
    href = href.slice(candidatePortalBaseURL.length)
  }

  if (/^\/(?!\/)/.test(href) && !/[\\\s<>"']/.test(href)) {
    return { href, internal: true }
  }

  try {
    const url = new URL(href)
    if (url.protocol !== "http:" && url.protocol !== "https:") return null
    return { href: url.toString(), internal: false }
  } catch {
    return null
  }
}

function renderInlineMarkdown(value: string) {
  const linkPattern = /\[([^\]\n]+)\]\(([^)\n]+)\)/g
  let rendered = ""
  let previousEnd = 0

  for (const match of value.matchAll(linkPattern)) {
    if (match.index === undefined) continue
    rendered += renderInlineText(value.slice(previousEnd, match.index))

    const link = safeMessageLink(match[2] || "")
    if (!link) {
      rendered += renderInlineText(match[0])
    } else if (link.internal) {
      rendered += `<a href="${escapeHtml(link.href)}" data-message-internal-link="true" class="text-primary underline underline-offset-2">${renderInlineText(match[1] || "")}</a>`
    } else {
      rendered += `<a href="${escapeHtml(link.href)}" target="_blank" rel="noopener noreferrer" class="text-primary underline underline-offset-2">${renderInlineText(match[1] || "")}</a>`
    }
    previousEnd = match.index + match[0].length
  }

  rendered += renderInlineText(value.slice(previousEnd))
  return rendered
}

export function messageMarkdownToHtml(markdown: string, lang: Lang) {
  const source = selectLocalizedMessageText(markdown, lang)
  if (!source) return ""

  const lines = source.split("\n")
  const html: string[] = []
  let listItems: string[] = []
  const flushList = () => {
    if (listItems.length === 0) return
    html.push(`<ul class="my-3 list-disc space-y-1 pl-5">${listItems.join("")}</ul>`)
    listItems = []
  }

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) {
      flushList()
      continue
    }

    const heading = line.match(/^(#{1,6})\s+(.+)$/)
    if (heading) {
      flushList()
      const level = Math.min(heading[1].length + 2, 6)
      html.push(`<h${level} class="mt-3 font-semibold text-foreground">${renderInlineMarkdown(heading[2])}</h${level}>`)
      continue
    }

    const bullet = line.match(/^[-*]\s+(.+)$/)
    if (bullet) {
      listItems.push(`<li>${renderInlineMarkdown(bullet[1])}</li>`)
      continue
    }

    flushList()
    html.push(`<p class="my-2">${renderInlineMarkdown(line)}</p>`)
  }
  flushList()
  return html.join("")
}
