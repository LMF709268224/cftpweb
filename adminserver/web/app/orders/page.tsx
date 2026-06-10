"use client"

import React from "react"
import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ShoppingCart, RefreshCw, Search } from "lucide-react"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"
import { statusBadgeClassForStatusValue } from "@cftpweb/shared"
import { Input } from "@/components/ui/input"

export default function AdminOrdersPage() {
  const { t } = useTranslation()
  const [orders, setOrders] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  // Filters
  const [candidateUlid, setCandidateUlid] = useState("")
  const [bizType, setBizType] = useState("")
  const [orderStatus, setOrderStatus] = useState("")
  const [paymentStatus, setPaymentStatus] = useState("")
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [totalCount, setTotalCount] = useState(0)

  const fetchOrders = async () => {
    setLoading(true)
    try {
      let query = `/api/mall/orders?page=${page}&limit=${pageSize}&offset=${(page-1)*pageSize}`
      if (candidateUlid) query += `&candidate_ulid=${candidateUlid}`
      if (bizType) query += `&biz_type=${bizType}`
      if (orderStatus) query += `&order_status=${orderStatus}`
      if (paymentStatus) query += `&payment_status=${paymentStatus}`

      const res = await apiClient(query)
      if (res?.items) {
        setOrders(res.items)
        setTotalCount(res.total || 0)
      } else {
        setOrders([])
        setTotalCount(0)
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchOrders()
  }, [page, pageSize])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setPage(1)
    fetchOrders()
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
          <Button onClick={fetchOrders} variant="outline" className="gap-2">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
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
              <option value="pipeline_enrollment">{t.adminOrdersPage?.pipelineOrder}</option>
              <option value="stage_enrollment">{t.adminOrdersPage?.stageOrder}</option>
              <option value="course_retake">{t.adminOrdersPage?.retakeOrder}</option>
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
              <option value="PENDING">PENDING</option>
              <option value="COMPLETED">COMPLETED</option>
              <option value="CANCELLED">CANCELLED</option>
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
              <option value="UNPAID">UNPAID</option>
              <option value="PAID">PAID</option>
              <option value="REFUNDED">REFUNDED</option>
            </select>
          </div>
          <Button type="submit" className="h-9">
            <Search className="h-4 w-4 mr-2" />
            {t.adminOrdersPage?.search}
          </Button>
        </form>

        {loading ? (
          <div>{t.common?.loading || "加载中..."}</div>
        ) : (
          <div className="bg-card rounded-xl border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b bg-muted/50 text-muted-foreground">
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.bizOrderUlid}</th>
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.bizType}</th>
                    <th className="px-4 py-3 font-medium">BizRef</th>
                    <th className="px-4 py-3 font-medium">Amount / Currency</th>
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.status}</th>
                    <th className="px-4 py-3 font-medium">{t.adminOrdersPage?.paymentStatus}</th>
                    <th className="px-4 py-3 font-medium">Created At</th>
                  </tr>
                </thead>
                <tbody>
                  {orders.length > 0 ? (
                    orders.map((o) => (
                      <tr key={o.logical_order_ulid} className="border-b last:border-0 hover:bg-muted/50 transition-colors">
                        <td className="px-4 py-3 font-medium text-xs font-mono">{o.logical_order_ulid}</td>
                        <td className="px-4 py-3">{o.biz_type}</td>
                        <td className="px-4 py-3 text-xs font-mono">{o.biz_ref_ulid}</td>
                        <td className="px-4 py-3 font-medium">{o.total_amount_cents ? (o.total_amount_cents / 100).toFixed(2) : "0.00"} {o.currency}</td>
                        <td className="px-4 py-3">
                          <Badge variant="outline" className={statusBadgeClassForStatusValue(o.order_status)}>
                            {o.order_status || t.common?.unknown}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant="outline" className={statusBadgeClassForStatusValue(o.payment_status)}>
                            {o.payment_status || t.common?.unknown}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-xs">{formatBackendDate(o.created_at)}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                        {t.adminOrdersPage?.noOrders}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            
            <div className="flex items-center justify-between p-4 border-t">
              <div className="text-sm text-muted-foreground">
                {t.adminOrdersPage?.totalCount ? t.adminOrdersPage.totalCount.replace("{{total}}", totalCount.toString()) : `共 ${totalCount} 条记录`}
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
