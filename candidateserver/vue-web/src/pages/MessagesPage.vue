<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { Bell, CheckCheck, ChevronRight, Circle, CreditCard, FileText, Gift, Loader2, Megaphone, MessageSquare, MoreHorizontal, Trash2 } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

type Message = { id: string; type: string; title: string; content: string; time: string; isRead: boolean }

const { t, lang } = useTranslation()
const selectedType = ref<string | null>(null)
const detailModalOpen = ref(false)
const selectedMessageDetail = ref<any>(null)
const messageList = ref<Message[]>([])
const openMenuId = ref<string | null>(null)
const loading = ref(true)

const typeConfig = computed(() => ({
  system: { icon: Bell, iconBg: "bg-primary/10", iconColor: "text-primary", label: t.value.messagesPage.systemNotice },
  announcement: { icon: Megaphone, iconBg: "bg-blue-500/10", iconColor: "text-blue-600", label: t.value.messagesPage.announcement },
  promotion: { icon: Gift, iconBg: "bg-amber-500/10", iconColor: "text-amber-600", label: t.value.messagesPage.promotion },
  payment: { icon: CreditCard, iconBg: "bg-emerald-500/10", iconColor: "text-emerald-600", label: t.value.messagesPage.payment },
  other: { icon: FileText, iconBg: "bg-zinc-500/10", iconColor: "text-zinc-600", label: t.value.messagesPage.other },
}))

const filteredMessages = computed(() => selectedType.value ? messageList.value.filter((m) => m.type === selectedType.value) : messageList.value)
const unreadCount = computed(() => messageList.value.filter((m) => !m.isRead).length)

function configFor(type: string) {
  return typeConfig.value[type as keyof typeof typeConfig.value] || typeConfig.value.system
}

function unreadCountText() {
  return t.value.messagesPage.unreadCount.replace("{{count}}", String(unreadCount.value))
}

function markReadMenuLabel() {
  return lang.value === "zh" ? "\u6807\u8bb0\u4e3a\u5df2\u8bfb" : "Mark as read"
}

function deleteMenuLabel() {
  return lang.value === "zh" ? "\u5220\u9664" : "Delete"
}

async function fetchMessages() {
  loading.value = true
  try {
    const res = await apiClient("/api/messages?limit=50")
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

        let content = m.payload || ""
        try {
          const parsed = JSON.parse(m.payload)
          title = parsed.title || title
          content = parsed.content || content
        } catch {
          // payload can be plain text.
        }

        return { id: String(m.message_id || m.id), type, title, content, time: formatBackendDate(m.created_at), isRead: m.status === 1 }
      })
    }
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

async function markAllAsRead() {
  const unreadIds = messageList.value.filter((m) => !m.isRead).map((m) => m.id)
  if (unreadIds.length === 0) return
  try {
    await apiClient("/api/messages/read", { method: "PUT", body: JSON.stringify({ message_ids: unreadIds }) })
    messageList.value = messageList.value.map((m) => ({ ...m, isRead: true }))
    toast.success(t.value.messagesPage.markReadSuccess)
  } catch {
    // apiClient handles toast.
  }
}

async function markAsRead(id: string) {
  try {
    await apiClient("/api/messages/read", { method: "PUT", body: JSON.stringify({ message_ids: [id] }) })
    messageList.value = messageList.value.map((m) => (m.id === id ? { ...m, isRead: true } : m))
    openMenuId.value = null
    toast.success(t.value.messagesPage.markReadSuccess)
  } catch {
    // apiClient handles toast.
  }
}

async function deleteMessage(id: string) {
  try {
    await apiClient("/api/messages/delete", { method: "POST", body: JSON.stringify({ message_ids: [id] }) })
    messageList.value = messageList.value.filter((m) => m.id !== id)
    openMenuId.value = null
    toast.success(t.value.messagesPage.deleteSuccess)
  } catch {
    // apiClient handles toast.
  }
}

async function handleViewDetail(message: Message) {
  try {
    if (!message.isRead) await markAsRead(message.id)
    selectedMessageDetail.value = await apiClient(`/api/messages/${message.id}`)
    detailModalOpen.value = true
  } catch {
    toast.error("Failed to load message detail")
  }
}

onMounted(fetchMessages)
</script>

