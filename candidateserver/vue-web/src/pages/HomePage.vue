<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { BookOpen, CheckCircle2, PanelLeft, MessageSquare } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import StatsCard from "@/components/StatsCard.vue"
import TodoList from "@/components/TodoList.vue"
import { apiClient } from "@/lib/apiClient"
import { getAccessToken } from "@/lib/authStorage"
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

const guideCopy = computed(() => lang.value === "zh"
  ? {
      title: "欢迎来到门户",
      subtitle: "探索我们的认证、课程、网络研讨会、洞察和报告，持续提升你的专业知识。",
    }
  : {
      title: "Welcome to Portal",
      subtitle: "Explore our collection of certifications, courses, webinars, insights and reports to advance your knowledge.",
    },
)
const statsActionCopy = computed(() => lang.value === "zh"
  ? {
      certifications: "查看认证详情",
      certificates: "查看证书",
      messages: "查看消息",
    }
  : {
      certifications: "View certifications",
      certificates: "View certificates",
      messages: "View messages",
    },
)
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
  const token = getAccessToken()
  if (!token) {
    const localName = localStorage.getItem("user_name")
    if (localName) userName.value = localName
    return
  }

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
      <div class="mb-4 px-1 py-3 md:py-5">
        <div class="flex items-start gap-3">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-accent text-primary">
            <PanelLeft class="h-6 w-6" />
          </div>
          <div>
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.sidebar.home }}</h1>
          </div>
        </div>
      </div>

      <section class="relative rounded-[16px] bg-white px-4 py-6 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)] md:px-8 md:py-8">
        <h2 class="text-2xl font-bold tracking-tight text-primary md:text-3xl">{{ guideCopy.title }}</h2>
        <p class="mx-auto mt-3 max-w-3xl text-sm leading-6 text-muted-foreground md:text-base">{{ guideCopy.subtitle }}</p>
      </section>

      <section class="rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <StatsCard :title="t.home.courseInProgress" :value="String(stats.courses_in_progress || 0)" :icon="BookOpen" variant="primary" :description="t.courses.tabs.my" :action-label="statsActionCopy.certifications" href="/certifications" />
          <StatsCard :title="t.home.certified" :value="String(stats.certifications_earned || 0)" :icon="CheckCircle2" variant="warning" :description="t.sidebar.certificates" :action-label="statsActionCopy.certificates" href="/certificates" />
          <StatsCard :title="t.home.unreadMessages" :value="String(unreadCount)" :icon="MessageSquare" variant="info" :description="t.home.unreadMessagesCount" :action-label="statsActionCopy.messages" href="/messages" />
        </div>
      </section>

      <TodoList :items="todoItems" />
    </div>
  </AppShell>
</template>
