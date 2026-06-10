"use client"

import { useEffect, useState } from "react"
import { AlertCircle, Building2, CheckCircle2, CreditCard, Loader2, Lock, ShoppingCart } from "lucide-react"
import { toast } from "sonner"

import { apiClient } from "@/lib/apiClient"
import { cn } from "@/lib/utils"
import { rememberPendingMallPayment } from "@/components/payment-return-handler"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { StripeEmbeddedCheckout } from "@/components/stripe-embedded-checkout"
import { useTranslation } from "@/lib/useLanguage"

interface PurchaseDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  courseName: string
  pipelineId: string
}

type PaymentMethod = "stripe" | "bank"
type MallAction = "purchase" | "unlock"

type EligibilityBlocker = {
  blocker_type?: string
  description?: string
  details?: unknown[]
}

type EligibilityPreview = {
  eligible?: boolean
  can_purchase?: boolean
  can_unlock?: boolean
  blockers?: EligibilityBlocker[]
}

type PaymentPreview = {
  subtotal?: number
  discount_total?: number
  tax_total?: number
  total?: number
  currency?: string
}

type ActiveOrder = {
  action: MallAction
  orderId: string
  status?: string
  payOrderId?: string
  message?: string
}

const selectedExemptionsJson = JSON.stringify({ stages: [] })

const stripeCheckoutUrl = (paymentKey: unknown) => {
  if (typeof paymentKey !== "string") return ""
  const value = paymentKey.trim()
  if (!value) return ""
  if (/^https:\/\/checkout\.stripe\.com\//i.test(value)) return value
  if (value.startsWith("/c/pay/")) return `https://checkout.stripe.com${value}`
  return ""
}

const stripeEmbeddedClientSecret = (paymentKey: unknown) => {
  if (typeof paymentKey !== "string") return ""
  const value = paymentKey.trim()
  return value.startsWith("cs_") ? value : ""
}

const formatMoney = (amount?: number, currency = "usd") => {
  if (typeof amount !== "number") return "-"
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "usd",
  }).format(amount / 100)
}

const detailText = (detail: unknown) => {
  if (typeof detail === "string") return detail
  if (detail && typeof detail === "object") {
    const record = detail as Record<string, unknown>
    return String(record.name || record.title || record.label || record.description || "")
  }
  return String(detail || "")
}

const normalizedStatus = (status: unknown) => String(status || "").trim().toUpperCase()

const isCompletedStatus = (status: unknown) => normalizedStatus(status).includes("COMPLETED")

const isFailedStatus = (status: unknown) => {
  const value = normalizedStatus(status)
  return value.includes("FAILED") || value.includes("CANCEL") || value.includes("REJECT")
}

