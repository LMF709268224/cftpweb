import { ApiClientError, apiClient } from "@/lib/apiClient"

export type InProgressStageOrder = {
  stageOrderId: string
  orderStatus: "WAIT_EXEMPTION_SELECTION" | "WAIT_STAGE_PAYMENT"
}

const recoverableStageOrderStatuses = new Set<InProgressStageOrder["orderStatus"]>([
  "WAIT_EXEMPTION_SELECTION",
  "WAIT_STAGE_PAYMENT",
])

export function isInProgressStagePurchaseConflict(error: unknown) {
  if (!(error instanceof ApiClientError)) return false
  const errorCode = String(error.errorCode || "").trim().toUpperCase()
  return error.status === 409
    && errorCode === "PRECONDITION_FAILED"
}

export function reportsInProgressStagePurchase(error: unknown) {
  if (!(error instanceof ApiClientError)) return false
  return /candidate has an in-progress stage purchase/i.test(String(error.rawMessage || "").trim())
}

export async function findInProgressStageOrder(): Promise<InProgressStageOrder | null> {
  const params = new URLSearchParams({
    biz_type: "STAGE_PAYMENT",
    page_size: "50",
  })
  const response = await apiClient(`/api/orders?${params.toString()}`, {
    suppressErrorToast: true,
  })
  const orders = Array.isArray(response?.orders) ? response.orders : []

  for (const order of orders) {
    const bizType = String(order?.biz_type || "").trim().toUpperCase()
    const orderStatus = String(order?.order_status || "").trim().toUpperCase()
    const stageOrderId = String(order?.biz_ref_ulid || "").trim()
    if (bizType !== "STAGE_PAYMENT" || !stageOrderId || !recoverableStageOrderStatuses.has(orderStatus as InProgressStageOrder["orderStatus"])) {
      continue
    }
    return {
      stageOrderId,
      orderStatus: orderStatus as InProgressStageOrder["orderStatus"],
    }
  }

  return null
}
