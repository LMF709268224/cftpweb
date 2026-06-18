<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { BookOpen, CheckCircle2, Home, MessageSquare } from "lucide-vue-next"
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
const router = useRouter()
const userName = ref("...")
const unreadCount = ref(0)
const stats = ref<DashboardStats>({})
const welcomeSeparator = computed(() => (lang.value === "zh" ? "\uFF0C" : ", "))

const guideCopy = computed(() => lang.value === "zh"
  ? {
      title: "欢迎来到 CFtP",
      subtitle: "查看你的认证、学习进度、考试和证书状态，快速进入下一步。",
      buyCourses: "购买认证",
    }
  : {
      title: "Welcome to CFtP",
      subtitle: "Review your certifications, learning progress, exams, and certificate status at a glance.",
      buyCourses: "Buy Certifications",
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

function goToCourses() {
  void router.push("/certifications")
}

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
            <Home class="h-6 w-6" />
          </div>
          <div>
            <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.sidebar.home }}</h1>
            <p class="mt-2 text-muted-foreground">{{ t.home.welcomeBack }}{{ welcomeSeparator }}{{ userName }}</p>
          </div>
        </div>
      </div>

      <section class="relative rounded-[16px] bg-white px-4 py-6 text-center shadow-[0_10px_24px_rgba(15,74,82,0.05)] md:px-8 md:py-8">
        <h2 class="text-2xl font-bold tracking-tight text-primary md:text-3xl">{{ guideCopy.title }}</h2>
        <p class="mx-auto mt-3 max-w-3xl text-sm leading-6 text-muted-foreground md:text-base">{{ guideCopy.subtitle }}</p>
        <button
          class="mt-5 inline-flex h-9 items-center justify-center rounded-lg border border-primary/20 bg-white/80 px-5 text-sm font-semibold text-primary shadow-sm transition-colors hover:bg-primary/10 md:absolute md:right-6 md:top-6 md:mt-0"
          @click="goToCourses"
        >
          {{ guideCopy.buyCourses }}
        </button>
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
