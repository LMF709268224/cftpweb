type MinorAmountFormatOptions = {
  fractionDigits?: number
  useGrouping?: boolean
}

function integerString(value: unknown) {
  if (typeof value === "bigint") return value.toString()
  if (typeof value === "number") return Number.isSafeInteger(value) ? String(value) : null
  if (typeof value !== "string") return null
  const normalized = value.trim()
  return /^[+-]?\d+$/.test(normalized) ? normalized : null
}

export function formatMinorAmount(value: unknown, options: MinorAmountFormatOptions = {}) {
  const fractionDigits = options.fractionDigits ?? 2
  if (!Number.isInteger(fractionDigits) || fractionDigits < 0 || fractionDigits > 20) return null

  const normalized = integerString(value)
  if (normalized === null) return null

  const amount = BigInt(normalized)
  const negative = amount < 0n
  const absolute = negative ? -amount : amount
  const divisor = 10n ** BigInt(fractionDigits)
  const whole = absolute / divisor
  const fraction = fractionDigits
    ? (absolute % divisor).toString().padStart(fractionDigits, "0").replace(/0+$/, "")
    : ""
  const wholeText = options.useGrouping ? whole.toLocaleString("zh-CN") : whole.toString()

  return `${negative ? "-" : ""}${wholeText}${fraction ? `.${fraction}` : ""}`
}

export function formatDecimalAmount(value: unknown) {
  if (typeof value === "number") {
    if (!Number.isFinite(value)) return null
    return Object.is(value, -0) ? "0" : String(value)
  }
  if (typeof value === "bigint") return value.toString()
  if (typeof value !== "string") return null

  const normalized = value.trim()
  const match = /^([+-]?)(\d+)(?:\.(\d+))?$/.exec(normalized)
  if (!match) return null

  const whole = match[2].replace(/^0+(?=\d)/, "")
  const fraction = (match[3] || "").replace(/0+$/, "")
  const sign = match[1] === "-" && (whole !== "0" || fraction) ? "-" : ""
  return `${sign}${whole}${fraction ? `.${fraction}` : ""}`
}

export function formatCurrencyAmount(value: unknown, currency = "USD") {
  const amount = formatDecimalAmount(value)
  if (amount === null) return null

  const normalizedCurrency = String(currency || "USD").trim().toUpperCase()
  const numericAmount = Number(amount)
  if (!Number.isFinite(numericAmount)) return null
  const fractionDigits = amount.includes(".") ? amount.length - amount.indexOf(".") - 1 : 0

  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: normalizedCurrency,
      minimumFractionDigits: 0,
      maximumFractionDigits: fractionDigits,
    }).format(numericAmount)
  } catch {
    return `${normalizedCurrency} ${numericAmount.toLocaleString(undefined, { maximumFractionDigits: fractionDigits })}`
  }
}

export function formatCurrencyMinorAmount(value: unknown, currency = "USD") {
  const amount = formatMinorAmount(value)
  return amount === null ? null : formatCurrencyAmount(amount, currency)
}
