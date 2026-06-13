<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { Bell, CheckCheck, ChevronRight, Circle, CreditCard, FileText, Gift, Megaphone, MessageSquare, MoreHorizontal, Trash2 } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
import { useTranslation } from "@/lib/language"

type Message = { id: string; type: string; title: string; content: string; time: string; isRead: boolean }

const { t } = useTranslation()
const selectedType = ref<string | null>(null)
const detailModalOpen = ref(false)
const selectedMessageDetail = ref<any>(null)
const messageList = ref<Message[]>([])
const openMenuId = ref<string | null>(null)

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

async function fetchMessages() {
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
  <AppShell>
    <div class="mb-8 flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.messagesPage.title }}</h1>
        <p class="mt-1 text-muted-foreground">{{ unreadCountText() }}</p>
      </div>
      <button v-if="unreadCount > 0" class="btn btn-outline" @click="markAllAsRead"><CheckCheck class="h-4 w-4" /> {{ t.messagesPage.markAllAsRead }}</button>
    </div>

    <div class="mb-6 flex flex-wrap gap-2">
      <button :class="['rounded-lg px-4 py-2 text-sm font-medium transition-all', selectedType === null ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:text-foreground']" @click="selectedType = null">
        {{ t.messagesPage.all }} <span class="ml-2 rounded-full bg-card/30 px-1.5">{{ messageList.length }}</span>
      </button>
      <button
        v-for="(config, type) in typeConfig"
        :key="type"
        :class="['flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all', selectedType === type ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:text-foreground']"
        @click="selectedType = type"
      >
        <component :is="config.icon" class="h-4 w-4" />
        {{ config.label }}
        <span class="rounded-full bg-card/30 px-1.5">{{ messageList.filter((m) => m.type === type).length }}</span>
      </button>
    </div>

    <div class="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
      <div v-if="filteredMessages.length === 0" class="flex flex-col items-center justify-center py-16 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted"><MessageSquare class="h-8 w-8 text-muted-foreground" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.messagesPage.noMessages }}</h3>
        <p class="text-muted-foreground">{{ t.messagesPage.noMessagesDesc }}</p>
      </div>
      <div v-else class="divide-y divide-border">
        <div
          v-for="message in filteredMessages"
          :key="message.id"
          :class="['group flex cursor-pointer items-start gap-4 p-6 transition-colors hover:bg-muted/50', !message.isRead && 'bg-primary/5']"
          @click="handleViewDetail(message)"
        >
          <div :class="['flex h-10 w-10 shrink-0 items-center justify-center rounded-xl', configFor(message.type).iconBg]">
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
              <button class="btn btn-ghost h-8 px-2" @click.stop="openMenuId = openMenuId === message.id ? null : message.id">
                <MoreHorizontal class="h-4 w-4" />
              </button>
              <div v-if="openMenuId === message.id" class="absolute right-0 top-9 z-50 min-w-36 overflow-hidden rounded-md border bg-card p-1 shadow-md" @click.stop>
                <button v-if="!message.isRead" class="flex w-full items-center rounded-sm px-2 py-1.5 text-sm hover:bg-muted" @click="markAsRead(message.id)">
                  <CheckCheck class="mr-2 h-4 w-4" />
                  标记为已读
                </button>
                <button class="flex w-full items-center rounded-sm px-2 py-1.5 text-sm text-destructive hover:bg-muted" @click="deleteMessage(message.id)">
                  <Trash2 class="mr-2 h-4 w-4" />
                  删除
                </button>
              </div>
            </div>
            <ChevronRight class="h-5 w-5 text-muted-foreground" />
          </div>
        </div>
      </div>
    </div>

    <div v-if="detailModalOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="detailModalOpen = false">
      <div class="w-full max-w-xl rounded-2xl bg-card p-6 shadow-2xl">
        <h2 class="mb-2 text-xl font-semibold">{{ selectedMessageDetail?.title || t.messagesPage.systemNotice }}</h2>
        <p class="text-sm text-muted-foreground">{{ selectedMessageDetail?.time }}</p>
        <div class="mt-4 border-t border-border pt-4 text-sm leading-relaxed text-foreground" v-html="selectedMessageDetail?.content || ''" />
      </div>
    </div>
  </AppShell>
</template>
