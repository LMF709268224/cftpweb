<script setup lang="ts">
import { AlertCircle, CheckCircle2, ChevronRight, Clock, FileText, GraduationCap, Plus } from "lucide-vue-next"
import { statusBadgeClassFromTone } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
const records: any[] = []
const statusConfig = {
  verified: { label: t.value.recordsPage.verified, icon: CheckCircle2, tone: "success" as const },
  pending: { label: t.value.recordsPage.pending, icon: Clock, tone: "warning" as const },
  rejected: { label: t.value.recordsPage.rejected, icon: AlertCircle, tone: "danger" as const },
}

function recordStatusConfig(status: keyof typeof statusConfig) {
  return statusConfig[status]
}
</script>

<template>
  <AppShell>
    <div class="mb-8 flex items-center justify-between">
      <div>
        <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.recordsPage.title }}</h1>
        <p class="mt-1 text-muted-foreground">{{ t.recordsPage.subtitle }}</p>
      </div>
      <button class="btn btn-primary" disabled><Plus class="h-4 w-4" /> {{ t.recordsPage.uploadNew }}</button>
    </div>
    <div class="mb-8 grid gap-4 sm:grid-cols-3">
      <div v-for="(config, status) in statusConfig" :key="status" class="flex items-center gap-4 rounded-xl border border-border bg-card p-4">
        <div :class="['flex h-10 w-10 items-center justify-center rounded-lg', status === 'verified' ? 'bg-blue-100' : status === 'pending' ? 'bg-yellow-100' : 'bg-red-100']">
          <component :is="config.icon" class="h-5 w-5 text-black" />
        </div>
        <div><p class="text-2xl font-bold text-card-foreground">0</p><p class="text-sm text-muted-foreground">{{ config.label }}</p></div>
      </div>
    </div>
    <div class="overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
      <div class="flex items-center gap-3 border-b border-border px-6 py-4">
        <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10"><GraduationCap class="h-4 w-4 text-primary" /></div>
        <h2 class="font-semibold text-card-foreground">{{ t.recordsPage.myRecords }}</h2>
      </div>
      <div v-if="records.length === 0" class="flex flex-col items-center justify-center px-6 py-14 text-center">
        <div class="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-muted"><FileText class="h-7 w-7 text-muted-foreground" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.recordsPage.noRecords }}</h3>
        <p class="max-w-md text-sm text-muted-foreground">{{ t.recordsPage.noRecordsDesc }}</p>
      </div>
      <div v-else class="divide-y divide-border">
        <div v-for="record in records" :key="record.id" class="group flex items-center justify-between p-6 transition-colors hover:bg-muted/50">
          <span :class="['badge', statusBadgeClassFromTone(recordStatusConfig(record.status).tone)]">{{ recordStatusConfig(record.status).label }}</span>
          <ChevronRight class="h-5 w-5 text-muted-foreground" />
        </div>
      </div>
    </div>
  </AppShell>
</template>
