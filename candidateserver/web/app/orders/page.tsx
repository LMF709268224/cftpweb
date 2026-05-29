"use client"

import React from "react"

import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import {
  ShoppingCart,
  Receipt,
  ChevronRight,
  CheckCircle2,
  Clock,
  Package,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"

type OrderItem = {
  id: string
  items: string[]
  date: string
  amount: string
  status: keyof typeof statusConfig
  paymentMethod: string
}

const orders: OrderItem[] = []

const statusConfig = {
  completed: {
    label: "已完成",
    icon: CheckCircle2,
    color: "bg-emerald-500/10 text-emerald-700",
    iconColor: "text-emerald-500",
  },
  pending: {
    label: "待支付",
    icon: Clock,
    color: "bg-amber-500/10 text-amber-700",
    iconColor: "text-amber-500",
  },
  processing: {
    label: "处理中",
    icon: Package,
    color: "bg-blue-500/10 text-blue-700",
    iconColor: "text-blue-500",
  },
}

export default function OrdersPage() {
  const { t, lang } = useTranslation()
  const totalSpent = orders.reduce((sum, order) => {
    const amount = parseFloat(order.amount.replace(/[¥,]/g, ""))
    return sum + amount
  }, 0)

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.orders.title}</h1>
            <p className="mt-1 text-muted-foreground">{t.orders.subtitle}</p>
          </div>

          {/* Stats */}
          <div className="mb-8 grid gap-4 sm:grid-cols-3">
            <div className="flex items-center gap-4 rounded-xl border border-border bg-card p-5">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10">
                <ShoppingCart className="h-6 w-6 text-primary" />
              </div>
              <div>
                <p className="text-2xl font-bold text-card-foreground">{orders.length}</p>
                <p className="text-sm text-muted-foreground">{t.orders.totalOrders}</p>
              </div>
            </div>
            <div className="flex items-center gap-4 rounded-xl border border-border bg-card p-5">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-500/10">
                <CheckCircle2 className="h-6 w-6 text-emerald-600" />
              </div>
              <div>
                <p className="text-2xl font-bold text-card-foreground">
                  {orders.filter((o) => o.status === "completed").length}
                </p>
                <p className="text-sm text-muted-foreground">{t.orders.completed}</p>
              </div>
            </div>
            <div className="flex items-center gap-4 rounded-xl border border-border bg-card p-5">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-500/10">
                <Receipt className="h-6 w-6 text-amber-600" />
              </div>
              <div>
                <p className="text-2xl font-bold text-card-foreground">
                  ¥{totalSpent.toLocaleString()}
                </p>
                <p className="text-sm text-muted-foreground">{t.orders.totalSpent}</p>
              </div>
            </div>
          </div>

          {/* Orders List */}
          <div className="rounded-2xl border border-border bg-card shadow-sm overflow-hidden">
            <div className="flex items-center gap-3 border-b border-border px-6 py-4">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10">
                <Receipt className="h-4 w-4 text-primary" />
              </div>
              <h2 className="font-semibold text-card-foreground">{t.orders.orderHistory}</h2>
            </div>
            
            {orders.length === 0 ? (
              <div className="flex flex-col items-center justify-center px-6 py-14 text-center">
                <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-muted">
                  <Package className="h-7 w-7 text-muted-foreground" />
                </div>
                <h3 className="mb-2 text-lg font-semibold text-foreground">
                  {lang === "zh" ? "暂无订单记录" : "No orders yet"}
                </h3>
                <p className="max-w-md text-sm text-muted-foreground">
                  {lang === "zh"
                    ? "订单接口接入后，这里会展示真实的购买记录和支付状态。"
                    : "Order records and payment status will appear here after the order API is connected."}
                </p>
              </div>
            ) : (
              <div className="divide-y divide-border">
                {orders.map((order) => {
                const config = statusConfig[order.status as keyof typeof statusConfig]
                return (
                  <div
                    key={order.id}
                    className="group flex items-center justify-between p-6 transition-colors hover:bg-muted/50"
                  >
                    <div className="flex items-center gap-4">
                      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-muted">
                        <Package className="h-6 w-6 text-muted-foreground" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="font-medium text-card-foreground">
                            {order.items.join(", ")}
                          </h3>
                          <Badge className={config.color}>
                            {lang === "zh" ? config.label : (
                              order.status === "completed" ? "Completed" : 
                              order.status === "pending" ? "Pending" : "Processing"
                            )}
                          </Badge>
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {order.id} · {order.date} · {order.paymentMethod}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right">
                        <p className="font-semibold text-card-foreground">{order.amount}</p>
                      </div>
                      <ChevronRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1" />
                    </div>
                  </div>
                )
                })}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
