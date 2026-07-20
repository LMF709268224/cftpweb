<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowLeft, Clock, Loader2 } from "lucide-vue-next"
import { timelineStatusLabel } from "@/lib/status-labels"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { formatBackendDate } from "@/lib/utils"
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
const pipelineId = computed(() => String(route.params.pipelineId || route.query.id || ""))
const logs = ref<TimelineLog[]>([])
const loading = ref(false)

async function loadTimeline() {
  if (!pipelineId.value) {
    logs.value = []
    return
  }
  loading.value = true
  try {
    const res = await apiClient(`/api/mall/pipelines/${pipelineId.value}/timeline`)
    logs.value = res?.logs || []
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

function learningLabel(key: string, fallback: string) {
  const learning = t.value.learning as Record<string, string>
  return learning?.[key] || fallback
}

function normalizeEnumValue(value?: string) {
  return String(value || "").trim().toUpperCase()
}

function timelineEntityLabel(entityType?: string) {
  switch (normalizeEnumValue(entityType)) {
    case "PIPELINE":
      return learningLabel("timelineEntityPipeline", "Certification Flow")
    case "STAGE":
      return learningLabel("timelineEntityStage", "Stage")
    case "COURSE_UNIT":
      return learningLabel("timelineEntityCourseUnit", "Course Unit")
    default:
      return entityType || t.value.common.unknown
  }
}

function timelineEventLabel(eventType?: string) {
  switch (normalizeEnumValue(eventType)) {
    case "STAGE_ADVANCED":
      return learningLabel("timelineEventStageAdvanced", "Stage advanced")
    case "STATUS_CHANGED":
    case "STATUS_CHANGE":
      return learningLabel("timelineEventStatusChanged", "Status changed")
    case "CREATED":
      return learningLabel("timelineEventCreated", "Created")
    case "UPDATED":
      return learningLabel("timelineEventUpdated", "Updated")
    default:
      return eventType || t.value.common.unknown
  }
}

function timelineSourceLabel(source?: string) {
  switch (normalizeEnumValue(source)) {
    case "SYSTEM":
      return learningLabel("timelineSourceSystem", "System update")
    case "ADMIN":
      return learningLabel("timelineSourceAdmin", "Admin action")
    case "CANDIDATE":
    case "USER":
      return learningLabel("timelineSourceCandidate", "Candidate action")
    default:
      return source || t.value.common.unknown
  }
}

onMounted(loadTimeline)
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <Clock class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.learning.pipelineTimelineEmpty || t.common.na }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <RouterLink :to="pipelineId ? `/certifications/${encodeURIComponent(pipelineId)}` : '/certifications'" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
          <ArrowLeft class="h-4 w-4" />
          {{ t.learning.timelineBackToCourseDetails || t.common.back }}
        </RouterLink>

    <div class="rounded-[16px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div v-if="loading" class="flex items-center justify-center gap-2 py-16 text-muted-foreground">
        <Loader2 class="h-5 w-5 animate-spin text-primary" />
        <span>{{ t.common.loading }}</span>
      </div>
      <div v-else-if="logs.length === 0" class="flex flex-col items-center justify-center py-14 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-muted">
          <Clock class="h-8 w-8 text-muted-foreground" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ t.learning.pipelineTimelineEmpty || t.common.na }}</h3>
        <p class="max-w-md text-muted-foreground">{{ t.learning.pipelineTimelineEmptyHint || t.common.na }}</p>
      </div>
      <div v-else class="space-y-4">
        <div v-for="log in logs" :key="log.transition_ulid || `${log.entity_type}-${log.created_at}`" class="rounded-lg bg-background p-4 shadow-sm">
          <div class="mb-2 flex flex-wrap items-center gap-2">
            <span class="badge">{{ timelineEntityLabel(log.entity_type) }}</span>
            <span class="badge border-0 bg-primary/10 text-primary">{{ timelineEventLabel(log.event_type) }}</span>
            <span class="text-sm text-muted-foreground">{{ formatBackendDate(log.created_at) || t.common.unknown }}</span>
          </div>
          <div class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <span>{{ timelineStatusLabel(t, log.entity_type, log.from_status) }}</span>
            <ArrowLeft class="h-3.5 w-3.5 rotate-180" />
            <span>{{ timelineStatusLabel(t, log.entity_type, log.to_status) }}</span>
          </div>
          <div class="mt-2 text-sm text-muted-foreground">{{ log.reason_message || log.reason_code || t.common.unknown }}</div>
          <div class="mt-2 text-xs text-muted-foreground">{{ timelineSourceLabel(log.trigger_source) }}</div>
        </div>
      </div>
    </div>
      </main>
    </div>
  </AppShell>
</template>
