import { formatCurrencyMinorAmount, formatMinorAmount } from "./display"

type BundleDisplayPrice = {
  display_amount_min?: number
  display_amount_max?: number
  display_currency?: string
}

export function bundleDisplayPriceLabel(bundle: BundleDisplayPrice): string {
  const min = bundle.display_amount_min
  const max = bundle.display_amount_max
  const currency = typeof bundle.display_currency === "string" ? bundle.display_currency.trim().toUpperCase() : ""
  if (typeof min !== "number" || typeof max !== "number" ||
    !Number.isSafeInteger(min) || !Number.isSafeInteger(max) || min < 0 || max < min ||
    !currency || !/^[A-Z]{3}$/.test(currency)) return ""

  // USD must remain identifiable regardless of the browser's regional settings.
  const format = (amount: number) => currency === "USD"
    ? `US$${formatMinorAmount(amount, { useGrouping: true })}`
    : formatCurrencyMinorAmount(amount, currency)
  const minimum = format(min)
  const maximum = format(max)
  if (!minimum || !maximum) return ""
  return max > min ? `${minimum} - ${maximum}` : minimum
}
