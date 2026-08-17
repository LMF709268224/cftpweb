import DOMPurify from "dompurify"

const COURSE_CONTENT_TAGS = [
  "a",
  "blockquote",
  "br",
  "code",
  "div",
  "em",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "hr",
  "img",
  "li",
  "ol",
  "p",
  "pre",
  "span",
  "strong",
  "sub",
  "sup",
  "table",
  "tbody",
  "td",
  "th",
  "thead",
  "tr",
  "u",
  "ul",
]

const COURSE_CONTENT_ATTRS = [
  "alt",
  "class",
  "colspan",
  "height",
  "href",
  "loading",
  "rel",
  "rowspan",
  "scope",
  "src",
  "start",
  "target",
  "title",
  "width",
]

function normalizeLinksAndImages(html: string) {
  const template = document.createElement("template")
  template.innerHTML = html

  template.content.querySelectorAll("a").forEach((link) => {
    if (link.target === "_blank") link.rel = "noopener noreferrer"
  })
  template.content.querySelectorAll("img").forEach((image) => {
    image.loading = "lazy"
  })

  return template.innerHTML
}

export function sanitizeCourseContent(value?: string | null) {
  const sanitized = DOMPurify.sanitize(String(value || ""), {
    ALLOWED_TAGS: COURSE_CONTENT_TAGS,
    ALLOWED_ATTR: COURSE_CONTENT_ATTRS,
    ALLOW_DATA_ATTR: false,
    FORBID_ATTR: ["srcdoc", "style"],
    FORBID_TAGS: ["button", "embed", "form", "iframe", "input", "math", "object", "script", "select", "style", "svg", "textarea"],
  })

  return normalizeLinksAndImages(sanitized)
}
