"use client"

import { useEffect } from "react"
import { toast } from "sonner"

import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"

const pendingMallPaymentKey = "candidate_pending_mall_payment"

type PendingMallPayment = {
  action?: string
  orderId?: string
  pipelineId?: string
  startedAt: number
}

type EligibilityBlocker = {
  blocker_type?: string
}

export const rememberPendingMallPayment = (payment: Omit<PendingMallPayment, "startedAt">) => {
  if (typeof window === "undefined") return
  localStorage.setItem(pendingMallPaymentKey, JSON.stringify({ ...payment, startedAt: Date.now() }))
}

export const clearPendingMallPayment = () => {
  if (typeof window === "undefined") return
  localStorage.removeItem(pendingMallPaymentKey)
}

export function PaymentReturnHandler() {
  const { t, lang } = useTranslation()

  useEffect(() => {
    if (typeof window === "undefined") return

    const url = new URL(window.location.href)
    const paymentStatus = url.searchParams.get("payment_status")

    // The courses page has extra refresh/tab handling, so leave that route to its page-level handler.
    if (url.pathname === "/courses") return

    const copy = t.paymentReturnHandler

    if (paymentStatus) {
      const paymentAction = url.searchParams.get("payment_action")
      if (paymentStatus === "success") {
        toast.success(paymentAction === "unlock" ? copy.unlockSuccess : copy.purchaseSuccess)
      } else if (paymentStatus === "cancelled") {
        toast.warning(copy.cancelled)
      } else if (paymentStatus === "failed") {
        toast.error(copy.failed)
      }

      clearPendingMallPayment()
      url.searchParams.delete("payment_status")
      url.searchParams.delete("payment_action")
      url.searchParams.delete("order_id")
      window.history.replaceState({}, "", `${url.pathname}${url.search}${url.hash}`)
      return
    }

    const pendingRaw = localStorage.getItem(pendingMallPaymentKey)
    if (url.pathname === "/" && pendingRaw) {
      const checkPendingStatus = async () => {
        try {
          const pending = JSON.parse(pendingRaw) as PendingMallPayment
          if (!pending.pipelineId) {
            toast.warning(copy.unknownDesc)
            return
          }

          const eligibility = await apiClient(`/api/mall/pipelines/${pending.pipelineId}/eligibility`)
          const blockers = Array.isArray(eligibility?.blockers) ? eligibility.blockers : []
          if (blockers.some((blocker: EligibilityBlocker) => blocker?.blocker_type === "ALREADY_PURCHASED")) {
            toast.success(pending.action === "unlock" ? copy.unlockSuccess : copy.purchaseSuccess)
          } else if (blockers.some((blocker: EligibilityBlocker) => blocker?.blocker_type === "IN_PROGRESS_PURCHASE")) {
            toast.warning(copy.inProgressDesc)
          } else {
            toast.warning(copy.unknownDesc)
          }
        } catch {
          toast.warning(copy.unknownDesc)
        } finally {
          clearPendingMallPayment()
        }
      }

      void checkPendingStatus()
    }
  }, [lang])

  return null
}
