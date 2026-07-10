export type LabelOption = {
  value: string
  label: string
}

export const bizTypeOptions: LabelOption[] = [
  { value: "PIPELINE_PAYMENT", label: "PIPELINE_PAYMENT" },
  { value: "STAGE_PAYMENT", label: "STAGE_PAYMENT" },
  { value: "COURSE_RETAKE_PAYMENT", label: "COURSE_RETAKE_PAYMENT" },
  { value: "PIPELINE_UNLOCK", label: "PIPELINE_UNLOCK" },
  { value: "CREDENTIAL_APPLICATION", label: "CREDENTIAL_APPLICATION" },
  { value: "BUNDLE_PURCHASE", label: "BUNDLE_PURCHASE" },
]

export const orderStatusOptions: LabelOption[] = [
  { value: "WAIT_PAYMENT", label: "WAIT_PAYMENT" },
  { value: "PENDING", label: "PENDING" },
  { value: "COMPLETED", label: "COMPLETED" },
  { value: "CANCELLED", label: "CANCELLED" },
  { value: "CLOSED", label: "CLOSED" },
]

export const paymentStatusOptions: LabelOption[] = [
  { value: "WAIT_PAY", label: "WAIT_PAY" },
  { value: "WAIT_PAYMENT", label: "WAIT_PAYMENT" },
  { value: "UNPAID", label: "UNPAID" },
  { value: "PAID", label: "PAID" },
  { value: "COMPLETED", label: "COMPLETED" },
  { value: "FAILED", label: "FAILED" },
  { value: "REFUNDED", label: "REFUNDED" },
  { value: "REFUND_OFFLINE", label: "REFUND_OFFLINE" },
  { value: "CANCELLED", label: "CANCELLED" },
]

export const applicationStatusOptions: LabelOption[] = [
  { value: "0", label: "ALL" },
  { value: "1", label: "PENDING" },
  { value: "2", label: "APPROVED" },
  { value: "3", label: "REJECTED" },
  { value: "4", label: "RESUBMIT_REQUIRED" },
]

export function normalizeStatus(value: unknown) {
  return String(value || "").trim().toUpperCase()
}

export function labelFor(options: LabelOption[], value: unknown) {
  const normalized = normalizeStatus(value)
  return options.find((item) => item.value === normalized)?.label || normalized || "-"
}

export function applicationStatusLabel(value: unknown) {
  const status = normalizeStatus(value)
  if (status.includes("APPROVED") || status === "2") return "APPROVED"
  if (status.includes("REJECTED") || status === "3") return "REJECTED"
  if (status.includes("RESUBMIT") || status.includes("REUPLOAD") || status === "4") return "RESUBMIT_REQUIRED"
  if (status.includes("PENDING") || status === "1") return "PENDING"
  return status || "-"
}

export function badgeClass(value: unknown) {
  const status = normalizeStatus(value)
  if (status === "UNPAID" || status === "WAIT_PAY" || status === "WAIT_PAYMENT" || status.includes("WAIT") || status.includes("PENDING") || status.includes("REVIEW") || status.includes("READY") || status === "1" || status === "4") {
    return "border-amber-200 bg-amber-50 text-amber-700"
  }
  if (status.includes("COMPLETED") || status.includes("APPROVED") || status === "PAID" || status.includes("RESOLVED") || status === "2") {
    return "border-emerald-200 bg-emerald-50 text-emerald-700"
  }
  if (status.includes("FAILED") || status.includes("REJECTED") || status.includes("CANCEL") || status.includes("EXPIRED") || status === "3") {
    return "border-red-200 bg-red-50 text-red-700"
  }
  return "border-slate-200 bg-slate-50 text-slate-600"
}

export function pickFirst(record: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = record[key]
    if (value !== undefined && value !== null && value !== "") {
      return value
    }
  }
  return undefined
}
