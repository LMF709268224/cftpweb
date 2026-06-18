<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { BookOpen, CheckCircle2, MessageSquare, PanelLeft } from "lucide-vue-next"
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
          title: lang.value === "zh" ? `你有 ${unreadCount.value} 条未读消息` : `You have ${unreadCount.value} unread messages`,
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
  <AppShell content-class="p-0">
    <div class="min-h-screen bg-white lg:m-4 lg:overflow-hidden lg:rounded-xl lg:border lg:border-border">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <PanelLeft class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.sidebar.home }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <section class="w-full text-center">
          <h1 class="text-3xl font-bold tracking-tight text-[#6847ff] md:text-4xl">{{ guideCopy.title }}</h1>
          <p class="mx-auto mt-4 max-w-4xl text-base leading-7 text-slate-700">{{ guideCopy.subtitle }}</p>
        </section>

        <section class="mt-10 w-full">
          <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            <StatsCard :title="t.home.courseInProgress" :value="String(stats.courses_in_progress || 0)" :icon="BookOpen" variant="primary" :description="t.courses.tabs.my" :action-label="statsActionCopy.certifications" href="/certifications" />
            <StatsCard :title="t.home.certified" :value="String(stats.certifications_earned || 0)" :icon="CheckCircle2" variant="warning" :description="t.sidebar.certificates" :action-label="statsActionCopy.certificates" href="/certificates" />
            <StatsCard :title="t.home.unreadMessages" :value="String(unreadCount)" :icon="MessageSquare" variant="info" :description="t.home.unreadMessagesCount" :action-label="statsActionCopy.messages" href="/messages" />
          </div>
        </section>

        <section class="mt-6 w-full">
          <TodoList :items="todoItems" />
        </section>
      </main>
    </div>
  </AppShell>
</template>
