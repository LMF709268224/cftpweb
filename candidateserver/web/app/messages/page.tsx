"use client"

import React from "react"

import { useState, useEffect } from "react"
import { apiClient } from "@/lib/apiClient"
import { Sidebar } from "@/components/sidebar"
import { cn, formatBackendDate } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  MessageSquare,
  Bell,
  Megaphone,
  Gift,
  CreditCard,
  FileText,
  ChevronRight,
  Circle,
  CheckCheck,
  Trash2,
  MoreHorizontal,
} from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useTranslation } from "@/lib/useLanguage"
import { toast } from "sonner"

// We will store loaded messages here.
type Message = {
  id: string;
  type: string;
  title: string;
  content: string;
  time: string;
  isRead: boolean;
};



export default function MessagesPage() {
  const { t } = useTranslation()
  const [selectedType, setSelectedType] = useState<string | null>(null)
  const [messageList, setMessageList] = useState<Message[]>([])
  const [loading, setLoading] = useState(false)

  const typeConfig = {
    system: {
      icon: Bell,
      iconBg: "bg-primary/10",
      iconColor: "text-primary",
      label: t.messagesPage.systemNotice,
    },
    announcement: {
      icon: Megaphone,
      iconBg: "bg-blue-500/10",
      iconColor: "text-blue-600",
      label: t.messagesPage.announcement,
    },
    promotion: {
      icon: Gift,
      iconBg: "bg-amber-500/10",
      iconColor: "text-amber-600",
      label: t.messagesPage.promotion,
    },
    payment: {
      icon: CreditCard,
      iconBg: "bg-emerald-500/10",
      iconColor: "text-emerald-600",
      label: t.messagesPage.payment,
    },
    other: {
      icon: FileText,
      iconBg: "bg-zinc-500/10",
      iconColor: "text-zinc-600",
      label: t.messagesPage.other,
    },
  }

  const fetchMessages = async () => {
    setLoading(true)
    try {
      const res = await apiClient("/api/messages?limit=50")
      if (res?.messages) {
        setMessageList(res.messages.map((m: any) => {
          let type = "system"
          if (m.msg_type === 2) type = "announcement"
          else if (m.msg_type === 3) type = "promotion"
          else if (m.msg_type === 4) type = "payment"
          else if (m.msg_type === 5) type = "other"

          let title = t.common.systemNotification
          if (type === "announcement") title = t.messagesPage.announcement
          else if (type === "promotion") title = t.messagesPage.promotion
          else if (type === "payment") title = t.messagesPage.payment
          else if (type === "other") title = t.messagesPage.other

          let content = m.payload || ""
          try {
            const parsed = JSON.parse(m.payload)
            title = parsed.title || title
            content = parsed.content || content
          } catch (e) { }

          return {
            id: String(m.message_id || m.id),
            type: type,
            title: title,
            content: content,
            time: formatBackendDate(m.created_at),
            isRead: m.status === 1, // Assuming 2 is read
          }
        }))
      }
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMessages()
  }, [])

  const filteredMessages = selectedType
    ? messageList.filter((m) => m.type === selectedType)
    : messageList

  const unreadCount = messageList.filter((m) => !m.isRead).length

  const markAllAsRead = async () => {
    const unreadIds = messageList.filter(m => !m.isRead).map(m => m.id)
    if (unreadIds.length === 0) return

    try {
      await apiClient("/api/messages/read", {
        method: "PUT",
        body: JSON.stringify({ message_ids: unreadIds })
      })
      setMessageList((prev) => prev.map((m) => ({ ...m, isRead: true })))
      toast.success(t.messagesPage.markReadSuccess)
    } catch (e) { }
  }

  const markAsRead = async (id: string) => {
    try {
      await apiClient("/api/messages/read", {
        method: "PUT",
        body: JSON.stringify({ message_ids: [id] })
      })
      setMessageList((prev) =>
        prev.map((m) => (m.id === id ? { ...m, isRead: true } : m))
      )
      toast.success(t.messagesPage.markReadSuccess)
    } catch (e) { }
  }

  const deleteMessage = async (id: string) => {
    try {
      await apiClient("/api/messages/delete", {
        method: "POST",
        body: JSON.stringify({ message_ids: [id] })
      })
      setMessageList((prev) => prev.filter((m) => m.id !== id))
      toast.success(t.messagesPage.deleteSuccess)
    } catch (e) { }
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />

      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8 flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.messagesPage.title}</h1>
              <p className="mt-1 text-muted-foreground">
                {t.messagesPage.unreadCount.replace("{{count}}", String(unreadCount))}
              </p>
            </div>
            {unreadCount > 0 && (
              <Button variant="outline" onClick={markAllAsRead} className="gap-2">
                <CheckCheck className="h-4 w-4" />
                {t.messagesPage.markAllAsRead}
              </Button>
            )}
          </div>

          {/* Filter Tabs */}
          <div className="mb-6 flex gap-2 flex-wrap">
            <button
              onClick={() => setSelectedType(null)}
              className={cn(
                "px-4 py-2 text-sm font-medium rounded-lg transition-all",
                selectedType === null
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:text-foreground"
              )}
            >
              {t.messagesPage.all}
              <Badge variant="secondary" className="ml-2 h-5 px-1.5">
                {messageList.length}
              </Badge>
            </button>
            {Object.entries(typeConfig).map(([type, config]) => {
              const count = messageList.filter((m) => m.type === type).length
              return (
                <button
                  key={type}
                  onClick={() => setSelectedType(type)}
                  className={cn(
                    "flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg transition-all",
                    selectedType === type
                      ? "bg-primary text-primary-foreground"
                      : "bg-muted text-muted-foreground hover:text-foreground"
                  )}
                >
                  <config.icon className="h-4 w-4" />
                  {config.label}
                  <Badge variant="secondary" className="h-5 px-1.5">
                    {count}
                  </Badge>
                </button>
              )
            })}
          </div>

          {/* Messages List */}
          <div className="rounded-2xl border border-border bg-card shadow-sm overflow-hidden">
            {filteredMessages.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-16 text-center">
                <div className="mb-4 h-16 w-16 rounded-full bg-muted flex items-center justify-center">
                  <MessageSquare className="h-8 w-8 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-semibold text-foreground mb-2">{t.messagesPage.noMessages}</h3>
                <p className="text-muted-foreground">{t.messagesPage.noMessagesDesc}</p>
              </div>
            ) : (
              <div className="divide-y divide-border">
                {filteredMessages.map((message) => {
                  const config = typeConfig[message.type as keyof typeof typeConfig]
                  return (
                    <div
                      key={message.id}
                      className={cn(
                        "group flex items-start gap-4 p-6 transition-colors hover:bg-muted/50",
                        !message.isRead && "bg-primary/5"
                      )}
                    >
                      <div className={cn(
                        "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
                        config.iconBg
                      )}>
                        <config.icon className={cn("h-5 w-5", config.iconColor)} />
                      </div>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          {!message.isRead && (
                            <Circle className="h-2 w-2 fill-primary text-primary" />
                          )}
                          <h3 className={cn(
                            "font-medium text-card-foreground",
                            !message.isRead && "font-semibold"
                          )}>
                            {message.title}
                          </h3>
                          <Badge variant="outline" className="text-xs">
                            {config.label}
                          </Badge>
                        </div>
                        <p className="text-sm text-muted-foreground line-clamp-2 mb-2">
                          {message.content}
                        </p>
                        <span className="text-xs text-muted-foreground">{message.time}</span>
                      </div>

                      <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-8 w-8">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            {!message.isRead && (
                              <DropdownMenuItem onClick={() => markAsRead(message.id)}>
                                <CheckCheck className="mr-2 h-4 w-4" />
                                标为已读
                              </DropdownMenuItem>
                            )}
                            <DropdownMenuItem
                              className="text-destructive"
                              onClick={() => deleteMessage(message.id)}
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              删除
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                        <ChevronRight className="h-5 w-5 text-muted-foreground" />
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
