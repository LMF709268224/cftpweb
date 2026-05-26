"use client"

import { useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { ShieldAlert, Search, ShieldCheck, UserX, Clock, AlertTriangle } from "lucide-react"

export default function PermissionsPage() {
  const [candidateId, setCandidateId] = useState("")
  const [credDefId, setCredDefId] = useState("")
  const [reason, setReason] = useState("")
  
  const [checkResult, setCheckResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  const handleCheck = async () => {
    if (!candidateId || !credDefId) return alert("Please provide both Candidate ID and Credential Def ID")
    
    setLoading(true)
    try {
      const res = await apiClient(`/api/permissions/check?candidate_id=${candidateId}&cred_def_id=${credDefId}`)
      setCheckResult(res)
    } catch (e) {
      alert("Check failed")
    } finally {
      setLoading(false)
    }
  }

  const handleAction = async (endpoint: string) => {
    if (!reason) return alert("Please provide a reason for this administrative action")
    
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
      alert("Action successful")
      setReason("")
      handleCheck() // Refresh status
    } catch (e) {
      alert("Action failed")
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
            Candidate Interventions
          </h1>
          <p className="text-muted-foreground mt-2">White-box manual intervention tools for candidate permissions and credential statuses.</p>
        </div>

        <div className="grid gap-6 md:grid-cols-12">
          {/* Query Panel */}
          <div className="md:col-span-5 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Query Target</CardTitle>
                <CardDescription>Enter candidate and qualification IDs to inspect current state.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label>Candidate ID (ULID)</Label>
                  <Input 
                    placeholder="e.g. 01H..." 
                    value={candidateId}
                    onChange={(e) => setCandidateId(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Credential Definition ID (ULID)</Label>
                  <Input 
                    placeholder="e.g. 01H..." 
                    value={credDefId}
                    onChange={(e) => setCredDefId(e.target.value)}
                  />
                </div>
                <Button className="w-full gap-2" onClick={handleCheck} disabled={loading}>
                  <Search className="h-4 w-4" />
                  {loading ? "Inspecting..." : "Inspect Status"}
                </Button>
              </CardContent>
            </Card>

            {checkResult && (
              <Card>
                <CardHeader className="bg-muted/50 pb-4 border-b">
                  <CardTitle className="text-lg">Administrative Action</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4 pt-6">
                  <div className="space-y-2">
                    <Label className="text-red-500 font-semibold">Audit/Intervention Reason (Required)</Label>
                    <Input 
                      placeholder="e.g. Manual verification approved by admin..." 
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      className="border-red-200 focus-visible:ring-red-500"
                    />
                  </div>
                  
                  <div className="grid grid-cols-2 gap-3 pt-2">
                    <Button 
                      variant="outline" 
                      className="gap-2 text-green-600 border-green-200 hover:bg-green-50"
                      onClick={() => handleAction("/api/permissions/grant")}
                    >
                      <ShieldCheck className="h-4 w-4" /> Grant Upload
                    </Button>
                    <Button 
                      variant="outline" 
                      className="gap-2 text-orange-600 border-orange-200 hover:bg-orange-50"
                      onClick={() => handleAction("/api/permissions/revoke")}
                    >
                      <UserX className="h-4 w-4" /> Revoke Upload
                    </Button>
                    <Button 
                      variant="outline" 
                      className="gap-2 text-yellow-600 border-yellow-200 hover:bg-yellow-50"
                      onClick={() => handleAction("/api/permissions/mark-expired")}
                    >
                      <Clock className="h-4 w-4" /> Mark Expired
                    </Button>
                    <Button 
                      variant="outline" 
                      className="gap-2 text-red-600 border-red-200 hover:bg-red-50"
                      onClick={() => handleAction("/api/permissions/revoke-credential")}
                    >
                      <AlertTriangle className="h-4 w-4" /> Revoke Cred
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
                  <CardTitle>Qualification State</CardTitle>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="flex items-center gap-4 p-4 bg-muted rounded-lg border">
                    <div>
                      <p className="text-sm text-muted-foreground">Is Eligible?</p>
                      <p className="text-2xl font-bold mt-1">
                        {checkResult.eligible ? (
                          <span className="text-green-500">YES</span>
                        ) : (
                          <span className="text-red-500">NO</span>
                        )}
                      </p>
                    </div>
                    <div className="h-10 w-px bg-border mx-4"></div>
                    <div>
                      <p className="text-sm text-muted-foreground">Credential Status</p>
                      <Badge className="mt-2" variant="outline">{checkResult.credential_status}</Badge>
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-2">System Message</h3>
                    <div className="p-3 bg-secondary rounded-md text-sm font-mono whitespace-pre-wrap">
                      {checkResult.message || "No specific message returned by gcreds engine."}
                    </div>
                  </div>

                  <div>
                    <h3 className="font-semibold mb-2 text-red-500">Danger Zone Information</h3>
                    <p className="text-sm text-muted-foreground">
                      Manual interventions bypass standard automation (Casdoor enforcer, examination results). 
                      Operations are permanently recorded in the audit logs of the `gcreds` service.
                    </p>
                  </div>
                </CardContent>
              </Card>
            ) : (
              <div className="h-full rounded-xl border border-dashed flex items-center justify-center p-8 text-center bg-muted/20">
                <div>
                  <Search className="h-12 w-12 text-muted-foreground mx-auto mb-4 opacity-50" />
                  <h3 className="text-lg font-medium">No Data Loaded</h3>
                  <p className="text-muted-foreground mt-2 max-w-sm">Enter IDs on the left panel to inspect the exact status of a candidate's qualification lifecycle.</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
