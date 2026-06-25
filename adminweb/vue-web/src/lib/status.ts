export type LabelOption = {
  value: string
  label: string
}

export const bizTypeOptions: LabelOption[] = [
  { value: "PIPELINE_PAYMENT", label: "管线订单" },
  { value: "STAGE_PAYMENT", label: "阶段订单" },
  { value: "COURSE_RETAKE_PAYMENT", label: "重考订单" },
  { value: "PIPELINE_UNLOCK", label: "管线解锁订单" },
  { value: "CREDENTIAL_APPLICATION", label: "资格申请订单" },
  { value: "BUNDLE_PURCHASE", label: "认证套餐订单" },
]

export const orderStatusOptions: LabelOption[] = [
  { value: "PENDING", label: "待处理" },
  { value: "WAIT_PAYMENT", label: "待支付" },
  { value: "WAIT_BUNDLE_PAYMENT", label: "等待套餐支付" },
  { value: "WAIT_PIPELINE_PAYMENT", label: "等待管线支付" },
  { value: "WAIT_PIPELINE_INSTANTIATE", label: "管线创建中" },
  { value: "WAIT_EXEMPTION_SELECTION", label: "等待选择免考" },
  { value: "WAIT_EXEMPTION_REVIEW", label: "等待免考审核" },
  { value: "WAIT_STAGE_PAYMENT", label: "等待阶段支付" },
  { value: "WAIT_REVIEW_FEE_PAYMENT", label: "等待审核费支付" },
  { value: "WAIT_RETAKE_PAYMENT", label: "等待重考支付" },
  { value: "WAIT_UNLOCK_PAYMENT", label: "等待解锁支付" },
  { value: "UPLOAD_READY", label: "可上传材料" },
  { value: "UNDER_REVIEW", label: "审核中" },
  { value: "RESOLVED", label: "已处理" },
  { value: "PAID", label: "已支付" },
  { value: "COMPLETED", label: "已完成" },
  { value: "CANCELLED", label: "已取消" },
  { value: "FAILED", label: "失败" },
  { value: "EXPIRED", label: "已过期" },
  { value: "PENDING_CREATE", label: "等待创建" },
  { value: "PENDING_PAYMENT", label: "等待支付" },
]

export const paymentStatusOptions: LabelOption[] = [
  { value: "WAIT_PAY", label: "待支付" },
  { value: "WAIT_PAYMENT", label: "待支付" },
  { value: "UNPAID", label: "待支付" },
  { value: "PAID", label: "已支付" },
  { value: "COMPLETED", label: "已支付" },
  { value: "FAILED", label: "支付失败" },
  { value: "REFUNDED", label: "已退款" },
  { value: "CANCELLED", label: "已取消" },
]

export const applicationStatusOptions: LabelOption[] = [
  { value: "0", label: "全部" },
  { value: "1", label: "待审核" },
  { value: "2", label: "已通过" },
  { value: "3", label: "已拒绝" },
  { value: "4", label: "需补交" },
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
  if (status.includes("APPROVED") || status === "2") return "已通过"
  if (status.includes("REJECTED") || status === "3") return "已拒绝"
  if (status.includes("RESUBMIT") || status.includes("REUPLOAD") || status === "4") return "需补交"
  if (status.includes("PENDING") || status === "1") return "待审核"
  return status || "-"
}

export function badgeClass(value: unknown) {
  const status = normalizeStatus(value)
  if (status.includes("COMPLETED") || status.includes("APPROVED") || status.includes("PAID") || status.includes("RESOLVED") || status === "2") {
    return "border-emerald-200 bg-emerald-50 text-emerald-700"
  }
  if (status.includes("WAIT") || status.includes("PENDING") || status.includes("REVIEW") || status.includes("READY") || status === "1" || status === "4") {
    return "border-amber-200 bg-amber-50 text-amber-700"
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
