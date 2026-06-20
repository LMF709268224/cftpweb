"use client"

import React, { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Activity, RefreshCw, Search, RotateCcw } from "lucide-react"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"
import { statusBadgeClassForStatusValue } from "@/lib/status-labels"

function WebhookDetailPreview({ msgFp }: { msgFp: string }) {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let active = true
    setLoading(true)
    apiClient(`/api/audit/webhooks/detail?msg_fp=${msgFp}`)
      .then(res => {
        if (active) setDetail(res)
      })
      .catch(console.error)
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => { active = false }
  }, [msgFp])

  if (loading) return <div className="mt-2 text-xs text-muted-foreground">Loading payload...</div>
  if (!detail) return <div className="mt-2 text-xs text-red-500">Failed to load payload</div>

  return (
    <div className="mt-3 bg-muted/30 p-4 rounded text-sm font-mono whitespace-pre-wrap break-words">
      <div className="mb-2 font-semibold">{t.auditWebhooksPage?.rawPayload}:</div>
      <pre className="text-xs overflow-x-auto bg-black/5 p-2 rounded">
        {JSON.stringify(detail.raw_payload ? JSON.parse(detail.raw_payload) : {}, null, 2)}
      </pre>
      {detail.parsed_metadata && (
        <>
          <div className="mt-4 mb-2 font-semibold">{t.auditWebhooksPage?.parsedMetadata}:</div>
          <pre className="text-xs overflow-x-auto bg-black/5 p-2 rounded">
            {JSON.stringify(detail.parsed_metadata, null, 2)}
          </pre>
        </>
      )}
    </div>
  )
}

