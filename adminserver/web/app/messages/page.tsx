"use client"

import React from "react"

import { useEffect, useState } from "react"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/useLanguage"
import { Sidebar } from "@/components/sidebar"
import { Send, List, FileText } from "lucide-react"

export default function AdminMessagesPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<"send" | "sent" | "templates">("send")

  // For users
  const [users, setUsers] = useState<any[]>([])
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([])
  const [userIds, setUserIds] = useState("")
  const [templateId, setTemplateId] = useState("")
  const [msgType, setMsgType] = useState<number>(1) // Default to 1 (System Notice)
  const [payload, setPayload] = useState("{\n}")
  const [sending, setSending] = useState(false)

  // For listing
  const [sentMessages, setSentMessages] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [statusFilter, setStatusFilter] = useState("")
  const [totalCount, setTotalCount] = useState(0)
  const [expandedMsgId, setExpandedMsgId] = useState<string | null>(null)

  const getMessageStatusText = (status: any) => {
    if (status === undefined || status === null) return t.messagesPage.statusUnread;
    const s = String(status).toUpperCase();
    if (s === "0" || s === "MESSAGE_STATUS_UNREAD" || s === "UNREAD") return t.messagesPage.statusUnread;
    if (s === "1" || s === "MESSAGE_STATUS_READ" || s === "READ") return t.messagesPage.statusRead;
    if (s === "2" || s === "MESSAGE_STATUS_DELETED" || s === "DELETED") return t.messagesPage.statusDeleted;
    if (s === "3" || s === "MESSAGE_STATUS_REVOKED" || s === "REVOKED") return t.messagesPage.statusRevoked;
    return String(status);
  }


  // For templates
  const [templates, setTemplates] = useState<any[]>([])
  const [loadingTemplates, setLoadingTemplates] = useState(false)
  const [newTplTitle, setNewTplTitle] = useState("")
  const [newTplContent, setNewTplContent] = useState("")
  const [newTplDesc, setNewTplDesc] = useState("")
  const [editingTemplateId, setEditingTemplateId] = useState<string | null>(null)
  const [editingTemplateVersion, setEditingTemplateVersion] = useState<number>(0)
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
        setUsers(res.users)
      }
    } catch (err) {
      console.error(err)
    }
  }

  const fetchTemplates = async () => {
    setLoadingTemplates(true)
    try {
      const res = await apiClient("/api/messages/templates")
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
    if (!newTplTitle || !newTplContent) {
      alert(t.messagesPage.alertFillFields)
      return
    }
    setCreatingTpl(true)
    try {
      if (editingTemplateId) {
        await apiClient("/api/messages/templates", {
          method: "PUT",
          body: JSON.stringify({
            template_id: editingTemplateId,
            title_tpl: newTplTitle,
            content_tpl: newTplContent,
            description: newTplDesc,
            current_version: editingTemplateVersion
          })
        })
        alert(t.messagesPage.alertUpdateSuccess)
      } else {
        await apiClient("/api/messages/templates", {
          method: "POST",
          body: JSON.stringify({
            title_tpl: newTplTitle,
            content_tpl: newTplContent,
            description: newTplDesc
          })
        })
        alert(t.messagesPage.alertCreateSuccess)
      }
      setNewTplTitle("")
      setNewTplContent("")
      setNewTplDesc("")
      setEditingTemplateId(null)
      setEditingTemplateVersion(0)
      fetchTemplates()
    } catch (err) {
      console.error(err)
    } finally {
      setCreatingTpl(false)
    }
  }

  const handleDeleteTemplate = async (id: string) => {
    if (!confirm(t.messagesPage.confirmDelete)) {
      return
    }
    try {
      await apiClient(`/api/messages/templates?template_id=${id}`, {
        method: "DELETE"
      })
      alert(t.messagesPage.alertDeleteSuccess)
      fetchTemplates()
    } catch (err: any) {
      if (err.message && err.message.includes("not implemented")) {
        alert(t.messagesPage.alertDeleteNotSupported)
      } else {
        alert(t.messagesPage.alertDeleteNotSupported)
      }
    }
  }

  const handleEditClick = (tpl: any) => {
    setEditingTemplateId(tpl.template_id)
    setEditingTemplateVersion(tpl.version || 0)
    setNewTplTitle(tpl.title_tpl || "")
    setNewTplContent(tpl.content_tpl || "")
    setNewTplDesc(tpl.description || "")
  }

  const fetchSentMessages = async () => {
    setLoading(true)
    try {
      let query = `/api/messages/sent?page=${page}&page_size=${pageSize}`
      if (statusFilter !== "") {
        query += `&status=${statusFilter}`
      }
      const res = await apiClient(query)
      if (res) {
        setSentMessages(res.messages || [])
        setTotalCount(res.total || 0)
      }
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleSend = async () => {
    if (selectedUserIds.length === 0) {
      alert(t.messagesPage.alertNoUsers)
      return
    }
    if (!templateId) {
      alert(t.messagesPage.alertNoTemplate)
      return
    }

    setSending(true)
    try {
      const res = await apiClient("/api/messages/send", {
        method: "POST",
        body: JSON.stringify({
          user_ids: selectedUserIds,
          template_id: templateId,
          payload: payload,
          msg_type: msgType,
        }),
      })

      alert(t.messagesPage.alertSendSuccess.replace('{{count}}', String(res.count || 0)))
      setSelectedUserIds([])
      setPayload("{\n}")
      setTemplateId("")
    } catch (err) {
      console.error(err)
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8 max-w-5xl mx-auto">
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-foreground mb-2">{t.messagesPage.title || t.sidebar.messages}</h1>
            <p className="text-muted-foreground">{t.messagesPage.subtitle}</p>
          </div>

          <div className="flex gap-4 border-b mb-6">
            <button
              onClick={() => setActiveTab("send")}
              className={`pb-3 flex items-center gap-2 font-medium transition-colors ${activeTab === "send" ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground"
                }`}
            >
              <Send className="w-4 h-4" />
              {t.messagesPage.sendTab}
            </button>
            <button
              onClick={() => setActiveTab("sent")}
              className={`pb-3 flex items-center gap-2 font-medium transition-colors ${activeTab === "sent" ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground"
                }`}
            >
              <List className="w-4 h-4" />
              {t.messagesPage.sentTab}
            </button>
            <button
              onClick={() => setActiveTab("templates")}
              className={`pb-3 flex items-center gap-2 font-medium transition-colors ${activeTab === "templates" ? "border-b-2 border-primary text-primary" : "text-muted-foreground hover:text-foreground"
                }`}
            >
              <FileText className="w-4 h-4" />
              {t.messagesPage.templatesTab}
            </button>
          </div>

          {activeTab === "send" && (
            <div className="bg-card rounded-xl border p-6">
              <h2 className="text-lg font-semibold mb-6">{t.messagesPage.createNewMessage}</h2>

              <div className="space-y-4">
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-sm font-medium">{t.messagesPage.selectUsers}</label>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => setSelectedUserIds(users.map(u => u.id))}
                        className="text-xs text-primary hover:underline"
                      >
                        {t.messagesPage.selectAll}
                      </button>
                      <button
                        type="button"
                        onClick={() => setSelectedUserIds([])}
                        className="text-xs text-muted-foreground hover:underline"
                      >
                        {t.messagesPage.clear}
                      </button>
                    </div>
                  </div>
                  <div className="max-h-40 overflow-y-auto border border-input rounded-md p-2 bg-transparent">
                    {users.length === 0 ? (
                      <div className="text-sm text-muted-foreground p-2">{t.messagesPage.loadingUsers}</div>
                    ) : (
                      users.map(u => (
                        <label key={u.id} className="flex items-center space-x-2 p-1 hover:bg-muted/50 rounded cursor-pointer">
                          <input
                            type="checkbox"
                            checked={selectedUserIds.includes(u.id)}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedUserIds(prev => [...prev, u.id])
                              } else {
                                setSelectedUserIds(prev => prev.filter(id => id !== u.id))
                              }
                            }}
                            className="rounded border-gray-300"
                          />
                          <span className="text-sm">{u.name} <span className="text-xs text-muted-foreground ml-1">({u.id})</span></span>
                        </label>
                      ))
                    )}
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    {t.messagesPage.selected.replace('{{count}}', String(selectedUserIds.length))}
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">{t.messagesPage.selectTemplate}</label>
                  <select
                    value={templateId}
                    onChange={(e) => {
                      const tId = e.target.value
                      setTemplateId(tId)
                      // Auto-fill payload template
                      const tpl = templates.find(t => t.template_id === tId)
                      if (tpl) {
                        const regex = /{{([^}]+)}}/g
                        const vars = new Set<string>()
                        let match
                        while ((match = regex.exec(tpl.title_tpl)) !== null) vars.add(match[1].trim().replace(/^\./, ''))
                        while ((match = regex.exec(tpl.content_tpl)) !== null) vars.add(match[1].trim().replace(/^\./, ''))

                        const obj: Record<string, string> = {}
                        vars.forEach(v => obj[v] = "")
                        setPayload(JSON.stringify(obj, null, 2))
                      } else {
                        setPayload("{\n}")
                      }
                    }}
                    className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  >
                    <option value="">{t.messagesPage.selectTemplatePlaceholder}</option>
                    {templates.map(t => (
                      <option key={t.template_id} value={t.template_id}>
                        {t.title_tpl} ({t.template_id})
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">{t.messagesPage.messageType}</label>
                  <select
                    value={msgType}
                    onChange={(e) => setMsgType(Number(e.target.value))}
                    className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  >
                    <option value={1}>{t.messagesPage.typeSystem}</option>
                    <option value={2}>{t.messagesPage.typeAnnouncement}</option>
                    <option value={3}>{t.messagesPage.typePromotion}</option>
                    <option value={4}>{t.messagesPage.typePayment}</option>
                    <option value={5}>{t.messagesPage.typeOther}</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium mb-1">{t.messagesPage.payloadJson}</label>
                  <textarea
                    rows={6}
                    value={payload}
                    onChange={(e) => setPayload(e.target.value)}
                    className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring font-mono"
                  />
                </div>

                <button
                  onClick={handleSend}
                  disabled={sending}
                  className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2"
                >
                  {sending ? t.messagesPage.sending : t.messagesPage.sendMessageBtn}
                </button>
              </div>
            </div>
          )}

          {activeTab === "sent" && (
            <div className="bg-card rounded-xl border p-6">
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-lg font-semibold">{t.messagesPage.sentHistory}</h2>
                <div className="flex items-center gap-4">
                  <label className="text-sm font-medium text-muted-foreground">{t.messagesPage.statusFilter}</label>
                  <select
                    value={statusFilter}
                    onChange={(e) => {
                      setPage(1)
                      setStatusFilter(e.target.value)
                    }}
                    className="rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  >
                    <option value="">{t.messagesPage.statusAll}</option>
                    <option value="0">{t.messagesPage.statusUnread}</option>
                    <option value="1">{t.messagesPage.statusRead}</option>
                    <option value="2">{t.messagesPage.statusDeleted}</option>
                    <option value="3">{t.messagesPage.statusRevoked}</option>
                  </select>
                </div>
              </div>

              {loading ? (
                <div className="text-muted-foreground py-4">{t.messagesPage.loading}</div>
              ) : sentMessages.length > 0 ? (
                <div className="space-y-4">
                  {sentMessages.map((msg, i) => {
                    const isExpanded = expandedMsgId === msg.message_id
                    return (
                      <div key={msg.message_id || i} className="border-b pb-4 last:border-0">
                        <div 
                          className="flex items-center justify-between cursor-pointer hover:bg-muted/50 p-2 rounded -mx-2 transition-colors"
                          onClick={() => setExpandedMsgId(isExpanded ? null : msg.message_id)}
                        >
                          <div className="font-medium text-sm">
                            <span className="text-muted-foreground mr-2">{msg.created_at ? msg.created_at.split('T')[0] : ''}</span>
                            UID: {msg.user_id} - TPL: {msg.template_id}
                          </div>
                          <div className="flex items-center gap-3">
                            <div className={`text-xs px-2 py-1 rounded border ${
                              (String(msg.status).toUpperCase() === "1" || String(msg.status).toUpperCase() === "MESSAGE_STATUS_READ" || String(msg.status).toUpperCase() === "READ") 
                              ? 'bg-green-100 text-green-700 border-green-200 dark:bg-green-900/30 dark:text-green-400' 
                              : 'bg-muted text-muted-foreground'
                            }`}>
                              Status: {getMessageStatusText(msg.status)}
                            </div>
                            <div className="text-muted-foreground text-xs">{isExpanded ? "▲" : "▼"}</div>
                          </div>
                        </div>
                        {isExpanded && (
                          <div className="mt-3 bg-muted/30 p-3 rounded text-sm font-mono whitespace-pre-wrap break-words">
                            <div className="mb-2 pb-2 border-b border-muted">
                              <span className="font-semibold">Message ID:</span> {msg.message_id}
                            </div>
                            <div className="mb-2 pb-2 border-b border-muted">
                              <span className="font-semibold">Sent At:</span> {msg.created_at}
                            </div>
                            <div className="flex gap-4">
                              <div><span className="font-semibold">Type:</span> {msg.msg_type}</div>
                              <div><span className="font-semibold">Source:</span> {msg.msg_source}</div>
                            </div>
                          </div>
                        )}
                      </div>
                    )
                  })}
                  
                  <div className="flex items-center justify-between pt-4 border-t mt-6">
                    <div className="text-sm text-muted-foreground">
                      {t.messagesPage.totalItems.replace('{{total}}', String(totalCount))}
                    </div>
                    <div className="flex gap-2">
                      <button
                        onClick={() => setPage(p => Math.max(1, p - 1))}
                        disabled={page === 1}
                        className="px-3 py-1 border rounded text-sm disabled:opacity-50 hover:bg-muted"
                      >
                        {t.messagesPage.prevPage}
                      </button>
                      <span className="px-3 py-1 text-sm">{page}</span>
                      <button
                        onClick={() => setPage(p => p + 1)}
                        disabled={page * pageSize >= totalCount}
                        className="px-3 py-1 border rounded text-sm disabled:opacity-50 hover:bg-muted"
                      >
                        {t.messagesPage.nextPage}
                      </button>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  {t.messagesPage.noSentMessages}
                </div>
              )}
            </div>
          )}

          {activeTab === "templates" && (
            <div className="grid gap-6 lg:grid-cols-2">
              <div className="bg-card rounded-xl border p-6">
                <h2 className="text-lg font-semibold mb-6">
                  {editingTemplateId ? t.messagesPage.edit : t.messagesPage.createTemplate}
                </h2>
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium mb-1">{t.messagesPage.titleTemplate}</label>
                    <input
                      type="text"
                      value={newTplTitle}
                      onChange={(e) => setNewTplTitle(e.target.value)}
                      placeholder={t.messagesPage.titleTemplatePlaceholder}
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">{t.messagesPage.contentTemplate}</label>
                    <textarea
                      rows={4}
                      value={newTplContent}
                      onChange={(e) => setNewTplContent(e.target.value)}
                      placeholder={t.messagesPage.contentTemplatePlaceholder}
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">描述 (Description)</label>
                    <input
                      type="text"
                      value={newTplDesc}
                      onChange={(e) => setNewTplDesc(e.target.value)}
                      placeholder="内部备注信息"
                      className="w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm"
                    />
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={handleCreateTemplate}
                      disabled={creatingTpl}
                      className="bg-primary text-primary-foreground hover:bg-primary/90 inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2"
                    >
                      {creatingTpl ? t.messagesPage.creating : (editingTemplateId ? t.messagesPage.saveChangesBtn : t.messagesPage.createTemplateBtn)}
                    </button>
                    {editingTemplateId && (
                      <button
                        onClick={() => {
                          setEditingTemplateId(null)
                          setEditingTemplateVersion(0)
                          setNewTplTitle("")
                          setNewTplContent("")
                          setNewTplDesc("")
                        }}
                        className="bg-secondary text-secondary-foreground hover:bg-secondary/80 inline-flex items-center justify-center rounded-md text-sm font-medium h-9 px-4 py-2"
                      >
                        {t.messagesPage.cancelEdit}
                      </button>
                    )}
                  </div>
                </div>
              </div>

              <div className="bg-card rounded-xl border p-6">
                <h2 className="text-lg font-semibold mb-6">{t.messagesPage.existingTemplates}</h2>
                {loadingTemplates ? (
                  <div className="text-muted-foreground">{t.messagesPage.loading}</div>
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
                            alert(t.messagesPage.copied);
                          }}>
                            {tpl.template_id}
                          </span>
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => handleEditClick(tpl)}
                              className="text-xs text-primary hover:underline"
                            >
                              {t.messagesPage.edit}
                            </button>
                            <button
                              onClick={() => handleDeleteTemplate(tpl.template_id)}
                              className="text-xs text-destructive hover:underline"
                            >
                              {t.messagesPage.delete}
                            </button>
                          </div>
                        </div>
                        <div className="text-sm text-foreground mt-2 font-semibold">{tpl.title_tpl}</div>
                        <div className="text-sm text-muted-foreground mt-1 whitespace-pre-wrap">{tpl.content_tpl}</div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-8 text-muted-foreground">{t.messagesPage.noTemplates}</div>
                )}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
