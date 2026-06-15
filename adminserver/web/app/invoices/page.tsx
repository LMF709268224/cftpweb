"use client"

import React from "react"
import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { FileText, RefreshCw } from "lucide-react"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"
import { statusBadgeClassForStatusValue } from "@cftpweb/shared"

export default function AdminInvoicesPage() {
  const { t } = useTranslation()
  const [invoices, setInvoices] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  // Filters
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [totalCount, setTotalCount] = useState(0)

  const fetchInvoices = async () => {
    setLoading(true)
    try {
      let query = `/api/mall/invoices?page=${page}&page_size=${pageSize}`
      const res = await apiClient(query)
      if (res?.invoices) {
        setInvoices(res.invoices)
        setTotalCount(res.total || 0)
      } else {
        setInvoices([])
        setTotalCount(0)
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchInvoices()
  }, [page, pageSize])

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="flex justify-between items-center mb-8">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <FileText className="h-8 w-8 text-primary" />
              {t.adminInvoicesPage?.title || "发票管理 (Invoices)"}
            </h1>
            <p className="text-muted-foreground mt-2">{t.adminInvoicesPage?.subtitle || "查看系统内所有产生的发票"}</p>
          </div>
          <Button onClick={fetchInvoices} variant="outline" className="gap-2">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            {t.adminOrdersPage?.refresh || "刷新"}
          </Button>
        </div>

        {loading ? (
          <div>{t.common?.loading || "加载中..."}</div>
        ) : (
          <div className="bg-card rounded-xl border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b bg-muted/50 text-muted-foreground">
                    <th className="px-4 py-3 font-medium">Invoice ID (Stripe)</th>
                    <th className="px-4 py-3 font-medium">Order ID</th>
                    <th className="px-4 py-3 font-medium">Customer ID</th>
                    <th className="px-4 py-3 font-medium">Amount / Currency</th>
                    <th className="px-4 py-3 font-medium">Status</th>
                    <th className="px-4 py-3 font-medium">Paid At</th>
                    <th className="px-4 py-3 font-medium">Created At</th>
                  </tr>
                </thead>
                <tbody>
                  {invoices.length > 0 ? (
                    invoices.map((inv) => (
                      <tr key={inv.id} className="border-b last:border-0 hover:bg-muted/50 transition-colors">
                        <td className="px-4 py-3 font-medium text-xs font-mono">{inv.id}</td>
                        <td className="px-4 py-3 text-xs font-mono">{inv.order_id}</td>
                        <td className="px-4 py-3 text-xs font-mono">{inv.email}</td>
                        <td className="px-4 py-3 font-medium">{inv.amount.toFixed(2)} {inv.currency?.toUpperCase()}</td>
                        <td className="px-4 py-3">
                          <Badge variant="outline" className={statusBadgeClassForStatusValue(inv.status)}>
                            {inv.status || "UNKNOWN"}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-xs">{inv.paid_at ? new Date(inv.paid_at).toLocaleString() : '-'}</td>
                        <td className="px-4 py-3 text-xs">{inv.created_at ? new Date(inv.created_at).toLocaleString() : '-'}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                        暂无发票记录
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            
            <div className="flex items-center justify-between p-4 border-t">
              <div className="text-sm text-muted-foreground">
                {`共 ${totalCount} 条记录`}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  上一页
                </Button>
                <span className="px-3 py-1 text-sm">{page}</span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => p + 1)}
                  disabled={page * pageSize >= totalCount}
                >
                  下一页
                </Button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
