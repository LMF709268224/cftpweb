<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { Search } from "lucide-vue-next"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate } from "@/lib/display"
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

function fieldLabel(key: string) {
  const labels = copy.value.filterLabels as Record<string, string>
  return labels[key] || key
}

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
      { key: "customer_ulid", label: fieldLabel("customer_ulid") },
      { key: "status", label: fieldLabel("status") },
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
      { key: "event_type", label: fieldLabel("event_type") },
      { key: "processed_status", label: fieldLabel("processed_status") },
      { key: "start_time", label: fieldLabel("start_time") },
      { key: "end_time", label: fieldLabel("end_time") },
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
    filters: [{ key: "order_ulid", label: fieldLabel("order_ulid"), placeholder: copy.value.placeholders.required }],
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
      { key: "candidate_ulid", label: fieldLabel("candidate_ulid") },
      { key: "order_ulid", label: fieldLabel("order_ulid") },
      { key: "task_status", label: fieldLabel("task_status") },
      { key: "mail_type", label: fieldLabel("mail_type") },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/mall/mail-tasks/${encodeURIComponent(id)}/retry` },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/mall/mail-tasks/${encodeURIComponent(id)}/ignore` },
    ],
  },
  {
    key: "mbrMailTasks",
    label: copy.value.tabs.mbrMailTasks,
    description: copy.value.descriptions.mbrMailTasks,
    listPath: "/api/memberships/mails",
    detailPath: (id) => `/api/memberships/mails/${encodeURIComponent(id)}`,
    itemKeys: ["mails", "items"],
    idKeys: ["mail_ulid", "mailUlid"],
    pagination: "page",
    filters: [
      { key: "candidate_ulid", label: fieldLabel("candidate_ulid") },
      { key: "task_status", label: fieldLabel("task_status") },
      { key: "notification_type", label: fieldLabel("notification_type") },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/memberships/mails/${encodeURIComponent(id)}/retry` },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/memberships/mails/${encodeURIComponent(id)}/ignore` },
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
      { key: "receive_status", label: fieldLabel("receive_status") },
      { key: "source_service", label: fieldLabel("source_service") },
      { key: "subject", label: fieldLabel("subject") },
      { key: "message_type", label: fieldLabel("message_type") },
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
      { key: "candidate_ulid", label: fieldLabel("candidate_ulid"), placeholder: copy.value.placeholders.required },
      { key: "pipeline_ulid", label: fieldLabel("pipeline_ulid") },
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
    filters: [{ key: "pipeline_ulid", label: fieldLabel("pipeline_ulid"), placeholder: copy.value.placeholders.required }],
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
      { key: "pipeline_ulid", label: fieldLabel("pipeline_ulid"), placeholder: copy.value.placeholders.required },
      { key: "stage_ulid", label: fieldLabel("stage_ulid") },
      { key: "status", label: fieldLabel("status") },
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
      { key: "entity_type", label: fieldLabel("entity_type") },
      { key: "entity_ulid", label: fieldLabel("entity_ulid") },
      { key: "event_status", label: fieldLabel("event_status") },
      { key: "event_type", label: fieldLabel("event_type") },
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
      { key: "receive_status", label: fieldLabel("receive_status") },
      { key: "source_service", label: fieldLabel("source_service") },
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
      { key: "processed_status", label: fieldLabel("processed_status") },
      { key: "event_type", label: fieldLabel("event_type") },
      { key: "start_time", label: fieldLabel("start_time") },
      { key: "end_time", label: fieldLabel("end_time") },
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
      { key: "exam_ulid", label: fieldLabel("exam_ulid") },
      { key: "status_type", label: fieldLabel("status_type") },
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
      { key: "exam_ulid", label: fieldLabel("exam_ulid") },
      { key: "task_status", label: fieldLabel("task_status") },
      { key: "delivery_status", label: fieldLabel("delivery_status") },
      { key: "candidate_email", label: fieldLabel("candidate_email") },
      { key: "reminder_type", label: fieldLabel("reminder_type") },
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
const showDetailModal = ref(false)
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
const missingRequiredFilterLabels = computed(() => missingRequiredFilters.value.map(fieldLabel))
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
  const status = getField(item, ["task_status", "taskStatus", "processed_status", "processedStatus", "event_status", "eventStatus", "status"])
  const time = getField(item, ["created_at", "createdAt", "scheduled_at", "scheduledAt"])
  const parts = [status, formatDate(time)].filter((value) => value !== undefined && value !== "")
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
  showDetailModal.value = false
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
      await openItem(items.value[0], false)
    }
  } catch (error) {
    items.value = []
    total.value = 0
    toast.error(error instanceof Error ? error.message : copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

async function openItem(item: JsonRecord, openModal = true) {
  selected.value = item
  detail.value = item
  detailNotice.value = ""
  showDetailModal.value = openModal
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

function closeDetailModal() {
  showDetailModal.value = false
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1600px] flex-col gap-5 px-8 py-6">
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

    <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
      <div class="flex flex-wrap gap-2">
        <button
          v-for="module in modules"
          :key="module.key"
          class="min-h-10 rounded-xl border px-3.5 py-2 text-left text-sm font-black transition"
          :class="module.key === activeKey ? 'border-sky-300 bg-sky-50 text-sky-700' : 'border-slate-200 bg-white text-slate-600 hover:border-slate-400'"
          @click="activeKey = module.key"
        >
          {{ module.label }}
        </button>
      </div>
    </div>

    <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div>
        <h2 class="text-xl font-black text-slate-950">{{ activeModule.label }}</h2>
        <p class="mt-1 text-sm font-medium text-slate-500">{{ activeModule.description }}</p>
      </div>

      <div v-if="activeModule.filters?.length" class="mt-4 flex flex-wrap items-end gap-3">
        <div v-for="filter in activeModule.filters" :key="filter.key" class="w-full space-y-1 sm:w-[200px] 2xl:w-[220px]">
          <label class="text-xs font-black text-slate-500">{{ filter.label }}</label>
          <input
            v-model="activeFilters[filter.key]"
            class="h-11 w-full rounded-xl border border-slate-200 px-3 text-sm font-bold outline-none focus:border-sky-400"
            :placeholder="filter.placeholder || copy.placeholders.optional"
            @keyup.enter="loadList(true)"
          />
        </div>
        <button class="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-black text-white hover:bg-blue-800 sm:w-auto" type="button" @click="loadList(true)">
          <Search class="h-4 w-4" />
          {{ copy.search }}
        </button>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-5 py-4">
        <div>
          <h2 class="text-xl font-black text-slate-950">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm font-medium text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <span class="shrink-0 rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.total(total) }}</span>
      </div>

      <div v-if="missingRequiredFilters.length" class="px-6 py-14 text-center text-sm font-bold text-amber-700">
        {{ copy.requiredPrefix }} {{ missingRequiredFilterLabels.join(", ") }}
      </div>
      <div v-else-if="loading" class="px-6 py-14 text-center text-sm font-bold text-slate-500">{{ copy.loading }}</div>
      <div v-else-if="!items.length" class="px-6 py-14 text-center text-sm font-bold text-slate-500">{{ copy.empty }}</div>
      <div v-else>
        <div class="grid grid-cols-[minmax(0,1fr)_112px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
          <span>{{ copy.record }}</span>
          <span class="text-right">{{ copy.operation }}</span>
        </div>
        <button
          v-for="item in items"
          :key="getItemID(item) || stringify(item)"
          class="grid w-full grid-cols-[minmax(0,1fr)_112px] items-center gap-4 border-b border-slate-100 p-5 text-left transition last:border-b-0 hover:bg-slate-50"
          :class="getItemID(item) === getItemID(selected) ? 'bg-sky-50' : ''"
          @click="openItem(item)"
        >
          <span class="min-w-0">
            <span class="block font-black text-slate-950">{{ getTitle(item) }}</span>
            <span class="mt-1 block break-all text-xs font-bold text-blue-700">{{ getItemID(item) || "-" }}</span>
            <span class="mt-1 block text-xs font-semibold text-slate-500">{{ getSubtitle(item) || "-" }}</span>
          </span>
          <span class="text-right text-sm font-bold text-blue-700">{{ copy.viewDetail }}</span>
        </button>
      </div>

      <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-5 py-4">
        <div class="flex gap-2">
          <button class="rounded-xl border border-slate-300 px-4 py-2 text-sm font-bold disabled:opacity-40" :disabled="page <= 1 && offset <= 0" @click="previousPage">
            {{ copy.previous }}
          </button>
          <button class="rounded-xl border border-slate-300 px-4 py-2 text-sm font-bold" :disabled="items.length < pageSize" @click="nextPage">
            {{ copy.next }}
          </button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="showDetailModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
        <div class="flex max-h-[88vh] w-full max-w-6xl flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div>
              <h2 class="text-2xl font-black text-slate-950">{{ copy.detailTitle }}</h2>
              <p class="mt-1 break-all text-xs font-bold text-blue-700">{{ getItemID(selected) || "-" }}</p>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-xl leading-none text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetailModal">
              ×
            </button>
          </div>

          <div class="flex-1 overflow-auto p-6">
            <div v-if="activeModule.actions?.length && selected" class="mb-5 flex flex-wrap justify-end gap-2">
              <button
                v-for="action in activeModule.actions"
                :key="action.key"
                class="inline-flex h-10 items-center justify-center rounded-xl border border-slate-300 px-4 text-sm font-black hover:border-slate-500 disabled:opacity-50"
                :disabled="!!actionLoading"
                @click="runAction(action)"
              >
                {{ actionLoading === action.key ? copy.processing : action.label }}
              </button>
            </div>

            <div v-if="detailNotice" class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm font-bold text-amber-800">
              {{ detailNotice }}
            </div>
            <div v-if="detailLoading" class="mt-10 text-center text-sm font-bold text-slate-500">{{ copy.loading }}</div>
            <pre v-else class="mt-6 max-h-[64vh] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs font-semibold leading-relaxed text-slate-100">{{ stringify(detail || selected || {}) }}</pre>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>
