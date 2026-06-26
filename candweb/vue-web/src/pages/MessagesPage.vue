<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { Bell, CheckCheck, ChevronRight, CreditCard, FileText, Gift, Loader2, Megaphone, MessageSquare, MoreHorizontal, Trash2 } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { fetchUnreadCount } from "@/lib/unreadCountCache"
import { useTranslation } from "@/lib/language"
import { usePolling } from "@/lib/polling"

type Message = { id: string; type: string; rawTitle: string; rawContent: string; time: string; isRead: boolean }

const { t, lang } = useTranslation()
const selectedType = ref<string | null>(null)
const detailModalOpen = ref(false)
const selectedMessageDetail = ref<any>(null)
const messageList = ref<Message[]>([])
const openMenuId = ref<string | null>(null)
const loading = ref(true)
const markAllLoading = ref(false)
const messageActionLoadingId = ref<string | null>(null)
const detailLoadingId = ref<string | null>(null)
const totalUnreadCount = ref(0)

const page = ref(1)
const pageSize = 10

const typeConfig = computed(() => ({
  system: { icon: Bell, iconBg: "bg-primary/10", iconColor: "text-primary", label: t.value.messagesPage.systemNotice },
  announcement: { icon: Megaphone, iconBg: "bg-blue-500/10", iconColor: "text-blue-600", label: t.value.messagesPage.announcement },
  promotion: { icon: Gift, iconBg: "bg-amber-500/10", iconColor: "text-amber-600", label: t.value.messagesPage.promotion },
  payment: { icon: CreditCard, iconBg: "bg-emerald-500/10", iconColor: "text-emerald-600", label: t.value.messagesPage.payment },
  other: { icon: FileText, iconBg: "bg-zinc-500/10", iconColor: "text-zinc-600", label: t.value.messagesPage.other },
}))

const filteredMessages = computed(() => selectedType.value ? messageList.value.filter((m) => m.type === selectedType.value) : messageList.value)
const listUnreadCount = computed(() => messageList.value.filter((m) => !m.isRead).length)

const paginatedMessages = computed(() => {
  const start = (page.value - 1) * pageSize
  return filteredMessages.value.slice(start, start + pageSize)
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredMessages.value.length / pageSize)))

const messageRangeLabel = computed(() => {
  if (filteredMessages.value.length === 0) return "0 / 0"
  const start = (page.value - 1) * pageSize + 1
  const end = Math.min(page.value * pageSize, filteredMessages.value.length)
  return `${start}-${end} / ${filteredMessages.value.length}`
})

function goToPage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value) return
  page.value = nextPage
  window.scrollTo({ top: 0, behavior: "smooth" })
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
  return t.value.messagesPage.unreadCount.replace("{{count}}", String(totalUnreadCount.value))
}

function markReadMenuLabel() {
  return lang.value === "zh" ? "\u6807\u8bb0\u4e3a\u5df2\u8bfb" : "Mark as read"
}

function deleteMenuLabel() {
  return lang.value === "zh" ? "\u5220\u9664" : "Delete"
}

function unreadLabel() {
  return lang.value === "zh" ? "\u672a\u8bfb" : "Unread"
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
  if (showLoading) loading.value = true
  try {
    const res = await apiClient("/api/messages?limit=500", { suppressErrorToast })
    if (res?.messages) {
      messageList.value = res.messages.map((m: any) => {
        let type = "system"
        if (m.msg_type === 2) type = "announcement"
        else if (m.msg_type === 3) type = "promotion"
        else if (m.msg_type === 4) type = "payment"
        else if (m.msg_type === 5) type = "other"

        let title = t.value.common.systemNotification
        if (type === "announcement") title = t.value.messagesPage.announcement
        else if (type === "promotion") title = t.value.messagesPage.promotion
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

        return {
          id: String(m.message_id || m.id),
          type,
          rawTitle: title,
          rawContent: content,
          time: formatBackendDate(m.created_at),
          isRead: m.status === 1,
        }
      })
    }
  } catch (e) {
    console.error(e)
  } finally {
    if (showLoading) loading.value = false
  }
}

async function markAllAsRead() {
  if (markAllLoading.value) return
  const unreadIds = messageList.value.filter((m) => !m.isRead).map((m) => m.id)
  if (unreadIds.length === 0) return
  markAllLoading.value = true
  try {
    await apiClient("/api/messages/read", { method: "PUT", body: JSON.stringify({ message_ids: unreadIds }) })
    messageList.value = messageList.value.map((m) => ({ ...m, isRead: true }))
    await syncUnreadCount()
    toast.success(t.value.messagesPage.markReadSuccess)
  } catch {
    // apiClient handles toast.
  } finally {
    markAllLoading.value = false
  }
}

