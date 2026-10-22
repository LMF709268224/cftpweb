import { formatCurrencyMinorAmount } from "./display"

type BundleDisplayPrice = {
  display_amount_min?: number
  display_amount_max?: number
  display_currency?: string
}

export function bundleDisplayPriceLabel(bundle: BundleDisplayPrice, fromLabel: string): string {
  const min = Number(bundle.display_amount_min || 0)
  const max = Number(bundle.display_amount_max || 0)
  if (min <= 0 && max <= 0) return ""

  const amount = max > min ? min : min || max
  const price = formatCurrencyMinorAmount(amount, bundle.display_currency || "USD") || ""
  return max > min && price ? `${fromLabel} ${price}` : price
}
