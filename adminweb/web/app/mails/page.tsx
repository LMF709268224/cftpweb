"use client"

import React from "react"

import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Badge } from "@/components/ui/badge"
import { Send, List, FileText } from "lucide-react"
import { statusBadgeClassForStatusValue } from "@/lib/status-labels"
import { toast } from "sonner"

function MailDetailPreview({ mailId }: { mailId: string }) {
  const { t } = useTranslation()
  const [detail, setDetail] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let active = true
    setLoading(true)
    apiClient(`/api/mails?mail_id=${mailId}`)
      .then(res => {
        if (active) setDetail(res)
      })
      .catch(console.error)
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => { active = false }
  }, [mailId])

  if (loading) return <div className="mt-2 text-xs text-muted-foreground">{t.common.loading}</div>
  if (!detail) return <div className="mt-2 text-xs text-muted-foreground">{t.common.error}</div>

  const htmlContent = detail.html_body || detail.plain_body || ""

  return (
    <div className="mt-4 border rounded overflow-hidden bg-white">
      <div className="bg-muted px-3 py-2 text-xs font-semibold border-b">
        {t.mailsPage.previewTitle}
      </div>
      <div className="w-full h-[400px]">
        <iframe
          title={`mail-preview-${mailId}`}
          className="w-full h-full border-0"
          srcDoc={htmlContent}
          sandbox="allow-same-origin"
        />
      </div>
    </div>
  )
}

