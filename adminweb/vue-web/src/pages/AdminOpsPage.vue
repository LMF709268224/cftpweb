<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { useAdminLanguage } from "@/lib/language"

type JsonRecord = Record<string, unknown>
type PaginationMode = "offset" | "page"

type OpsFilter = {
  key: string
  label: string
  placeholder?: string
}

type OpsAction = {
  key: "retry" | "ignore"
  label: string
  path: (id: string) => string
  method?: "POST"
}

type OpsModule = {
  key: string
  label: string
  description: string
  listPath: string
  itemKeys: string[]
  idKeys: string[]
  pagination: PaginationMode
  filters?: OpsFilter[]
  requiredFilters?: string[]
  detailPath?: (id: string) => string
  detailIDFormat?: "ulid"
  actions?: OpsAction[]
}

const { t } = useAdminLanguage()
const copy = computed(() => t.value.adminOps)

const modules = computed<OpsModule[]>(() => [
  {
    key: "paySubscriptions",
    label: copy.value.tabs.paySubscriptions,
    description: copy.value.descriptions.paySubscriptions,
    listPath: "/api/pay/subscriptions",
    itemKeys: ["subscriptions", "items"],
    idKeys: ["subscription_ulid", "subscriptionUlid", "stripe_subscription_id", "stripeSubscriptionId", "order_ulid", "orderUlid"],
    pagination: "page",
    filters: [
      { key: "customer_ulid", label: "customer_ulid" },
      { key: "status", label: "status" },
    ],
  },
  {
    key: "payWebhooks",
    label: copy.value.tabs.payWebhooks,
    description: copy.value.descriptions.payWebhooks,
    listPath: "/api/pay/webhook-events",
    detailPath: (id) => `/api/pay/webhook-events/${encodeURIComponent(id)}`,
    itemKeys: ["events", "items"],
    idKeys: ["event_id", "eventId", "stripe_event_id", "stripeEventId"],
    pagination: "page",
    filters: [
      { key: "event_type", label: "event_type" },
      { key: "processed_status", label: "processed_status" },
      { key: "start_time", label: "start_time" },
      { key: "end_time", label: "end_time" },
    ],
  },
  {
    key: "payOrderItems",
    label: copy.value.tabs.payOrderItems,
    description: copy.value.descriptions.payOrderItems,
    listPath: "/api/pay/order-items",
    itemKeys: ["items"],
    idKeys: ["id", "item_id", "itemId"],
    pagination: "page",
    requiredFilters: ["order_ulid"],
    filters: [{ key: "order_ulid", label: "order_ulid", placeholder: copy.value.placeholders.required }],
  },
  {
    key: "mallMailTasks",
    label: copy.value.tabs.mallMailTasks,
    description: copy.value.descriptions.mallMailTasks,
    listPath: "/api/mall/mail-tasks",
    detailPath: (id) => `/api/mall/mail-tasks/${encodeURIComponent(id)}`,
    itemKeys: ["items"],
    idKeys: ["mail_task_ulid", "mailTaskUlid"],
    pagination: "offset",
    filters: [
      { key: "candidate_ulid", label: "candidate_ulid" },
      { key: "order_ulid", label: "order_ulid" },
      { key: "task_status", label: "task_status" },
      { key: "mail_type", label: "mail_type" },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/mall/mail-tasks/${encodeURIComponent(id)}/retry` },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/mall/mail-tasks/${encodeURIComponent(id)}/ignore` },
    ],
  },
  {
    key: "mallNats",
    label: copy.value.tabs.mallNats,
    description: copy.value.descriptions.mallNats,
    listPath: "/api/mall/nats-messages",
    detailPath: (id) => `/api/mall/nats-messages/${encodeURIComponent(id)}`,
    itemKeys: ["items"],
    idKeys: ["message_ulid", "messageUlid"],
    pagination: "offset",
    filters: [
      { key: "receive_status", label: "receive_status" },
      { key: "source_service", label: "source_service" },
      { key: "subject", label: "subject" },
      { key: "message_type", label: "message_type" },
    ],
  },
  {
    key: "progMailTasks",
    label: copy.value.tabs.progMailTasks,
    description: copy.value.descriptions.progMailTasks,
    listPath: "/api/prog/mail-tasks",
    detailPath: (id) => `/api/prog/mail-tasks/${encodeURIComponent(id)}`,
    itemKeys: ["tasks", "items"],
    idKeys: ["mail_task_ulid", "mailTaskUlid"],
    pagination: "offset",
    requiredFilters: ["candidate_ulid"],
    filters: [
      { key: "candidate_ulid", label: "candidate_ulid", placeholder: copy.value.placeholders.required },
      { key: "pipeline_ulid", label: "pipeline_ulid" },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/prog/mail-tasks/${encodeURIComponent(id)}/retry` },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/prog/mail-tasks/${encodeURIComponent(id)}/ignore` },
    ],
  },
  {
    key: "progStages",
    label: copy.value.tabs.progStages,
    description: copy.value.descriptions.progStages,
    listPath: "/api/prog/stages",
    detailPath: (id) => `/api/prog/stages/${encodeURIComponent(id)}`,
    itemKeys: ["stages", "items"],
    idKeys: ["stage_ulid", "stageUlid"],
    pagination: "offset",
    requiredFilters: ["pipeline_ulid"],
    filters: [{ key: "pipeline_ulid", label: "pipeline_ulid", placeholder: copy.value.placeholders.required }],
  },
  {
    key: "progCourseUnits",
    label: copy.value.tabs.progCourseUnits,
    description: copy.value.descriptions.progCourseUnits,
    listPath: "/api/prog/course-units",
    detailPath: (id) => `/api/prog/course-units/${encodeURIComponent(id)}`,
    itemKeys: ["course_units", "courseUnits", "items"],
    idKeys: ["course_unit_ulid", "courseUnitUlid"],
    pagination: "offset",
    requiredFilters: ["pipeline_ulid"],
    filters: [
      { key: "pipeline_ulid", label: "pipeline_ulid", placeholder: copy.value.placeholders.required },
      { key: "stage_ulid", label: "stage_ulid" },
      { key: "status", label: "status" },
    ],
  },
  {
    key: "progDriverEvents",
    label: copy.value.tabs.progDriverEvents,
    description: copy.value.descriptions.progDriverEvents,
    listPath: "/api/prog/driver-events",
    detailPath: (id) => `/api/prog/driver-events/${encodeURIComponent(id)}`,
    itemKeys: ["items"],
    idKeys: ["event_ulid", "eventUlid"],
    pagination: "offset",
    filters: [
      { key: "entity_type", label: "entity_type" },
      { key: "entity_ulid", label: "entity_ulid" },
      { key: "event_status", label: "event_status" },
      { key: "event_type", label: "event_type" },
    ],
  },
  {
    key: "progNats",
    label: copy.value.tabs.progNats,
    description: copy.value.descriptions.progNats,
    listPath: "/api/prog/nats-messages",
    detailPath: (id) => `/api/prog/nats-messages/${encodeURIComponent(id)}`,
    itemKeys: ["items"],
    idKeys: ["message_ulid", "messageUlid"],
    pagination: "offset",
    filters: [
      { key: "receive_status", label: "receive_status" },
      { key: "source_service", label: "source_service" },
    ],
  },
  {
    key: "examAudit",
    label: copy.value.tabs.examAudit,
    description: copy.value.descriptions.examAudit,
    listPath: "/api/exam-ops/audit-messages",
    detailPath: (id) => `/api/exam-ops/audit-messages/${encodeURIComponent(id)}`,
    detailIDFormat: "ulid",
    itemKeys: ["audit_messages", "auditMessages", "items"],
    idKeys: ["message_ulid", "messageUlid"],
    pagination: "page",
    filters: [
      { key: "processed_status", label: "processed_status" },
      { key: "event_type", label: "event_type" },
      { key: "start_time", label: "start_time" },
      { key: "end_time", label: "end_time" },
    ],
  },
  {
    key: "examTransitions",
    label: copy.value.tabs.examTransitions,
    description: copy.value.descriptions.examTransitions,
    listPath: "/api/exam-ops/status-transitions",
    itemKeys: ["transitions", "items"],
    idKeys: ["transition_ulid", "transitionUlid", "id"],
    pagination: "page",
    filters: [
      { key: "exam_ulid", label: "exam_ulid" },
      { key: "status_type", label: "status_type" },
    ],
  },
  {
    key: "examReminders",
    label: copy.value.tabs.examReminders,
    description: copy.value.descriptions.examReminders,
    listPath: "/api/exam-ops/reminder-mails",
    detailPath: (id) => `/api/exam-ops/reminder-mails/${encodeURIComponent(id)}`,
    itemKeys: ["mails", "items"],
    idKeys: ["mail_ulid", "mailUlid"],
    pagination: "page",
    filters: [
      { key: "exam_ulid", label: "exam_ulid" },
      { key: "task_status", label: "task_status" },
      { key: "delivery_status", label: "delivery_status" },
      { key: "candidate_email", label: "candidate_email" },
      { key: "reminder_type", label: "reminder_type" },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/exam-ops/reminder-mails/${encodeURIComponent(id)}/retry` },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/exam-ops/reminder-mails/${encodeURIComponent(id)}/ignore` },
    ],
  },
])

