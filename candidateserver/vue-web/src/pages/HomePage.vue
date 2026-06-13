<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { BookOpen, CheckCircle2, Crown, GraduationCap, MessageSquare } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import StatsCard from "@/components/StatsCard.vue"
import TodoList from "@/components/TodoList.vue"
import { apiClient } from "@/lib/apiClient"
import { getCachedDashboard } from "@/lib/dashboardCache"
import { useTranslation } from "@/lib/language"

type DashboardStats = {
  courses_in_progress?: number
  certifications_earned?: number
  membership_level?: string
  unread_messages?: number
}

const { t, lang } = useTranslation()
const userName = ref("...")
const unreadCount = ref(0)
const stats = ref<DashboardStats>({})

const todoItems = computed(() =>
  unreadCount.value > 0
    ? [
        {
          id: "message-unread",
          icon: "message" as const,
          title: lang.value === "zh" ? `你有 ${unreadCount.value} 条未读消息` : `You have ${unreadCount.value} unread messages`,
          action: { label: t.value.home.view, href: "/messages" },
        },
      ]
    : [],
)

onMounted(async () => {
  try {
    const payload = await apiClient("/api/user/me")
    const nameToSet = payload?.display_name || payload?.name
    if (nameToSet) {
      userName.value = nameToSet
      localStorage.setItem("user_name", nameToSet)
    }
  } catch {
    const localName = localStorage.getItem("user_name")
    if (localName) userName.value = localName
  }

  try {
    const dashboard = await getCachedDashboard()
    if (dashboard?.unread_messages_count !== undefined) unreadCount.value = dashboard.unread_messages_count
    if (dashboard?.stats) stats.value = dashboard.stats
  } catch (err) {
    console.error(err)
  }
})
</script>

<template>
  <AppShell content-class="px-4 py-4">
    <div class="mb-4 overflow-hidden rounded-3xl border border-border bg-card shadow-sm">
      <div class="bg-[#eef8fa] p-4">
        <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-primary/20 bg-white px-3 py-1 text-xs font-medium text-primary">
          <GraduationCap class="h-3.5 w-3.5" />
          {{ t.sidebar.systemName }}
        </div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.sidebar.home }}</h1>
        <p class="mt-2 text-muted-foreground">{{ t.home.welcomeBack }}，{{ userName }}</p>
      </div>
    </div>

    <div class="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatsCard :title="t.home.courseInProgress" :value="String(stats.courses_in_progress || 0)" :icon="BookOpen" variant="primary" :description="t.courses.tabs.my" href="/courses" />
      <StatsCard :title="t.home.certified" :value="String(stats.certifications_earned || 0)" :icon="CheckCircle2" variant="success" :description="t.sidebar.certificates" href="/certificates" />
      <StatsCard :title="t.home.memberLevel" :value="stats.membership_level || t.common.na" :icon="Crown" variant="warning" :description="t.membership.title" href="/membership" />
      <StatsCard :title="t.home.unreadMessages" :value="String(unreadCount)" :icon="MessageSquare" variant="info" :description="t.home.unreadMessagesCount" href="/messages" />
    </div>

    <TodoList :items="todoItems" />
  </AppShell>
</template>
