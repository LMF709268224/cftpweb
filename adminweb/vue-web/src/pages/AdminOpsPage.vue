<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue"
import { Search, X } from "lucide-vue-next"
import { toast } from "vue-sonner"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

type JsonRecord = Record<string, unknown>
type PaginationMode = "offset" | "page"

type OpsFilter = {
  key: string
  label: string
  placeholder?: string
  options?: { value: string; label: string }[]
  inputType?: "text" | "datetime-local"
  queryFormat?: "rfc3339" | "unixSeconds"
}

type OpsAction = {
  key: "retry" | "ignore"
  label: string
  path: (id: string) => string
  method?: "POST"
  showIf?: (item: JsonRecord) => boolean
}

type OpsModule = {
  key: string
  groupKey: "mail" | "nats" | "runtime" | "pay" | "audit"
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

type DetailEntry = {
  key: string
  label: string
  value: string
  mono?: boolean
}

type DetailSection = {
  key: string
  title: string
  entries: DetailEntry[]
}

const { t } = useAdminLanguage()
const copy = computed(() => t.value.adminOps)

function fieldLabel(key: string) {
  const labels = copy.value.filterLabels as Record<string, string>
  return labels[key] || key
}

function statusLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.statusLabels as Record<string, string>
  return labels[raw] || labels[raw.toUpperCase()] || raw
}

function mailTaskStatusOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["SENT", "FAILED", "PENDING", "IGNORED", "CANCELLED"].map((value) => ({ value, label: statusLabel(value) })),
  ]
}

function deliveryStatusOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["PENDING", "DELIVERED", "FAILED"].map((value) => ({ value, label: statusLabel(value) })),
  ]
}

function receiveStatusOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["PROCESSED", "FAILED", "PENDING", "IGNORED", "SKIPPED"].map((value) => ({ value, label: statusLabel(value) })),
  ]
}

function natsMessageTypeLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.natsMessageTypeLabels as Record<string, string>
  return labels[raw] || labels[raw.toUpperCase()] || labels[raw.toLowerCase()] || raw
}

function mallNatsMessageTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["status_update"].map((value) => ({ value, label: natsMessageTypeLabel(value) })),
  ]
}

function payWebhookEventTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...Object.entries(copy.value.payWebhookDetail.eventTypes as Record<string, string>).map(([value, label]) => ({ value, label })),
  ]
}

function payWebhookProcessedStatusLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.payWebhookDetail.processedStatuses as Record<string, string>
  return labels[raw] || labels[raw.toLowerCase()] || raw
}

function payWebhookProcessedStatusOptions() {
  return [
    { value: "", label: copy.value.all },
    ...Object.keys(copy.value.payWebhookDetail.processedStatuses as Record<string, string>).map((value) => ({ value, label: payWebhookProcessedStatusLabel(value) })),
  ]
}

function examAuditEventTypeLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.examAuditDetail.eventTypes as Record<string, string>
  return labels[raw] || labels[raw.toLowerCase()] || raw
}

function examAuditEventTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...Object.keys(copy.value.examAuditDetail.eventTypes as Record<string, string>).map((value) => ({ value, label: examAuditEventTypeLabel(value) })),
  ]
}

function examAuditProcessedStatusLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.examAuditDetail.processedStatuses as Record<string, string>
  return labels[raw] || labels[raw.toUpperCase()] || raw
}

function examAuditProcessedStatusOptions() {
  return [
    { value: "", label: copy.value.all },
    ...Object.keys(copy.value.examAuditDetail.processedStatuses as Record<string, string>).map((value) => ({ value, label: examAuditProcessedStatusLabel(value) })),
  ]
}

function mailTypeLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.mailTypeLabels as Record<string, string>
  return labels[raw] || labels[raw.toUpperCase()] || raw
}

function mallMailTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["BUNDLE_PAYMENT_REQUIRED"].map((value) => ({ value, label: mailTypeLabel(value) })),
  ]
}

function notificationTypeLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.notificationTypeLabels as Record<string, string>
  return labels[raw] || labels[raw.toLowerCase()] || raw
}

function membershipNotificationTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["membership_activated"].map((value) => ({ value, label: notificationTypeLabel(value) })),
  ]
}

function reminderTypeLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.reminderTypeLabels as Record<string, string>
  return labels[raw] || labels[raw.toUpperCase()] || raw
}

function examReminderTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["ONE_HOUR"].map((value) => ({ value, label: reminderTypeLabel(value) })),
  ]
}

function driverEntityTypeLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.driverEntityTypeLabels as Record<string, string>
  return labels[raw] || labels[raw.toUpperCase()] || raw
}

function driverEntityTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["PIPELINE", "STAGE", "COURSE_UNIT", "EXAM", "RESULT"].map((value) => ({ value, label: driverEntityTypeLabel(value) })),
  ]
}

function driverEventStatusOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["PROCESSED", "FAILED", "PENDING", "IGNORED", "SKIPPED"].map((value) => ({ value, label: statusLabel(value) })),
  ]
}

function driverEventTypeLabel(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return ""
  const labels = copy.value.driverEventTypeLabels as Record<string, string>
  return labels[raw] || labels[raw.toUpperCase()] || raw
}

function driverEventTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["PIPELINE_NEXT_STAGE"].map((value) => ({ value, label: driverEventTypeLabel(value) })),
  ]
}

function examTransitionStatusTypeLabel(value: unknown) {
  const raw = String(value || "").trim().toUpperCase()
  if (!raw) return ""
  const labels = copy.value.examTransitionDetail.statusTypes as Record<string, string>
  return labels[raw] || raw
}

function examTransitionStatusTypeOptions() {
  return [
    { value: "", label: copy.value.all },
    ...["EXAM", "RESULT"].map((value) => ({ value, label: examTransitionStatusTypeLabel(value) })),
  ]
}

