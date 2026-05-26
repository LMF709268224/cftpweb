"use client"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Award, FileText, CheckCircle, XCircle, Clock, AlertCircle } from "lucide-react"
import { useTranslation } from "@/lib/useLanguage"

export default function CredentialsPage() {
  const { t } = useTranslation()
  const [definitions, setDefinitions] = useState<any[]>([])
  const [applications, setApplications] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  // Application flow state
  const [selectedDef, setSelectedDef] = useState<any>(null)
  const [isApplyOpen, setIsApplyOpen] = useState(false)
  const [uploadedFiles, setUploadedFiles] = useState<Record<string, {name: string, url: string, ext: string, hash: string, size: number}>>({})
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

  const handleApplyClick = (def: any) => {
    setSelectedDef(def)
    setUploadedFiles({})
    setIsApplyOpen(true)
  }

  const handleFileUpload = async (constraintName: string, file: File) => {
    // 1. Request presigned URL
    const fileExt = file.name.includes('.') ? '.' + file.name.split('.').pop() : ''
    const fileHash = "hash-" + Date.now() // Mock hash for now
    try {
      const res = await apiClient("/api/credentials/upload-url", {
        method: "POST",
        body: JSON.stringify({
          cred_def_id: selectedDef.cred_def_id,
          file_name: file.name,
          file_ext: fileExt,
          file_hash: fileHash,
          content_type: file.type,
          file_usage: "credential_evidence"
        })
      })
      
      // 2. Mocking S3 direct upload for now (in real world, PUT to res.upload_url)
      // await fetch(res.upload_url, { method: "PUT", body: file })
      
      setUploadedFiles(prev => ({
        ...prev,
        [constraintName]: { 
          name: file.name, 
          url: res.file_url,
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
      file_usage: "credential_evidence",
      file_type: selectedDef.file_constraints.find((c:any) => c.name === k)?.type || 1
    }))

    try {
      await apiClient("/api/credentials/applications", {
        method: "POST",
        body: JSON.stringify({
          cred_def_id: selectedDef.cred_def_id,
          files: evidenceFiles
        })
      })
      setIsApplyOpen(false)
      fetchData() // refresh list
    } catch (e) {
      alert(t.credentialsPage.submitFailed)
    } finally {
      setIsSubmitting(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "PENDING": return <Clock className="h-5 w-5 text-yellow-500" />
      case "APPROVED": return <CheckCircle className="h-5 w-5 text-green-500" />
      case "REJECTED": return <XCircle className="h-5 w-5 text-red-500" />
      case "NEEDS_RESUBMIT": return <AlertCircle className="h-5 w-5 text-orange-500" />
      default: return <FileText className="h-5 w-5" />
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case "PENDING": return t.credentialsPage.appStatusPending
      case "APPROVED": return t.credentialsPage.appStatusApproved
      case "REJECTED": return t.credentialsPage.appStatusRejected
      case "NEEDS_RESUBMIT": return t.credentialsPage.appStatusResubmit
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
                <div className="grid gap-4 md:grid-cols-2">
                  {applications.map(app => (
                    <Card key={app.app_id}>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-lg flex justify-between items-center">
                          <span>{app.credential_definition?.name || "Unknown"}</span>
                          <Badge variant="outline" className="flex items-center gap-1">
                            {getStatusIcon(app.status)}
                            <span className="ml-1">{getStatusText(app.status)}</span>
                          </Badge>
                        </CardTitle>
                      </CardHeader>
                      <CardContent>
                        <p className="text-sm text-muted-foreground mb-2">ID: {app.app_id}</p>
                        {app.audit_remark && (
                          <div className="mt-4 p-3 bg-muted rounded-md text-sm border-l-4 border-primary">
                            <strong>{t.credentialsPage.auditRemark}: </strong> {app.audit_remark}
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </section>

            <hr />

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
                    <div className="flex items-center gap-2">
                      <Input 
                        type="file" 
                        onChange={(e) => e.target.files && handleFileUpload(constraint.name, e.target.files[0])}
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
