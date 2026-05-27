"use client"

import React from "react"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Sidebar } from "@/components/sidebar"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"

interface FileInfo {
  file_name: string
  view_url?: string
}

interface Application {
  app_id: string
  candidate_id: string
  cred_def_id: string
  status: string
  files: FileInfo[]
  audit_remark: string
  created_at: string
}

const getStatusMap = (t: any) => ({
  PENDING: { label: t.applicationsPage.statusPending, color: "bg-yellow-500/10 text-yellow-500" },
  APPROVED: { label: t.applicationsPage.statusApproved, color: "bg-green-500/10 text-green-500" },
  REJECTED: { label: t.applicationsPage.statusRejected, color: "bg-red-500/10 text-red-500" },
  RESUBMIT: { label: t.applicationsPage.statusResubmit, color: "bg-orange-500/10 text-orange-500" },
  REUPLOAD: { label: t.applicationsPage.statusResubmit, color: "bg-orange-500/10 text-orange-500" },
  APPLICATION_STATUS_PENDING: { label: t.applicationsPage.statusPending, color: "bg-yellow-500/10 text-yellow-500" },
  APPLICATION_STATUS_APPROVED: { label: t.applicationsPage.statusApproved, color: "bg-green-500/10 text-green-500" },
  APPLICATION_STATUS_REJECTED: { label: t.applicationsPage.statusRejected, color: "bg-red-500/10 text-red-500" },
  APPLICATION_STATUS_RESUBMIT: { label: t.applicationsPage.statusResubmit, color: "bg-orange-500/10 text-orange-500" },
  APPLICATION_STATUS_REUPLOAD: { label: t.applicationsPage.statusResubmit, color: "bg-orange-500/10 text-orange-500" },
})

