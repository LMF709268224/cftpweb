import { en } from "./locales/en"
import { zh } from "./locales/zh"

type Language = "zh" | "en"
type LocalizedCatalog = typeof zh

function catalogFor(lang: Language): LocalizedCatalog {
  return lang === "zh" ? zh : en
}

export const ErrorMessages: Record<string, { zh: string; en: string }> = Object.fromEntries(
  Object.keys(zh.apiErrors).map((code) => [
    code,
    {
      zh: zh.apiErrors[code as keyof typeof zh.apiErrors],
      en: en.apiErrors[code as keyof typeof en.apiErrors],
    },
  ]),
)

export function getErrorMessage(errorCode: string | undefined | null, lang: Language = "zh"): string {
  if (!errorCode) return ErrorMessages["UNKNOWN_ERROR"][lang]
  const tip = ErrorMessages[errorCode]
  if (tip) {
    return tip[lang]
  }
  return ErrorMessages["UNKNOWN_ERROR"][lang]
}

function getFieldLabel(field: string, lang: Language): string {
  const normalized = field.replace(/\[\d+\]/g, "")
  const labels = catalogFor(lang).apiFieldLabels as Record<string, string>
  return labels[field] || labels[normalized] || field
}

function formatMessage(template: string, values: Record<string, string>) {
  return Object.entries(values).reduce(
    (message, [key, value]) => message.replace(`{{${key}}}`, value),
    template,
  )
}

export function localizeApiErrorMessage(
  errorCode: string | undefined | null,
  message: string | undefined | null,
  lang: Language = "zh"
): string {
  if (!message) return getErrorMessage(errorCode, lang)
  const validation = catalogFor(lang).apiValidation

  let match = message.match(/^(.+) is required$/)
  if (match) {
    return formatMessage(validation.fieldRequired, { field: getFieldLabel(match[1], lang) })
  }

  match = message.match(/^(.+) are required$/)
  if (match) {
    const fields = match[1].split(/\s+and\s+/).map((field) => getFieldLabel(field, lang))
    return formatMessage(validation.fieldsRequired, { fields: fields.join(validation.fieldsJoiner) })
  }

  match = message.match(/^(.+) must be greater than 0$/)
  if (match) {
    return formatMessage(validation.greaterThanZero, { field: getFieldLabel(match[1], lang) })
  }

  match = message.match(/^(.+) is invalid$/)
  if (match) {
    return formatMessage(validation.invalid, { field: getFieldLabel(match[1], lang) })
  }

  match = message.match(/^course "([^"]+)" must contain at least one chapter before publishing$/)
  if (match) {
    return formatMessage(validation.courseNeedsChapter, { name: match[1] })
  }

  match = message.match(/^chapter "([^"]+)" must contain at least one lesson or quiz before publishing$/)
  if (match) {
    return formatMessage(validation.chapterNeedsContent, { name: match[1] })
  }

  match = message.match(/^published course "([^"]+)" cannot be modified$/)
  if (match) {
    return formatMessage(validation.publishedCourseImmutable, { name: match[1] })
  }

  const isGenericMessage = /^(Bad Request|Unauthorized|Forbidden|Not Found|Method Not Allowed|Internal Server Error|Bad Gateway|Service Unavailable|Error)$/i.test(message || "")

  if (errorCode && ErrorMessages[errorCode] && (!message || isGenericMessage)) {
    return getErrorMessage(errorCode, lang)
  }

  const isUntranslatedEnglish = lang === "zh"
    && /[A-Za-z]/.test(message)
    && !/[\u3400-\u9fff]/.test(message)
  if (isUntranslatedEnglish) {
    return getErrorMessage(errorCode, lang)
  }

  return message
}
