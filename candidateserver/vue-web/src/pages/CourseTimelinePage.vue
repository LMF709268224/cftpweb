<script setup lang="ts">
import { onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowLeft, Clock } from "lucide-vue-next"
import { timelineStatusLabel } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

type TimelineLog = {
  transition_ulid?: string
  entity_type?: string
  from_status?: string
  to_status?: string
  reason_code?: string
  reason_message?: string
  trigger_source?: string
  event_type?: string
  created_at?: string
}

const route = useRoute()
const { t } = useTranslation()
const pipelineId = String(route.query.id || "")
const logs = ref<TimelineLog[]>([])
const loading = ref(false)

async function loadTimeline() {
  if (!pipelineId) {
    logs.value = []
    return
  }
  loading.value = true
  try {
    const res = await apiClient(`/api/mall/pipelines/${pipelineId}/timeline`)
    logs.value = res?.logs || []
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadTimeline)
</script>

<template>
  <AppShell content-class="p-4">
    <RouterLink :to="pipelineId ? `/courses/detail?id=${encodeURIComponent(pipelineId)}` : '/courses'" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
      <ArrowLeft class="h-4 w-4" />
      {{ t.learning.timelineBackToCourseDetails || t.common.back }}
    </RouterLink>

    <div class="rounded-[22px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div v-if="loading" class="py-16 text-center text-muted-foreground">{{ t.common.loading }}</div>
      <div v-else-if="logs.length === 0" class="flex flex-col items-center justify-center py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
          <Clock class="h-8 w-8 text-muted-foreground" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.learning.pipelineTimelineEmpty || t.common.na }}</h3>
        <p class="max-w-md text-muted-foreground">{{ t.learning.pipelineTimelineEmptyHint || t.common.na }}</p>
      </div>
      <div v-else class="space-y-4">
        <div v-for="log in logs" :key="log.transition_ulid || `${log.entity_type}-${log.created_at}`" class="rounded-xl bg-background p-4 shadow-sm">
          <div class="mb-2 flex flex-wrap items-center gap-2">
            <span class="badge">{{ log.entity_type || t.common.unknown }}</span>
            <span class="badge border-0 bg-primary/10 text-primary">{{ log.event_type || t.common.unknown }}</span>
            <span class="text-sm text-muted-foreground">{{ log.created_at || t.common.unknown }}</span>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <span>{{ timelineStatusLabel(t, log.entity_type, log.from_status) }}</span>
            <ArrowLeft class="h-3.5 w-3.5 rotate-180" />
            <span>{{ timelineStatusLabel(t, log.entity_type, log.to_status) }}</span>
          </div>
          <div class="mt-2 text-sm text-muted-foreground">{{ log.reason_message || log.reason_code || t.common.unknown }}</div>
          <div class="mt-2 text-xs text-muted-foreground">{{ log.trigger_source || t.common.unknown }}</div>
        </div>
      </div>
    </div>
  </AppShell>
</template>