const activeKey = ref(modules.value[0]?.key || "")
const filters = reactive<Record<string, Record<string, string>>>({})
const items = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const detail = ref<unknown>(null)
const detailNotice = ref("")
const loading = ref(false)
const detailLoading = ref(false)
const actionLoading = ref("")
const total = ref(0)
const page = ref(1)
const offset = ref(0)
const pageSize = 20

const activeModule = computed(() => modules.value.find((module) => module.key === activeKey.value) || modules.value[0])
const activeFilters = computed(() => filters[activeModule.value.key] || (filters[activeModule.value.key] = {}))
const missingRequiredFilters = computed(() =>
  (activeModule.value.requiredFilters || []).filter((key) => !String(activeFilters.value[key] || "").trim()),
)
const canLoad = computed(() => missingRequiredFilters.value.length === 0)

function isRecord(value: unknown): value is JsonRecord {
  return !!value && typeof value === "object" && !Array.isArray(value)
}

function getField(item: JsonRecord | null, keys: string[]): unknown {
  if (!item) return undefined
  for (const key of keys) {
    if (item[key] !== undefined && item[key] !== null && item[key] !== "") return item[key]
  }
  return undefined
}

function getItemID(item: JsonRecord | null, module = activeModule.value) {
  const value = getField(item, module.idKeys)
  return value === undefined ? "" : String(value)
}

