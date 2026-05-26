"use client"

import React from "react"
import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Send, List, FileText } from "lucide-react"

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
    switch (status) {
      case "SCHEDULING": return t.mailsPage.statusScheduling;
      case "SENT": return t.mailsPage.statusSent;
      case "FAILED": return t.mailsPage.statusFailed;
      case "CANCELLED": return t.mailsPage.statusCancelled;
      default: return status || t.mailsPage.statusScheduling;
    }
  }


  // For templates
  const [templates, setTemplates] = useState<any[]>([])
  const [loadingTemplates, setLoadingTemplates] = useState(false)
  const [newTplName, setNewTplName] = useState("")
  const [newTplSubject, setNewTplSubject] = useState("")
  const [newTplContent, setNewTplContent] = useState("")
  const [newTplDesc, setNewTplDesc] = useState("")
  const [editingTemplateId, setEditingTemplateId] = useState<string | null>(null)
  const [creatingTpl, setCreatingTpl] = useState(false)

  useEffect(() => {
    fetchTemplates()
    fetchUsers()
  }, [])

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
    if (!newTplName || !newTplContent || !newTplSubject) {
      alert(t.mailsPage.alertFillFields)
      return
    }
    setCreatingTpl(true)
    try {
      if (editingTemplateId) {
        await apiClient("/api/mails/templates", {
          method: "PUT",
          body: JSON.stringify({
            template_id: editingTemplateId,
            name: newTplName,
            subject_template: newTplSubject,
            template_body: newTplContent,
            is_html: true,
            description: newTplDesc,
          })
        })
        alert(t.mailsPage.alertUpdateSuccess)
      } else {
        await apiClient("/api/mails/templates", {
          method: "POST",
          body: JSON.stringify({
            name: newTplName,
            subject_template: newTplSubject,
            template_body: newTplContent,
            is_html: true,
            description: newTplDesc,
          })
        })
        alert(t.mailsPage.alertCreateSuccess)
      }
      
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
      await apiClient(`/api/mails/templates?template_id=${id}`, {
        method: "DELETE"
      })
      alert(t.mailsPage.alertDeleteSuccess)
      fetchTemplates()
    } catch (err: any) {
      // If the backend returns 501 or not implemented
      if (err.message && err.message.includes("not implemented")) {
        alert(t.mailsPage.alertDeleteNotSupported)
      } else {
        alert(t.mailsPage.alertDeleteNotSupported) // Fallback since grpc has no delete
      }
    }
  }

  const handleEditClick = (tpl: any) => {
    setEditingTemplateId(tpl.template_id)
    setNewTplName(tpl.name || "")
    setNewTplSubject(tpl.subject_template || "")
    setNewTplContent(tpl.template_body || "")
    setNewTplDesc(tpl.description || "")
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
      alert(t.mailsPage.alertNoUsers)
      return
    }

    if (!templateId) {
      if (!subject) {
        alert(t.mailsPage.alertNoSubject)
        return
      }
      if (!payload || payload.trim() === "{\n}" || payload.trim() === "{}") {
        alert(t.mailsPage.alertNoPayload || t.common.error)
        return
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
            template_id: templateId,
            payload: payload,
            is_html: isHtml,
          }),
        })
        successCount++;
      }

      alert(t.mailsPage.alertSendSuccess)
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
                        const tpl = templates.find(t => t.template_id === tId)
                        if (tpl) {
                          const regex = /{{([^}]+)}}/g
                          const vars = new Set<string>()
                          let match
                          while ((match = regex.exec(tpl.subject_template)) !== null) vars.add(match[1].trim().replace(/^\./, ''))
                          while ((match = regex.exec(tpl.template_body)) !== null) vars.add(match[1].trim().replace(/^\./, ''))

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
                    {templates.map(t => (
                      <option key={t.template_id} value={t.template_id}>
                        {t.name || t.subject_template} ({t.template_id})
                      </option>
                    ))}
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
                            To: {msg.to_email} {msg.template_id ? `- TPL: ${msg.template_id}` : ''}
                          </div>
                          <div className="flex items-center gap-3">
                            <div className={`text-xs px-2 py-1 rounded border ${
                              msg.status === "SENT" 
                              ? 'bg-green-100 text-green-700 border-green-200 dark:bg-green-900/30 dark:text-green-400' 
                              : msg.status === "FAILED"
                              ? 'bg-red-100 text-red-700 border-red-200 dark:bg-red-900/30 dark:text-red-400'
                              : 'bg-muted text-muted-foreground'
                            }`}>
                              Status: {getMailStatusText(msg.status)}
                            </div>
                            <div className="text-muted-foreground text-xs">{isExpanded ? "�? : "�?}</div>
                          </div>
                        </div>
                        {isExpanded && (
                          <div className="mt-3 bg-muted/30 p-3 rounded text-sm font-mono whitespace-pre-wrap break-words">
                            <div className="mb-2 pb-2 border-b border-muted">
                              <span className="font-semibold">Mail ID:</span> {msg.mail_id}
                            </div>
                            <div className="mb-2 pb-2 border-b border-muted">
                              <span className="font-semibold">Subject:</span> {msg.subject}
                            </div>
                            {msg.from_email && (
                              <div className="mb-2 pb-2 border-b border-muted">
                                <span className="font-semibold">From:</span> {msg.from_name} &lt;{msg.from_email}&gt;
                              </div>
                            )}
                            {msg.error_message && (
                              <div className="mt-2 text-red-500">
                                <span className="font-semibold">Error:</span> {msg.error_message}
                              </div>
                            )}
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
                  <div>
                    <label className="block text-sm font-medium mb-1">模板名称 (Name)</label>
                    <input
                      type="text"
                      value={newTplName}
                      onChange={(e) => setNewTplName(e.target.value)}
                      placeholder="例如：注册验证码模板"
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
                    <label className="block text-sm font-medium mb-1">描述 (Description)</label>
                    <input
                      type="text"
                      value={newTplDesc}
                      onChange={(e) => setNewTplDesc(e.target.value)}
                      placeholder="内部备注信息，如：用于新用户注册时发送验证码"
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
                    {templates.map((tpl) => (
                      <div key={tpl.template_id} className="border-b pb-4 last:border-0 relative">
                        <div className="flex items-center justify-between gap-2">
                          <span className="font-mono text-xs bg-muted px-2 py-1 rounded select-all cursor-pointer" title="Click to select and copy" onClick={(e) => {
                            const target = e.target as HTMLElement;
                            const selection = window.getSelection();
                            const range = document.createRange();
                            range.selectNodeContents(target);
                            selection?.removeAllRanges();
                            selection?.addRange(range);
                            document.execCommand('copy');
                            alert(t.mailsPage.copied);
                          }}>
                            {tpl.template_id}
                          </span>
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => handleEditClick(tpl)}
                              className="text-xs text-primary hover:underline"
                            >
                              {t.mailsPage.edit}
                            </button>
                            <button
                              onClick={() => handleDeleteTemplate(tpl.template_id)}
                              className="text-xs text-destructive hover:underline"
                            >
                              {t.mailsPage.delete}
                            </button>
                          </div>
                        </div>
                        <div className="text-sm text-foreground mt-2 font-semibold">{tpl.name || tpl.subject_template}</div>
                        <div className="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">{tpl.template_body}</div>
                      </div>
                    ))}
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