export default function ApplicationsPage() {
  const { t } = useTranslation()
  const [applications, setApplications] = useState<Application[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState("0")

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [selectedApp, setSelectedApp] = useState<Application | null>(null)
  const [auditRemark, setAuditRemark] = useState("")

  const statusMap = getStatusMap(t)

  const fetchApplications = async () => {
    try {
      setLoading(true)
      const res = await apiClient(`/api/applications?page_number=${page}&page_size=20&status=${statusFilter}`)
      if (res && res.applications) {
        setApplications(res.applications)
        setTotal(res.total || 0)
      } else {
        setApplications([])
        setTotal(0)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchApplications()
  }, [page, statusFilter])

  const handleOpenAudit = (app: Application) => {
    setSelectedApp(app)
    setAuditRemark("")
    setIsDialogOpen(true)
  }

  const handleAudit = async (action: "approve" | "reject" | "resubmit") => {
    if (!selectedApp) return

    if ((action === "reject" || action === "resubmit") && !auditRemark.trim()) {
      toast.error(t.applicationsPage.auditRemarkRequired)
      return
    }

    let approved = false
    let requireResubmit = false
    if (action === "approve") approved = true
    if (action === "resubmit") requireResubmit = true

    try {
      await apiClient("/api/applications/audit", {
        method: "POST",
        body: JSON.stringify({
          application_id: selectedApp.app_id,
          approved,
          reject_reason: auditRemark,
          require_resubmit: requireResubmit,
        }),
      })
      setIsDialogOpen(false)
      fetchApplications()
    } catch (err) {
      console.error("Audit failed", err)
    }
  }

  const getStatusDisplay = (status: string) => {
    const upperStatus = String(status).toUpperCase()
    const s = statusMap[upperStatus as keyof typeof statusMap]
    if (s) {
      return <Badge className={`hover:bg-transparent ${s.color}`} variant="outline">{s.label}</Badge>
    }
    return <Badge variant="outline">{status}</Badge>
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-foreground mb-2">{t.applicationsPage.title}</h1>
          <p className="text-muted-foreground">{t.applicationsPage.subtitle}</p>
        </div>
      </div>

      <div className="flex items-center gap-4 mb-4">
        <select 
          className="flex h-9 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm"
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value)
            setPage(1)
          }}
        >
          <option value="0">{t.applicationsPage.statusAll}</option>
          <option value="1">{t.applicationsPage.statusPending}</option>
          <option value="2">{t.applicationsPage.statusApproved}</option>
          <option value="3">{t.applicationsPage.statusRejected}</option>
          <option value="4">{t.applicationsPage.statusResubmit}</option>
        </select>
        <div className="text-sm text-muted-foreground">
          {t.applicationsPage.total.replace('{{total}}', total.toString())}
        </div>
      </div>

      <div className="rounded-md border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t.applicationsPage.appId}</TableHead>
              <TableHead>{t.applicationsPage.candidate}</TableHead>
              <TableHead>{t.applicationsPage.credential}</TableHead>
              <TableHead>{t.applicationsPage.status}</TableHead>
              <TableHead>{t.applicationsPage.createdAt}</TableHead>
              <TableHead className="text-right">{t.applicationsPage.action}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8">{t.common.loading}</TableCell>
              </TableRow>
            ) : applications.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  {t.applicationsPage.noApplications}
                </TableCell>
              </TableRow>
            ) : (
              applications.map((app) => (
                <TableRow key={app.app_id}>
                  <TableCell className="font-medium text-xs">{app.app_id}</TableCell>
                  <TableCell className="text-xs">{app.candidate_id}</TableCell>
                  <TableCell className="text-xs">{app.cred_def_id}</TableCell>
                  <TableCell>{getStatusDisplay(app.status)}</TableCell>
                  <TableCell className="text-xs">{app.created_at || '-'}</TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => handleOpenAudit(app)}>
                      {t.applicationsPage.review}
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination Controls */}
      <div className="flex items-center justify-end space-x-2 py-4">
        <Button
          variant="outline"
          size="sm"
          onClick={() => setPage(p => Math.max(1, p - 1))}
          disabled={page === 1}
        >
          {t.applicationsPage.prevPage}
        </Button>
        <div className="text-sm">{t.applicationsPage.page.replace('{{page}}', page.toString())}</div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setPage(p => p + 1)}
          disabled={applications.length < 20}
        >
          {t.applicationsPage.nextPage}
        </Button>
      </div>

      {/* Audit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t.applicationsPage.auditTitle}</DialogTitle>
          </DialogHeader>
          {selectedApp && (
            <div className="grid gap-4 py-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div><span className="text-muted-foreground">{t.applicationsPage.appId}:</span> {selectedApp.app_id}</div>
                <div><span className="text-muted-foreground">{t.applicationsPage.status}:</span> {getStatusDisplay(selectedApp.status)}</div>
                <div><span className="text-muted-foreground">{t.applicationsPage.candidate}:</span> {selectedApp.candidate_id}</div>
                <div><span className="text-muted-foreground">{t.applicationsPage.credential}:</span> {selectedApp.cred_def_id}</div>
              </div>

              <div className="mt-4">
                <Label className="mb-2 block">{t.applicationsPage.uploadedFiles}</Label>
                <ul className="space-y-2">
                  {selectedApp.files && selectedApp.files.length > 0 ? (
                    selectedApp.files.map((file, i) => (
                      <li key={i} className="flex items-center justify-between p-2 border rounded-md">
                        <span className="text-sm">{file.file_name}</span>
                        {file.view_url ? (
                          <a href={file.view_url} target="_blank" rel="noreferrer" className="text-blue-500 hover:underline text-sm">
                            {t.applicationsPage.viewFile}
                          </a>
                        ) : (
                          <span className="text-xs text-muted-foreground">{t.applicationsPage.noUrl}</span>
                        )}
                      </li>
                    ))
                  ) : (
                    <li className="text-sm text-muted-foreground">{t.applicationsPage.noFiles}</li>
                  )}
                </ul>
              </div>

              {(String(selectedApp.status).toUpperCase() === "APPLICATION_STATUS_PENDING" || String(selectedApp.status).toUpperCase() === "PENDING" || String(selectedApp.status) === "1") && (
                <div className="grid gap-2 mt-4">
                  <Label>{t.applicationsPage.auditRemark}</Label>
                  <Input 
                    placeholder={t.applicationsPage.remarkPlaceholder} 
                    value={auditRemark}
                    onChange={e => setAuditRemark(e.target.value)}
                  />
                  <div className="flex justify-end gap-2 mt-4">
                    <Button variant="outline" onClick={() => handleAudit("reject")} className="text-red-500 border-red-200 hover:bg-red-50">
                      {t.applicationsPage.reject}
                    </Button>
                    <Button variant="outline" onClick={() => handleAudit("resubmit")} className="text-orange-500 border-orange-200 hover:bg-orange-50">
                      {t.applicationsPage.requireResubmit}
                    </Button>
                    <Button onClick={() => handleAudit("approve")} className="bg-green-600 hover:bg-green-700">
                      {t.applicationsPage.approve}
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
      </main>
    </div>
  )
}
