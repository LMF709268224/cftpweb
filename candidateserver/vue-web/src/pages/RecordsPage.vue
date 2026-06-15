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
  <AppShell content-class="p-4">
    <div class="mb-4 overflow-hidden rounded-[16px] bg-white shadow-[0_12px_30px_rgba(15,74,82,0.06)]">
      <div class="flex flex-col gap-4 bg-gradient-to-r from-[#ecfbf7] via-white to-[#f4fbff] p-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div class="mb-3 inline-flex items-center gap-2 rounded-full bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
            <GraduationCap class="h-3.5 w-3.5" />
            {{ t.sidebar.records }}
          </div>
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.recordsPage.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.recordsPage.subtitle }}</p>
        </div>
        <button class="btn btn-primary rounded-lg shadow-sm shadow-primary/20" disabled><Plus class="h-4 w-4" /> {{ t.recordsPage.uploadNew }}</button>
      </div>
    </div>

    <div class="mb-4 grid gap-4 sm:grid-cols-3">
      <div v-for="(config, status) in statusConfig" :key="status" class="group relative overflow-hidden rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:ring-primary/25 hover:shadow-md hover:shadow-primary/10">
        <div :class="['absolute left-0 top-0 h-full w-1', status === 'verified' ? 'bg-primary' : status === 'pending' ? 'bg-amber-500' : 'bg-red-500']" />
        <div class="flex items-center gap-4">
          <div :class="['flex h-11 w-11 items-center justify-center rounded-lg transition-transform group-hover:scale-105', status === 'verified' ? 'bg-primary/10' : status === 'pending' ? 'bg-amber-100' : 'bg-red-100']">
            <component
              :is="config.icon"
              :class="[
                'h-5 w-5',
                config.tone === 'success' && 'text-primary',
                config.tone === 'warning' && 'text-amber-600',
                config.tone === 'danger' && 'text-red-600',
              ]"
            />
          </div>
          <div>
            <p class="text-2xl font-bold text-card-foreground">0</p>
            <p class="text-sm text-muted-foreground">{{ config.label }}</p>
          </div>
        </div>
      </div>
    </div>

    <div class="overflow-hidden rounded-[16px] bg-white shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="flex items-center gap-3 bg-white px-4 py-4">
        <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10"><GraduationCap class="h-4 w-4 text-primary" /></div>
        <h2 class="font-semibold text-card-foreground">{{ t.recordsPage.myRecords }}</h2>
      </div>
      <div v-if="records.length === 0" class="flex flex-col items-center justify-center px-4 py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10"><FileText class="h-8 w-8 text-primary" /></div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.recordsPage.noRecords }}</h3>
        <p class="max-w-md text-sm text-muted-foreground">{{ t.recordsPage.noRecordsDesc }}</p>
      </div>
      <div v-else class="space-y-2">
        <div v-for="record in records" :key="record.id" class="group flex items-center justify-between px-4 py-4 transition-colors hover:bg-primary/10">
          <span :class="['badge', statusBadgeClassFromTone(recordStatusConfig(record.status).tone)]">{{ recordStatusConfig(record.status).label }}</span>
          <ChevronRight class="h-5 w-5 text-muted-foreground" />
        </div>
      </div>
    </div>
  </AppShell>
</template>