const modules = computed<OpsModule[]>(() => [
  {
    key: "paySubscriptions",
    groupKey: "pay",
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
    groupKey: "pay",
    label: copy.value.tabs.payWebhooks,
    description: copy.value.descriptions.payWebhooks,
    listPath: "/api/pay/webhook-events",
    detailPath: (id) => `/api/pay/webhook-events/${encodeURIComponent(id)}`,
    itemKeys: ["events", "items"],
    idKeys: ["event_id", "eventId", "stripe_event_id", "stripeEventId"],
    pagination: "page",
    filters: [
      { key: "event_type", label: fieldLabel("event_type"), options: payWebhookEventTypeOptions() },
      { key: "processed_status", label: fieldLabel("processed_status"), options: payWebhookProcessedStatusOptions() },
      { key: "start_time", label: fieldLabel("start_time"), inputType: "datetime-local", queryFormat: "unixSeconds" },
      { key: "end_time", label: fieldLabel("end_time"), inputType: "datetime-local", queryFormat: "unixSeconds" },
    ],
  },
  {
    key: "payOrderItems",
    groupKey: "pay",
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
    groupKey: "mail",
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
      { key: "task_status", label: fieldLabel("task_status"), options: mailTaskStatusOptions() },
      { key: "mail_type", label: fieldLabel("mail_type"), options: mallMailTypeOptions() },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/mall/mail-tasks/${encodeURIComponent(id)}/retry`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/mall/mail-tasks/${encodeURIComponent(id)}/ignore`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" },
    ],
  },
  {
    key: "mbrMailTasks",
    groupKey: "mail",
    label: copy.value.tabs.mbrMailTasks,
    description: copy.value.descriptions.mbrMailTasks,
    listPath: "/api/memberships/mails",
    detailPath: (id) => `/api/memberships/mails/${encodeURIComponent(id)}`,
    itemKeys: ["mails", "items"],
    idKeys: ["mail_ulid", "mailUlid"],
    pagination: "page",
    filters: [
      { key: "candidate_ulid", label: fieldLabel("candidate_ulid") },
      { key: "task_status", label: fieldLabel("task_status"), options: mailTaskStatusOptions() },
      { key: "notification_type", label: fieldLabel("notification_type"), options: membershipNotificationTypeOptions() },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/memberships/mails/${encodeURIComponent(id)}/retry`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/memberships/mails/${encodeURIComponent(id)}/ignore`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" },
    ],
  },
  {
    key: "mallNats",
    groupKey: "nats",
    label: copy.value.tabs.mallNats,
    description: copy.value.descriptions.mallNats,
    listPath: "/api/mall/nats-messages",
    detailPath: (id) => `/api/mall/nats-messages/${encodeURIComponent(id)}`,
    itemKeys: ["items"],
    idKeys: ["message_ulid", "messageUlid"],
    pagination: "offset",
    filters: [
      { key: "receive_status", label: fieldLabel("receive_status"), options: receiveStatusOptions() },
      { key: "source_service", label: fieldLabel("source_service") },
      { key: "subject", label: fieldLabel("subject") },
      { key: "message_type", label: fieldLabel("message_type"), options: mallNatsMessageTypeOptions() },
    ],
  },
  {
    key: "progMailTasks",
    groupKey: "mail",
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
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/prog/mail-tasks/${encodeURIComponent(id)}/retry`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/prog/mail-tasks/${encodeURIComponent(id)}/ignore`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" },
    ],
  },
  {
    key: "progStages",
    groupKey: "runtime",
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
    groupKey: "runtime",
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
    groupKey: "nats",
    label: copy.value.tabs.progDriverEvents,
    description: copy.value.descriptions.progDriverEvents,
    listPath: "/api/prog/driver-events",
    detailPath: (id) => `/api/prog/driver-events/${encodeURIComponent(id)}`,
    itemKeys: ["items"],
    idKeys: ["event_ulid", "eventUlid"],
    pagination: "offset",
    filters: [
      { key: "entity_type", label: fieldLabel("entity_type"), options: driverEntityTypeOptions() },
      { key: "entity_ulid", label: fieldLabel("entity_ulid") },
      { key: "event_status", label: fieldLabel("event_status"), options: driverEventStatusOptions() },
      { key: "event_type", label: fieldLabel("event_type"), options: driverEventTypeOptions() },
    ],
  },
  {
    key: "progNats",
    groupKey: "nats",
    label: copy.value.tabs.progNats,
    description: copy.value.descriptions.progNats,
    listPath: "/api/prog/nats-messages",
    detailPath: (id) => `/api/prog/nats-messages/${encodeURIComponent(id)}`,
    itemKeys: ["items"],
    idKeys: ["message_ulid", "messageUlid"],
    pagination: "offset",
    filters: [
      { key: "receive_status", label: fieldLabel("receive_status"), options: receiveStatusOptions() },
      { key: "source_service", label: fieldLabel("source_service") },
    ],
  },
  {
    key: "examAudit",
    groupKey: "audit",
    label: copy.value.tabs.examAudit,
    description: copy.value.descriptions.examAudit,
    listPath: "/api/exam-ops/audit-messages",
    detailPath: (id) => `/api/exam-ops/audit-messages/${encodeURIComponent(id)}`,
    detailIDFormat: "ulid",
    itemKeys: ["audit_messages", "auditMessages", "items"],
    idKeys: ["message_ulid", "messageUlid"],
    pagination: "page",
    filters: [
      { key: "processed_status", label: fieldLabel("processed_status"), options: examAuditProcessedStatusOptions() },
      { key: "event_type", label: fieldLabel("event_type"), options: examAuditEventTypeOptions() },
    ],
  },
  {
    key: "examTransitions",
    groupKey: "runtime",
    label: copy.value.tabs.examTransitions,
    description: copy.value.descriptions.examTransitions,
    listPath: "/api/exam-ops/status-transitions",
    itemKeys: ["transitions", "items"],
    idKeys: ["transition_ulid", "transitionUlid", "id"],
    pagination: "page",
    filters: [
      { key: "exam_ulid", label: fieldLabel("exam_ulid") },
      { key: "status_type", label: fieldLabel("status_type"), options: examTransitionStatusTypeOptions() },
    ],
  },
  {
    key: "examReminders",
    groupKey: "mail",
    label: copy.value.tabs.examReminders,
    description: copy.value.descriptions.examReminders,
    listPath: "/api/exam-ops/reminder-mails",
    detailPath: (id) => `/api/exam-ops/reminder-mails/${encodeURIComponent(id)}`,
    itemKeys: ["mails", "items"],
    idKeys: ["mail_ulid", "mailUlid"],
    pagination: "page",
    filters: [
      { key: "exam_ulid", label: fieldLabel("exam_ulid") },
      { key: "task_status", label: fieldLabel("task_status"), options: mailTaskStatusOptions() },
      { key: "delivery_status", label: fieldLabel("delivery_status"), options: deliveryStatusOptions() },
      { key: "candidate_email", label: fieldLabel("candidate_email") },
      { key: "reminder_type", label: fieldLabel("reminder_type"), options: examReminderTypeOptions() },
    ],
    actions: [
      { key: "retry", label: copy.value.actions.retry, path: (id) => `/api/exam-ops/reminder-mails/${encodeURIComponent(id)}/retry`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" || item.delivery_status === "FAILED" || item.deliveryStatus === "FAILED" },
      { key: "ignore", label: copy.value.actions.ignore, path: (id) => `/api/exam-ops/reminder-mails/${encodeURIComponent(id)}/ignore`, showIf: (item) => item.task_status === "FAILED" || item.taskStatus === "FAILED" || item.delivery_status === "FAILED" || item.deliveryStatus === "FAILED" },
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
let listRequestId = 0
let detailRequestId = 0

const activeModule = computed(() => modules.value.find((module) => module.key === activeKey.value) || modules.value[0])
const activeFilters = computed(() => filters[activeModule.value.key] || (filters[activeModule.value.key] = {}))
const missingRequiredFilters = computed(() =>
  (activeModule.value.requiredFilters || []).filter((key) => !String(activeFilters.value[key] || "").trim()),
)
const missingRequiredFilterLabels = computed(() => missingRequiredFilters.value.map(fieldLabel))
const canLoad = computed(() => missingRequiredFilters.value.length === 0)
const structuredDetailSections = computed(() => buildStructuredDetailSections(detail.value || selected.value))

function ensureOptionFilterDefaults(module = activeModule.value) {
  const moduleFilters = filters[module.key] || (filters[module.key] = {})
  for (const filter of module.filters || []) {
    if (!filter.options?.length || moduleFilters[filter.key] !== undefined) continue
    const defaultOption = filter.options.find((option) => option.value === "") || filter.options[0]
    moduleFilters[filter.key] = defaultOption?.value || ""
  }
}

function clearFilterInput(key: string) {
  if (!activeFilters.value[key]) return
  activeFilters.value[key] = ""
  void loadList(true)
}

const moduleGroups = computed(() => {
  const groupsMap: Record<string, OpsModule[]> = {
    mail: [],
    nats: [],
    runtime: [],
    pay: [],
    audit: [],
  }
  modules.value.forEach((m) => {
    if (groupsMap[m.groupKey]) {
      groupsMap[m.groupKey].push(m)
    }
  })
  return [
    { key: "mail", modules: groupsMap.mail },
    { key: "nats", modules: groupsMap.nats },
    { key: "runtime", modules: groupsMap.runtime },
    { key: "pay", modules: groupsMap.pay },
    { key: "audit", modules: groupsMap.audit },
  ].filter((g) => g.modules.length > 0)
})

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

function getDetailField(item: JsonRecord, keys: string[]): unknown {
  for (const key of keys) {
    if (item[key] !== undefined && item[key] !== null && item[key] !== "") return item[key]
  }
  return undefined
}

function getItemID(item: JsonRecord | null, module = activeModule.value) {
  const value = getField(item, module.idKeys)
  return value === undefined ? "" : String(value)
}

function isCurrentListRequest(requestId: number, moduleKey: string) {
  return requestId === listRequestId && activeKey.value === moduleKey
}

function isCurrentDetailRequest(requestId: number, module: OpsModule, id: string) {
  return requestId === detailRequestId && activeKey.value === module.key && getItemID(selected.value, module) === id
}

function invalidateDetailRequest() {
  detailRequestId += 1
  detailLoading.value = false
}

function isULID(value: string) {
  return /^[0-9A-HJKMNP-TV-Z]{26}$/i.test(value)
}

function getTitle(item: JsonRecord) {
  if (activeModule.value.key === "progDriverEvents") {
    const eventType = getField(item, ["event_type", "eventType"])
    if (eventType !== undefined) return driverEventTypeLabel(eventType)
  }
  if (activeModule.value.key === "examAudit") {
    const eventType = getField(item, ["event_type", "eventType"])
    if (eventType !== undefined) return examAuditEventTypeLabel(eventType)
  }
  const title = getField(item, ["title", "name", "subject", "event_type", "eventType", "message_type", "messageType"])
  return title === undefined ? getItemID(item) || copy.value.untitled : String(title)
}

function getSubtitle(item: JsonRecord) {
  const rawStatus = getField(item, ["task_status", "taskStatus", "processed_status", "processedStatus", "event_status", "eventStatus", "status"])
  const mappedStatus = rawStatus
    ? activeModule.value.key === "payWebhooks"
      ? payWebhookProcessedStatusLabel(rawStatus)
      : activeModule.value.key === "examAudit"
        ? examAuditProcessedStatusLabel(rawStatus)
        : statusLabel(rawStatus)
    : undefined
  const time = getField(item, ["created_at", "createdAt", "scheduled_at", "scheduledAt"])
  const parts = [mappedStatus, formatDate(time)].filter((value) => value !== undefined && value !== "")
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

function unwrapMallMailTaskDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const summaryRecord = isRecord(detailRecord.summary) ? detailRecord.summary : {}
  return { ...detailRecord, ...summaryRecord }
}

function unwrapPayWebhookDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const eventRecord = isRecord(detailRecord.event) ? detailRecord.event : detailRecord
  const payload = parseJsonValue(getDetailField(eventRecord, ["payload_json", "payloadJson"]))
  const payloadRecord = isRecord(payload) ? payload : {}
  const dataRecord = isRecord(payloadRecord.data) ? payloadRecord.data : {}
  const objectRecord = isRecord(dataRecord.object) ? dataRecord.object : {}
  const objectType = getDetailField(objectRecord, ["object"])
  const invoiceID =
    objectType === "invoice" ? getDetailField(objectRecord, ["id"]) : getDetailField(objectRecord, ["invoice", "invoice_id", "invoiceId"])

  return {
    ...detailRecord,
    ...eventRecord,
    event_id: getDetailField(eventRecord, ["event_id", "eventId", "stripe_event_id", "stripeEventId"]) || getDetailField(payloadRecord, ["id"]),
    event_type: getDetailField(eventRecord, ["event_type", "eventType"]) || getDetailField(payloadRecord, ["type"]),
    stripe_created_at: getDetailField(payloadRecord, ["created"]),
    object_type: objectType,
    object_id: getDetailField(objectRecord, ["id"]),
    amount_paid: getDetailField(objectRecord, ["amount_paid", "amountPaid", "amount_total", "amountTotal", "amount"]),
    amount_due: getDetailField(objectRecord, ["amount_due", "amountDue", "total"]),
    billing_reason: getDetailField(objectRecord, ["billing_reason", "billingReason"]),
    currency: getDetailField(objectRecord, ["currency"]),
    customer_email: getDetailField(objectRecord, ["customer_email", "customerEmail"]),
    customer_id: getDetailField(objectRecord, ["customer", "customer_id", "customerId"]),
    customer_name: getDetailField(objectRecord, ["customer_name", "customerName"]),
    invoice_id: invoiceID,
    invoice_number: getDetailField(objectRecord, ["number"]),
    livemode: getDetailField(payloadRecord, ["livemode"]) ?? getDetailField(objectRecord, ["livemode"]),
    paid: getDetailField(objectRecord, ["paid"]),
    payment_status: getDetailField(objectRecord, ["payment_status", "paymentStatus", "status"]),
    subscription_id: getDetailField(objectRecord, ["subscription", "subscription_id", "subscriptionId"]),
  }
}

function unwrapMallNatsDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const summaryRecord = isRecord(detailRecord.summary) ? detailRecord.summary : {}
  return { ...detailRecord, ...summaryRecord }
}

function unwrapProgDriverEventDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const summaryRecord = isRecord(detailRecord.summary) ? detailRecord.summary : {}
  return { ...detailRecord, ...summaryRecord }
}

function unwrapProgNatsDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const summaryRecord = isRecord(detailRecord.summary) ? detailRecord.summary : {}
  return { ...detailRecord, ...summaryRecord }
}

function unwrapExamTransitionDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const summaryRecord = isRecord(detailRecord.summary) ? detailRecord.summary : {}
  return { ...detailRecord, ...summaryRecord }
}

function unwrapExamAuditDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const messageRecord = isRecord(detailRecord.message) ? detailRecord.message : detailRecord
  const summaryRecord = isRecord(detailRecord.summary) ? detailRecord.summary : {}
  return { ...detailRecord, ...messageRecord, ...summaryRecord }
}

function unwrapMbrMailTaskDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const mailRecord = isRecord(detailRecord.mail) ? detailRecord.mail : detailRecord
  return { ...detailRecord, ...mailRecord }
}

function unwrapExamReminderDetail(value: unknown) {
  if (!isRecord(value)) return null
  const root = value
  const detailRecord = isRecord(root.detail) ? root.detail : root
  const mailRecord = isRecord(detailRecord.mail) ? detailRecord.mail : detailRecord
  return { ...detailRecord, ...mailRecord }
}

function parseJsonValue(value: unknown) {
  if (typeof value !== "string") return value
  const trimmed = value.trim()
  if (!trimmed) return value
  try {
    return JSON.parse(trimmed)
  } catch {
    return value
  }
}

