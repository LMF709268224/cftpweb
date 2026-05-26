"use client"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { FileText, RefreshCw } from "lucide-react"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/useLanguage"

export default function PdfRequestsPage() {
  const { t } = useTranslation()
  const [requests, setRequests] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  const fetchRequests = async () => {
    setLoading(true)
    try {
      const res = await apiClient("/api/pdf-requests")
      if (res?.requests) {
        setRequests(res.requests)
      } else {
        setRequests([])
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchRequests()
  }, [])

  const getStatusBadge = (status: number) => {
    // 0: UNSPECIFIED, 1: PENDING, 2: GENERATING, 3: SUCCESS, 4: FAILED
    switch (status) {
      case 1:
        return <Badge variant="outline" className="text-yellow-600 bg-yellow-50 border-yellow-200">排队中 (Pending)</Badge>
      case 2:
        return <Badge variant="outline" className="text-blue-600 bg-blue-50 border-blue-200">生成中 (Generating)</Badge>
      case 3:
        return <Badge className="bg-green-600 hover:bg-green-700">成功 (Success)</Badge>
      case 4:
        return <Badge variant="destructive">失败 (Failed)</Badge>
      default:
        return <Badge variant="secondary">未知 (Unknown)</Badge>
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="flex justify-between items-center mb-8">
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <FileText className="h-8 w-8 text-primary" />
              {t.sidebar?.pdfRequests || "证书流水记录"}
            </h1>
            <p className="text-muted-foreground mt-2">查看后台自动或手动触发的 PDF 证书生成情况</p>
          </div>
          <Button onClick={fetchRequests} variant="outline" className="gap-2">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            刷新
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
                    <th className="px-4 py-3 font-medium">请求ID</th>
                    <th className="px-4 py-3 font-medium">考生ID</th>
                    <th className="px-4 py-3 font-medium">业务编号(DegreeNo)</th>
                    <th className="px-4 py-3 font-medium">证书哈希(SHA256)</th>
                    <th className="px-4 py-3 font-medium">状态</th>
                    <th className="px-4 py-3 font-medium">错误信息</th>
                    <th className="px-4 py-3 font-medium">创建时间</th>
                  </tr>
                </thead>
                <tbody>
                  {requests.length > 0 ? (
                    requests.map((req) => (
                      <tr key={req.request_id} className="border-b last:border-0 hover:bg-muted/50 transition-colors">
                        <td className="px-4 py-3 font-medium text-xs">{req.request_id}</td>
                        <td className="px-4 py-3 text-xs">{req.candidate_id}</td>
                        <td className="px-4 py-3 font-medium">{req.degree_no}</td>
                        <td className="px-4 py-3 font-mono text-xs max-w-[150px] truncate" title={req.pdf_file_hash}>
                          {req.pdf_file_hash || "-"}
                        </td>
                        <td className="px-4 py-3">{getStatusBadge(req.status)}</td>
                        <td className="px-4 py-3 text-red-500 max-w-[200px] truncate" title={req.error_message}>
                          {req.error_message || "-"}
                        </td>
                        <td className="px-4 py-3 text-xs">{formatBackendDate(req.created_at)}</td>
                      </tr>
                    ))
                  ) : (
                    <tr>
                      <td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">
                        暂无生成请求记录
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
