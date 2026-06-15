<script setup lang="ts">
import { computed } from "vue"
import { RouterLink } from "vue-router"
import { ChevronRight, Clock, FileCheck, MessageSquare, XCircle } from "lucide-vue-next"
import { useTranslation } from "@/lib/language"

type TodoItem = {
  id: string
  icon: "message" | "file" | "rejected" | "pending"
  title: string
  description?: string
  action: { label: string; href: string }
}

defineProps<{ items: TodoItem[] }>()
const { t } = useTranslation()

const iconMap = { message: MessageSquare, file: FileCheck, rejected: XCircle, pending: Clock }
const iconStyles = {
  message: "bg-blue-500/10 text-blue-600",
  file: "bg-amber-500/10 text-amber-600",
  rejected: "bg-red-500/10 text-red-500",
  pending: "bg-slate-500/10 text-slate-600",
}
const todoCopy = computed(() => ({
  items: t.value.todoList?.items || "items",
  noPendingTasks: t.value.todoList?.noPendingTasks || t.value.common.na,
}))
</script>

<template>
  <div class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
    <div class="flex items-center justify-between bg-white px-5 py-4">
      <div class="flex items-center gap-3">
        <div class="flex h-9 w-9 items-center justify-center rounded-full bg-primary/10">
          <Clock class="h-4 w-4 text-primary" />
        </div>
        <h3 class="font-semibold text-card-foreground">{{ t.home.pendingTasks }}</h3>
      </div>
      <span class="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary">{{ items.length }} {{ todoCopy.items }}</span>
    </div>
    <div class="px-5 pb-5">
      <div v-if="items.length === 0" class="rounded-xl bg-[#f7fbfb] px-4 py-10 text-center text-sm text-muted-foreground">
        {{ todoCopy.noPendingTasks }}
      </div>
      <div v-for="item in items" :key="item.id" class="group flex items-center justify-between rounded-xl bg-[#f7fbfb] px-4 py-4 transition-colors hover:bg-primary/10">
        <div class="flex items-center gap-4">
          <div :class="['flex h-10 w-10 items-center justify-center rounded-lg transition-transform group-hover:scale-105', iconStyles[item.icon]]">
            <component :is="iconMap[item.icon]" class="h-5 w-5" />
          </div>
          <div>
            <p class="font-medium text-card-foreground">{{ item.title }}</p>
            <p v-if="item.description" class="text-sm text-muted-foreground">{{ item.description }}</p>
          </div>
        </div>
        <RouterLink :to="item.action.href" class="flex items-center gap-1 text-sm font-medium text-primary transition-colors hover:text-primary/80">
          {{ item.action.label }}
          <ChevronRight class="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </RouterLink>
      </div>
    </div>
  </div>
</template>