async function markAsRead(id: string, showToast = true) {
  if (messageActionLoadingId.value === id) return
  messageActionLoadingId.value = id
  try {
    await apiClient("/api/messages/read", { method: "PUT", body: JSON.stringify({ message_ids: [id] }) })
    messageList.value = messageList.value.map((m) => (m.id === id ? { ...m, isRead: true } : m))
    await syncUnreadCount()
    openMenuId.value = null
    if (showToast) toast.success(t.value.messagesPage.markReadSuccess)
  } catch {
    // apiClient handles toast.
  } finally {
    if (messageActionLoadingId.value === id) messageActionLoadingId.value = null
  }
}

async function deleteMessage(id: string) {
  if (messageActionLoadingId.value === id) return
  messageActionLoadingId.value = id
  try {
    await apiClient("/api/messages/delete", { method: "POST", body: JSON.stringify({ message_ids: [id] }) })
    messageList.value = messageList.value.filter((m) => m.id !== id)
    await syncUnreadCount()
    openMenuId.value = null
    toast.success(t.value.messagesPage.deleteSuccess)
  } catch {
    // apiClient handles toast.
  } finally {
    if (messageActionLoadingId.value === id) messageActionLoadingId.value = null
  }
}

async function handleViewDetail(message: Message) {
  if (detailLoadingId.value || messageActionLoadingId.value === message.id) return
  detailLoadingId.value = message.id
  try {
    if (!message.isRead) await markAsRead(message.id, false)
    const detail = await apiClient(`/api/messages/${message.id}`)
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
    toast.error("Failed to load message detail")
  } finally {
    if (detailLoadingId.value === message.id) detailLoadingId.value = null
  }
}

