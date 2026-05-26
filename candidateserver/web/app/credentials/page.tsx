"use client"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Award, FileText, CheckCircle2, Clock, XCircle, AlertCircle, UploadCloud } from "lucide-react"

interface FileConstraint {
  name: string
  type: number
  is_required: boolean
}

interface CredentialDefinition {
  cred_def_id: string
  name: string
  description: string
  category: string
  file_constraints: FileConstraint[]
}

interface Application {
  app_id: string
  cred_def_id: string
  status: string
  audit_remark?: string
  created_at: string
}

const STATUS_MAP: Record<string, { label: string; icon: any; color: string }> = {
  APPLICATION_STATUS_PENDING: { label: "Pending Audit", icon: Clock, color: "text-yellow-500 bg-yellow-500/10" },
  APPLICATION_STATUS_APPROVED: { label: "Approved", icon: CheckCircle2, color: "text-green-500 bg-green-500/10" },
  APPLICATION_STATUS_REJECTED: { label: "Rejected", icon: XCircle, color: "text-red-500 bg-red-500/10" },
  APPLICATION_STATUS_RESUBMIT: { label: "Needs Resubmit", icon: AlertCircle, color: "text-orange-500 bg-orange-500/10" },
}

export default function CredentialsPage() {
  const [definitions, setDefinitions] = useState<CredentialDefinition[]>([])
  const [applications, setApplications] = useState<Application[]>([])
  const [loading, setLoading] = useState(true)

  const [isApplyOpen, setIsApplyOpen] = useState(false)
  const [selectedDef, setSelectedDef] = useState<CredentialDefinition | null>(null)
  
  // State for file uploads during application
  const [uploadedFiles, setUploadedFiles] = useState<Record<string, { hash: string, name: string, type: number, ext: string }>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  const fetchData = async () => {
    setLoading(true)
    try {
      const [defsRes, appsRes] = await Promise.all([
        apiClient("/api/credentials/definitions"),
        apiClient("/api/credentials/applications")
      ])
      if (defsRes?.definitions) setDefinitions(defsRes.definitions)
      if (appsRes?.applications) setApplications(appsRes.applications)
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleOpenApply = (def: CredentialDefinition) => {
    setSelectedDef(def)
    setUploadedFiles({})
    setIsApplyOpen(true)
  }

  const handleMockUpload = async (constraint: FileConstraint, file: File) => {
    if (!selectedDef) return
    // In a real flow, we would upload the file to S3 using the presigned URL
    // Here we just mock it for the BFF test
    const fakeHash = "hash_" + Date.now().toString()
    const ext = file.name.split('.').pop() || "unknown"

    try {
      // Still call RequestUploadUrl to test the BFF integration
      await apiClient("/api/credentials/upload-url", {
        method: "POST",
        body: JSON.stringify({
          cred_def_id: selectedDef.cred_def_id,
          file_hash: fakeHash,
          file_ext: ext,
          content_type: file.type || "application/octet-stream",
          file_usage: constraint.name
        })
      })

      // Store in local state for final submission
      setUploadedFiles(prev => ({
        ...prev,
        [constraint.name]: {
          hash: fakeHash,
          name: file.name,
          type: constraint.type,
          ext: ext
        }
      }))
    } catch (e) {
      alert("Failed to request upload URL")
    }
  }

  const handleSubmitApplication = async () => {
    if (!selectedDef) return
    
    // Check required constraints
    for (const constraint of selectedDef.file_constraints) {
      if (constraint.is_required && !uploadedFiles[constraint.name]) {
        alert(`Please upload required file: ${constraint.name}`)
        return
      }
    }

    setIsSubmitting(true)
    try {
      const filesPayload = Object.keys(uploadedFiles).map(usageName => ({
        file_hash: uploadedFiles[usageName].hash,
        file_name: uploadedFiles[usageName].name,
        file_type: uploadedFiles[usageName].type,
        file_ext: uploadedFiles[usageName].ext,
        file_size: 1024, // Mock size
        file_usage: usageName
      }))

      await apiClient("/api/credentials/apply", {
        method: "POST",
        body: JSON.stringify({
          cred_def_id: selectedDef.cred_def_id,
          files: filesPayload
        })
      })

      setIsApplyOpen(false)
      fetchData() // Refresh list
    } catch (e) {
      alert("Submit failed")
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold tracking-tight text-foreground mb-2">Qualifications & Applications</h1>
          <p className="text-muted-foreground">Apply for new qualifications and track your application status.</p>
        </div>

        {/* My Applications Section */}
        <div className="mb-12">
          <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
            <FileText className="h-5 w-5 text-primary" />
            My Applications
          </h2>
          {loading ? (
            <div className="text-muted-foreground">Loading...</div>
          ) : applications.length === 0 ? (
            <div className="rounded-xl border border-dashed p-8 text-center bg-card">
              <p className="text-muted-foreground">You have no active applications.</p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {applications.map((app) => {
                const def = definitions.find(d => d.cred_def_id === app.cred_def_id)
                const status = STATUS_MAP[app.status] || { label: app.status, icon: Clock, color: "bg-gray-100 text-gray-800" }
                const StatusIcon = status.icon
                
                return (
                  <div key={app.app_id} className="rounded-xl border bg-card p-5 shadow-sm">
                    <div className="flex justify-between items-start mb-3">
                      <div>
                        <h3 className="font-semibold text-lg">{def?.name || app.cred_def_id}</h3>
                        <p className="text-xs text-muted-foreground mt-1">App ID: {app.app_id}</p>
                      </div>
                      <Badge className={`${status.color} border-0 hover:bg-transparent flex items-center gap-1`}>
                        <StatusIcon className="h-3 w-3" />
                        {status.label}
                      </Badge>
                    </div>
                    {app.audit_remark && (
                      <div className="mt-4 p-3 bg-red-50/50 rounded-lg border border-red-100 text-sm">
                        <span className="font-semibold text-red-800">Review Note: </span>
                        <span className="text-red-600">{app.audit_remark}</span>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Available Credentials Section */}
        <div>
          <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
            <Award className="h-5 w-5 text-primary" />
            Available Qualifications
          </h2>
          {loading ? (
            <div className="text-muted-foreground">Loading...</div>
          ) : definitions.length === 0 ? (
            <div className="rounded-xl border border-dashed p-8 text-center bg-card">
              <p className="text-muted-foreground">No qualifications available to apply for at the moment.</p>
            </div>
          ) : (
            <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
              {definitions.map((def) => {
                const hasPendingApp = applications.some(a => a.cred_def_id === def.cred_def_id && (a.status === 'APPLICATION_STATUS_PENDING' || a.status === 'APPLICATION_STATUS_RESUBMIT'))
                
                return (
                  <div key={def.cred_def_id} className="rounded-2xl border bg-card p-6 shadow-sm hover:shadow-md transition-shadow">
                    <div className="mb-4">
                      <Badge variant="outline" className="mb-3">{def.category}</Badge>
                      <h3 className="text-xl font-bold mb-2">{def.name}</h3>
                      <p className="text-sm text-muted-foreground line-clamp-2 min-h-[40px]">{def.description}</p>
                    </div>
                    
                    <div className="mb-6 rounded-lg bg-muted/50 p-3">
                      <p className="text-xs font-semibold mb-2">Required Materials:</p>
                      <ul className="text-xs space-y-1">
                        {def.file_constraints?.length > 0 ? (
                          def.file_constraints.map((fc, i) => (
                            <li key={i} className="flex justify-between items-center text-muted-foreground">
                              <span>• {fc.name}</span>
                              {fc.is_required && <span className="text-red-500">*</span>}
                            </li>
                          ))
                        ) : (
                          <li className="text-muted-foreground">No files required</li>
                        )}
                      </ul>
                    </div>
                    
                    <Button 
                      className="w-full" 
                      onClick={() => handleOpenApply(def)}
                      disabled={hasPendingApp}
                    >
                      {hasPendingApp ? "Application in Progress" : "Apply Now"}
                    </Button>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Application Dialog */}
        <Dialog open={isApplyOpen} onOpenChange={setIsApplyOpen}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>Apply for {selectedDef?.name}</DialogTitle>
            </DialogHeader>
            <div className="py-4">
              <p className="text-sm text-muted-foreground mb-6">
                Please upload the required documents to submit your application.
              </p>
              
              <div className="space-y-4">
                {selectedDef?.file_constraints?.map((constraint, i) => (
                  <div key={i} className="space-y-2">
                    <Label className="flex justify-between">
                      <span>{constraint.name} {constraint.is_required && <span className="text-red-500">*</span>}</span>
                      {uploadedFiles[constraint.name] && <span className="text-xs text-green-500 flex items-center gap-1"><CheckCircle2 className="h-3 w-3"/> Uploaded</span>}
                    </Label>
                    <div className="flex items-center gap-2">
                      <Input 
                        type="file" 
                        onChange={(e) => {
                          if (e.target.files && e.target.files[0]) {
                            handleMockUpload(constraint, e.target.files[0])
                          }
                        }}
                      />
                    </div>
                  </div>
                ))}
                
                {(!selectedDef?.file_constraints || selectedDef.file_constraints.length === 0) && (
                  <p className="text-sm text-center py-4 bg-muted rounded-md">No files required. You can submit directly.</p>
                )}
              </div>
            </div>
            
            <div className="flex justify-end gap-3 mt-4 border-t pt-4">
              <Button variant="outline" onClick={() => setIsApplyOpen(false)}>Cancel</Button>
              <Button onClick={handleSubmitApplication} disabled={isSubmitting}>
                {isSubmitting ? "Submitting..." : "Submit Application"}
              </Button>
            </div>
          </DialogContent>
        </Dialog>

      </main>
    </div>
  )
}