function formatDetailValue(key: string, value: unknown) {
  if (value === undefined || value === null || value === "") return "-"
  if (key.endsWith("_at")) return formatDate(value)
  if (typeof value === "boolean") return value ? copy.value.booleanLabels.true : copy.value.booleanLabels.false
  if (key === "task_status" || key === "delivery_status" || key === "receive_status" || key === "event_status") {
    return statusLabel(value)
  }
  if (key === "mail_type") return mailTypeLabel(value)
  if (key === "notification_type") return notificationTypeLabel(value)
  if (key === "reminder_type") return reminderTypeLabel(value)
  if (key === "message_type") return natsMessageTypeLabel(value)
  const parsedValue = key.endsWith("_json") || key === "message_payload" ? parseJsonValue(value) : value
  if (typeof parsedValue === "object") return stringify(parsedValue)
  return String(parsedValue)
}

function formatProgDriverEventDetailValue(key: string, value: unknown) {
  if (key === "entity_type") return driverEntityTypeLabel(value)
  if (key === "event_type") return driverEventTypeLabel(value)
  return formatDetailValue(key, value)
}

function formatStripeAmount(value: unknown, currency: unknown) {
  const amount = typeof value === "number" ? value : Number(value)
  if (!Number.isFinite(amount)) return formatDetailValue("", value)
  const currencyCode = String(currency || "").toUpperCase()
  const zeroDecimalCurrencies = new Set(["BIF", "CLP", "DJF", "GNF", "JPY", "KMF", "KRW", "MGA", "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF"])
  const fractionDigits = zeroDecimalCurrencies.has(currencyCode) ? 0 : 2
  const divisor = fractionDigits === 0 ? 1 : 100
  const formatted = (amount / divisor).toLocaleString("zh-CN", {
    minimumFractionDigits: fractionDigits,
    maximumFractionDigits: fractionDigits,
  })
  return currencyCode ? `${currencyCode} ${formatted}` : formatted
}

function formatStripeTimestamp(value: unknown) {
  const timestamp = typeof value === "number" ? value : Number(value)
  if (!Number.isFinite(timestamp) || timestamp <= 0) return formatDetailValue("", value)
  return new Date(timestamp * 1000).toLocaleString("zh-CN", { hour12: false })
}

function normalizeDetailCode(value: unknown) {
  return String(value || "").trim().toUpperCase()
}

function lookupDetailLabel(value: unknown, labels: Record<string, string>, prefixes: string[] = []) {
  const code = normalizeDetailCode(value)
  if (!code) return "-"
  const normalizedCodes = [code, ...prefixes.map((prefix) => code.replace(new RegExp(`^${prefix}`), ""))]
  for (const normalizedCode of normalizedCodes) {
    if (labels[normalizedCode]) return labels[normalizedCode]
  }
  return String(value)
}

function formatExamTransitionDetailValue(record: JsonRecord, key: string, value: unknown) {
  if (key === "event_type") {
    const labels = copy.value.examTransitionDetail.eventTypes as Record<string, string>
    const event = String(value || "").trim().toLowerCase()
    return event ? labels[event] || String(value) : "-"
  }
  if (key === "status_type") {
    const labels = copy.value.examTransitionDetail.statusTypes as Record<string, string>
    const type = normalizeDetailCode(value)
    if (type.includes("EXAM")) return labels.EXAM
    if (type.includes("RESULT")) return labels.RESULT
    return type || "-"
  }
  if (key === "from_status" || key === "to_status") {
    const type = normalizeDetailCode(getDetailField(record, ["status_type", "statusType"]))
    if (type.includes("EXAM")) {
      return lookupDetailLabel(value, copy.value.examTransitionDetail.examStatuses as Record<string, string>, ["EXAM_STATUS_"])
    }
    if (type.includes("RESULT")) {
      return lookupDetailLabel(value, copy.value.examTransitionDetail.resultStatuses as Record<string, string>, ["RESULT_STATUS_"])
    }
  }
  return formatDetailValue(key, value)
}

