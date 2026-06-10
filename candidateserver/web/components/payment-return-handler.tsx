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
  const { lang } = useTranslation()

  useEffect(() => {
    if (typeof window === "undefined") return

    const url = new URL(window.location.href)
    const paymentStatus = url.searchParams.get("payment_status")

    // The courses page has extra refresh/tab handling, so leave that route to its page-level handler.
    if (url.pathname === "/courses") return

    const copy = {
      purchaseSuccess: lang === "zh" ? "购买成功，课程列表已刷新。" : "Purchase successful. The course list has been refreshed.",
      unlockSuccess: lang === "zh" ? "解锁成功，课程列表已刷新。" : "Unlock successful. The course list has been refreshed.",
      cancelled: lang === "zh" ? "支付已取消，你可以稍后继续处理订单。" : "Payment cancelled. You can continue the order later.",
      failed: lang === "zh" ? "支付失败，请稍后重试或联系管理员。" : "Payment failed. Please try again later or contact support.",
      stillPending: lang === "zh"
        ? "支付尚未完成，订单仍在处理中。请回到认证中心继续支付或重新检查状态。"
        : "Payment is not complete yet. Go back to Certifications to continue payment or recheck the order.",
      unknownReturn: lang === "zh"
        ? "支付流程已返回，但没有收到支付结果。请回到认证中心重新检查订单状态。"
        : "The payment flow returned without a result. Go back to Certifications and recheck the order status.",
    }

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
            toast.warning(copy.unknownReturn)
            return
          }

          const eligibility = await apiClient(`/api/mall/pipelines/${pending.pipelineId}/eligibility`)
          const blockers = Array.isArray(eligibility?.blockers) ? eligibility.blockers : []
          if (blockers.some((blocker: EligibilityBlocker) => blocker?.blocker_type === "ALREADY_PURCHASED")) {
            toast.success(pending.action === "unlock" ? copy.unlockSuccess : copy.purchaseSuccess)
          } else if (blockers.some((blocker: EligibilityBlocker) => blocker?.blocker_type === "IN_PROGRESS_PURCHASE")) {
            toast.warning(copy.stillPending)
          } else {
            toast.warning(copy.unknownReturn)
          }
        } catch {
          toast.warning(copy.unknownReturn)
        } finally {
          clearPendingMallPayment()
        }
      }

      void checkPendingStatus()
    }
  }, [lang])

  return null
}
