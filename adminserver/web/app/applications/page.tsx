"use client"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"

interface FileInfo {
  file_name: string
  view_url?: string
}

interface Application {
  app_id: string
  candidate_id: string
  cred_def_id: string
  status: string // "PENDING", "APPROVED", "REJECTED", "RESUBMIT"
  files: FileInfo[]
  audit_remark: string
  created_at: string
}

const STATUS_MAP: Record<string, { label: string; color: string }> = {
  PENDING: { label: "Pending", color: "bg-yellow-500/10 text-yellow-500" },
  APPROVED: { label: "Approved", color: "bg-green-500/10 text-green-500" },
  REJECTED: { label: "Rejected", color: "bg-red-500/10 text-red-500" },
  RESUBMIT: { label: "Resubmit", color: "bg-orange-500/10 text-orange-500" },
  APPLICATION_STATUS_PENDING: { label: "Pending", color: "bg-yellow-500/10 text-yellow-500" },
  APPLICATION_STATUS_APPROVED: { label: "Approved", color: "bg-green-500/10 text-green-500" },
  APPLICATION_STATUS_REJECTED: { label: "Rejected", color: "bg-red-500/10 text-red-500" },
  APPLICATION_STATUS_RESUBMIT: { label: "Resubmit", color: "bg-orange-500/10 text-orange-500" },
}

export default function ApplicationsPage() {
  const { t } = useTranslation()
  const [applications, setApplications] = useState<Application[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  // Filters
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState("0")

  // Audit Dialog State
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [selectedApp, setSelectedApp] = useState<Application | null>(null)
  const [auditRemark, setAuditRemark] = useState("")

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
    const s = STATUS_MAP[status]
    if (s) {
      return <Badge className={`hover:bg-transparent ${s.color}`} variant="outline">{s.label}</Badge>
    }
    return <Badge variant="outline">{status}</Badge>
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-foreground mb-2">{t.sidebar?.applications || "Audit Center"}</h1>
          <p className="text-muted-foreground">Review and audit candidate credential applications.</p>
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
          <option value="0">All</option>
          <option value="1">Pending</option>
          <option value="2">Approved</option>
          <option value="3">Rejected</option>
          <option value="4">Resubmit Required</option>
        </select>
        <div className="text-sm text-muted-foreground">
          Total: {total}
        </div>
      </div>

      <div className="rounded-md border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>App ID</TableHead>
              <TableHead>Candidate</TableHead>
              <TableHead>Credential</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created At</TableHead>
              <TableHead className="text-right">Action</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8">Loading...</TableCell>
              </TableRow>
            ) : applications.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                  No applications found.
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
                      Review
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
          Previous
        </Button>
        <div className="text-sm">Page {page}</div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => setPage(p => p + 1)}
          disabled={applications.length < 20}
        >
          Next
        </Button>
      </div>

      {/* Audit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Audit Application</DialogTitle>
          </DialogHeader>
          {selectedApp && (
            <div className="grid gap-4 py-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div><span className="text-muted-foreground">App ID:</span> {selectedApp.app_id}</div>
                <div><span className="text-muted-foreground">Status:</span> {getStatusDisplay(selectedApp.status)}</div>
                <div><span className="text-muted-foreground">Candidate:</span> {selectedApp.candidate_id}</div>
                <div><span className="text-muted-foreground">Credential:</span> {selectedApp.cred_def_id}</div>
              </div>

              <div className="mt-4">
                <Label className="mb-2 block">Uploaded Files:</Label>
                <ul className="space-y-2">
                  {selectedApp.files && selectedApp.files.length > 0 ? (
                    selectedApp.files.map((file, i) => (
                      <li key={i} className="flex items-center justify-between p-2 border rounded-md">
                        <span className="text-sm">{file.file_name}</span>
                        {file.view_url ? (
                          <a href={file.view_url} target="_blank" rel="noreferrer" className="text-blue-500 hover:underline text-sm">
                            View File
                          </a>
                        ) : (
                          <span className="text-xs text-muted-foreground">No URL</span>
                        )}
                      </li>
                    ))
                  ) : (
                    <li className="text-sm text-muted-foreground">No files uploaded.</li>
                  )}
                </ul>
              </div>

              {(selectedApp.status === "APPLICATION_STATUS_PENDING" || selectedApp.status === "PENDING") && (
                <div className="grid gap-2 mt-4">
                  <Label>Audit Remark / Reject Reason</Label>
                  <Input 
                    placeholder="Provide a reason if rejecting or requesting resubmission..." 
                    value={auditRemark}
                    onChange={e => setAuditRemark(e.target.value)}
                  />
                  <div className="flex justify-end gap-2 mt-4">
                    <Button variant="outline" onClick={() => handleAudit("reject")} className="text-red-500 border-red-200 hover:bg-red-50">
                      Reject
                    </Button>
                    <Button variant="outline" onClick={() => handleAudit("resubmit")} className="text-orange-500 border-orange-200 hover:bg-orange-50">
                      Require Resubmit
                    </Button>
                    <Button onClick={() => handleAudit("approve")} className="bg-green-600 hover:bg-green-700">
                      Approve
                    </Button>
                  </div>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
