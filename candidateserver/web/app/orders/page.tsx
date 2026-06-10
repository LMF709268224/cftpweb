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
import { useTranslation } from "@/lib/useLanguage"
import { statusBadgeClassForStatusValue } from "@cftpweb/shared"

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
    labelKey: "statusCompleted",
    icon: CheckCircle2,
    statusValue: "SUCCESS",
  },
  pending: {
    labelKey: "statusPending",
    icon: Clock,
    statusValue: "PENDING",
  },
  processing: {
    labelKey: "statusProcessing",
    icon: Package,
    statusValue: "PROCESSING",
  },
} as const

export default function OrdersPage() {
  const { t } = useTranslation()
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
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-100">
                <CheckCircle2 className="h-6 w-6 text-black" />
              </div>
              <div>
                <p className="text-2xl font-bold text-card-foreground">
                  {orders.filter((o) => o.status === "completed").length}
                </p>
                <p className="text-sm text-muted-foreground">{t.orders.completed}</p>
              </div>
            </div>
            <div className="flex items-center gap-4 rounded-xl border border-border bg-card p-5">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-yellow-100">
                <Receipt className="h-6 w-6 text-black" />
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
                  {t.orders.noOrders}
                </h3>
                <p className="max-w-md text-sm text-muted-foreground">
                  {t.orders.noOrdersDesc}
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
                          <Badge className={statusBadgeClassForStatusValue(config.statusValue)}>
                            {t.orders[config.labelKey]}
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