<template>
  <AppShell content-class="p-4">
    <div class="mb-4 px-1 py-3 md:py-5">
      <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.messagesPage.title }}</h1>
      <p class="mt-2 text-muted-foreground">{{ unreadCountText() }}</p>
      <div v-if="unreadCount > 0" class="mt-4 flex justify-end">
        <button class="btn btn-outline rounded-lg bg-white/80 shadow-sm hover:border-primary/25 hover:bg-primary/10 hover:text-primary" @click="markAllAsRead"><CheckCheck class="h-4 w-4" /> {{ t.messagesPage.markAllAsRead }}</button>
      </div>
    </div>

    <div class="mb-4 rounded-md bg-white px-8 pt-6">
      <div class="flex flex-wrap gap-10 border-b border-[#edf0f2]">
        <button
          :class="['relative cursor-pointer whitespace-nowrap px-1 pb-7 text-base font-medium transition-colors duration-200', selectedType === null ? 'text-primary' : 'text-[#111827] hover:text-primary']"
          @click="selectedType = null"
        >
          {{ t.messagesPage.all }} <span class="ml-2 text-sm text-muted-foreground">{{ messageList.length }}</span>
          <span v-if="selectedType === null" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
        </button>
        <button
          v-for="(config, type) in typeConfig"
          :key="type"
          :class="['relative inline-flex cursor-pointer items-center gap-2 whitespace-nowrap px-1 pb-7 text-base font-medium transition-colors duration-200', selectedType === type ? 'text-primary' : 'text-[#111827] hover:text-primary']"
          @click="selectedType = type"
        >
          <component :is="config.icon" class="h-4 w-4" />
          {{ config.label }}
          <span class="text-sm text-muted-foreground">{{ messageList.filter((m) => m.type === type).length }}</span>
          <span v-if="selectedType === type" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
        </button>
      </div>
    </div>

    <div class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div v-if="loading" class="flex items-center justify-center gap-2 px-4 py-16 text-muted-foreground">
        <Loader2 class="h-5 w-5 animate-spin text-primary" />
        <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="filteredMessages.length === 0" class="flex flex-col items-center justify-center px-4 py-16 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><MessageSquare class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.messagesPage.noMessages }}</h3>
        <p class="text-muted-foreground">{{ t.messagesPage.noMessagesDesc }}</p>
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="message in filteredMessages"
          :key="message.id"
          :class="['group flex cursor-pointer items-start gap-4 px-4 py-4 transition-colors hover:bg-primary/10', !message.isRead && 'bg-primary/5']"
          @click="handleViewDetail(message)"
        >
          <div :class="['flex h-10 w-10 shrink-0 items-center justify-center rounded-lg', configFor(message.type).iconBg]">
            <component :is="configFor(message.type).icon" :class="['h-5 w-5', configFor(message.type).iconColor]" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="mb-1 flex items-center gap-2">
              <Circle v-if="!message.isRead" class="h-2 w-2 fill-primary text-primary" />
              <h3 :class="['font-medium text-card-foreground', !message.isRead && 'font-semibold']">{{ message.title || configFor(message.type).label }}</h3>
              <span class="badge">{{ configFor(message.type).label }}</span>
            </div>
            <p class="mb-2 line-clamp-2 text-sm text-muted-foreground">{{ message.content }}</p>
            <span class="text-xs text-muted-foreground">{{ message.time }}</span>
          </div>
          <div class="flex items-center gap-2 opacity-0 transition-opacity group-hover:opacity-100">
            <div class="relative">
              <button class="btn btn-ghost h-8 rounded-lg px-2" @click.stop="openMenuId = openMenuId === message.id ? null : message.id">
                <MoreHorizontal class="h-4 w-4" />
              </button>
              <div v-if="openMenuId === message.id" class="absolute right-0 top-9 z-50 min-w-36 overflow-hidden rounded-lg bg-white p-1 shadow-md" @click.stop>
                <button v-if="!message.isRead" class="flex w-full items-center rounded-lg px-2 py-1.5 text-sm hover:bg-muted" @click="markAsRead(message.id)">
                  <CheckCheck class="mr-2 h-4 w-4" />
                  {{ markReadMenuLabel() }}
                </button>
                <button class="flex w-full items-center rounded-lg px-2 py-1.5 text-sm text-destructive hover:bg-muted" @click="deleteMessage(message.id)">
                  <Trash2 class="mr-2 h-4 w-4" />
                  {{ deleteMenuLabel() }}
                </button>
              </div>
            </div>
            <ChevronRight class="h-5 w-5 text-muted-foreground" />
          </div>
        </div>
      </div>
    </div>

    <div v-if="detailModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="detailModalOpen = false">
      <div class="w-full max-w-xl rounded-[16px] bg-white p-4 shadow-2xl">
        <div class="flex items-start justify-between gap-4">
          <div>
            <h2 class="mb-2 text-xl font-semibold">{{ selectedMessageDetail?.title || t.messagesPage.systemNotice }}</h2>
            <p class="text-sm text-muted-foreground">{{ selectedMessageDetail?.time }}</p>
          </div>
          <button class="text-xl leading-none text-muted-foreground transition-colors hover:text-foreground" @click="detailModalOpen = false">x</button>
        </div>
        <div class="mt-4 border-t border-border pt-4 text-sm leading-relaxed text-foreground" v-html="selectedMessageDetail?.content || ''" />
      </div>
    </div>
  </AppShell>
</template>
