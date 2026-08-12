<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { AlertCircle, Bell, CheckCheck, ChevronRight, Circle, CreditCard, FileText, Gift, Loader2, Megaphone, MessageSquare, MoreHorizontal, RefreshCw, Trash2, X } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { useDialogAccessibility } from "@/lib/dialogAccessibility"
import { formatBackendDate } from "@/lib/utils"
import { fetchUnreadCount } from "@/lib/unreadCountCache"
import { useTranslation } from "@/lib/language"
import { usePolling } from "@/lib/polling"
type Message = { id: string; type: string; rawTitle: string; rawContent: string; time: string; isRead: boolean; isUnread: boolean }
type MessageStatusFilter = "unread" | "read"
type MessageAction = "read" | "delete"

const { t, lang } = useTranslation()
const selectedStatus = ref<MessageStatusFilter | null>(null)
const detailModalOpen = ref(false)
useBodyScrollLock(() => detailModalOpen.value)
const messageDetailDialogRef = ref<HTMLElement | null>(null)
const selectedMessageDetail = ref<any>(null)
const messageList = ref<Message[]>([])
const openMenuId = ref<string | null>(null)
const loading = ref(true)
const loadError = ref(false)
const markAllLoading = ref(false)
const messageActionLoadingId = ref<string | null>(null)
const messageActionError = ref<{ id: string; action: MessageAction } | null>(null)
const detailLoadingId = ref<string | null>(null)
const detailLoadError = ref(false)
const detailSourceMessage = ref<Message | null>(null)
const totalUnreadCount = ref(0)
let messagesRequestId = 0
let detailRequestId = 0

const page = ref(1)
const lastPage = ref(1)
const prevCursor = ref("")
const nextCursor = ref("")
const hasMore = ref(false)
const pageSize = 10

const typeConfig = computed(() => ({
  system: { icon: Bell, iconBg: "bg-primary/10", iconColor: "text-primary", label: t.value.messagesPage.systemNotice },
  announcement: { icon: Megaphone, iconBg: "bg-blue-500/10", iconColor: "text-blue-600", label: t.value.messagesPage.announcement },
  score: { icon: Gift, iconBg: "bg-amber-500/10", iconColor: "text-amber-600", label: t.value.messagesPage.promotion },
  payment: { icon: CreditCard, iconBg: "bg-emerald-500/10", iconColor: "text-emerald-600", label: t.value.messagesPage.payment },
  other: { icon: FileText, iconBg: "bg-zinc-500/10", iconColor: "text-zinc-600", label: t.value.messagesPage.other },
}))

const statusTabs = computed(() => [
  { value: null, label: t.value.messagesPage.all, icon: MessageSquare },
  { value: "unread" as const, label: t.value.messagesPage.statusUnread, icon: Bell },
  { value: "read" as const, label: t.value.messagesPage.statusRead, icon: Circle },
])

const unreadCount = computed(() => messageList.value.filter((m) => m.isUnread).length)

function nextPage() {
  if (!hasMore.value || loading.value) return
  page.value++
  void fetchMessages()
  window.scrollTo({ top: 0, behavior: "smooth" })
}

function prevPage() {
  if (page.value <= 1 || loading.value) return
  page.value--
  void fetchMessages()
  window.scrollTo({ top: 0, behavior: "smooth" })
}

function resetPagination() {
  page.value = 1
  lastPage.value = 1
  prevCursor.value = ""
  nextCursor.value = ""
  hasMore.value = false
}

async function selectStatus(status: MessageStatusFilter | null) {
  if (selectedStatus.value === status && !loading.value) return
  selectedStatus.value = status
  resetPagination()
  await fetchMessages()
}

async function syncUnreadCount(suppressErrorToast = true) {
  try {
    totalUnreadCount.value = await fetchUnreadCount(suppressErrorToast)
  } catch {
    // The message list should remain usable even if the count refresh fails.
  }
}

function configFor(type: string) {
  return typeConfig.value[type as keyof typeof typeConfig.value] || typeConfig.value.system
}

function unreadCountText() {
  return t.value.messagesPage.unreadCount.replace("{{count}}", unreadCountBadgeLabel())
}

function unreadCountBadgeLabel() {
  return totalUnreadCount.value >= 99 ? "99+" : String(totalUnreadCount.value)
}

