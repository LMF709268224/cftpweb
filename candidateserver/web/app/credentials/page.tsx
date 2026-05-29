"use client"

import React from "react"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { formatBackendDate } from "@/lib/utils"
import { Award, FileText, CheckCircle, XCircle, Clock, AlertCircle } from "lucide-react"
import { useTranslation } from "@/lib/useLanguage"

async function sha256Hex(file: File) {
  const buffer = await file.arrayBuffer()
  const hash = await crypto.subtle.digest("SHA-256", buffer)
  return Array.from(new Uint8Array(hash)).map((byte) => byte.toString(16).padStart(2, "0")).join("")
}

function uploadHeaders(signedHeaders?: Record<string, string>) {
  return new Headers(signedHeaders || {})
}

export default function CredentialsPage() {
  const { t } = useTranslation()
  const [definitions, setDefinitions] = useState<any[]>([])
  const [applications, setApplications] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  // Application flow state
  const [selectedDef, setSelectedDef] = useState<any>(null)
  const [resubmitAppId, setResubmitAppId] = useState<string>("")
  const [isApplyOpen, setIsApplyOpen] = useState(false)
  const [uploadedFiles, setUploadedFiles] = useState<Record<string, { name: string, url: string, ext: string, hash: string, size: number }>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [defsRes, appsRes] = await Promise.all([
        apiClient("/api/credentials/definitions"),
        apiClient("/api/credentials/applications")
      ])

      setDefinitions(defsRes?.definitions || [])
      setApplications(appsRes?.applications || [])
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleApplyClick = (def: any, appId?: string) => {
    setResubmitAppId(appId || "")
    setSelectedDef(def)
    setUploadedFiles({})
    setIsApplyOpen(true)
  }

  const handleFileUpload = async (constraintName: string, file: File) => {
    // 1. Request presigned URL
    const fileExt = file.name.includes('.') ? '.' + file.name.split('.').pop() : ''
    try {
      const fileHash = await sha256Hex(file)
      const contentType = file.type || "application/octet-stream"
      const res = await apiClient("/api/credentials/upload-url", {
        method: "POST",
        body: JSON.stringify({
          cred_def_id: selectedDef.cred_def_id,
          file_name: file.name,
          file_ext: fileExt,
          file_hash: fileHash,
          content_type: contentType,
          file_usage: constraintName
        })
      })

      // 2. Upload file directly to S3 using the presigned URL
      const uploadRes = await fetch(res.upload_url, {
        method: "PUT",
        headers: uploadHeaders(res.signed_headers),
        body: file
      })

      if (!uploadRes.ok) {
        throw new Error("S3 upload failed")
      }

      setUploadedFiles(prev => ({
        ...prev,
        [constraintName]: {
          name: file.name,
          url: res.file_key,
          ext: fileExt,
          hash: fileHash,
          size: file.size
        }
      }))
    } catch (e) {
      alert(t.credentialsPage.uploadFailed)
    }
  }

  const handleSubmitApplication = async () => {
    setIsSubmitting(true)

    const evidenceFiles = Object.keys(uploadedFiles).map(k => ({
      file_name: uploadedFiles[k].name,
      file_url: uploadedFiles[k].url,
      file_hash: uploadedFiles[k].hash,
      file_ext: uploadedFiles[k].ext,
      file_size: uploadedFiles[k].size,
      file_usage: k,
      file_type: selectedDef.file_constraints.find((c: any) => c.name === k)?.type || 1
    }))

    try {
      if (resubmitAppId) {
        await apiClient("/api/credentials/update", {
          method: "PUT",
          body: JSON.stringify({
            app_id: resubmitAppId,
            files: evidenceFiles
          })
        })
      } else {
        await apiClient("/api/credentials/submit", {
          method: "POST",
          body: JSON.stringify({
            cred_def_id: selectedDef.cred_def_id,
            files: evidenceFiles
          })
        })
      }
      setIsApplyOpen(false)
      fetchData() // refresh list
    } catch (e) {
      alert(t.credentialsPage.submitFailed)
    } finally {
      setIsSubmitting(false)
    }
  }

  const getStatusIcon = (status: string) => {
    const s = String(status).toUpperCase()
    switch (s) {
      case "PENDING":
      case "APPLICATION_STATUS_PENDING":
        return <Clock className="h-5 w-5 text-yellow-500" />
      case "APPROVED":
      case "APPLICATION_STATUS_APPROVED":
        return <CheckCircle className="h-5 w-5 text-green-500" />
      case "REJECTED":
      case "APPLICATION_STATUS_REJECTED":
        return <XCircle className="h-5 w-5 text-red-500" />
      case "NEEDS_RESUBMIT":
      case "RESUBMIT":
      case "REUPLOAD":
      case "APPLICATION_STATUS_RESUBMIT":
      case "APPLICATION_STATUS_REUPLOAD":
        return <AlertCircle className="h-5 w-5 text-orange-500" />
      default: return <FileText className="h-5 w-5" />
    }
  }

  const getStatusText = (status: string) => {
    const s = String(status).toUpperCase()
    switch (s) {
      case "PENDING":
      case "APPLICATION_STATUS_PENDING":
        return t.credentialsPage.appStatusPending
      case "APPROVED":
      case "APPLICATION_STATUS_APPROVED":
        return t.credentialsPage.appStatusApproved
      case "REJECTED":
      case "APPLICATION_STATUS_REJECTED":
        return t.credentialsPage.appStatusRejected
      case "NEEDS_RESUBMIT":
      case "RESUBMIT":
      case "REUPLOAD":
      case "APPLICATION_STATUS_RESUBMIT":
      case "APPLICATION_STATUS_REUPLOAD":
        return t.credentialsPage.appStatusResubmit
      default: return t.credentialsPage.appStatusUnknown
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Award className="h-8 w-8 text-primary" />
            {t.credentialsPage.title}
          </h1>
          <p className="text-muted-foreground mt-2">{t.credentialsPage.subtitle}</p>
        </div>

        {loading ? (
          <div>{t.common.loading}</div>
        ) : (
          <div className="space-y-10">
            {/* Available Credentials Section */}
            <section>
              <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
                <Award className="h-5 w-5" /> {t.credentialsPage.availableQualifications}
              </h2>
              <div className="grid gap-6 md:grid-cols-3">
                {definitions.map(def => (
                  <Card key={def.cred_def_id} className="flex flex-col">
                    <CardHeader>
                      <CardTitle className="text-xl">{def.name}</CardTitle>
                      <Badge className="w-fit" variant="secondary">{def.category}</Badge>
                    </CardHeader>
                    <CardContent className="flex-1 flex flex-col">
                      <p className="text-sm text-muted-foreground mb-4 flex-1">
                        {def.description}
                      </p>
                      <Button onClick={() => handleApplyClick(def)} className="w-full mt-4">
                        {t.credentialsPage.applyNow}
                      </Button>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </section>

            <hr />

            {/* My Applications Section */}
            <section>
              <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
                <FileText className="h-5 w-5" /> {t.credentialsPage.myApplications}
              </h2>
              {applications.length === 0 ? (
                <div className="p-8 text-center border border-dashed rounded-lg text-muted-foreground">
                  {t.credentialsPage.noApplications}
                </div>
              ) : (
                <div className="overflow-hidden rounded-md border bg-card">
                  <div className="divide-y">
                    {applications.map(app => {
                      const def = definitions.find(d => d.cred_def_id === app.cred_def_id)
                      const canResubmit = ["REUPLOAD", "RESUBMIT", "NEEDS_RESUBMIT", "APPLICATION_STATUS_REUPLOAD", "APPLICATION_STATUS_RESUBMIT"].includes(String(app.status).toUpperCase())

                      return (
                        <div key={app.app_id} className="grid grid-cols-[minmax(180px,1.5fr)_minmax(220px,2fr)_minmax(120px,1fr)_auto] items-center gap-4 px-4 py-3 text-sm">
                          <div className="min-w-0">
                            <div className="truncate font-medium text-foreground">{def?.name || t.common.unknown}</div>
                            <div className="truncate text-xs text-muted-foreground">{app.app_id}</div>
                          </div>
                          <div className="min-w-0 truncate text-muted-foreground">
                            {app.audit_remark ? `${t.credentialsPage.auditRemark}: ${app.audit_remark}` : t.common.na}
                          </div>
                          <Badge variant="outline" className="w-fit gap-1">
                            {getStatusIcon(app.status)}
                            <span>{getStatusText(app.status)}</span>
                          </Badge>
                          {canResubmit ? (
                            <Button
                              size="sm"
                              onClick={() => {
                                if (def) handleApplyClick(def, app.app_id)
                              }}
                            >
                              {t.credentialsPage.appStatusResubmit}
                            </Button>
                          ) : (
                            <span className="text-xs text-muted-foreground">{formatBackendDate(app.created_at).split(" ")[0] || t.common.na}</span>
                          )}
                        </div>
                      )
                    })}
                  </div>
                </div>
              )}
            </section>
          </div>
        )}

        {/* Application Dialog */}
        <Dialog open={isApplyOpen} onOpenChange={setIsApplyOpen}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>{selectedDef?.name}</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <p className="text-sm text-muted-foreground">{t.credentialsPage.description}: {selectedDef?.description}</p>

              <div className="space-y-4 border-t pt-4">
                <h4 className="font-semibold text-sm">{t.credentialsPage.uploadMaterials}</h4>
                {selectedDef?.file_constraints?.map((constraint: any) => (
                  <div key={constraint.name} className="space-y-2 p-3 bg-muted rounded-lg">
                    <div className="flex justify-between">
                      <Label className="font-medium">{constraint.name}</Label>
                      {constraint.is_required ?
                        <Badge variant="destructive">{t.credentialsPage.required}</Badge> :
                        <Badge variant="secondary">{t.credentialsPage.optional}</Badge>}
                    </div>
                    <div className="flex items-center gap-2 mt-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => document.getElementById(`file-${constraint.name}`)?.click()}
                      >
                        {t.credentialsPage.chooseFile}
                      </Button>
                      <span className="text-sm text-muted-foreground truncate max-w-[200px]" title={uploadedFiles[constraint.name] ? uploadedFiles[constraint.name].name : t.credentialsPage.noFileChosen}>
                        {uploadedFiles[constraint.name] ? uploadedFiles[constraint.name].name : t.credentialsPage.noFileChosen}
                      </span>
                      <Input
                        id={`file-${constraint.name}`}
                        type="file"
                        className="hidden"
                        onChange={(e) => {
                          if (e.target.files && e.target.files[0]) {
                            handleFileUpload(constraint.name, e.target.files[0])
                          }
                        }}
                      />
                    </div>
                    {uploadedFiles[constraint.name] && (
                      <p className="text-xs text-green-600 flex items-center gap-1">
                        <CheckCircle className="h-3 w-3" /> {uploadedFiles[constraint.name].name} uploaded
                      </p>
                    )}
                  </div>
                ))}
              </div>

            </div>
            <div className="flex justify-end gap-3">
              <Button variant="outline" onClick={() => setIsApplyOpen(false)}>{t.common.cancel}</Button>
              <Button onClick={handleSubmitApplication} disabled={isSubmitting}>
                {isSubmitting ? t.credentialsPage.submitting : t.credentialsPage.submitApplication}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </main>
    </div>
  )
}