export default function AuditWebhooksPage() {
  const { t } = useTranslation()
  const [messages, setMessages] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  // Filters
  const [provider, setProvider] = useState("")
  const [status, setStatus] = useState("")
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [totalCount, setTotalCount] = useState(0)

  const [expandedMsg, setExpandedMsg] = useState<string | null>(null)
  const [reprocessing, setReprocessing] = useState<string | null>(null)

  const fetchMessages = async () => {
    setLoading(true)
    try {
      let query = `/api/audit/webhooks?page=${page}&page_size=${pageSize}`
      if (provider) query += `&provider=${provider}`
      if (status) query += `&status=${status}`

      const res = await apiClient(query)
      if (res?.messages) {
        setMessages(res.messages)
        setTotalCount(res.total || 0)
      } else {
        setMessages([])
        setTotalCount(0)
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMessages()
  }, [page, pageSize])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setPage(1)
    fetchMessages()
  }

  const handleReprocess = async (id: number, e: React.MouseEvent) => {
    e.stopPropagation()
    setReprocessing(id.toString())
    try {
      await apiClient("/api/audit/webhooks/reprocess", {
        method: "POST",
        body: JSON.stringify({ webhook_msg_id: id })
      })
      alert("Reprocess requested successfully")
      fetchMessages()
    } catch (err) {
      console.error(err)
      alert("Failed to reprocess webhook")
    } finally {
      setReprocessing(null)
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="flex justify-between items-center mb-8">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <Activity className="h-8 w-8 text-primary" />
              {t.auditWebhooksPage?.title}
            </h1>
            <p className="text-muted-foreground mt-2">{t.auditWebhooksPage?.subtitle}</p>
          </div>
          <Button onClick={fetchMessages} variant="outline" className="gap-2">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            {t.auditWebhooksPage?.refresh}
          </Button>
        </div>

        <form onSubmit={handleSearch} className="bg-card p-4 rounded-xl border mb-6 flex flex-wrap gap-4 items-end">
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">{t.auditWebhooksPage?.provider}</label>
            <Input 
              placeholder="e.g. pearson" 
              value={provider} 
              onChange={e => setProvider(e.target.value)} 
              className="h-9 w-48"
            />
          </div>
          <div className="space-y-1">
            <label className="text-xs font-medium text-muted-foreground">{t.auditWebhooksPage?.status}</label>
            <select
              className="flex h-9 w-40 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
              value={status}
              onChange={e => setStatus(e.target.value)}
            >
              <option value="">{t.auditWebhooksPage?.allStatus}</option>
              <option value="SUCCESS">SUCCESS</option>
              <option value="FAILED">FAILED</option>
              <option value="IGNORED">IGNORED</option>
              <option value="PENDING">PENDING</option>
            </select>
          </div>
          <Button type="submit" className="h-9">
            <Search className="h-4 w-4 mr-2" />
            {t.auditWebhooksPage?.search}
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
                    <th className="px-4 py-3 font-medium">MsgFp</th>
                    <th className="px-4 py-3 font-medium">{t.auditWebhooksPage?.providerName}</th>
                    <th className="px-4 py-3 font-medium">{t.auditWebhooksPage?.statusLabel}</th>
                    <th className="px-4 py-3 font-medium">Error</th>
                    <th className="px-4 py-3 font-medium">{t.auditWebhooksPage?.receivedAt}</th>
                    <th className="px-4 py-3 font-medium text-right">{t.auditWebhooksPage?.action}</th>
                  </tr>
                </thead>
                <tbody>
                  {messages.length > 0 ? (
                    messages.map((msg) => {
                      const isExpanded = expandedMsg === msg.msg_fp
                      return (
                        <React.Fragment key={msg.msg_fp}>
                          <tr 
                            className="border-b last:border-0 hover:bg-muted/50 transition-colors cursor-pointer"
                            onClick={() => setExpandedMsg(isExpanded ? null : msg.msg_fp)}
                          >
                            <td className="px-4 py-3 font-medium text-xs font-mono max-w-[200px] truncate" title={msg.msg_fp}>
                              {msg.msg_fp}
                            </td>
                            <td className="px-4 py-3 uppercase font-semibold">{msg.provider}</td>
                            <td className="px-4 py-3">
                              <Badge variant="outline" className={statusBadgeClassForStatusValue(msg.status)}>
                                {msg.status || t.common?.unknown}
                              </Badge>
                            </td>
                            <td className="px-4 py-3 text-red-500 text-xs max-w-[200px] truncate" title={msg.error_message}>
                              {msg.error_message || "-"}
                            </td>
                            <td className="px-4 py-3 text-xs">{formatBackendDate(msg.created_at)}</td>
                            <td className="px-4 py-3 text-right">
                              <Button 
                                variant="outline" 
                                size="sm" 
                                className="h-7 text-xs gap-1"
                                disabled={reprocessing === msg.id?.toString()}
                                onClick={(e) => handleReprocess(msg.id, e)}
                              >
                                <RotateCcw className={`h-3 w-3 ${reprocessing === msg.id?.toString() ? 'animate-spin' : ''}`} />
                                {t.auditWebhooksPage?.retry}
                              </Button>
                            </td>
                          </tr>
                          {isExpanded && (
                            <tr className="border-b bg-muted/10">
                              <td colSpan={6} className="px-4 py-3">
                                <WebhookDetailPreview msgFp={msg.msg_fp} />
                              </td>
                            </tr>
                          )}
                        </React.Fragment>
                      )
                    })
                  ) : (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                        {t.auditWebhooksPage?.noRecords}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
            
            <div className="flex items-center justify-between p-4 border-t">
              <div className="text-sm text-muted-foreground">
                {t.auditWebhooksPage?.totalCount ? t.auditWebhooksPage.totalCount.replace("{{total}}", totalCount.toString()) : `共 ${totalCount} 条记录`}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                >
                  {t.auditWebhooksPage?.prevPage}
                </Button>
                <span className="px-3 py-1 text-sm">{page}</span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage(p => p + 1)}
                  disabled={page * pageSize >= totalCount}
                >
                  {t.auditWebhooksPage?.nextPage}
                </Button>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