function formatPayWebhookDetailValue(record: JsonRecord, key: string, value: unknown) {
  if (key === "event_type") {
    const labels = copy.value.payWebhookDetail.eventTypes as Record<string, string>
    const event = String(value || "").trim()
    return event ? labels[event] || event : "-"
  }
  if (key === "processed_status") {
    return payWebhookProcessedStatusLabel(value) || "-"
  }
  if (key === "payment_status") {
    const labels = copy.value.payWebhookDetail.paymentStatuses as Record<string, string>
    const status = String(value || "").trim().toLowerCase()
    return status ? labels[status] || String(value) : "-"
  }
  if (key === "billing_reason") {
    const labels = copy.value.payWebhookDetail.billingReasons as Record<string, string>
    const reason = String(value || "").trim()
    return reason ? labels[reason] || String(value) : "-"
  }
  if (key === "object_type") {
    const labels = copy.value.payWebhookDetail.objectTypes as Record<string, string>
    const type = String(value || "").trim()
    return type ? labels[type] || String(value) : "-"
  }
  if (key === "amount_paid" || key === "amount_due") return formatStripeAmount(value, getDetailField(record, ["currency"]))
  if (key === "currency") return String(value || "-").toUpperCase()
  if (key === "stripe_created_at") return formatStripeTimestamp(value)
  return formatDetailValue(key, value)
}

function formatExamAuditDetailValue(key: string, value: unknown) {
  if (key === "event_type") {
    return examAuditEventTypeLabel(value) || "-"
  }
  if (key === "audit_status") {
    const labels = copy.value.examAuditDetail.auditStatuses as Record<string, string>
    const status = String(value || "").trim().toLowerCase()
    return status ? labels[status] || String(value) : "-"
  }
  if (key === "processed_status") {
    return examAuditProcessedStatusLabel(value) || "-"
  }
  if (key === "event_timestamp" || key === "audit_date") return formatDate(value)
  return formatDetailValue(key, value)
}

function buildStructuredDetailSections(value: unknown): DetailSection[] {
  if (activeModule.value.key === "payWebhooks") return buildPayWebhookDetailSections(value)
  if (activeModule.value.key === "mallMailTasks") return buildMallMailTaskDetailSections(value)
  if (activeModule.value.key === "mallNats") return buildMallNatsDetailSections(value)
  if (activeModule.value.key === "progDriverEvents") return buildProgDriverEventDetailSections(value)
  if (activeModule.value.key === "progNats") return buildProgNatsDetailSections(value)
  if (activeModule.value.key === "mbrMailTasks") return buildMbrMailTaskDetailSections(value)
  if (activeModule.value.key === "examTransitions") return buildExamTransitionDetailSections(value)
  if (activeModule.value.key === "examAudit") return buildExamAuditDetailSections(value)
  if (activeModule.value.key === "examReminders") return buildExamReminderDetailSections(value)
  return []
}