function countBadgeClass(active: boolean) {
  return active
    ? "bg-primary text-primary-foreground"
    : "bg-slate-100 text-slate-600"
}

function markReadMenuLabel() {
  return t.value.messagesPage.markAsRead
}

function deleteMenuLabel() {
  return t.value.messagesPage.delete
}

function unreadLabel() {
  return t.value.messagesPage.unread
}

function messageActionErrorText(action: MessageAction) {
  return action === "read"
    ? t.value.messagesPage.markReadFailed
    : t.value.messagesPage.deleteFailed
}

function retryMessageAction(message: Message) {
  if (messageActionError.value?.id !== message.id) return
  if (messageActionError.value.action === "read") void markAsRead(message.id)
  else void deleteMessage(message.id)
}

function messageStatusValue(status: unknown) {
  if (typeof status === "number") return status
  const normalized = String(status || "").toUpperCase()
  if (normalized === "UNREAD" || normalized === "MESSAGE_STATUS_UNREAD") return 0
  if (normalized === "READ" || normalized === "MESSAGE_STATUS_READ") return 1
  if (normalized === "DELETED" || normalized === "MESSAGE_STATUS_DELETED") return 2
  if (normalized === "REVOKED" || normalized === "MESSAGE_STATUS_REVOKED") return 3
  const numeric = Number(status)
  return Number.isFinite(numeric) ? numeric : -1
}

function splitBilingualText(value: string) {
  const normalized = value
    .replace(/\r\n/g, "\n")
    .replace(/\s*\/\s*/g, "\n")
    .split("\n")
    .map((part) => part.trim())
    .filter(Boolean)

  if (normalized.length <= 1) return value.trim()

  const scoreChinese = (part: string) => (part.match(/[\u3400-\u9fff]/g)?.length || 0) * 2 - (part.match(/[A-Za-z]/g)?.length || 0) * 0.15
  const scoreEnglish = (part: string) => (part.match(/[A-Za-z]/g)?.length || 0) - (part.match(/[\u3400-\u9fff]/g)?.length || 0) * 1.5

  if (lang.value === "zh") {
    const preferred = [...normalized].sort((a, b) => scoreChinese(b) - scoreChinese(a))[0]
    if (preferred) return preferred
  }
  if (lang.value !== "zh") {
    const preferred = [...normalized].sort((a, b) => scoreEnglish(b) - scoreEnglish(a))[0]
    if (preferred) return preferred
  }
  return normalized[0]
}

