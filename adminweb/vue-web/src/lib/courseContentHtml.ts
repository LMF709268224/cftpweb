import DOMPurify from "dompurify"

// Keep these rules aligned with candweb/src/lib/sanitizeHtml.ts.
export function courseContentHtml(value: string) {
  const html = DOMPurify.sanitize(value, {
    ALLOWED_TAGS: ["a", "blockquote", "br", "code", "div", "em", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "img", "li", "ol", "p", "pre", "span", "strong", "sub", "sup", "table", "tbody", "td", "th", "thead", "tr", "u", "ul"],
    ALLOWED_ATTR: ["alt", "class", "colspan", "height", "href", "loading", "rel", "rowspan", "scope", "src", "start", "target", "title", "width"],
    ALLOW_DATA_ATTR: false,
    FORBID_ATTR: ["srcdoc", "style"],
    FORBID_TAGS: ["button", "embed", "form", "iframe", "input", "math", "object", "script", "select", "style", "svg", "textarea"],
  })
  const template = document.createElement("template")
  template.innerHTML = html
  template.content.querySelectorAll("a").forEach((link) => {
    if (link.target === "_blank") link.rel = "noopener noreferrer"
  })
  template.content.querySelectorAll("img").forEach((image) => { image.loading = "lazy" })
  return template.innerHTML
}

export function hasCourseBodyContent(value: string) {
  const template = document.createElement("template")
  template.innerHTML = courseContentHtml(value)
  return !!template.content.textContent?.trim() || !!template.content.querySelector("img[src]")
}

// Do not silently flatten custom layouts when starting visual editing.
export function supportsVisualCourseContent(value: string) {
  const template = document.createElement("template")
  template.innerHTML = value
  const tags = new Set(["A", "BLOCKQUOTE", "BR", "CODE", "EM", "H1", "H2", "H3", "H4", "H5", "H6", "HR", "IMG", "LI", "OL", "P", "PRE", "STRONG", "TABLE", "TBODY", "TD", "TH", "THEAD", "TR", "U", "UL"])
  const attrs = new Set(["href", "target", "rel", "src", "alt", "title", "colspan", "rowspan", "start"])
  return [...template.content.querySelectorAll("*")].every((element) => tags.has(element.tagName) && [...element.attributes].every((attribute) => attrs.has(attribute.name)))
}