function buildPayWebhookDetailSections(value: unknown): DetailSection[] {
  const record = unwrapPayWebhookDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.payWebhookDetail.fields as Record<string, string>
  const sections = copy.value.payWebhookDetail.sections
  const definitions = [
    {
      key: "event",
      title: sections.event,
      fields: [
        ["event_id", ["event_id", "eventId", "stripe_event_id", "stripeEventId"]],
        ["event_type", ["event_type", "eventType"]],
        ["processed_status", ["processed_status", "processedStatus"]],
        ["stripe_created_at", ["stripe_created_at"]],
      ],
    },
    {
      key: "payment",
      title: sections.payment,
      fields: [
        ["object_type", ["object_type"]],
        ["payment_status", ["payment_status"]],
        ["amount_paid", ["amount_paid"]],
        ["amount_due", ["amount_due"]],
        ["paid", ["paid"]],
        ["livemode", ["livemode"]],
      ],
    },
    {
      key: "customer",
      title: sections.customer,
      fields: [
        ["customer_email", ["customer_email"]],
        ["customer_name", ["customer_name"]],
        ["customer_id", ["customer_id"]],
      ],
    },
    {
      key: "invoice",
      title: sections.invoice,
      fields: [
        ["invoice_id", ["invoice_id"]],
        ["invoice_number", ["invoice_number"]],
        ["subscription_id", ["subscription_id"]],
        ["billing_reason", ["billing_reason"]],
      ],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["created_at", ["created_at", "createdAt"]],
        ["processed_at", ["processed_at", "processedAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
        ["error_message", ["error_message", "errorMessage", "last_error", "lastError"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields
        .map(([key, keys]) => ({
          key,
          label: fieldLabels[key] || key.replaceAll("_", " "),
          value: formatPayWebhookDetailValue(record, key, getDetailField(record, [...keys])),
          mono: key.endsWith("_id") || key.endsWith("_email") || key === "event_id",
        }))
        .filter((entry) => entry.value !== "-"),
    }))
    .filter((section) => section.entries.length > 0)
}

function buildMallMailTaskDetailSections(value: unknown): DetailSection[] {
  if (activeModule.value.key !== "mallMailTasks") return []
  const record = unwrapMallMailTaskDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.mallMailTaskDetail.fields as Record<string, string>
  const sections = copy.value.mallMailTaskDetail.sections
  const definitions = [
    {
      key: "task",
      title: sections.task,
      fields: [
        ["mail_task_ulid", ["mail_task_ulid", "mailTaskUlid", "mail_ulid", "mailUlid"]],
        ["task_status", ["task_status", "taskStatus"]],
        ["mail_type", ["mail_type", "mailType"]],
        ["subject", ["subject"]],
      ],
    },
    {
      key: "business",
      title: sections.business,
      fields: [
        ["candidate_ulid", ["candidate_ulid", "candidateUlid"]],
        ["order_ulid", ["order_ulid", "orderUlid"]],
        ["template_path", ["template_path", "templatePath"]],
        ["template_params_json", ["template_params_json", "templateParamsJson"]],
      ],
    },
    {
      key: "recipient",
      title: sections.recipient,
      fields: [
        ["recipient_name", ["recipient_name", "recipientName"]],
        ["recipient_email", ["recipient_email", "recipientEmail", "to_email", "toEmail"]],
      ],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["version", ["version"]],
        ["created_at", ["created_at", "createdAt"]],
        ["final_at", ["final_at", "finalAt"]],
        ["last_reconciled_at", ["last_reconciled_at", "lastReconciledAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatDetailValue(key, getDetailField(record, [...keys])),
        mono: key.endsWith("_ulid") || key === "recipient_email" || key === "template_params_json",
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function buildMallNatsDetailSections(value: unknown): DetailSection[] {
  const record = unwrapMallNatsDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.mallNatsDetail.fields as Record<string, string>
  const sections = copy.value.mallNatsDetail.sections
  const definitions = [
    {
      key: "message",
      title: sections.message,
      fields: [
        ["message_ulid", ["message_ulid", "messageUlid"]],
        ["subject", ["subject"]],
        ["source_service", ["source_service", "sourceService"]],
        ["message_type", ["message_type", "messageType"]],
      ],
    },
    {
      key: "processing",
      title: sections.processing,
      fields: [
        ["receive_status", ["receive_status", "receiveStatus"]],
        ["process_attempts", ["process_attempts", "processAttempts"]],
        ["received_at", ["received_at", "receivedAt"]],
        ["last_processing_at", ["last_processing_at", "lastProcessingAt"]],
      ],
    },
    {
      key: "payload",
      title: sections.payload,
      fields: [["message_payload", ["message_payload", "messagePayload"]]],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["created_at", ["created_at", "createdAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatDetailValue(key, getDetailField(record, [...keys])),
        mono: key.endsWith("_ulid") || key === "message_payload",
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function buildProgDriverEventDetailSections(value: unknown): DetailSection[] {
  const record = unwrapProgDriverEventDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.progDriverEventDetail.fields as Record<string, string>
  const sections = copy.value.progDriverEventDetail.sections
  const definitions = [
    {
      key: "event",
      title: sections.event,
      fields: [
        ["event_ulid", ["event_ulid", "eventUlid"]],
        ["event_type", ["event_type", "eventType"]],
        ["event_status", ["event_status", "eventStatus"]],
        ["source", ["source"]],
      ],
    },
    {
      key: "entity",
      title: sections.entity,
      fields: [
        ["entity_type", ["entity_type", "entityType"]],
        ["entity_ulid", ["entity_ulid", "entityUlid"]],
      ],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["created_at", ["created_at", "createdAt"]],
        ["last_reconciled_at", ["last_reconciled_at", "lastReconciledAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatProgDriverEventDetailValue(key, getDetailField(record, [...keys])),
        mono: key.endsWith("_ulid"),
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function buildProgNatsDetailSections(value: unknown): DetailSection[] {
  const record = unwrapProgNatsDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.progNatsDetail.fields as Record<string, string>
  const sections = copy.value.progNatsDetail.sections
  const definitions = [
    {
      key: "message",
      title: sections.message,
      fields: [
        ["message_ulid", ["message_ulid", "messageUlid"]],
        ["subject", ["subject"]],
        ["source_service", ["source_service", "sourceService"]],
        ["message_type", ["message_type", "messageType"]],
      ],
    },
    {
      key: "processing",
      title: sections.processing,
      fields: [
        ["receive_status", ["receive_status", "receiveStatus"]],
        ["process_attempts", ["process_attempts", "processAttempts"]],
        ["received_at", ["received_at", "receivedAt"]],
        ["last_processing_at", ["last_processing_at", "lastProcessingAt"]],
      ],
    },
    {
      key: "payload",
      title: sections.payload,
      fields: [["message_payload", ["message_payload", "messagePayload"]]],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["created_at", ["created_at", "createdAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatDetailValue(key, getDetailField(record, [...keys])),
        mono: key.endsWith("_ulid") || key === "message_payload",
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function buildMbrMailTaskDetailSections(value: unknown): DetailSection[] {
  const record = unwrapMbrMailTaskDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.mbrMailTaskDetail.fields as Record<string, string>
  const sections = copy.value.mbrMailTaskDetail.sections
  const definitions = [
    {
      key: "task",
      title: sections.task,
      fields: [
        ["mail_ulid", ["mail_ulid", "mailUlid", "mail_task_ulid", "mailTaskUlid"]],
        ["task_status", ["task_status", "taskStatus"]],
        ["notification_type", ["notification_type", "notificationType"]],
        ["subject", ["subject"]],
      ],
    },
    {
      key: "business",
      title: sections.business,
      fields: [
        ["candidate_ulid", ["candidate_ulid", "candidateUlid"]],
        ["membership_record_ulid", ["membership_record_ulid", "membershipRecordUlid"]],
        ["reference_id", ["reference_id", "referenceId"]],
        ["payload_json", ["payload_json", "payloadJson"]],
      ],
    },
    {
      key: "recipient",
      title: sections.recipient,
      fields: [
        ["to_name", ["to_name", "toName", "recipient_name", "recipientName"]],
        ["to_email", ["to_email", "toEmail", "recipient_email", "recipientEmail"]],
      ],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["created_at", ["created_at", "createdAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatDetailValue(key, getDetailField(record, [...keys])),
        mono: key.endsWith("_ulid") || key.endsWith("_email") || key.endsWith("_json"),
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function buildExamTransitionDetailSections(value: unknown): DetailSection[] {
  const record = unwrapExamTransitionDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.examTransitionDetail.fields as Record<string, string>
  const sections = copy.value.examTransitionDetail.sections
  const definitions = [
    {
      key: "transition",
      title: sections.transition,
      fields: [
        ["transition_ulid", ["transition_ulid", "transitionUlid", "id"]],
        ["msg_fp", ["msg_fp", "msgFp"]],
        ["event_type", ["event_type", "eventType"]],
        ["status_type", ["status_type", "statusType"]],
      ],
    },
    {
      key: "exam",
      title: sections.exam,
      fields: [
        ["exam_ulid", ["exam_ulid", "examUlid"]],
        ["from_status", ["from_status", "fromStatus"]],
        ["to_status", ["to_status", "toStatus"]],
        ["transitioned_at", ["transitioned_at", "transitionedAt"]],
      ],
    },
    {
      key: "metadata",
      title: sections.metadata,
      fields: [["metadata_json", ["metadata_json", "metadataJson"]]],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["created_at", ["created_at", "createdAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatExamTransitionDetailValue(record, key, getDetailField(record, [...keys])),
        mono: key.endsWith("_ulid") || key.endsWith("_json") || key === "msg_fp",
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function buildExamAuditDetailSections(value: unknown): DetailSection[] {
  const record = unwrapExamAuditDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.examAuditDetail.fields as Record<string, string>
  const sections = copy.value.examAuditDetail.sections
  const definitions = [
    {
      key: "message",
      title: sections.message,
      fields: [
        ["message_ulid", ["message_ulid", "messageUlid"]],
        ["event_type", ["event_type", "eventType"]],
      ],
    },
    {
      key: "status",
      title: sections.status,
      fields: [
        ["audit_status", ["audit_status", "auditStatus"]],
        ["processed_status", ["processed_status", "processedStatus"]],
        ["event_timestamp", ["event_timestamp", "eventTimestamp"]],
        ["audit_date", ["audit_date", "auditDate"]],
      ],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["id", ["id"]],
        ["created_at", ["created_at", "createdAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatExamAuditDetailValue(key, getDetailField(record, [...keys])),
        mono: key === "message_ulid",
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function buildExamReminderDetailSections(value: unknown): DetailSection[] {
  const record = unwrapExamReminderDetail(value)
  if (!record) return []

  const fieldLabels = copy.value.examReminderDetail.fields as Record<string, string>
  const sections = copy.value.examReminderDetail.sections
  const definitions = [
    {
      key: "task",
      title: sections.task,
      fields: [
        ["mail_ulid", ["mail_ulid", "mailUlid"]],
        ["task_status", ["task_status", "taskStatus"]],
        ["delivery_status", ["delivery_status", "deliveryStatus"]],
        ["reminder_type", ["reminder_type", "reminderType"]],
        ["subject", ["subject"]],
      ],
    },
    {
      key: "exam",
      title: sections.exam,
      fields: [
        ["exam_ulid", ["exam_ulid", "examUlid"]],
        ["scheduled_at", ["scheduled_at", "scheduledAt"]],
        ["payload_json", ["payload_json", "payloadJson"]],
      ],
    },
    {
      key: "recipient",
      title: sections.recipient,
      fields: [
        ["to_name", ["to_name", "toName", "recipient_name", "recipientName"]],
        ["to_email", ["to_email", "toEmail", "recipient_email", "recipientEmail"]],
      ],
    },
    {
      key: "system",
      title: sections.system,
      fields: [
        ["created_at", ["created_at", "createdAt"]],
        ["updated_at", ["updated_at", "updatedAt"]],
      ],
    },
  ] as const

  return definitions
    .map((section) => ({
      key: section.key,
      title: section.title,
      entries: section.fields.map(([key, keys]) => ({
        key,
        label: fieldLabels[key] || key.replaceAll("_", " "),
        value: formatDetailValue(key, getDetailField(record, [...keys])),
        mono: key.endsWith("_ulid") || key.endsWith("_email") || key.endsWith("_json"),
      })),
    }))
    .filter((section) => section.entries.some((entry) => entry.value !== "-"))
}

function filterQueryValue(filter: OpsFilter, value: string) {
  if (!filter.queryFormat) return value
  const timestamp = new Date(value).getTime()
  if (!Number.isFinite(timestamp)) return ""
  if (filter.queryFormat === "rfc3339") return new Date(timestamp).toISOString()
  return String(Math.floor(timestamp / 1000))
}

function buildListURL(module: OpsModule) {
  const params = new URLSearchParams()
  for (const filter of module.filters || []) {
    const value = String(activeFilters.value[filter.key] || "").trim()
    const queryValue = value ? filterQueryValue(filter, value) : ""
    if (queryValue) params.set(filter.key, queryValue)
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
  const requestId = ++listRequestId
  invalidateDetailRequest()
  const module = activeModule.value
  const moduleKey = module.key
  ensureOptionFilterDefaults(module)
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
    loading.value = false
    return
  }
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>(buildListURL(module))
    if (!isCurrentListRequest(requestId, moduleKey)) return
    items.value = extractItems(data, module)
    total.value = Number(data.total || items.value.length || 0)
    if (items.value.length) {
      void openItem(items.value[0], false)
    }
  } catch (error) {
    if (!isCurrentListRequest(requestId, moduleKey)) return
    items.value = []
    total.value = 0
    toast.error(apiErrorMessage(error, copy.value.toasts.loadFailed))
  } finally {
    if (isCurrentListRequest(requestId, moduleKey)) loading.value = false
  }
}

async function openItem(item: JsonRecord, openModal = true) {
  const module = activeModule.value
  const moduleKey = module.key
  const id = getItemID(item, module)
  const requestId = ++detailRequestId
  detailLoading.value = false
  selected.value = item
  detail.value = item
  detailNotice.value = ""
  showDetailModal.value = openModal
  if (!openModal) return
  if (!module.detailPath || !id) return
  if (module.detailIDFormat === "ulid" && !isULID(id)) {
    detailNotice.value = copy.value.invalidDetailId(id)
    return
  }
  detailLoading.value = true
  try {
    const response = await apiClient<JsonRecord>(module.detailPath(id))
    if (!isCurrentDetailRequest(requestId, module, id)) return
    const responseId = getItemID(response, module)
    if (responseId && responseId !== id) throw new Error(`Detail response ID does not match ${moduleKey}:${id}`)
    detail.value = response
  } catch (error) {
    if (!isCurrentDetailRequest(requestId, module, id)) return
    toast.error(apiErrorMessage(error, copy.value.toasts.detailFailed))
  } finally {
    if (isCurrentDetailRequest(requestId, module, id)) detailLoading.value = false
  }
}

function closeDetailModal() {
  invalidateDetailRequest()
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
    toast.error(apiErrorMessage(error, copy.value.toasts.actionFailed))
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1600px] flex-col gap-5 px-4 py-5 md:px-8 md:py-6">
    <header class="flex flex-col items-start gap-4 md:flex-row md:items-end md:justify-between">
      <div class="min-w-0">
        <p class="text-xs font-black uppercase tracking-[0.2em] text-slate-400">{{ copy.eyebrow }}</p>
        <h1 class="mt-2 text-3xl font-black text-slate-950">{{ copy.title }}</h1>
        <p class="mt-2 text-sm font-medium text-slate-500">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center justify-center rounded-xl border border-slate-300 bg-white px-5 py-3 text-sm font-black text-slate-900 shadow-sm hover:border-slate-500" @click="loadList(true)">
        {{ copy.refresh }}
      </button>
    </header>

    <div class="space-y-5 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <div v-for="group in moduleGroups" :key="group.key">
        <div class="mb-2 text-xs font-black tracking-wider text-slate-400 uppercase">{{ (copy.groups as Record<string, string>)[group.key] }}</div>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="module in group.modules"
            :key="module.key"
            class="min-h-10 min-w-[128px] flex-1 rounded-xl border px-3.5 py-2 text-left text-sm font-black transition md:flex-none"
            :class="module.key === activeKey ? 'border-sky-300 bg-sky-50 text-sky-700' : 'border-slate-200 bg-white text-slate-600 hover:border-slate-400'"
            @click="activeKey = module.key"
          >
            {{ module.label }}
          </button>
        </div>
      </div>
    </div>

    <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:p-5">
      <div class="min-w-0">
        <h2 class="text-xl font-black text-slate-950">{{ activeModule.label }}</h2>
        <p class="mt-1 text-sm font-medium text-slate-500">{{ activeModule.description }}</p>
      </div>

      <div v-if="activeModule.filters?.length" class="mt-4 flex flex-wrap items-end gap-3">
        <div v-for="filter in activeModule.filters" :key="filter.key" class="w-full space-y-1 sm:w-[200px] 2xl:w-[220px]">
          <label class="text-xs font-black text-slate-500">{{ filter.label }}</label>
          <select
            v-if="filter.options?.length"
            v-model="activeFilters[filter.key]"
            class="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold outline-none focus:border-sky-400"
            @change="loadList(true)"
          >
            <option v-for="option in filter.options" :key="option.value || 'all'" :value="option.value">{{ option.label }}</option>
          </select>
          <div v-else class="relative">
            <input
              v-model="activeFilters[filter.key]"
              class="h-11 w-full rounded-xl border border-slate-200 px-3 pr-10 text-sm font-bold outline-none focus:border-sky-400"
              :type="filter.inputType || 'text'"
              :placeholder="filter.placeholder || ''"
              @keyup.enter="loadList(true)"
            />
            <button
              v-if="activeFilters[filter.key]"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
              type="button"
              :aria-label="copy.clearInput"
              :title="copy.clearInput"
              @click="clearFilterInput(filter.key)"
            >
              <X class="h-4 w-4" />
            </button>
          </div>
        </div>
        <button class="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 text-sm font-black text-white hover:bg-blue-800 sm:w-auto" type="button" @click="loadList(true)">
          <Search class="h-4 w-4" />
          {{ copy.search }}
        </button>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-4 md:px-5">
        <div class="min-w-0">
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
        <div class="grid grid-cols-[minmax(0,1fr)_96px] gap-3 border-b border-slate-100 bg-slate-50 px-4 py-3 text-xs font-black text-slate-500 md:grid-cols-[minmax(0,1fr)_112px] md:gap-4 md:px-5">
          <span>{{ copy.record }}</span>
          <span class="text-right">{{ copy.operation }}</span>
        </div>
        <button
          v-for="item in items"
          :key="getItemID(item) || stringify(item)"
          class="grid w-full gap-3 border-b border-slate-100 p-4 text-left transition last:border-b-0 hover:bg-slate-50 md:grid-cols-[minmax(0,1fr)_112px] md:items-center md:gap-4 md:p-5"
          :class="getItemID(item) === getItemID(selected) ? 'bg-sky-50' : ''"
          @click="openItem(item)"
        >
          <span class="min-w-0">
            <span class="block break-words font-black text-slate-950">{{ getTitle(item) }}</span>
            <span class="mt-1 block break-all text-xs font-bold text-blue-700">{{ getItemID(item) || "-" }}</span>
            <span class="mt-1 block break-words text-xs font-semibold text-slate-500">{{ getSubtitle(item) || "-" }}</span>
          </span>
          <span class="inline-flex items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 md:justify-self-end md:border-0 md:bg-transparent md:px-0 md:py-0 md:text-right">{{ copy.viewDetail }}</span>
        </button>
      </div>

      <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-4 py-4 md:px-5">
        <div class="flex w-full gap-2 md:w-auto">
          <button class="flex-1 rounded-xl border border-slate-300 px-4 py-2 text-sm font-bold disabled:opacity-40 md:flex-none" :disabled="page <= 1 && offset <= 0" @click="previousPage">
            {{ copy.previous }}
          </button>
          <button class="flex-1 rounded-xl border border-slate-300 px-4 py-2 text-sm font-bold md:flex-none" :disabled="items.length < pageSize" @click="nextPage">
            {{ copy.next }}
          </button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="showDetailModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <div class="flex h-full max-h-none w-full max-w-6xl flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black text-slate-950 md:text-2xl">{{ copy.detailTitle }}</h2>
              <p class="mt-1 break-all text-xs font-bold text-blue-700">{{ getItemID(selected) || "-" }}</p>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-xl leading-none text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closeDetailModal">
              ×
            </button>
          </div>

          <div class="min-h-0 flex-1 overflow-auto p-4 md:p-6">
            <div v-if="activeModule.actions?.length && selected" class="mb-5 flex flex-wrap justify-end gap-2">
              <template v-for="action in activeModule.actions" :key="action.key">
                <button
                  v-if="!action.showIf || action.showIf(selected)"
                  class="inline-flex h-10 w-full items-center justify-center rounded-xl border border-slate-300 px-4 text-sm font-black hover:border-slate-500 disabled:opacity-50 md:w-auto"
                  :disabled="!!actionLoading"
                  @click="runAction(action)"
                >
                  {{ actionLoading === action.key ? copy.processing : action.label }}
                </button>
              </template>
            </div>

            <div v-if="detailNotice" class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm font-bold text-amber-800">
              {{ detailNotice }}
            </div>
            <div v-if="detailLoading" class="mt-10 text-center text-sm font-bold text-slate-500">{{ copy.loading }}</div>
            <div v-else class="mt-6 space-y-5">
              <div v-if="structuredDetailSections.length" class="space-y-4">
                <section
                  v-for="section in structuredDetailSections"
                  :key="section.key"
                  class="rounded-2xl border border-slate-200 bg-white p-4"
                >
                  <h3 class="text-sm font-black text-slate-950">{{ section.title }}</h3>
                  <dl class="mt-3 grid gap-3 md:grid-cols-2">
                    <div
                      v-for="entry in section.entries"
                      :key="entry.key"
                      class="rounded-xl bg-slate-50 px-3 py-2"
                      :class="entry.key.endsWith('_json') || entry.key === 'message_payload' ? 'md:col-span-2' : ''"
                    >
                      <dt class="text-xs font-black text-slate-500">{{ entry.label }}</dt>
                      <dd
                        class="mt-1 whitespace-pre-wrap break-words text-sm font-bold text-slate-900"
                        :class="entry.mono ? 'font-mono text-xs leading-relaxed' : ''"
                      >
                        {{ entry.value }}
                      </dd>
                    </div>
                  </dl>
                </section>
              </div>
              <div>
                <h3 v-if="structuredDetailSections.length" class="mb-3 text-sm font-black text-slate-700">{{ copy.rawJson }}</h3>
                <pre class="max-h-[60vh] overflow-auto rounded-2xl bg-slate-950 p-4 text-xs font-semibold leading-relaxed text-slate-100 md:max-h-[64vh] md:p-5">{{ stringify(detail || selected || {}) }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </section>
</template>