export function PurchaseDialog({
  open,
  onOpenChange,
  courseName,
  pipelineId,
}: PurchaseDialogProps) {
  const { t, lang } = useTranslation()
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>("stripe")
  const [eligibilityLoading, setEligibilityLoading] = useState(false)
  const [actionLoading, setActionLoading] = useState(false)
  const [paymentLoading, setPaymentLoading] = useState(false)
  const [eligibility, setEligibility] = useState<EligibilityPreview | null>(null)
  const [activeOrder, setActiveOrder] = useState<ActiveOrder | null>(null)
  const [paymentPreview, setPaymentPreview] = useState<PaymentPreview | null>(null)
  const [previewError, setPreviewError] = useState("")
  const [embeddedClientSecret, setEmbeddedClientSecret] = useState("")

  const blockers = eligibility?.blockers || []
  const canPurchase = Boolean(eligibility?.can_purchase)
  const canUnlock = Boolean(eligibility?.can_unlock)
  const cannotContinue = Boolean(eligibility && !canPurchase && !canUnlock)
  const hasInProgressOrder = blockers.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE")

  const copy = {
    title: lang === "zh" ? "认证购买状态" : "Certification purchase status",
    checking: lang === "zh" ? "正在检查你是否可以购买或解锁..." : "Checking whether you can buy or unlock this pipeline...",
    canPurchaseTitle: lang === "zh" ? "可以购买认证" : "Ready to purchase",
    canPurchaseDesc: lang === "zh" ? "你已满足购买条件，可以创建购买订单并查看价格。" : "You meet the requirements. Create an order to preview the price.",
    canUnlockTitle: lang === "zh" ? "需要先解锁认证" : "Unlock required first",
    canUnlockDesc: lang === "zh" ? "这个认证需要先解锁。解锁完成后，系统会重新检查，然后才可以购买认证。" : "This certification must be unlocked before purchase. After unlocking, we will check again before purchase.",
    blockedTitle: lang === "zh" ? "暂时不能购买或解锁" : "Action unavailable",
    blockedDesc: lang === "zh" ? "请先处理下面的阻塞原因。" : "Resolve the blockers below first.",
    blockersTitle: lang === "zh" ? "阻塞原因" : "Blockers",
    requiredItems: lang === "zh" ? "需要完成" : "Required",
    missingQualification: lang === "zh" ? "缺少解锁资格" : "Missing unlock qualification",
    alreadyPurchased: lang === "zh" ? "你已经购买过该认证" : "You have already purchased this certification",
    inProgressPurchase: lang === "zh" ? "已有未完成的购买订单" : "Purchase already in progress",
    inProgressPurchaseDesc: lang === "zh" ? "你已有未完成订单，可以继续查看价格并完成支付。" : "You have an unfinished order. Continue to review the price and complete payment.",
    pipelineNotFound: lang === "zh" ? "该认证已不可用" : "This certification is no longer available",
    unknownBlocker: lang === "zh" ? "暂时不能继续" : "Unable to continue",
    createPurchaseOrder: lang === "zh" ? "创建购买订单" : "Create purchase order",
    createUnlockOrder: lang === "zh" ? "创建解锁订单" : "Create unlock order",
    refreshEligibility: lang === "zh" ? "重新检查状态" : "Recheck status",
    pricePreviewTitle: lang === "zh" ? "价格预览" : "Price preview",
    pricePreviewFailed: lang === "zh" ? "暂时无法获取价格预览，请稍后重试。价格未确认前不能发起支付。" : "Price preview is temporarily unavailable. Payment cannot be started until the price is confirmed.",
    retryPreview: lang === "zh" ? "重新获取价格" : "Retry price preview",
    orderCreated: lang === "zh" ? "订单已创建" : "Order created",
    activeOrder: lang === "zh" ? "未完成订单" : "Unfinished order",
    unlockCompleted: lang === "zh" ? "解锁已完成，请重新检查购买状态。" : "Unlock completed. Recheck purchase status.",
    subtotal: lang === "zh" ? "原价" : "Subtotal",
    discount: lang === "zh" ? "优惠" : "Discount",
    tax: lang === "zh" ? "税费" : "Tax",
    total: lang === "zh" ? "应付合计" : "Total due",
    stripe: lang === "zh" ? "Stripe 在线支付" : "Stripe online payment",
    bank: lang === "zh" ? "银行转账" : "Bank transfer",
    payNow: lang === "zh" ? "去支付" : "Pay now",
    embeddedCheckoutTitle: lang === "zh" ? "请在下方完成支付" : "Complete payment below",
    embeddedCheckoutDesc: lang === "zh" ? "支付会话已创建。支付完成后，Stripe 会返回认证中心并刷新订单状态。" : "The payment session is ready. After payment, Stripe will return you to Certifications and refresh the order status.",
    embeddedCheckoutLoading: lang === "zh" ? "正在加载 Stripe 支付组件..." : "Loading Stripe checkout...",
    stripePublishableKeyMissing: lang === "zh" ? "缺少 Stripe Publishable Key，请配置 STRIPE_PUBLISHABLE_KEY。" : "Missing Stripe publishable key. Configure STRIPE_PUBLISHABLE_KEY.",
    stripeEmbeddedFailed: lang === "zh" ? "支付组件加载失败，请刷新后重试。" : "Failed to load the payment component. Please refresh and try again.",
    paymentSessionFailed: lang === "zh" ? "支付会话创建失败，请稍后重试。" : "Payment session could not be created. Please try again later.",
    unsupportedPaymentKey: lang === "zh" ? "暂不支持的支付凭证类型，请稍后重试。" : "Unsupported payment credential type. Please try again later.",
    purchaseCompleted: lang === "zh" ? "购买成功，认证已开通。" : "Purchase successful. The certification is now active.",
    purchaseFailed: lang === "zh" ? "购买失败，请稍后重试或联系管理员。" : "Purchase failed. Please try again later or contact support.",
    unlockFailed: lang === "zh" ? "解锁失败，请稍后重试或联系管理员。" : "Unlock failed. Please try again later or contact support.",
  }

  const blockerTitle = (blocker: EligibilityBlocker) => {
    if (blocker.blocker_type === "MISSING_UNLOCK_QUALIFICATION") return copy.missingQualification
    if (blocker.blocker_type === "ALREADY_PURCHASED") return copy.alreadyPurchased
    if (blocker.blocker_type === "IN_PROGRESS_PURCHASE") return copy.inProgressPurchase
    if (blocker.blocker_type === "PIPELINE_NOT_FOUND") return copy.pipelineNotFound
    return blocker.description || blocker.blocker_type || copy.unknownBlocker
  }

  const loadEligibility = async () => {
    setEligibilityLoading(true)
    setActiveOrder(null)
    setPaymentPreview(null)
    setPreviewError("")
    setEmbeddedClientSecret("")
    try {
      const res: EligibilityPreview = await apiClient(`/api/mall/pipelines/${pipelineId}/eligibility`)
      setEligibility(res)
      if (res.blockers?.some((blocker) => blocker.blocker_type === "IN_PROGRESS_PURCHASE")) {
        await loadActiveOrder()
      }
    } finally {
      setEligibilityLoading(false)
    }
  }

  useEffect(() => {
    if (!open || !pipelineId) return
    void loadEligibility()
  }, [open, pipelineId])

  const previewPayment = async (action: MallAction, orderId: string) => {
    setPreviewError("")
    const bizType = action === "unlock" ? "PIPELINE_UNLOCK" : "PIPELINE_PAYMENT"
    try {
      const res = await apiClient("/api/mall/payments/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          biz_type: bizType,
          biz_ref_ulid: orderId,
          coupon_codes: [],
        }),
      })
      setPaymentPreview(res)
    } catch {
      setPaymentPreview(null)
      setPreviewError(copy.pricePreviewFailed)
    }
  }

  const orderIdFromDetail = (order: any) => {
    return order?.pipeline_order_ulid || order?.summary?.pipeline_order_ulid || ""
  }

  const orderStatusFromDetail = (order: any) => {
    return order?.order_status || order?.summary?.order_status || ""
  }

  const loadActiveOrder = async () => {
    setPreviewError("")
    setPaymentPreview(null)
    try {
      const order = await apiClient(`/api/mall/pipelines/${pipelineId}/active-order`)
      const orderId = orderIdFromDetail(order)
      if (!orderId) return
      setActiveOrder({
        action: "purchase",
        orderId,
        status: orderStatusFromDetail(order),
        payOrderId: order.pipeline_pay_order_ulid,
        message: copy.inProgressPurchaseDesc,
      })
      await previewPayment("purchase", orderId)
    } catch (error) {
      console.error(error)
    }
  }

  const createPurchaseOrder = async () => {
    setActionLoading(true)
    try {
      const latest: EligibilityPreview = await apiClient(`/api/mall/pipelines/${pipelineId}/eligibility`)
      setEligibility(latest)
      if (!latest.can_purchase) return

      const order = await apiClient(`/api/mall/pipelines/${pipelineId}/purchase`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          payment_mode: "FULL_PIPELINE",
          candidate_selected_exemptions_json: selectedExemptionsJson,
        }),
      })
      const orderId = order.pipeline_order_ulid
      const orderStatus = order.order_status
      setActiveOrder({
        action: "purchase",
        orderId,
        status: orderStatus,
        payOrderId: order.pipeline_pay_order_ulid,
        message: order.message,
      })
      if (isCompletedStatus(orderStatus)) {
        toast.success(copy.purchaseCompleted)
        onOpenChange(false)
        window.setTimeout(() => window.location.reload(), 800)
        return
      }
      if (isFailedStatus(orderStatus)) {
        toast.error(copy.purchaseFailed)
        return
      }
      if (orderId) await previewPayment("purchase", orderId)
    } catch (error) {
      console.error(error)
    } finally {
      setActionLoading(false)
    }
  }

  const createUnlockOrder = async () => {
    setActionLoading(true)
    try {
      const latest: EligibilityPreview = await apiClient(`/api/mall/pipelines/${pipelineId}/eligibility`)
      setEligibility(latest)
      if (!latest.can_unlock) return

      const order = await apiClient(`/api/mall/pipelines/${pipelineId}/unlock`, { method: "POST" })
      const orderId = order.pipeline_unlock_order_ulid
      const paymentKey = order.payment_key
      const orderStatus = order.order_status
      setActiveOrder({
        action: "unlock",
        orderId,
        status: orderStatus,
        payOrderId: order.pay_order_ulid,
        message: order.message,
      })

      if (isCompletedStatus(orderStatus)) {
        toast.success(copy.unlockCompleted)
        await loadEligibility()
        return
      }
      if (isFailedStatus(orderStatus)) {
        toast.error(copy.unlockFailed)
        return
      }
      if (orderId && (paymentKey || order.pay_order_ulid || normalizedStatus(orderStatus).includes("PAYMENT"))) {
        await previewPayment("unlock", orderId)
      } else {
        toast.info(copy.refreshEligibility)
      }
    } catch (error) {
      console.error(error)
    } finally {
      setActionLoading(false)
    }
  }

  const initiatePayment = async () => {
    if (!activeOrder?.orderId) return
    const bizType = activeOrder.action === "unlock" ? "PIPELINE_UNLOCK" : "PIPELINE_PAYMENT"
    setPaymentLoading(true)
    try {
      setEmbeddedClientSecret("")
      const origin = window.location.origin
      const successParams = new URLSearchParams({
        payment_status: "success",
        payment_action: activeOrder.action,
        order_id: activeOrder.orderId,
        pipeline_id: pipelineId,
      })
      const cancelParams = new URLSearchParams({
        payment_status: "cancelled",
        payment_action: activeOrder.action,
        order_id: activeOrder.orderId,
        pipeline_id: pipelineId,
      })
      const res = await apiClient("/api/mall/payments/initiate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          biz_type: bizType,
          biz_ref_ulid: activeOrder.orderId,
          success_url: `${origin}/courses?${successParams.toString()}`,
          cancel_url: `${origin}/courses?${cancelParams.toString()}`,
          coupon_codes: [],
        }),
      })
      const paymentKey = res.payment_key
      if (!paymentKey) {
        toast.error(copy.paymentSessionFailed)
        return
      }
      const checkoutUrl = stripeCheckoutUrl(paymentKey)
      if (paymentMethod === "stripe" && checkoutUrl) {
        rememberPendingMallPayment({
          action: activeOrder.action,
          orderId: activeOrder.orderId,
          pipelineId,
        })
        window.location.href = checkoutUrl
        return
      }
      const clientSecret = stripeEmbeddedClientSecret(paymentKey)
      if (paymentMethod === "stripe" && clientSecret) {
        rememberPendingMallPayment({
          action: activeOrder.action,
          orderId: activeOrder.orderId,
          pipelineId,
        })
        setEmbeddedClientSecret(clientSecret)
        return
      }
      toast.error(copy.unsupportedPaymentKey)
    } catch (error) {
      console.error(error)
    } finally {
      setPaymentLoading(false)
    }
  }

  const statusPanel = () => {
    if (eligibilityLoading && !eligibility) {
      return (
        <div className="rounded-xl border border-border bg-muted/30 p-4">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            {copy.checking}
          </div>
        </div>
      )
    }

    if (canPurchase) {
      return (
        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-4">
          <div className="flex items-center gap-2 font-semibold text-emerald-900">
            <CheckCircle2 className="h-4 w-4" />
            {copy.canPurchaseTitle}
          </div>
          <p className="mt-2 text-sm text-emerald-800">{copy.canPurchaseDesc}</p>
        </div>
      )
    }

    if (canUnlock) {
      return (
        <div className="rounded-xl border border-blue-200 bg-blue-50 p-4">
          <div className="flex items-center gap-2 font-semibold text-blue-900">
            <Lock className="h-4 w-4" />
            {copy.canUnlockTitle}
          </div>
          <p className="mt-2 text-sm text-blue-800">{copy.canUnlockDesc}</p>
        </div>
      )
    }

    if (cannotContinue) {
      if (hasInProgressOrder) {
        return (
          <div className="rounded-xl border border-blue-200 bg-blue-50 p-4">
            <div className="flex items-center gap-2 font-semibold text-blue-900">
              <CreditCard className="h-4 w-4" />
              {copy.inProgressPurchase}
            </div>
            <p className="mt-2 text-sm text-blue-800">{copy.inProgressPurchaseDesc}</p>
          </div>
        )
      }
      return (
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-4">
          <div className="flex items-center gap-2 font-semibold text-amber-900">
            <AlertCircle className="h-4 w-4" />
            {copy.blockedTitle}
          </div>
          <p className="mt-2 text-sm text-amber-800">{copy.blockedDesc}</p>
        </div>
      )
    }

    return null
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[620px] p-0 gap-0 overflow-hidden">
        <DialogHeader className="px-6 pt-6 pb-4 border-b border-border">
          <DialogTitle className="text-xl font-semibold">
            {copy.title}: {courseName}
          </DialogTitle>
        </DialogHeader>

        <div className="max-h-[72vh] overflow-y-auto px-6 py-5 space-y-5">
          <div className="flex justify-between items-center py-2 border-b border-border/50">
            <span className="text-sm text-muted-foreground">{t.common.purchaseDialogCourse}</span>
            <span className="text-sm font-medium text-foreground">{courseName}</span>
          </div>

          {statusPanel()}

          {blockers.length > 0 && (
            <div className="rounded-xl border border-amber-200 bg-amber-50/70 p-4">
              <div className="mb-3 text-sm font-semibold text-amber-950">{copy.blockersTitle}</div>
              <ul className="space-y-2">
                {blockers.map((blocker, index) => {
                  const details = Array.isArray(blocker.details) ? blocker.details.map(detailText).filter(Boolean) : []
                  return (
                    <li key={`${blocker.blocker_type || "blocker"}-${index}`} className="rounded-lg border border-amber-200 bg-white/80 p-3">
                      <div className="font-medium text-amber-950">{blockerTitle(blocker)}</div>
                      {details.length > 0 && (
                        <div className="mt-2">
                          <div className="mb-1 text-xs font-medium text-muted-foreground">{copy.requiredItems}</div>
                          <ul className="space-y-1">
                            {details.map((detail, detailIndex) => (
                              <li key={`${detail}-${detailIndex}`} className="flex items-center gap-2 rounded-md bg-amber-100/70 px-2 py-1.5 text-sm font-medium text-amber-950">
                                <AlertCircle className="h-3.5 w-3.5 shrink-0 text-amber-600" />
                                <span>{detail}</span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </li>
                  )
                })}
              </ul>
            </div>
          )}

          {activeOrder && (
            <div className="rounded-xl border border-border bg-muted/30 p-4">
              <div className="mb-2 flex items-center justify-between gap-3">
                <div className="text-sm font-semibold text-foreground">{activeOrder.message === copy.inProgressPurchaseDesc ? copy.activeOrder : copy.orderCreated}</div>
                {activeOrder.status && <Badge variant="outline">{activeOrder.status}</Badge>}
              </div>
              <div className="text-xs text-muted-foreground break-all">{activeOrder.orderId}</div>
              {activeOrder.message && <p className="mt-2 text-sm text-muted-foreground">{activeOrder.message}</p>}
            </div>
          )}

          {paymentPreview && (
            <div className="rounded-xl border border-border bg-muted/30 p-4">
              <div className="mb-3 text-sm font-semibold text-foreground">{copy.pricePreviewTitle}</div>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{copy.subtotal}</span>
                  <span className="font-medium">{formatMoney(paymentPreview.subtotal, paymentPreview.currency)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{copy.discount}</span>
                  <span className="font-medium">-{formatMoney(paymentPreview.discount_total || 0, paymentPreview.currency)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{copy.tax}</span>
                  <span className="font-medium">{formatMoney(paymentPreview.tax_total || 0, paymentPreview.currency)}</span>
                </div>
                <div className="mt-2 flex justify-between border-t border-border pt-2">
                  <span className="font-semibold text-foreground">{copy.total}</span>
                  <span className="text-lg font-bold text-foreground">{formatMoney(paymentPreview.total, paymentPreview.currency)}</span>
                </div>
              </div>
            </div>
          )}

          {activeOrder && previewError && (
            <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
              <div className="flex items-center gap-2 font-semibold">
                <AlertCircle className="h-4 w-4" />
                {copy.pricePreviewTitle}
              </div>
              <p className="mt-2">{previewError}</p>
            </div>
          )}

          {embeddedClientSecret && (
            <div className="space-y-3">
              <div className="rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
                <div className="flex items-center gap-2 font-semibold">
                  <CreditCard className="h-4 w-4" />
                  {copy.embeddedCheckoutTitle}
                </div>
                <p className="mt-2">{copy.embeddedCheckoutDesc}</p>
              </div>
              <StripeEmbeddedCheckout
                clientSecret={embeddedClientSecret}
                loadingText={copy.embeddedCheckoutLoading}
                missingKeyText={copy.stripePublishableKeyMissing}
                failedText={copy.stripeEmbeddedFailed}
              />
            </div>
          )}

          {activeOrder && paymentPreview && !embeddedClientSecret && (
            <div className="space-y-3">
              <label className="text-sm font-medium text-foreground">{t.common.purchaseDialogPaymentMethod}</label>
              <div className="space-y-2">
                <button
                  type="button"
                  onClick={() => setPaymentMethod("stripe")}
                  className={cn(
                    "w-full flex items-center gap-3 p-3 rounded-xl border transition-all",
                    paymentMethod === "stripe" ? "border-primary bg-primary/5" : "border-border hover:border-primary/50"
                  )}
                >
                  <div className={cn("h-5 w-5 rounded-full border-2 flex items-center justify-center transition-colors", paymentMethod === "stripe" ? "border-primary" : "border-muted-foreground/30")}>
                    {paymentMethod === "stripe" && <div className="h-2.5 w-2.5 rounded-full bg-primary" />}
                  </div>
                  <CreditCard className="h-4 w-4 text-primary" />
                  <span className="text-sm font-medium text-foreground">{copy.stripe}</span>
                  <Badge className="ml-auto bg-amber-500/10 text-amber-700 border-0 text-xs">
                    {t.common.purchaseDialogStripeBadge}
                  </Badge>
                </button>

                <button
                  type="button"
                  onClick={() => setPaymentMethod("bank")}
                  className={cn(
                    "w-full flex items-center gap-3 p-3 rounded-xl border transition-all",
                    paymentMethod === "bank" ? "border-primary bg-primary/5" : "border-border hover:border-primary/50"
                  )}
                >
                  <div className={cn("h-5 w-5 rounded-full border-2 flex items-center justify-center transition-colors", paymentMethod === "bank" ? "border-primary" : "border-muted-foreground/30")}>
                    {paymentMethod === "bank" && <div className="h-2.5 w-2.5 rounded-full bg-primary" />}
                  </div>
                  <Building2 className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium text-foreground">{copy.bank}</span>
                </button>
              </div>
            </div>
          )}
        </div>

        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-border bg-muted/30">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t.common.cancel}
          </Button>
          {cannotContinue && (
            <Button variant="outline" onClick={loadEligibility} disabled={eligibilityLoading}>
              {eligibilityLoading && <Loader2 className="h-4 w-4 animate-spin" />}
              {copy.refreshEligibility}
            </Button>
          )}
          {canUnlock && !activeOrder && (
            <Button onClick={createUnlockOrder} className="gap-2" disabled={actionLoading}>
              {actionLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Lock className="h-4 w-4" />}
              {copy.createUnlockOrder}
            </Button>
          )}
          {canPurchase && !activeOrder && (
            <Button onClick={createPurchaseOrder} className="gap-2" disabled={actionLoading}>
              {actionLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <ShoppingCart className="h-4 w-4" />}
              {copy.createPurchaseOrder}
            </Button>
          )}
          {activeOrder && previewError && (
            <Button variant="outline" onClick={() => previewPayment(activeOrder.action, activeOrder.orderId)} disabled={actionLoading}>
              {copy.retryPreview}
            </Button>
          )}
          {activeOrder && paymentPreview && !embeddedClientSecret && (
            <Button onClick={initiatePayment} className="gap-2" disabled={paymentLoading}>
              {paymentLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <CreditCard className="h-4 w-4" />}
              {copy.payNow}
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
