"use client"

import React from "react"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { useTranslation } from "@/lib/useLanguage"
import { ShieldAlert, Search, ShieldCheck, UserX, Clock, AlertTriangle } from "lucide-react"
import { statusBadgeClassForStatusValue } from "@/lib/status-labels"

export default function PermissionsPage() {
  const { t } = useTranslation()
  const [candidateId, setCandidateId] = useState("")
  const [credDefId, setCredDefId] = useState("")
  const [reason, setReason] = useState("")
  const [definitions, setDefinitions] = useState<any[]>([])
  
  const [checkResult, setCheckResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const fetchDefs = async () => {
      try {
        const res = await apiClient("/api/credentials/definitions")
        if (res && res.definitions) {
          setDefinitions(res.definitions)
        }
      } catch (err) {
        console.error(err)
      }
    }
    fetchDefs()
  }, [])

  const handleCheck = async () => {
    if (!candidateId || !credDefId) return alert(t.permissionsPage.alertProvideIds)
    
    setLoading(true)
    try {
      const res = await apiClient(`/api/permissions/check?candidate_id=${candidateId}&cred_def_id=${credDefId}`)
      setCheckResult(res)
    } catch (e) {
      alert(t.common.error)
    } finally {
      setLoading(false)
    }
  }

  const handleAction = async (endpoint: string) => {
    if (!reason) return alert(t.permissionsPage.alertProvideReason)
    
    setLoading(true)
    try {
      await apiClient(endpoint, {
        method: "POST",
        body: JSON.stringify({
          candidate_id: candidateId,
          cred_def_id: credDefId,
          reason: reason
        })
      })
      alert(t.common.success)
      setReason("")
      handleCheck() // Refresh status
    } catch (e) {
      alert(t.common.error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-background flex">
      <Sidebar />
      <main className="flex-1 ml-64 p-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <ShieldAlert className="h-8 w-8 text-primary" />
            {t.permissionsPage.title}
          </h1>
          <p className="text-muted-foreground mt-2">{t.permissionsPage.subtitle}</p>
        </div>

        <div className="grid gap-6 md:grid-cols-12">
          {/* Query Panel */}
          <div className="md:col-span-5 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>{t.permissionsPage.queryTarget}</CardTitle>
                <CardDescription>{t.permissionsPage.queryDesc}</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>{t.permissionsPage.candidateId}</Label>
                  <Input 
                    placeholder="e.g. 01H..." 
                    value={candidateId}
                    onChange={(e) => setCandidateId(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t.permissionsPage.credDefId}</Label>
                  <select
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                    value={credDefId}
                    onChange={(e) => setCredDefId(e.target.value)}
                  >
                    <option value="" disabled>{t.permissionsPage.selectCredentialDef}</option>
                    {definitions.map((def) => (
                      <option key={def.cred_def_id} value={def.cred_def_id}>
                        {def.name}
                      </option>
                    ))}
                  </select>
                </div>
                <Button className="w-full gap-2" onClick={handleCheck} disabled={loading}>
                  <Search className="h-4 w-4" />
                  {loading ? t.permissionsPage.inspecting : t.permissionsPage.inspectStatus}
                </Button>
              </CardContent>
            </Card>

            {checkResult && (
              <Card>
                <CardHeader className="bg-muted/50 pb-4 border-b">
                  <CardTitle className="text-lg">{t.permissionsPage.adminAction}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4 pt-6">
                  <div className="space-y-2">
                    <Label className="text-red-500 font-semibold">{t.permissionsPage.reason}</Label>
                    <Input 
                      placeholder={t.permissionsPage.reasonPlaceholder}
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      className="border-red-200 focus-visible:ring-red-500"
                    />
                  </div>
                  
                  <div className="grid grid-cols-2 gap-3 pt-2">
                    <Button 
                      variant="outline" 
                      className="gap-2 text-green-600 border-green-200 hover:bg-green-50 hover:text-green-700"
                      onClick={() => handleAction("/api/permissions/grant")}
                    >
                      <ShieldCheck className="h-4 w-4" /> {t.permissionsPage.grantUpload}
                    </Button>
                    <Button 
                      variant="outline" 
                      className="gap-2 text-orange-600 border-orange-200 hover:bg-orange-50 hover:text-orange-700"
                      onClick={() => handleAction("/api/permissions/revoke")}
                    >
                      <UserX className="h-4 w-4" /> {t.permissionsPage.revokeUpload}
                    </Button>
                    <Button 
                      variant="outline" 
                      className="gap-2 text-yellow-600 border-yellow-200 hover:bg-yellow-50 hover:text-yellow-700"
                      onClick={() => handleAction("/api/permissions/mark-expired")}
                    >
                      <Clock className="h-4 w-4" /> {t.permissionsPage.markExpired}
                    </Button>
                    <Button 
                      variant="outline" 
                      className="gap-2 text-red-600 border-red-200 hover:bg-red-50 hover:text-red-700"
                      onClick={() => handleAction("/api/permissions/revoke-credential")}
                    >
                      <AlertTriangle className="h-4 w-4" /> {t.permissionsPage.revokeCred}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}
          </div>

          {/* Result Panel */}
          <div className="md:col-span-7">
            {checkResult ? (
              <Card className="h-full">
                <CardHeader>
                  <CardTitle>{t.permissionsPage.qualState}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="flex items-center gap-4 p-4 bg-muted rounded-lg border">
                    <div>
                      <p className="text-sm text-muted-foreground">{t.permissionsPage.isEligible}</p>
                      <p className="text-2xl font-bold mt-1">
                        {checkResult.eligible ? (
                          <span className="text-green-500">{t.permissionsPage.yes}</span>
                        ) : (
                          <span className="text-red-500">{t.permissionsPage.no}</span>
                        )}
                      </p>
                    </div>
                    <div className="h-10 w-px bg-border mx-4"></div>
                    <div>
                      <p className="text-sm text-muted-foreground">{t.permissionsPage.uploadPermission}</p>
                      <p className="text-sm font-medium mt-2 text-orange-500 bg-orange-50 px-2 py-1 rounded inline-block">
                        {t.permissionsPage.uploadPermissionUnknown}
                      </p>
                    </div>
                    <div className="h-10 w-px bg-border mx-4"></div>
                    <div>
                      <p className="text-sm text-muted-foreground">{t.permissionsPage.credStatus}</p>
                      <Badge className={`mt-2 ${statusBadgeClassForStatusValue(checkResult.credential_status)}`} variant="outline">{checkResult.credential_status}</Badge>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-2">{t.permissionsPage.sysMessage}</h3>
                    <div className="p-3 bg-secondary rounded-md text-sm font-mono whitespace-pre-wrap">
                      {checkResult.message || t.permissionsPage.noSysMessage}
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-2 text-red-500">{t.permissionsPage.dangerZone}</h3>
                    <p className="text-sm text-muted-foreground">
                      {t.permissionsPage.dangerDesc}
                    </p>
                  </div>
                </CardContent>
              </Card>
            ) : (
              <div className="h-full rounded-xl border border-dashed flex items-center justify-center p-8 text-center bg-muted/20">
                <div>
                  <Search className="h-12 w-12 text-muted-foreground mx-auto mb-4 opacity-50" />
                  <h3 className="text-lg font-medium">{t.permissionsPage.noData}</h3>
                  <p className="text-muted-foreground mt-2 max-w-sm">{t.permissionsPage.noDataDesc}</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
