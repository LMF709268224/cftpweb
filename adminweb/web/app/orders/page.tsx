"use client"

import React from "react"
import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ShoppingCart, RefreshCw, Search, Trash2 } from "lucide-react"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"
import { statusBadgeClassForStatusValue } from "@/lib/status-labels"
import { Input } from "@/components/ui/input"
import { toast } from "sonner"

type AdminOrder = Record<string, any>
type LabelOption = { value: string; zh: string; en: string }
type Lang = "zh" | "en"

const BIZ_TYPE_OPTIONS: LabelOption[] = [
  { value: "PIPELINE_PAYMENT", zh: "\u7ba1\u7ebf\u8ba2\u5355", en: "Pipeline Order" },
  { value: "STAGE_PAYMENT", zh: "\u9636\u6bb5\u8ba2\u5355", en: "Stage Order" },
  { value: "COURSE_RETAKE_PAYMENT", zh: "\u91cd\u8003\u8ba2\u5355", en: "Retake Order" },
  { value: "PIPELINE_UNLOCK", zh: "\u7ba1\u7ebf\u89e3\u9501\u8ba2\u5355", en: "Pipeline Unlock Order" },
  { value: "CREDENTIAL_APPLICATION", zh: "\u8d44\u683c\u7533\u8bf7\u8ba2\u5355", en: "Credential Application Order" },
  { value: "BUNDLE_PURCHASE", zh: "\u8ba4\u8bc1\u5957\u9910\u8ba2\u5355", en: "Bundle Purchase Order" },
]

const ORDER_STATUS_OPTIONS: LabelOption[] = [
  { value: "PENDING", zh: "\u5f85\u5904\u7406", en: "Pending" },
  { value: "COMPLETED", zh: "\u5df2\u5b8c\u6210", en: "Completed" },
  { value: "CANCELLED", zh: "\u5df2\u53d6\u6d88", en: "Cancelled" },
  { value: "FAILED", zh: "\u5931\u8d25", en: "Failed" },
  { value: "EXPIRED", zh: "\u5df2\u8fc7\u671f", en: "Expired" },
]

const PAYMENT_STATUS_OPTIONS: LabelOption[] = [
  { value: "WAIT_PAY", zh: "\u5f85\u652f\u4ed8", en: "Waiting Payment" },
  { value: "UNPAID", zh: "\u5f85\u652f\u4ed8", en: "Unpaid" },
  { value: "PAID", zh: "\u5df2\u652f\u4ed8", en: "Paid" },
  { value: "FAILED", zh: "\u652f\u4ed8\u5931\u8d25", en: "Failed" },
  { value: "REFUNDED", zh: "\u5df2\u9000\u6b3e", en: "Refunded" },
  { value: "CANCELLED", zh: "\u5df2\u53d6\u6d88", en: "Cancelled" },
]

const normalizeCode = (value: unknown) => String(value || "").trim().toUpperCase()

const pickFirst = (order: AdminOrder, keys: string[]) => {
  for (const key of keys) {
    const value = order[key]
    if (value !== undefined && value !== null && value !== "") {
      return value
    }
  }
  return undefined
}

const findLabel = (options: LabelOption[], value: unknown, lang: Lang) => {
  const normalized = normalizeCode(value)
  const option = options.find(item => item.value === normalized)
  return option ? option[lang] : normalized || "-"
}

const getOrderUlid = (order: AdminOrder) => String(pickFirst(order, [
  "order_ulid",
  "orderUlid",
  "logical_order_ulid",
  "logicalOrderUlid",
  "biz_order_ulid",
  "bizOrderUlid",
]) || "-")

const getBizType = (order: AdminOrder) => pickFirst(order, ["biz_type", "bizType"])
const getBizRefUlid = (order: AdminOrder) => String(pickFirst(order, ["biz_ref_ulid", "bizRefUlid"]) || "")
const getCandidateUlid = (order: AdminOrder) => String(pickFirst(order, ["candidate_ulid", "candidateUlid"]) || "")
const getOrderStatus = (order: AdminOrder) => pickFirst(order, ["order_status", "orderStatus", "status"])
const getPaymentStatus = (order: AdminOrder) => pickFirst(order, ["payment_status", "paymentStatus"])
const isBundlePurchaseOrder = (order: AdminOrder) => normalizeCode(getBizType(order)) === "BUNDLE_PURCHASE"

const getCurrency = (order: AdminOrder) => String(pickFirst(order, [
  "currency_code",
  "currencyCode",
  "currency",
]) || "")

const getOrderAmount = (order: AdminOrder) => {
  const directTotal = pickFirst(order, ["total", "total_amount", "totalAmount"])
  if (directTotal !== undefined) {
    const value = Number(directTotal)
    return Number.isFinite(value) ? value : 0
  }

  const minorAmount = pickFirst(order, [
    "amount_minor",
    "amountMinor",
    "total_amount_cents",
    "totalAmountCents",
  ])
  if (minorAmount !== undefined) {
    const value = Number(minorAmount)
    return Number.isFinite(value) ? value / 100 : 0
  }

  const amount = Number(pickFirst(order, ["amount"]) || 0)
  return Number.isFinite(amount) ? amount / 100 : 0
}

