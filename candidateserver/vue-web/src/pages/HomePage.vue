<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
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
const welcomeSeparator = computed(() => (lang.value === "zh" ? "\uFF0C" : ", "))

const todoItems = computed(() =>
  unreadCount.value > 0
    ? [
        {
          id: "message-unread",
          icon: "message" as const,
          title: lang.value === "zh" ? `\u4f60\u6709 ${unreadCount.value} \u6761\u672a\u8bfb\u6d88\u606f` : `You have ${unreadCount.value} unread messages`,
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
  <AppShell content-class="p-4">
    <div class="space-y-4">
      <section class="overflow-hidden rounded-[16px] bg-white shadow-[0_12px_30px_rgba(15,74,82,0.06)]">
        <div class="relative flex flex-col gap-4 bg-gradient-to-r from-[#ecfbf7] via-white to-[#f4fbff] p-4 md:flex-row md:items-center md:justify-between">
          <div class="absolute right-8 top-0 h-24 w-24 rounded-full bg-primary/10 blur-3xl" />
          <div class="relative">
            <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
              <GraduationCap class="h-3.5 w-3.5" />
              {{ t.sidebar.systemName }}
            </div>
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.sidebar.home }}</h1>
            <p class="mt-2 text-muted-foreground">{{ t.home.welcomeBack }}{{ welcomeSeparator }}{{ userName }}</p>
          </div>

          <div class="relative grid grid-cols-2 gap-4 sm:min-w-[340px]">
            <RouterLink to="/courses" class="rounded-xl bg-white/85 px-4 py-3 shadow-sm transition-colors hover:bg-white">
              <p class="text-xs text-muted-foreground">{{ t.home.courseInProgress }}</p>
              <p class="mt-1 text-2xl font-bold text-foreground">{{ stats.courses_in_progress || 0 }}</p>
            </RouterLink>
            <RouterLink to="/messages" class="rounded-xl bg-white/85 px-4 py-3 shadow-sm transition-colors hover:bg-white">
              <p class="text-xs text-muted-foreground">{{ t.home.unreadMessages }}</p>
              <p class="mt-1 text-2xl font-bold text-primary">{{ unreadCount }}</p>
            </RouterLink>
          </div>
        </div>
      </section>

      <section class="rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatsCard :title="t.home.courseInProgress" :value="String(stats.courses_in_progress || 0)" :icon="BookOpen" variant="primary" :description="t.courses.tabs.my" href="/courses" />
          <StatsCard :title="t.home.certified" :value="String(stats.certifications_earned || 0)" :icon="CheckCircle2" variant="success" :description="t.sidebar.certificates" href="/certificates" />
          <StatsCard :title="t.home.memberLevel" :value="stats.membership_level || t.common.na" :icon="Crown" variant="warning" :description="t.membership.title" href="/membership" />
          <StatsCard :title="t.home.unreadMessages" :value="String(unreadCount)" :icon="MessageSquare" variant="info" :description="t.home.unreadMessagesCount" href="/messages" />
        </div>
      </section>

      <TodoList :items="todoItems" />
    </div>
  </AppShell>
</template>