function isULID(value: string) {
  return /^[0-9A-HJKMNP-TV-Z]{26}$/i.test(value)
}

function getTitle(item: JsonRecord) {
  const title = getField(item, ["title", "name", "subject", "event_type", "eventType", "message_type", "messageType"])
  return title === undefined ? getItemID(item) || copy.value.untitled : String(title)
}

function getSubtitle(item: JsonRecord) {
  const parts = [
    getField(item, ["task_status", "taskStatus", "processed_status", "processedStatus", "event_status", "eventStatus", "status"]),
    getField(item, ["created_at", "createdAt", "scheduled_at", "scheduledAt"]),
  ].filter((value) => value !== undefined && value !== "")
  return parts.map(String).join(" · ")
}

function extractItems(data: JsonRecord, module: OpsModule) {
  for (const key of module.itemKeys) {
    const value = data[key]
    if (Array.isArray(value)) {
      return value.filter(isRecord)
    }
  }
  return []
}

function stringify(value: unknown) {
  return JSON.stringify(value, null, 2)
}

function buildListURL(module: OpsModule) {
  const params = new URLSearchParams()
  for (const filter of module.filters || []) {
    const value = String(activeFilters.value[filter.key] || "").trim()
    if (value) params.set(filter.key, value)
  }
  if (module.pagination === "page") {
    params.set("page", String(page.value))
    params.set("page_size", String(pageSize))
  } else {
    params.set("limit", String(pageSize))
    params.set("offset", String(offset.value))
  }
  return `${module.listPath}?${params.toString()}`
}