function cleanMarkdown(value: string) {
  return value
    .replace(/^#{1,6}\s*/gm, "")
    .replace(/\*\*(.*?)\*\*/g, "$1")
    .replace(/\[(.*?)\]\((.*?)\)/g, "$1")
    .replace(/\{\{.*?\}\}/g, "")
    .replace(/\s+/g, " ")
    .trim()
}

function localizedMessageTitle(value: string, fallback: string) {
  const cleaned = cleanMarkdown(value || "")
  if (!cleaned) return fallback
  return splitBilingualText(cleaned)
}

function escapeHtml(value: string) {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;")
}

function renderInlineMarkdown(value: string) {
  return escapeHtml(value)
    .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
    .replace(/\[(.*?)\]\((https?:\/\/[^\s)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer" class="text-primary underline underline-offset-2">$1</a>')
    .replace(/\{\{.*?\}\}/g, "")
}

function markdownToHtml(markdown: string) {
  const source = String(markdown || "").replace(/\r\n/g, "\n").trim()
  if (!source) return ""

  const lines = source.split("\n")
  const html: string[] = []
  let listItems: string[] = []
  const flushList = () => {
    if (listItems.length === 0) return
    html.push(`<ul class="my-3 list-disc space-y-1 pl-5">${listItems.join("")}</ul>`)
    listItems = []
  }

  for (const rawLine of lines) {
    const line = rawLine.trim()
    if (!line) {
      flushList()
      continue
    }

    const heading = line.match(/^(#{1,6})\s+(.+)$/)
    if (heading) {
      flushList()
      const level = Math.min(heading[1].length + 2, 6)
      html.push(`<h${level} class="mt-3 font-semibold text-foreground">${renderInlineMarkdown(heading[2])}</h${level}>`)
      continue
    }

    const bullet = line.match(/^[-*]\s+(.+)$/)
    if (bullet) {
      listItems.push(`<li>${renderInlineMarkdown(bullet[1])}</li>`)
      continue
    }

    flushList()
    html.push(`<p class="my-2">${renderInlineMarkdown(line)}</p>`)
  }
  flushList()
  return html.join("")
}

function formatPayloadSummary(payload: unknown) {
  if (!payload) return ""
  if (typeof payload !== "string") return String(payload)
  const trimmed = payload.trim()
  if (!trimmed || trimmed === "{}") return ""
  try {
    const parsed = JSON.parse(trimmed)
    if (!parsed || typeof parsed !== "object") return trimmed
    if (typeof parsed.content === "string" && parsed.content.trim()) return parsed.content.trim()
    if (typeof parsed.message === "string" && parsed.message.trim()) return parsed.message.trim()
    if (typeof parsed.description === "string" && parsed.description.trim()) return parsed.description.trim()
    return Object.entries(parsed)
      .filter(([, value]) => value !== null && value !== undefined && String(value).trim() !== "")
      .slice(0, 4)
      .map(([key, value]) => `${key}: ${value}`)
      .join(" / ")
  } catch {
    return trimmed
  }
}

async function fetchMessages(showLoading = true, suppressErrorToast = false) {
  const requestId = ++messagesRequestId
  const requestedPage = page.value
  const requestedLastPage = lastPage.value
  const requestedStatus = selectedStatus.value
  if (showLoading) {
    loading.value = true
    loadError.value = false
  }
  try {
    const params = new URLSearchParams({
      page_size: String(pageSize),
    })
    let cursor = ""
    if (requestedPage > requestedLastPage) {
      cursor = nextCursor.value
    } else if (requestedPage < requestedLastPage) {
      cursor = prevCursor.value
    }

    if (cursor) params.set("cursor", cursor)
    if (requestedStatus) params.set("status", requestedStatus)
    const res = await apiClient(`/api/messages?${params.toString()}`, { suppressErrorToast })
    if (requestId !== messagesRequestId) return
    if (!Array.isArray(res?.messages)) throw new Error("MESSAGES_INVALID_RESPONSE")

    const isBackward = requestedPage < requestedLastPage
    hasMore.value = isBackward ? true : Boolean(res?.has_more)
    nextCursor.value = String(res?.next_cursor || "")
    prevCursor.value = String(res?.prev_cursor || "")
    lastPage.value = requestedPage
    messageList.value = res.messages.map((m: any) => {
      let type = "system"
      if (m.msg_type === 2) type = "announcement"
      else if (m.msg_type === 3) type = "score"
      else if (m.msg_type === 4) type = "payment"
      else if (m.msg_type === 5) type = "other"

      let title = t.value.common.systemNotification
      if (type === "announcement") title = t.value.messagesPage.announcement
      else if (type === "score") title = t.value.messagesPage.promotion
      else if (type === "payment") title = t.value.messagesPage.payment
      else if (type === "other") title = t.value.messagesPage.other

      const payload = m.template_payload || m.payload
      let content = m.content || formatPayloadSummary(payload)
      try {
        const parsed = JSON.parse(payload)
        title = m.title || parsed.title || title
        content = m.content || parsed.content || content
      } catch {
        // payload can be plain text.
        title = m.title || title
      }

      const statusValue = messageStatusValue(m.status)
      return {
        id: String(m.message_id || m.id),
        type,
        rawTitle: title,
        rawContent: content,
        time: formatBackendDate(m.created_at),
        isRead: statusValue === 1,
        isUnread: statusValue === 0,
      }
    })
    loadError.value = false
  } catch (e) {
    if (requestId !== messagesRequestId) return
    console.error(e)
    if (showLoading) loadError.value = true
  } finally {
    if (showLoading && requestId === messagesRequestId) loading.value = false
  }
}

async function markAllAsRead() {
  if (markAllLoading.value) return
  const unreadIds = messageList.value.filter((m) => m.isUnread).map((m) => m.id)
  if (unreadIds.length === 0) return
  markAllLoading.value = true
  try {
    messagesRequestId += 1
    await apiClient("/api/messages/read", { method: "PUT", body: JSON.stringify({ message_ids: unreadIds }), suppressErrorToast: true })
    messageList.value = messageList.value.map((m) => (m.isUnread ? { ...m, isRead: true, isUnread: false } : m))
    await syncUnreadCount()
    toast.success(t.value.messagesPage.markReadSuccess)
  } catch {
    toast.error(t.value.messagesPage.markAllFailed)
  } finally {
    markAllLoading.value = false
  }
}

async function markAsRead(id: string, showToast = true) {
  if (messageActionLoadingId.value === id) return
  messageActionLoadingId.value = id
  messageActionError.value = null
  try {
    messagesRequestId += 1
    await apiClient("/api/messages/read", { method: "PUT", body: JSON.stringify({ message_ids: [id] }), suppressErrorToast: true })
    messageList.value = messageList.value.map((m) => (m.id === id ? { ...m, isRead: true, isUnread: false } : m))
    await syncUnreadCount()
    openMenuId.value = null
    if (showToast) toast.success(t.value.messagesPage.markReadSuccess)
  } catch {
    messageActionError.value = { id, action: "read" }
    openMenuId.value = null
  } finally {
    if (messageActionLoadingId.value === id) messageActionLoadingId.value = null
  }
}

async function deleteMessage(id: string) {
  if (messageActionLoadingId.value === id) return
  messageActionLoadingId.value = id
  messageActionError.value = null
  try {
    messagesRequestId += 1
    await apiClient("/api/messages/delete", { method: "POST", body: JSON.stringify({ message_ids: [id] }), suppressErrorToast: true })
    messageList.value = messageList.value.filter((m) => m.id !== id)
    await syncUnreadCount()
    openMenuId.value = null
    toast.success(t.value.messagesPage.deleteSuccess)
  } catch {
    messageActionError.value = { id, action: "delete" }
    openMenuId.value = null
  } finally {
    if (messageActionLoadingId.value === id) messageActionLoadingId.value = null
  }
}

async function handleViewDetail(message: Message, markUnread = true) {
  if (detailLoadingId.value || messageActionLoadingId.value === message.id) return
  const requestId = ++detailRequestId
  detailSourceMessage.value = message
  detailLoadError.value = false
  detailLoadingId.value = message.id
  try {
    if (markUnread && message.isUnread) await markAsRead(message.id, false)
    const detail = await apiClient(`/api/messages/${message.id}`, { suppressErrorToast: true })
    if (requestId !== detailRequestId) return
    const detailType = message.type
    selectedMessageDetail.value = {
      ...detail,
      rawTitle: detail?.title || message.rawTitle,
      rawContent: detail?.content || message.rawContent,
      typeLabel: configFor(detailType).label,
      time: formatBackendDate(detail?.created_at || ""),
    }
    detailModalOpen.value = true
  } catch {
    if (requestId !== detailRequestId) return
    selectedMessageDetail.value = {
      rawTitle: message.rawTitle,
      rawContent: message.rawContent,
      typeLabel: configFor(message.type).label,
      time: message.time,
    }
    detailLoadError.value = true
    detailModalOpen.value = true
  } finally {
    if (requestId === detailRequestId && detailLoadingId.value === message.id) detailLoadingId.value = null
  }
}

function retryMessageDetail() {
  if (!detailSourceMessage.value || detailLoadingId.value) return
  void handleViewDetail(detailSourceMessage.value, false)
}

function closeMessageDetail() {
  detailRequestId += 1
  detailModalOpen.value = false
  detailLoadingId.value = null
  detailLoadError.value = false
  detailSourceMessage.value = null
  selectedMessageDetail.value = null
}

useDialogAccessibility(() => detailModalOpen.value, messageDetailDialogRef, closeMessageDetail)

const messagesPolling = usePolling(
  async () => {
    await fetchMessages(false, true)
    await syncUnreadCount()
  },
  { shouldPoll: () => !loading.value && !detailModalOpen.value && !markAllLoading.value && !messageActionLoadingId.value && !detailLoadingId.value },
)

onMounted(() => {
  void fetchMessages()
  void syncUnreadCount()
  messagesPolling.start()
})
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <MessageSquare class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.messagesPage.title }}</span>
      </header>

      <main class="messages-main w-full px-5 py-8 md:px-8 lg:px-10">
        <div class="messages-page-intro mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 class="text-2xl font-bold tracking-tight text-foreground md:text-3xl">{{ t.messagesPage.title }}</h1>
            <p class="mt-2 text-muted-foreground">{{ unreadCountText() }}</p>
          </div>
          <div v-if="unreadCount > 0" class="flex justify-end">
        <button class="btn btn-outline w-full justify-center rounded-lg bg-white/80 shadow-sm hover:border-primary/25 hover:bg-primary/10 hover:text-primary sm:w-auto" :disabled="markAllLoading" @click="markAllAsRead">
          <Loader2 v-if="markAllLoading" class="h-4 w-4 animate-spin" />
          <CheckCheck v-else class="h-4 w-4" />
          {{ t.messagesPage.markAllAsRead }}
        </button>
          </div>
        </div>

    <div class="messages-panel w-full overflow-hidden rounded-t-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="w-full bg-white">
        <div class="messages-tabs grid min-h-[60px] w-full grid-cols-3 items-stretch gap-1 rounded-t-xl bg-slate-100 p-1 md:flex md:min-h-[68px] md:flex-wrap md:items-stretch md:gap-x-8 md:gap-y-0 md:rounded-none md:border-b md:border-border md:bg-transparent md:p-0 md:pl-4">
          <button
            v-for="tab in statusTabs"
            :key="tab.value || 'all'"
            :class="['message-tab relative inline-flex min-h-12 cursor-pointer items-center justify-center gap-1.5 whitespace-nowrap rounded-lg px-2 py-3 text-sm font-semibold transition-colors duration-200 md:min-h-[68px] md:justify-start md:gap-2 md:rounded-none md:px-1 md:py-0 md:text-base md:font-medium', selectedStatus === tab.value ? 'bg-white text-primary shadow-sm md:bg-transparent md:shadow-none' : 'text-slate-600 hover:text-primary md:text-foreground']"
            @click="selectStatus(tab.value)"
          >
            <component :is="tab.icon" class="h-4 w-4" />
            {{ tab.label }}
            <span v-if="tab.value === 'unread'" :class="['rounded-full px-2 py-0.5 text-xs font-bold', countBadgeClass(selectedStatus === tab.value)]">{{ unreadCountBadgeLabel() }}</span>
            <span v-if="selectedStatus === tab.value" class="absolute bottom-[-1px] left-0 hidden h-0.5 w-full rounded-full bg-primary md:block" />
          </button>
        </div>
      </div>

      <div v-if="loading" class="messages-state flex items-center justify-center gap-2 px-4 py-16 text-muted-foreground">
        <Loader2 class="h-5 w-5 animate-spin text-primary" />
        <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="loadError" class="messages-state flex flex-col items-center justify-center px-4 py-16 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-red-50">
          <AlertCircle class="h-8 w-8 text-red-600" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.messagesPage.loadFailed }}</h3>
        <p class="mb-5 max-w-md text-muted-foreground">{{ t.messagesPage.loadFailedDesc }}</p>
        <button class="btn btn-primary min-w-32 justify-center" @click="fetchMessages()">
          <RefreshCw class="h-4 w-4" />
          {{ t.messagesPage.retry }}
        </button>
      </div>
      <div v-else-if="messageList.length === 0" class="messages-state flex flex-col items-center justify-center px-4 py-16 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><MessageSquare class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.messagesPage.noMessages }}</h3>
        <p class="text-muted-foreground">{{ t.messagesPage.noMessagesDesc }}</p>
      </div>
      <div v-else>
        <div
          v-for="message in messageList"
          :key="message.id"
          :class="['message-row group relative flex cursor-pointer items-start gap-3 border-b border-slate-100 px-3 py-4 transition-colors hover:bg-primary/10 md:gap-4 md:px-4', message.isUnread ? 'bg-primary/5' : '', detailLoadingId === message.id ? 'pointer-events-none opacity-75' : '']"
          @click="handleViewDetail(message)"
        >
          <div :class="['flex h-9 w-9 shrink-0 items-center justify-center rounded-lg md:h-10 md:w-10', configFor(message.type).iconBg, message.isUnread && 'ring-2 ring-primary/25']">
            <component :is="configFor(message.type).icon" :class="['h-4 w-4 md:h-5 md:w-5', configFor(message.type).iconColor]" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="message-heading mb-1.5 flex flex-wrap items-center gap-2 md:mb-2">
              <span v-if="message.isUnread" class="rounded-full bg-primary px-2 py-0.5 text-[11px] font-bold text-primary-foreground shadow-sm shadow-primary/20">
                {{ unreadLabel() }}
              </span>
              <h3 :class="['line-clamp-2 text-sm leading-snug text-card-foreground md:text-base', message.isUnread ? 'font-bold' : 'font-semibold']">{{ localizedMessageTitle(message.rawTitle, configFor(message.type).label) }}</h3>
              <span class="message-type-badge badge">{{ configFor(message.type).label }}</span>
            </div>
            <span class="text-xs text-muted-foreground">{{ message.time }}</span>
            <div
              v-if="messageActionError?.id === message.id"
              class="message-row-feedback mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 rounded-lg border border-red-200 bg-red-50 px-2.5 py-2 text-xs text-red-700"
              role="alert"
              @click.stop
            >
              <AlertCircle class="h-4 w-4 shrink-0" />
              <span class="min-w-0 flex-1">{{ messageActionErrorText(messageActionError.action) }}</span>
              <button type="button" class="font-semibold text-red-800 underline underline-offset-2 disabled:opacity-60" :disabled="messageActionLoadingId === message.id" @click.stop="retryMessageAction(message)">
                {{ t.messagesPage.actionRetry }}
              </button>
            </div>
          </div>
          <div class="message-actions flex shrink-0 items-center gap-1 md:gap-2">
            <div class="relative">
              <button type="button" class="btn btn-ghost h-9 w-9 rounded-lg border border-transparent p-0 text-slate-500 transition-colors hover:border-primary/20 hover:bg-primary/10 hover:text-primary" :aria-label="t.messagesPage.moreActions" :title="t.messagesPage.moreActions" :disabled="messageActionLoadingId === message.id || detailLoadingId === message.id" @click.stop="openMenuId = openMenuId === message.id ? null : message.id">
                <MoreHorizontal class="h-5 w-5" />
              </button>
              <div v-if="openMenuId === message.id" class="absolute right-0 top-9 z-50 min-w-36 overflow-hidden rounded-lg bg-white p-1 shadow-md" @click.stop>
                <button v-if="message.isUnread" class="flex w-full items-center rounded-lg px-2 py-1.5 text-sm hover:bg-muted disabled:cursor-not-allowed disabled:opacity-60" :disabled="messageActionLoadingId === message.id" @click="markAsRead(message.id)">
                  <Loader2 v-if="messageActionLoadingId === message.id" class="mr-2 h-4 w-4 animate-spin" />
                  <CheckCheck v-else class="mr-2 h-4 w-4" />
                  {{ markReadMenuLabel() }}
                </button>
                <button class="flex w-full items-center rounded-lg px-2 py-1.5 text-sm text-destructive hover:bg-muted disabled:cursor-not-allowed disabled:opacity-60" :disabled="messageActionLoadingId === message.id" @click="deleteMessage(message.id)">
                  <Loader2 v-if="messageActionLoadingId === message.id" class="mr-2 h-4 w-4 animate-spin" />
                  <Trash2 v-else class="mr-2 h-4 w-4" />
                  {{ deleteMenuLabel() }}
                </button>
              </div>
            </div>
            <Loader2 v-if="detailLoadingId === message.id" class="message-detail-indicator h-5 w-5 animate-spin text-primary" />
            <ChevronRight v-else class="message-detail-indicator h-5 w-5 text-slate-500 transition-colors group-hover:text-primary" />
          </div>
        </div>
        <div class="messages-pagination flex items-center justify-between px-4 py-3 text-sm text-muted-foreground">
          <span></span>
          <div class="flex items-center gap-2">
            <button class="rounded-lg border border-slate-200 px-3 py-1.5 font-medium transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :disabled="page <= 1" @click="prevPage()">
              {{ t.messagesPage.previous }}
            </button>
            <span class="min-w-20 text-center">{{ page }}</span>
            <button class="rounded-lg border border-slate-200 px-3 py-1.5 font-medium transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :disabled="!hasMore" @click="nextPage()">
              {{ t.messagesPage.next }}
            </button>
          </div>
        </div>
      </div>
    </div>

      </main>
    </div>

    <div v-if="detailModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-[2px]">
      <div
        ref="messageDetailDialogRef"
        class="flex max-h-[86vh] w-full max-w-2xl flex-col overflow-hidden rounded-[20px] bg-white shadow-[0_24px_70px_rgba(15,23,42,0.28)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="message-detail-dialog-title"
        tabindex="-1"
      >
        <div class="flex items-start justify-between gap-4 border-b border-slate-100 px-5 py-5 sm:px-6">
          <div class="min-w-0">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <h2 id="message-detail-dialog-title" class="min-w-0 flex-1 text-xl font-bold leading-snug text-slate-950 sm:text-2xl">{{ localizedMessageTitle(selectedMessageDetail?.rawTitle || '', t.messagesPage.systemNotice) }}</h2>
              <span v-if="selectedMessageDetail?.typeLabel" class="badge shrink-0 border-primary/15 bg-primary/5 text-primary">{{ selectedMessageDetail.typeLabel }}</span>
            </div>
            <p class="text-sm font-medium text-slate-500">{{ selectedMessageDetail?.time }}</p>
          </div>
          <button type="button" class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white/90 text-slate-500 transition hover:border-primary/25 hover:text-primary md:h-10 md:w-10" :aria-label="t.common.close" :title="t.common.close" @click="closeMessageDetail">
            <X class="h-5 w-5" />
          </button>
        </div>
        <div class="min-h-0 overflow-y-auto bg-slate-50/70 px-5 py-5 sm:px-6">
          <div v-if="detailLoadingId" class="flex min-h-48 items-center justify-center gap-2 text-muted-foreground" role="status" aria-live="polite">
            <Loader2 class="h-5 w-5 animate-spin text-primary" />
            <span>{{ t.common.loading }}</span>
          </div>
          <div v-else-if="detailLoadError" class="flex min-h-48 flex-col items-center justify-center rounded-xl border border-red-100 bg-white px-5 py-8 text-center" role="alert">
            <div class="mb-4 flex h-14 w-14 items-center justify-center rounded-xl bg-red-50">
              <AlertCircle class="h-7 w-7 text-red-600" />
            </div>
            <h3 class="text-lg font-semibold text-slate-950">{{ t.messagesPage.detailLoadFailed }}</h3>
            <p class="mt-2 max-w-md text-sm leading-6 text-slate-600">{{ t.messagesPage.detailLoadFailedDesc }}</p>
            <button type="button" class="btn btn-primary mt-5 min-w-28 justify-center" @click="retryMessageDetail">
              <RefreshCw class="h-4 w-4" />
              {{ t.messagesPage.retry }}
            </button>
          </div>
          <div v-else class="message-detail-content rounded-xl border border-slate-100 bg-white px-5 py-4 text-sm leading-7 text-slate-800 shadow-sm shadow-slate-200/70" v-html="markdownToHtml(selectedMessageDetail?.rawContent || '')" />
        </div>
      </div>
    </div>
  </AppShell>
</template>

<style scoped>
.message-detail-content :deep(p:first-child) {
  margin-top: 0;
}

.message-detail-content :deep(p:last-child),
.message-detail-content :deep(ul:last-child) {
  margin-bottom: 0;
}

.message-detail-content :deep(strong) {
  color: rgb(15 23 42);
  font-weight: 700;
}

@media (max-width: 767px) {
  .messages-page-intro {
    gap: 12px;
    margin-bottom: 16px;
  }

  .messages-tabs {
    height: 42px;
    min-height: 42px;
    padding-block: 0;
  }

  .message-tab {
    height: 42px;
    min-height: 42px;
    gap: 4px;
    padding: 4px;
  }

  .messages-state {
    padding-block: 32px;
  }

  .message-row {
    gap: 10px;
    padding: 12px;
  }

  .message-heading {
    gap: 6px;
    margin-bottom: 4px;
  }

  .message-type-badge {
    padding: 2px 8px;
  }

  .message-actions {
    gap: 0;
  }

  .message-detail-indicator {
    width: 16px;
    height: 16px;
  }

  .messages-pagination {
    padding: 8px 12px;
  }
}
</style>

