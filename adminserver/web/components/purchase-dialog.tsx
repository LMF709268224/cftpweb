"use client"

import { useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { CreditCard, Building2 } from "lucide-react"
import { toast } from "sonner"
import { useTranslation } from "@/lib/useLanguage"

interface PurchaseDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  courseName: string
  price: number
  pipelineId: string
}

type PaymentMethod = "stripe" | "bank"

export function PurchaseDialog({
  open,
  onOpenChange,
  courseName,
  price,
  pipelineId,
}: PurchaseDialogProps) {
  const { t } = useTranslation()
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>("stripe")
  const [loading, setLoading] = useState(false)

  const handlePayment = async () => {
    setLoading(true)
    try {
      const data = await apiClient(`/api/mall/pipelines/${pipelineId}/purchase`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          payment_mode: "FULL_PIPELINE",
          candidate_selected_exemptions_json: "{}",
        }),
      })

      if (paymentMethod === "stripe" && data.payment_url) {
        window.location.href = data.payment_url
        return
      }

      const orderMessage = data.pipeline_order_ulid
        ? t.common.purchaseDialogOrderCreatedWithId.replace("{{id}}", data.pipeline_order_ulid)
        : t.common.purchaseDialogOrderCreatedBankHint

      toast.success(orderMessage)
      onOpenChange(false)
    } catch (error) {
      console.error(error)
      toast.error(t.common.error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px] p-0 gap-0 overflow-hidden">
        <DialogHeader className="px-6 pt-6 pb-4 border-b border-border">
          <DialogTitle className="text-xl font-semibold">
            {t.common.purchaseDialogTitle} {courseName}
          </DialogTitle>
        </DialogHeader>

        <div className="px-6 py-5 space-y-6">
          <div className="space-y-3">
            <div className="flex justify-between items-center py-2 border-b border-border/50">
              <span className="text-sm text-muted-foreground">{t.common.purchaseDialogCourse}</span>
              <span className="text-sm font-medium text-foreground">{courseName}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-border/50">
              <span className="text-sm text-muted-foreground">{t.common.purchaseDialogPrice}</span>
              <div className="text-right">
                <span className="text-lg font-bold text-foreground">${price}</span>
              </div>
            </div>
          </div>

          <div className="space-y-3">
            <label className="text-sm font-medium text-foreground">{t.common.purchaseDialogPaymentMethod}</label>
            <div className="space-y-2">
              <button
                type="button"
                onClick={() => setPaymentMethod("stripe")}
                className={cn(
                  "w-full flex items-center gap-3 p-3 rounded-xl border transition-all",
                  paymentMethod === "stripe"
                    ? "border-primary bg-primary/5"
                    : "border-border hover:border-primary/50"
                )}
              >
                <div
                  className={cn(
                    "h-5 w-5 rounded-full border-2 flex items-center justify-center transition-colors",
                    paymentMethod === "stripe" ? "border-primary" : "border-muted-foreground/30"
                  )}
                >
                  {paymentMethod === "stripe" && <div className="h-2.5 w-2.5 rounded-full bg-primary" />}
                </div>
                <CreditCard className="h-4 w-4 text-primary" />
                <span className="text-sm font-medium text-foreground">
                  {t.common.purchaseDialogStripeMethod}
                </span>
                <Badge className="ml-auto bg-amber-500/10 text-amber-700 border-0 text-xs">
                  {t.common.purchaseDialogStripeBadge}
                </Badge>
              </button>

              <button
                type="button"
                onClick={() => setPaymentMethod("bank")}
                className={cn(
                  "w-full flex items-center gap-3 p-3 rounded-xl border transition-all",
                  paymentMethod === "bank"
                    ? "border-primary bg-primary/5"
                    : "border-border hover:border-primary/50"
                )}
              >
                <div
                  className={cn(
                    "h-5 w-5 rounded-full border-2 flex items-center justify-center transition-colors",
                    paymentMethod === "bank" ? "border-primary" : "border-muted-foreground/30"
                  )}
                >
                  {paymentMethod === "bank" && <div className="h-2.5 w-2.5 rounded-full bg-primary" />}
                </div>
                <Building2 className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium text-foreground">
                  {t.common.purchaseDialogBankMethod}
                </span>
              </button>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-border bg-muted/30">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t.common.cancel}
          </Button>
          <Button onClick={handlePayment} className="gap-2" disabled={loading}>
            <CreditCard className="h-4 w-4" />
            {loading
              ? t.common.loading
              : paymentMethod === "stripe"
                ? t.common.purchaseDialogStripeSubmit
                : t.common.purchaseDialogBankSubmit}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