export default function AdminMailsPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<"send" | "sent" | "templates">("send")

  // For users
  const [users, setUsers] = useState<any[]>([])
  const [selectedUsers, setSelectedUsers] = useState<any[]>([])

  const [templateId, setTemplateId] = useState("")
  const [subject, setSubject] = useState("")
  const [payload, setPayload] = useState("{\n}")
  const [isHtml, setIsHtml] = useState(false)
  const [sending, setSending] = useState(false)

  // For listing
  const [sentMessages, setSentMessages] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState("")
  const [totalCount, setTotalCount] = useState(0)
  const [expandedMailId, setExpandedMailId] = useState<string | null>(null)

  const getMailStatusText = (status: string) => {
    const s = String(status).toUpperCase()
    switch (s) {
      case "SCHEDULING":
      case "MAIL_STATUS_SCHEDULING":
        return t.mailsPage.statusScheduling;
      case "SENT":
      case "MAIL_STATUS_SENT":
        return t.mailsPage.statusSent;
      case "FAILED":
      case "MAIL_STATUS_FAILED":
        return t.mailsPage.statusFailed;
      case "CANCELLED":
      case "MAIL_STATUS_CANCELLED":
        return t.mailsPage.statusCancelled;
      default: return t.mailsPage.statusUnknown;
    }
  }


  // For templates
  const [templates, setTemplates] = useState<any[]>([])
  const [loadingTemplates, setLoadingTemplates] = useState(false)
  const [newTplPath, setNewTplPath] = useState("")
  const [newTplName, setNewTplName] = useState("")
  const [newTplSubject, setNewTplSubject] = useState("")
  const [newTplContent, setNewTplContent] = useState("")
  const [newTplDesc, setNewTplDesc] = useState("")
  const [editingTemplateId, setEditingTemplateId] = useState<string | null>(null)
  const [creatingTpl, setCreatingTpl] = useState(false)

  const templatePathOf = (tpl: any) => tpl?.path || tpl?.template_path || tpl?.template_id || ""
  const templateBodyOf = (tpl: any) => tpl?.html_body || tpl?.template_body || tpl?.plain_body || ""

  useEffect(() => {
    fetchUsers()
    fetchTemplates()
  }, [])

  useEffect(() => {
    if (templateId) {
      const fetchTemplateDetail = async () => {
        try {
          const res = await apiClient(`/api/mails/templates/detail?path=${encodeURIComponent(templateId)}`)
          if (res && res.subject_template) {
            const html = res.html_body || res.template_body || res.plain_body || "";
            const matches = [...(res.subject_template.matchAll(/\{\{([^}]+)\}\}/g)), ...(html.matchAll(/\{\{([^}]+)\}\}/g))];
            const vars = Array.from(new Set(matches.map(m => m[1].trim())));
            if (vars.length > 0) {
              const obj: any = {};
              vars.forEach(v => obj[v] = "");
              setPayload(JSON.stringify(obj, null, 2));
            } else {
              setPayload("{\n}");
            }
          }
        } catch (err) {
          console.error("Failed to fetch template detail", err)
        }
      }
      fetchTemplateDetail()
    } else {
      setPayload("{\n}")
    }
  }, [templateId])

  useEffect(() => {
    if (activeTab === "sent") {
      fetchSentMessages()
    }
  }, [activeTab, page, statusFilter])

  const fetchUsers = async () => {
    try {
      const res = await apiClient("/api/user/list")
      if (res && res.users) {
        // Only keep users with email
        setUsers(res.users.filter((u: any) => u.email))
      }
    } catch (err) {
      console.error(err)
    }
  }

  const fetchTemplates = async () => {
    setLoadingTemplates(true)
    try {
      const res = await apiClient("/api/mails/templates")
      if (res && res.templates) {
        setTemplates(res.templates)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoadingTemplates(false)
    }
  }

  const handleCreateTemplate = async () => {
    if (!editingTemplateId && !newTplPath) {
      toast.error(t.mailsPage.alertTemplatePathRequired)
      return
    }
    if (!newTplName || !newTplContent || !newTplSubject) {
      toast.error(t.mailsPage.alertFillFields)
      return
    }
    setCreatingTpl(true)
    try {
      if (editingTemplateId) {
        await apiClient("/api/mails/templates", {
          method: "PUT",
          body: JSON.stringify({
            path: editingTemplateId,
            name: newTplName,
            subject_template: newTplSubject,
            html_body: newTplContent,
            plain_body: newTplContent.replace(/<[^>]+>/g, ""),
            description: newTplDesc,
          })
        })
        toast.success(t.mailsPage.alertUpdateSuccess)
      } else {
        await apiClient("/api/mails/templates", {
          method: "POST",
          body: JSON.stringify({
            path: newTplPath,
            name: newTplName,
            subject_template: newTplSubject,
            html_body: newTplContent,
            plain_body: newTplContent.replace(/<[^>]+>/g, ""),
            description: newTplDesc,
          })
        })
        toast.success(t.mailsPage.alertCreateSuccess)
      }

      setNewTplPath("")
      setNewTplName("")
      setNewTplSubject("")
      setNewTplContent("")
      setNewTplDesc("")
      setEditingTemplateId(null)
      fetchTemplates()
    } catch (err) {
      console.error(err)
    } finally {
      setCreatingTpl(false)
    }
  }

  const handleDeleteTemplate = async (id: string) => {
    if (!confirm(t.mailsPage.confirmDelete)) {
      return
    }
    try {
      await apiClient(`/api/mails/templates?path=${encodeURIComponent(id)}`, {
        method: "DELETE"
      })
      toast.success(t.mailsPage.alertDeleteSuccess)
      fetchTemplates()
    } catch (err: any) {
      // If the backend returns 501 or not implemented
      if (err.message && err.message.includes("not implemented")) {
        toast.error(t.mailsPage.alertDeleteNotSupported)
      } else {
        toast.error(t.mailsPage.alertDeleteNotSupported) // Fallback since grpc has no delete
      }
    }
  }

  const handleEditClick = async (tpl: any) => {
    const path = templatePathOf(tpl)
    setEditingTemplateId(path)
    setNewTplName(tpl.name || "")
    setNewTplSubject(t.common.loading)
    setNewTplContent("")
    setNewTplDesc(tpl.description || "")

    try {
      const res = await apiClient(`/api/mails/templates/detail?path=${encodeURIComponent(path)}`)
      if (res) {
        setNewTplName(res.name || tpl.name || "")
        setNewTplSubject(res.subject_template || "")
        setNewTplContent(res.html_body || res.template_body || res.plain_body || "")
      }
    } catch (err) {
      console.error("Failed to fetch template detail", err)
      toast.error(t.mailsPage.alertLoadTemplateFailed)
      setNewTplSubject("")
    }
  }

  const fetchSentMessages = async () => {
    setLoading(true)
    try {
      let query = `/api/mails/sent?page=${page}&page_size=${pageSize}`
      if (statusFilter !== "") {
        query += `&status=${statusFilter}`
      }
      const res = await apiClient(query)
      if (res) {
        setSentMessages(res.mails || [])
        setTotalCount(res.total || 0)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleSend = async () => {
    if (selectedUsers.length === 0) {
      toast.error(t.mailsPage.alertNoUsers)
      return
    }

    if (!templateId) {
      if (!subject) {
        toast.error(t.mailsPage.alertNoSubject)
        return
      }
      if (!payload || payload.trim() === "{\n}" || payload.trim() === "{}") {
        toast.error(t.mailsPage.alertNoPayload || t.common.error)
        return
      }
    }

    // Validate payload is valid JSON object if not empty and using a template
    if (templateId && payload.trim() !== "") {
      try {
        const parsed = JSON.parse(payload);
        if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
          toast.error(t.mailsPage.alertPayloadObject);
          return;
        }
      } catch (e) {
        toast.error(t.mailsPage.alertPayloadJson.replace("{{error}}", (e as Error).message));
        return;
      }
    }

    setSending(true)
    let successCount = 0;
    try {
      // Since gmail API CreateMail only accepts one ToEmail per request, we loop
      for (const u of selectedUsers) {
        await apiClient("/api/mails/send", {
          method: "POST",
          body: JSON.stringify({
            to_email: u.email,
            to_name: u.name,
            subject: subject,
            template_path: templateId,
            payload: payload,
            html_body: templateId ? "" : payload,
            plain_body: templateId ? "" : payload,
          }),
        })
        successCount++;
      }

      toast.success(t.mailsPage.alertSendSuccess)
      setSelectedUsers([])
      setPayload("{\n}")
      setTemplateId("")
      setSubject("")
    } catch (err) {
      console.error(err)
    } finally {
      setSending(false)
    }
  }

  const isUserSelected = (id: string) => selectedUsers.some(u => u.id === id)

  const toggleUserSelection = (user: any, checked: boolean) => {
    if (checked) {
      setSelectedUsers(prev => [...prev, user])
    } else {
      setSelectedUsers(prev => prev.filter(u => u.id !== user.id))
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8 max-w-5xl mx-auto">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-foreground mb-2">{t.mailsPage.title || t.sidebar.mails}</h1>
            <p className="text-muted-foreground">{t.mailsPage.subtitle}</p>
          </div>

          <div className="flex gap-4 border-b mb-6">
            <button
              onClick={() => setActiveTab("send")}
              className={`pb-3 flex items-center gap-2 font-medium transition-colors ${activeTab === "send" ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground"
                }`}
            >
              <Send className="w-4 h-4" />
              {t.mailsPage.sendTab}
            </button>
            <button
              onClick={() => setActiveTab("sent")}
              className={`pb-3 flex items-center gap-2 font-medium transition-colors ${activeTab === "sent" ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground"
                }`}
            >
              <List className="w-4 h-4" />
              {t.mailsPage.sentTab}
            </button>
            <button
              onClick={() => setActiveTab("templates")}
              className={`pb-3 flex items-center gap-2 font-medium transition-colors ${activeTab === "templates" ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground"
                }`}
            >
              <FileText className="w-4 h-4" />
              {t.mailsPage.templatesTab}
            </button>
          </div>

          {activeTab === "send" && (
            <div className="bg-card rounded-xl border p-6">
              <h2 className="text-lg font-semibold mb-6">{t.mailsPage.createNewMessage}</h2>

              <div className="space-y-4">
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-sm font-medium">{t.mailsPage.selectUsers}</label>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => setSelectedUsers([...users])}
                        className="text-xs text-primary hover:underline"
                      >
                        {t.mailsPage.selectAll}
                      </button>
                      <button
                        type="button"
                        onClick={() => setSelectedUsers([])}
                        className="text-xs text-muted-foreground hover:underline"
                      >
                        {t.mailsPage.clear}
                      </button>
                    </div>
                  </div>
                  <div className="max-h-40 overflow-y-auto border border-input rounded-md p-2 bg-transparent">
                    {users.length === 0 ? (
                      <div className="text-sm text-muted-foreground p-2">{t.mailsPage.loadingUsers}</div>
                    ) : (
                      users.map(u => (
                        <label key={u.id} className="flex items-center space-x-2 p-1 hover:bg-muted/50 rounded cursor-pointer">
                          <input
                            type="checkbox"
                            checked={isUserSelected(u.id)}
                            onChange={(e) => toggleUserSelection(u, e.target.checked)}
                            className="rounded border-gray-300"
                          />
                          <span className="text-sm">{u.name} <span className="text-xs text-muted-foreground ml-1">({u.email})</span></span>
                        </label>
                      ))
                    )}
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {t.mailsPage.selected.replace('{{count}}', String(selectedUsers.length))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">{t.mailsPage.selectTemplate}</label>
                  <select
                    value={templateId}
                    onChange={(e) => {
                      const tId = e.target.value
                      setTemplateId(tId)

                      if (tId) {
                        const tpl = templates.find(t => templatePathOf(t) === tId)
                        if (tpl) {
                          const regex = /{{([^}]+)}}/g
                          const vars = new Set<string>()
                          let match
                          while ((match = regex.exec(tpl.subject_template)) !== null) vars.add(match[1].trim().replace(/^\./, ''))
                          while ((match = regex.exec(templateBodyOf(tpl))) !== null) vars.add(match[1].trim().replace(/^\./, ''))

                          const obj: Record<string, string> = {}
                          vars.forEach(v => obj[v] = "")
                          setPayload(JSON.stringify(obj, null, 2))
                          setSubject("") // Template overrides subject
                        }
                      } else {
                        setPayload("{\n}")
                      }
                    }}
                    className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  >
                    <option value="">{t.mailsPage.selectTemplatePlaceholder}</option>
                    {templates.map((tpl) => {
                      const path = templatePathOf(tpl)
                      return (
                        <option key={path} value={path}>
                          {tpl.name || tpl.subject_template} ({path})
                        </option>
                      )
                    })}
                  </select>
                </div>

                {!templateId && (
                  <div>
                    <label className="block text-sm font-medium mb-1">{t.mailsPage.subject}</label>
                    <input
                      type="text"
                      value={subject}
                      onChange={(e) => setSubject(e.target.value)}
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    />
                  </div>
                )}

                <div>
                  <label className="block text-sm font-medium mb-1">{t.mailsPage.payloadJson}</label>
                  <textarea
                    rows={6}
                    value={payload}
                    onChange={(e) => setPayload(e.target.value)}
                    className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring font-mono"
                  />
                </div>

                {!templateId && (
                  <label className="flex items-center space-x-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={isHtml}
                      onChange={(e) => setIsHtml(e.target.checked)}
                      className="rounded border-gray-300"
                    />
                    <span className="text-sm font-medium">{t.mailsPage.isHtml}</span>
                  </label>
                )}

                <button
                  onClick={handleSend}
                  disabled={sending}
                  className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2"
                >
                  {sending ? t.mailsPage.sending : t.mailsPage.sendMessageBtn}
                </button>
              </div>
            </div>
          )}

          {activeTab === "sent" && (
            <div className="bg-card rounded-xl border p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-lg font-semibold">{t.mailsPage.sentHistory}</h2>
                <div className="flex items-center gap-4">
                  <label className="text-sm font-medium text-muted-foreground">{t.mailsPage.statusFilter}</label>
                  <select
                    value={statusFilter}
                    onChange={(e) => {
                      setPage(1)
                      setStatusFilter(e.target.value)
                    }}
                    className="rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  >
                    <option value="">{t.mailsPage.statusAll}</option>
                    <option value="SCHEDULING">{t.mailsPage.statusScheduling}</option>
                    <option value="SENT">{t.mailsPage.statusSent}</option>
                    <option value="FAILED">{t.mailsPage.statusFailed}</option>
                    <option value="CANCELLED">{t.mailsPage.statusCancelled}</option>
                  </select>
                </div>
              </div>

              {loading ? (
                <div className="text-muted-foreground py-4">{t.mailsPage.loading}</div>
              ) : sentMessages.length > 0 ? (
                <div className="space-y-4">
                  {sentMessages.map((msg, i) => {
                    const isExpanded = expandedMailId === msg.mail_id
                    return (
                      <div key={msg.mail_id || i} className="border-b pb-4 last:border-0">
                        <div
                          className="flex items-center justify-between cursor-pointer hover:bg-muted/50 p-2 rounded -mx-2 transition-colors"
                          onClick={() => setExpandedMailId(isExpanded ? null : msg.mail_id)}
                        >
                          <div className="font-medium text-sm">
                            <span className="text-muted-foreground mr-2">{msg.created_at ? msg.created_at.split('T')[0] : ''}</span>
                            {t.mailsPage.toLabel}: {msg.to_email} {msg.template_path || msg.template_id ? `- ${t.mailsPage.templateShort}: ${msg.template_path || msg.template_id}` : ''}
                            <Badge
                              variant="outline"
                              className={statusBadgeClassForStatusValue(msg.status)}
                            >
                              {t.mailsPage.statusLabel}: {getMailStatusText(msg.status)}
                            </Badge>
                            <div className="text-muted-foreground text-xs">{isExpanded ? "▲" : "▼"}</div>
                          </div>
                        </div>
                        {isExpanded && (
                          <div className="mt-3 bg-muted/30 p-3 rounded text-sm font-mono whitespace-pre-wrap break-words">
                            <div className="mb-2 pb-2 border-b border-muted">
                              <span className="font-semibold">{t.mailsPage.mailId}:</span> {msg.mail_id}
                            </div>
                            <div className="mb-2 pb-2 border-b border-muted">
                              <span className="font-semibold">{t.mailsPage.subjectLabel}:</span> {msg.subject}
                            </div>
                            {msg.from_email && (
                              <div className="mb-2 pb-2 border-b border-muted">
                                <span className="font-semibold">{t.mailsPage.fromLabel}:</span> {msg.from_name} &lt;{msg.from_email}&gt;
                              </div>
                            )}
                            {msg.error_message && (
                              <div className="mt-2 text-red-500">
                                <span className="font-semibold">{t.mailsPage.errorLabel}:</span> {msg.error_message}
                              </div>
                            )}
                            <MailDetailPreview mailId={msg.mail_id} />
                          </div>
                        )}
                      </div>
                    )
                  })}

                  <div className="flex items-center justify-between pt-4 border-t mt-6">
                    <div className="text-sm text-muted-foreground">
                      {t.mailsPage.totalItems.replace('{{total}}', String(totalCount))}
                    </div>
                    <div className="flex gap-2">
                      <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="px-3 py-1 border rounded text-sm disabled:opacity-50 hover:bg-muted"
                      >
                        {t.mailsPage.prevPage}
                      </button>
                      <span className="px-3 py-1 text-sm">{page}</span>
                      <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={page * pageSize >= totalCount}
                        className="px-3 py-1 border rounded text-sm disabled:opacity-50 hover:bg-muted"
                      >
                        {t.mailsPage.nextPage}
                      </button>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  {t.mailsPage.noSentMessages}
                </div>
              )}
            </div>
          )}

          {activeTab === "templates" && (
            <div className="grid gap-6 lg:grid-cols-2">
              <div className="bg-card rounded-xl border p-6">
                <h2 className="text-lg font-semibold mb-6">
                  {editingTemplateId ? t.mailsPage.edit : t.mailsPage.createTemplate}
                </h2>
                <div className="space-y-4">
                  {!editingTemplateId && (
                    <div>
                      <label className="block text-sm font-medium mb-1">{t.mailsPage.templatePath}</label>
                      <input
                        type="text"
                        value={newTplPath}
                        onChange={(e) => setNewTplPath(e.target.value)}
                        placeholder={t.mailsPage.templatePathPlaceholder}
                        className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                      />
                    </div>
                  )}
                  <div>
                    <label className="block text-sm font-medium mb-1">{t.mailsPage.templateName}</label>
                    <input
                      type="text"
                      value={newTplName}
                      onChange={(e) => setNewTplName(e.target.value)}
                      placeholder={t.mailsPage.templateNamePlaceholder}
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">{t.mailsPage.titleTemplate}</label>
                    <input
                      type="text"
                      value={newTplSubject}
                      onChange={(e) => setNewTplSubject(e.target.value)}
                      placeholder={t.mailsPage.titleTemplatePlaceholder}
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">{t.mailsPage.contentTemplate}</label>
                    <textarea
                      rows={4}
                      value={newTplContent}
                      onChange={(e) => setNewTplContent(e.target.value)}
                      placeholder={t.mailsPage.contentTemplatePlaceholder}
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">{t.mailsPage.templateDescription}</label>
                    <input
                      type="text"
                      value={newTplDesc}
                      onChange={(e) => setNewTplDesc(e.target.value)}
                      placeholder={t.mailsPage.templateDescriptionPlaceholder}
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    />
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={handleCreateTemplate}
                      disabled={creatingTpl}
                      className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2"
                    >
                      {creatingTpl ? t.mailsPage.creating : (editingTemplateId ? t.mailsPage.saveChangesBtn : t.mailsPage.createTemplateBtn)}
                    </button>
                    {editingTemplateId && (
                      <button
                        onClick={() => {
                          setEditingTemplateId(null)
                          setNewTplName("")
                          setNewTplSubject("")
                          setNewTplContent("")
                          setNewTplDesc("")
                        }}
                        className="bg-secondary text-secondary-foreground hover:bg-secondary/80 inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2"
                      >
                        {t.mailsPage.cancelEdit}
                      </button>
                    )}
                  </div>
                </div>
              </div>

              <div className="bg-card rounded-xl border p-6">
                <h2 className="text-lg font-semibold mb-6">{t.mailsPage.existingTemplates}</h2>
                {loadingTemplates ? (
                  <div className="text-muted-foreground">{t.mailsPage.loading}</div>
                ) : templates.length > 0 ? (
                  <div className="space-y-4 max-h-[500px] overflow-y-auto pr-2">
                    {templates.map((tpl) => {
                      const path = templatePathOf(tpl)
                      return (
                        <div key={path} className="border-b pb-4 last:border-0 relative">
                          <div className="flex items-center justify-between gap-2">
                            <span className="font-mono text-xs bg-muted px-2 py-1 rounded select-all cursor-pointer" title={t.mailsPage.templatePathCopyTitle} onClick={(e) => {
                              const target = e.target as HTMLElement;
                              const selection = window.getSelection();
                              const range = document.createRange();
                              range.selectNodeContents(target);
                              selection?.removeAllRanges();
                              selection?.addRange(range);
                              document.execCommand('copy');
                              toast.success(t.mailsPage.copied);
                            }}>
                              {path}
                            </span>
                            <div className="flex items-center gap-2">
                              <button
                                onClick={() => handleEditClick(tpl)}
                                className="text-xs text-primary hover:underline"
                              >
                                {t.mailsPage.edit}
                              </button>
                              <button
                                onClick={() => handleDeleteTemplate(path)}
                                className="text-xs text-destructive hover:underline"
                              >
                                {t.mailsPage.delete}
                              </button>
                            </div>
                          </div>
                          <div className="text-sm text-foreground mt-2 font-semibold">{tpl.name || tpl.subject_template}</div>
                          <div className="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">{templateBodyOf(tpl)}</div>
                        </div>
                      )
                    })}
                  </div>
                ) : (
                  <div className="text-center py-8 text-muted-foreground">{t.mailsPage.noTemplates}</div>
                )}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