async function loadList(reset = false) {
  if (reset) {
    page.value = 1
    offset.value = 0
  }
  selected.value = null
  detail.value = null
  detailNotice.value = ""
  if (!canLoad.value) {
    items.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const module = activeModule.value
    const data = await apiClient<JsonRecord>(buildListURL(module))
    items.value = extractItems(data, module)
    total.value = Number(data.total || items.value.length || 0)
    if (items.value.length) {
      await openItem(items.value[0])
    }
  } catch (error) {
    items.value = []
    total.value = 0
    toast.error(error instanceof Error ? error.message : copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

async function openItem(item: JsonRecord) {
  selected.value = item
  detail.value = item
  detailNotice.value = ""
  const module = activeModule.value
  const id = getItemID(item, module)
  if (!module.detailPath || !id) return
  if (module.detailIDFormat === "ulid" && !isULID(id)) {
    detailNotice.value = copy.value.invalidDetailId(id)
    return
  }
  detailLoading.value = true
  try {
    detail.value = await apiClient<JsonRecord>(module.detailPath(id))
  } catch (error) {
    toast.error(error instanceof Error ? error.message : copy.value.toasts.detailFailed)
  } finally {
    detailLoading.value = false
  }
}

async function runAction(action: OpsAction) {
  const module = activeModule.value
  const id = getItemID(selected.value, module)
  if (!id) return
  if (!window.confirm(copy.value.confirmAction(action.label))) return
  actionLoading.value = action.key
  try {
    await apiClient(action.path(id), { method: action.method || "POST", body: JSON.stringify({}) })
    toast.success(copy.value.toasts.actionSuccess)
    await loadList()
  } catch (error) {
    toast.error(error instanceof Error ? error.message : copy.value.toasts.actionFailed)
  } finally {
    actionLoading.value = ""
  }
}

function previousPage() {
  if (activeModule.value.pagination === "page") {
    if (page.value <= 1) return
    page.value -= 1
  } else {
    offset.value = Math.max(0, offset.value - pageSize)
  }
  void loadList()
}

function nextPage() {
  if (activeModule.value.pagination === "page") {
    page.value += 1
  } else {
    offset.value += pageSize
  }
  void loadList()
}

watch(activeKey, () => {
  page.value = 1
  offset.value = 0
  void loadList()
})

void loadList()
</script>

<template>
  <section class="space-y-6">
    <header class="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
      <div>
        <p class="text-xs font-black uppercase tracking-[0.2em] text-slate-400">{{ copy.eyebrow }}</p>
        <h1 class="mt-2 text-3xl font-black text-slate-950">{{ copy.title }}</h1>
        <p class="mt-2 text-sm font-medium text-slate-500">{{ copy.subtitle }}</p>
      </div>
      <button class="rounded-xl border border-slate-300 bg-white px-5 py-3 text-sm font-black text-slate-900 shadow-sm hover:border-slate-500" @click="loadList(true)">
        {{ copy.refresh }}
      </button>
    </header>

    <div class="rounded-[1.5rem] border border-slate-200 bg-white p-4 shadow-sm">
      <div class="flex gap-3 overflow-x-auto pb-2">
        <button
          v-for="module in modules"
          :key="module.key"
          class="shrink-0 rounded-2xl border px-4 py-3 text-left text-sm font-black transition"
          :class="module.key === activeKey ? 'border-sky-300 bg-sky-50 text-sky-700' : 'border-slate-200 bg-white text-slate-600 hover:border-slate-400'"
          @click="activeKey = module.key"
        >
          {{ module.label }}
        </button>
      </div>
    </div>

    <div class="grid gap-6 xl:grid-cols-[420px_1fr]">
      <div class="overflow-hidden rounded-[1.5rem] border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-100 p-5">
          <h2 class="text-xl font-black text-slate-950">{{ activeModule.label }}</h2>
          <p class="mt-1 text-sm font-medium text-slate-500">{{ activeModule.description }}</p>
        </div>

        <div v-if="activeModule.filters?.length" class="space-y-3 border-b border-slate-100 p-5">
          <div v-for="filter in activeModule.filters" :key="filter.key" class="space-y-1">
            <label class="text-xs font-black text-slate-500">{{ filter.label }}</label>
            <input
              v-model="activeFilters[filter.key]"
              class="h-11 w-full rounded-xl border border-slate-200 px-3 text-sm font-bold outline-none focus:border-sky-400"
              :placeholder="filter.placeholder || copy.placeholders.optional"
              @keyup.enter="loadList(true)"
            />
          </div>
          <button class="h-11 w-full rounded-xl bg-blue-700 text-sm font-black text-white hover:bg-blue-800" @click="loadList(true)">
            {{ copy.search }}
          </button>
        </div>

        <div v-if="missingRequiredFilters.length" class="p-6 text-sm font-bold text-amber-700">
          {{ copy.requiredPrefix }} {{ missingRequiredFilters.join(", ") }}
        </div>
        <div v-else-if="loading" class="p-10 text-center text-sm font-bold text-slate-500">{{ copy.loading }}</div>
        <div v-else-if="!items.length" class="p-10 text-center text-sm font-bold text-slate-500">{{ copy.empty }}</div>
        <div v-else class="divide-y divide-slate-100">
          <button
            v-for="item in items"
            :key="getItemID(item) || stringify(item)"
            class="block w-full p-5 text-left transition hover:bg-slate-50"
            :class="getItemID(item) === getItemID(selected) ? 'bg-sky-50' : ''"
            @click="openItem(item)"
          >
            <p class="font-black text-slate-950">{{ getTitle(item) }}</p>
            <p class="mt-1 break-all text-xs font-bold text-blue-700">{{ getItemID(item) || "-" }}</p>
            <p class="mt-1 text-xs font-semibold text-slate-500">{{ getSubtitle(item) || "-" }}</p>
          </button>
        </div>

        <div class="flex items-center justify-between border-t border-slate-100 p-5">
          <span class="text-xs font-black text-slate-500">{{ copy.total(total) }}</span>
          <div class="flex gap-2">
            <button class="rounded-xl border border-slate-300 px-4 py-2 text-sm font-bold disabled:opacity-40" :disabled="page <= 1 && offset <= 0" @click="previousPage">
              {{ copy.previous }}
            </button>
            <button class="rounded-xl border border-slate-300 px-4 py-2 text-sm font-bold" :disabled="items.length < pageSize" @click="nextPage">
              {{ copy.next }}
            </button>
          </div>
        </div>
      </div>

      <div class="min-h-[520px] rounded-[1.5rem] border border-slate-200 bg-white p-6 shadow-sm">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 class="text-xl font-black text-slate-950">{{ copy.detailTitle }}</h2>
            <p class="mt-1 break-all text-xs font-bold text-blue-700">{{ getItemID(selected) || "-" }}</p>
          </div>
          <div v-if="activeModule.actions?.length && selected" class="flex flex-wrap gap-2">
            <button
              v-for="action in activeModule.actions"
              :key="action.key"
              class="rounded-xl border border-slate-300 px-4 py-2 text-sm font-black hover:border-slate-500 disabled:opacity-50"
              :disabled="!!actionLoading"
              @click="runAction(action)"
            >
              {{ actionLoading === action.key ? copy.processing : action.label }}
            </button>
          </div>
        </div>

        <div v-if="detailNotice" class="mt-6 rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm font-bold text-amber-800">
          {{ detailNotice }}
        </div>
        <div v-if="detailLoading" class="mt-10 text-center text-sm font-bold text-slate-500">{{ copy.loading }}</div>
        <pre v-else class="mt-6 max-h-[640px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs font-semibold leading-relaxed text-slate-100">{{ stringify(detail || selected || {}) }}</pre>
      </div>
    </div>
  </section>
</template>