const formatOrderCreatedAt = (value: unknown) => {
  if (typeof value === "number") {
    const ms = value > 1_000_000_000_000 ? value : value * 1000
    return formatBackendDate(new Date(ms).toISOString())
  }
  return formatBackendDate(value ? String(value) : "")
}

export default function AdminOrdersPage() {
  const { t, lang } = useTranslation()
  const [orders, setOrders] = useState<AdminOrder[]>([])
  const [loading, setLoading] = useState(true)

  const [candidateUlid, setCandidateUlid] = useState("")
  const [bizType, setBizType] = useState("")
  const [orderStatus, setOrderStatus] = useState("")
  const [paymentStatus, setPaymentStatus] = useState("")
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [totalCount, setTotalCount] = useState(0)
  const [purgingOrderUlid, setPurgingOrderUlid] = useState("")

  const fetchOrders = async (targetPage = page) => {
    setLoading(true)
    try {
      let query = `/api/mall/orders?page=${targetPage}&limit=${pageSize}&offset=${(targetPage - 1) * pageSize}`
      if (candidateUlid) query += `&candidate_ulid=${encodeURIComponent(candidateUlid)}`
      if (bizType) query += `&biz_type=${encodeURIComponent(bizType)}`
      if (orderStatus) query += `&order_status=${encodeURIComponent(orderStatus)}`
      if (paymentStatus) query += `&payment_status=${encodeURIComponent(paymentStatus)}`

      const res = await apiClient(query)
      const items = Array.isArray(res?.items)
        ? res.items
        : Array.isArray(res?.orders)
          ? res.orders
          : []

      setOrders(items)
      setTotalCount(Number(res?.total ?? res?.total_count ?? res?.totalCount ?? items.length) || 0)
    } catch (e) {
      console.error(e)
      setOrders([])
      setTotalCount(0)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchOrders(page)
  }, [page, pageSize])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (page === 1) {
      fetchOrders(1)
    } else {
      setPage(1)
    }
  }

  const handlePurgeBundleOrder = async (order: AdminOrder) => {
    const candidate = getCandidateUlid(order)
    const bundleOrderUlid = getBizRefUlid(order) || getOrderUlid(order)
    if (!candidate || !bundleOrderUlid || bundleOrderUlid === "-") {
      toast.error(lang === "zh" ? "缺少 candidate_ulid 或 bundle_order_ulid" : "Missing candidate_ulid or bundle_order_ulid")
      return
    }
    const confirmed = window.confirm(
      lang === "zh"
        ? "这会清理该认证套餐订单及其关联的测试数据，确认继续？"
        : "This will purge the bundle order and related test data. Continue?",
    )
    if (!confirmed) return

    setPurgingOrderUlid(bundleOrderUlid)
    try {
      const res = await apiClient("/api/mall/bundle-orders/purge", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          candidate_ulid: candidate,
          bundle_order_ulid: bundleOrderUlid,
        }),
      })
      toast.success(res?.message || (lang === "zh" ? "认证数据已清理" : "Bundle data purged"))
      await fetchOrders(page)
    } finally {
      setPurgingOrderUlid("")
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="flex justify-between items-center mb-8">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <ShoppingCart className="h-8 w-8 text-primary" />
              {t.adminOrdersPage?.title}
            </h1>
            <p className="text-muted-foreground mt-2">{t.adminOrdersPage?.subtitle}</p>
          </div>
          <Button onClick={() => fetchOrders(page)} variant="outline" className="gap-2">
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            {t.adminOrdersPage?.refresh}
          </Button>
        </div>

        <form onSubmit={handleSearch} className="bg-card p-4 rounded-xl border mb-6 flex flex-wrap gap-4 items-end">
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">{t.adminOrdersPage?.userUlid}</label>
            <Input
              placeholder="Candidate ULID"
              value={candidateUlid}
              onChange={e => setCandidateUlid(e.target.value)}
              className="h-9 w-64"
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">{t.adminOrdersPage?.bizType}</label>
            <select
              className="flex h-9 w-40 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
              value={bizType}
              onChange={e => setBizType(e.target.value)}
            >
              <option value="">{t.adminOrdersPage?.allTypes}</option>
              {BIZ_TYPE_OPTIONS.map(option => (
                <option key={option.value} value={option.value}>
                  {option[lang]}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">{t.adminOrdersPage?.status}</label>
            <select
              className="flex h-9 w-40 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
              value={orderStatus}
              onChange={e => setOrderStatus(e.target.value)}
            >
              <option value="">{t.adminOrdersPage?.allStatus}</option>
              {ORDER_STATUS_OPTIONS.map(option => (
                <option key={option.value} value={option.value}>
                  {option[lang]}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">{t.adminOrdersPage?.paymentStatus}</label>
            <select
              className="flex h-9 w-40 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
              value={paymentStatus}
              onChange={e => setPaymentStatus(e.target.value)}
            >
              <option value="">{t.adminOrdersPage?.allPaymentStatus}</option>
              {PAYMENT_STATUS_OPTIONS.map(option => (
                <option key={option.value} value={option.value}>
                  {option[lang]}
                </option>
              ))}
            </select>
          </div>
          <Button type="submit" className="h-9">
            <Search className="h-4 w-4 mr-2" />
            {t.adminOrdersPage?.search}
          </Button>
        </form>

        {loading ? (
          <div>{t.common?.loading || "Loading..."}</div>
        ) : (
          <div className="bg-card rounded-xl border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b bg-muted/50 text-muted-foreground">
                    <th className="px-4 py-3 font-medium">{lang === "zh" ? "\u8ba2\u5355 ULID" : "Order ULID"}</th>
                    <th className="px-4 py-3 font-medium">{lang === "zh" ? "\u5019\u9009\u4eba ULID" : "Candidate ULID"}</th>
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.bizType}</th>
                    <th className="px-4 py-3 font-medium">{lang === "zh" ? "\u4e1a\u52a1\u5f15\u7528 ULID" : "Biz Ref ULID"}</th>
                    <th className="px-4 py-3 font-medium">{lang === "zh" ? "\u91d1\u989d / \u8d27\u5e01" : "Amount / Currency"}</th>
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.status}</th>
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.paymentStatus}</th>
                    <th className="px-4 py-3 font-medium">Created At</th>
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.action || (lang === "zh" ? "\u64cd\u4f5c" : "Action")}</th>
                  </tr>
                </thead>
                <tbody>
                  {orders.length > 0 ? (
                    orders.map((order, index) => {
                      const orderUlid = getOrderUlid(order)
                      const bizTypeValue = getBizType(order)
                      const bizRefUlid = getBizRefUlid(order)
                      const bundleOrderUlid = bizRefUlid || orderUlid
                      const orderStatusValue = getOrderStatus(order)
                      const paymentStatusValue = getPaymentStatus(order)

                      return (
                        <tr key={`${orderUlid}-${index}`} className="border-b last:border-0 hover:bg-muted/50 transition-colors">
                          <td className="px-4 py-3 font-medium text-xs font-mono">{orderUlid}</td>
                          <td className="px-4 py-3 text-xs font-mono">{getCandidateUlid(order) || "-"}</td>
                          <td className="px-4 py-3">{findLabel(BIZ_TYPE_OPTIONS, bizTypeValue, lang)}</td>
                          <td className="px-4 py-3 text-xs font-mono">{bizRefUlid || "-"}</td>
                          <td className="px-4 py-3 font-medium">{getOrderAmount(order).toFixed(2)} {getCurrency(order)}</td>
                          <td className="px-4 py-3">
                            <Badge variant="outline" className={statusBadgeClassForStatusValue(orderStatusValue)}>
                              {findLabel(ORDER_STATUS_OPTIONS, orderStatusValue, lang)}
                            </Badge>
                          </td>
                          <td className="px-4 py-3">
                            <Badge variant="outline" className={statusBadgeClassForStatusValue(paymentStatusValue)}>
                              {findLabel(PAYMENT_STATUS_OPTIONS, paymentStatusValue, lang)}
                            </Badge>
                          </td>
                          <td className="px-4 py-3 text-xs">{formatOrderCreatedAt(order.created_at ?? order.createdAt)}</td>
                          <td className="px-4 py-3">
                            {isBundlePurchaseOrder(order) ? (
                              <Button
                                variant="destructive"
                                size="sm"
                                className="gap-2"
                                disabled={purgingOrderUlid === bundleOrderUlid}
                                onClick={() => handlePurgeBundleOrder(order)}
                              >
                                <Trash2 className="h-4 w-4" />
                                {purgingOrderUlid === bundleOrderUlid
                                  ? t.common?.loading || "Loading..."
                                  : lang === "zh" ? "\u6e05\u7406\u8ba4\u8bc1\u6570\u636e" : "Purge bundle data"}
                              </Button>
                            ) : (
                              <span className="text-xs text-muted-foreground">-</span>
                            )}
                          </td>
                        </tr>
                      )
                    })
                  ) : (
                    <tr>
                      <td colSpan={9} className="px-4 py-8 text-center text-muted-foreground">
                        {t.adminOrdersPage?.noOrders}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>

            <div className="flex items-center justify-between p-4 border-t">
              <div className="text-sm text-muted-foreground">
                {t.adminOrdersPage?.totalCount
                  ? t.adminOrdersPage.totalCount.replace("{{total}}", totalCount.toString())
                  : `Total ${totalCount} Records`}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  {t.adminOrdersPage?.prevPage}
                </Button>
                <span className="px-3 py-1 text-sm">{page}</span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => p + 1)}
                  disabled={page * pageSize >= totalCount}
                >
                  {t.adminOrdersPage?.nextPage}
                </Button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
