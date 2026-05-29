"use client"

import { useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { CreditCard, Building2 } from "lucide-react"

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
      } else {
        alert("订单已创建，请按指引完成银行转账\n订单号：" + data.pipeline_order_ulid)
        onOpenChange(false)
      }
    } catch (error) {
      console.error(error)
      alert("购买失败: " + error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px] p-0 gap-0 overflow-hidden">
        <DialogHeader className="px-6 pt-6 pb-4 border-b border-border">
          <DialogTitle className="text-xl font-semibold">
            购买 {courseName}
          </DialogTitle>
        </DialogHeader>

        <div className="px-6 py-5 space-y-6">
          {/* Course & Price Info */}
          <div className="space-y-3">
            <div className="flex justify-between items-center py-2 border-b border-border/50">
              <span className="text-sm text-muted-foreground">课程</span>
              <span className="text-sm font-medium text-foreground">{courseName}</span>
            </div>
            <div className="flex justify-between items-center py-2 border-b border-border/50">
              <span className="text-sm text-muted-foreground">总费用</span>
              <div className="text-right">
                <span className="text-lg font-bold text-foreground">
                  ${price}
                </span>
              </div>
            </div>
          </div>

          {/* Payment Method */}
          <div className="space-y-3">
            <label className="text-sm font-medium text-foreground">支付方式</label>
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
                    paymentMethod === "stripe"
                      ? "border-primary"
                      : "border-muted-foreground/30"
                  )}
                >
                  {paymentMethod === "stripe" && (
                    <div className="h-2.5 w-2.5 rounded-full bg-primary" />
                  )}
                </div>
                <CreditCard className="h-4 w-4 text-primary" />
                <span className="text-sm font-medium text-foreground">
                  在线支付 (Stripe)
                </span>
                <Badge className="ml-auto bg-amber-500/10 text-amber-700 border-0 text-xs">
                  推荐 · 即时生效
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
                    paymentMethod === "bank"
                      ? "border-primary"
                      : "border-muted-foreground/30"
                  )}
                >
                  {paymentMethod === "bank" && (
                    <div className="h-2.5 w-2.5 rounded-full bg-primary" />
                  )}
                </div>
                <Building2 className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium text-foreground">
                  银行转账 / 线下支付
                </span>
              </button>
            </div>
          </div>

        </div>

        {/* Footer Actions */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-border bg-muted/30">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button onClick={handlePayment} className="gap-2" disabled={loading}>
            <CreditCard className="h-4 w-4" />
            {loading ? "处理中..." : paymentMethod === "stripe" ? "前往在线支付" : "提交订单"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