const messagesPolling = usePolling(
  async () => {
    await fetchMessages(false, true)
    await syncUnreadCount()
  },
  { shouldPoll: () => !detailModalOpen.value && !markAllLoading.value && !messageActionLoadingId.value && !detailLoadingId.value },
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

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.messagesPage.title }}</h1>
            <p class="mt-2 text-muted-foreground">{{ unreadCountText() }}</p>
          </div>
          <div v-if="listUnreadCount > 0" class="flex justify-end">
        <button class="btn btn-outline rounded-lg bg-white/80 shadow-sm hover:border-primary/25 hover:bg-primary/10 hover:text-primary" :disabled="markAllLoading" @click="markAllAsRead">
          <Loader2 v-if="markAllLoading" class="h-4 w-4 animate-spin" />
          <CheckCheck v-else class="h-4 w-4" />
          {{ t.messagesPage.markAllAsRead }}
        </button>
          </div>
        </div>

    <div class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="border-b border-slate-100 bg-white px-4 pt-4 md:px-8 md:pt-6">
        <div class="flex flex-wrap gap-10">
          <button
            :class="['relative cursor-pointer whitespace-nowrap px-1 pb-7 text-base font-medium transition-colors duration-200', selectedType === null ? 'text-primary' : 'text-[#111827] hover:text-primary']"
            @click="selectedType = null; page = 1"
          >
            {{ t.messagesPage.all }} <span class="ml-2 text-sm text-muted-foreground">{{ messageList.length }}</span>
            <span v-if="selectedType === null" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
          </button>
          <button
            v-for="(config, type) in typeConfig"
            :key="type"
            :class="['relative inline-flex cursor-pointer items-center gap-2 whitespace-nowrap px-1 pb-7 text-base font-medium transition-colors duration-200', selectedType === type ? 'text-primary' : 'text-[#111827] hover:text-primary']"
            @click="selectedType = type; page = 1"
          >
            <component :is="config.icon" class="h-4 w-4" />
            {{ config.label }}
            <span class="text-sm text-muted-foreground">{{ messageList.filter((m) => m.type === type).length }}</span>
            <span v-if="selectedType === type" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
          </button>
        </div>
      </div>

      <div v-if="loading" class="flex items-center justify-center gap-2 px-4 py-16 text-muted-foreground">
        <Loader2 class="h-5 w-5 animate-spin text-primary" />
        <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="filteredMessages.length === 0" class="flex flex-col items-center justify-center px-4 py-16 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><MessageSquare class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.messagesPage.noMessages }}</h3>
        <p class="text-muted-foreground">{{ t.messagesPage.noMessagesDesc }}</p>
      </div>
      <div v-else>
        <div
          v-for="message in paginatedMessages"
          :key="message.id"
          :class="['group relative flex cursor-pointer items-start gap-4 border-b border-slate-100 px-4 py-4 transition-colors hover:bg-primary/10', !message.isRead ? 'bg-primary/5' : '', detailLoadingId === message.id ? 'pointer-events-none opacity-75' : '']"
          @click="handleViewDetail(message)"
        >
          <div :class="['flex h-10 w-10 shrink-0 items-center justify-center rounded-lg', configFor(message.type).iconBg, !message.isRead && 'ring-2 ring-primary/25']">
            <component :is="configFor(message.type).icon" :class="['h-5 w-5', configFor(message.type).iconColor]" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <span v-if="!message.isRead" class="rounded-full bg-primary px-2 py-0.5 text-[11px] font-bold text-primary-foreground shadow-sm shadow-primary/20">
                {{ unreadLabel() }}
              </span>
              <h3 :class="['line-clamp-2 text-base text-card-foreground', !message.isRead ? 'font-bold' : 'font-semibold']">{{ localizedMessageTitle(message.rawTitle, configFor(message.type).label) }}</h3>
              <span class="badge">{{ configFor(message.type).label }}</span>
            </div>
            <span class="text-xs text-muted-foreground">{{ message.time }}</span>
          </div>
          <div :class="['flex items-center gap-2 transition-opacity group-hover:opacity-100', detailLoadingId === message.id ? 'opacity-100' : 'opacity-0']">
            <div class="relative">
              <button class="btn btn-ghost h-8 rounded-lg px-2" :disabled="messageActionLoadingId === message.id || detailLoadingId === message.id" @click.stop="openMenuId = openMenuId === message.id ? null : message.id">
                <MoreHorizontal class="h-4 w-4" />
              </button>
              <div v-if="openMenuId === message.id" class="absolute right-0 top-9 z-50 min-w-36 overflow-hidden rounded-lg bg-white p-1 shadow-md" @click.stop>
                <button v-if="!message.isRead" class="flex w-full items-center rounded-lg px-2 py-1.5 text-sm hover:bg-muted disabled:cursor-not-allowed disabled:opacity-60" :disabled="messageActionLoadingId === message.id" @click="markAsRead(message.id)">
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
            <Loader2 v-if="detailLoadingId === message.id" class="h-5 w-5 animate-spin text-primary" />
            <ChevronRight v-else class="h-5 w-5 text-muted-foreground" />
          </div>
        </div>
        <div class="flex items-center justify-between px-4 py-3 text-sm text-muted-foreground">
          <span>{{ messageRangeLabel }}</span>
          <div class="flex items-center gap-2">
            <button class="rounded-lg border border-slate-200 px-3 py-1.5 font-medium transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :disabled="page <= 1" @click="goToPage(page - 1)">
              {{ lang === "zh" ? "上一页" : "Previous" }}
            </button>
            <span class="min-w-20 text-center">{{ page }} / {{ totalPages }}</span>
            <button class="rounded-lg border border-slate-200 px-3 py-1.5 font-medium transition-colors hover:border-primary hover:text-primary disabled:cursor-not-allowed disabled:opacity-50" :disabled="page >= totalPages" @click="goToPage(page + 1)">
              {{ lang === "zh" ? "下一页" : "Next" }}
            </button>
          </div>
        </div>
      </div>
    </div>

      </main>
    </div>

    <div v-if="detailModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="detailModalOpen = false">
      <div class="w-full max-w-xl rounded-[16px] bg-white p-4 shadow-2xl">
        <div class="flex items-start justify-between gap-4">
          <div>
            <div class="mb-2 flex flex-wrap items-start gap-2">
              <h2 class="min-w-0 flex-1 text-xl font-semibold leading-snug">{{ localizedMessageTitle(selectedMessageDetail?.rawTitle || '', t.messagesPage.systemNotice) }}</h2>
              <span v-if="selectedMessageDetail?.typeLabel" class="badge shrink-0">{{ selectedMessageDetail.typeLabel }}</span>
            </div>
            <p class="text-sm text-muted-foreground">{{ selectedMessageDetail?.time }}</p>
          </div>
          <button class="text-xl leading-none text-muted-foreground transition-colors hover:text-foreground" @click="detailModalOpen = false">x</button>
        </div>
        <div class="mt-4 border-t border-border pt-4 text-sm leading-relaxed text-foreground" v-html="markdownToHtml(selectedMessageDetail?.rawContent || '')" />
      </div>
    </div>
  </AppShell>
</template>

