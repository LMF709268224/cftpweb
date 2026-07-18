<script setup lang="ts">
import { ChevronRight, FileText, List, Loader2, MessageSquare, Plus, RefreshCw, Search, Send, Users, X } from "lucide-vue-next"
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import ReadonlyField from "@/components/ReadonlyField.vue"
import UlidChip from "@/components/UlidChip.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type TabKey = "send" | "sent" | "templates"

const pageSize = 20
const templatePageSize = 10

const activeTab = ref<TabKey>("send")
const users = ref<JsonRecord[]>([])
const templates = ref<JsonRecord[]>([])
const selectedTemplate = ref<JsonRecord | null>(null)
const selectedUserIds = ref<string[]>([])
const templatePath = ref("")
const payload = ref("{\n}")
const msgType = ref(1)
const sending = ref(false)

const messages = ref<JsonRecord[]>([])
const messagesLoading = ref(false)
const messagePage = ref(1)
const statusFilter = ref("")
const total = ref(0)
const selectedMessage = ref<JsonRecord | null>(null)
const messageDetailOpen = ref(false)
const templateDetailOpen = ref(false)
const templateEditOpen = ref(false)

const usersLoading = ref(false)
const templatesLoading = ref(false)
const templateSaving = ref(false)
const templatePage = ref(1)
const totalTemplates = ref(0)
const editingTemplatePath = ref("")

const messageCursorStack = ref<string[]>([""])
const messageNextCursor = ref("")
const messageHasMore = ref(false)

const templateCursorStack = ref<string[]>([""])
const templateNextCursor = ref("")
const templateHasMore = ref(false)
const formPath = ref("")
const formTitle = ref("")
const formContent = ref("")
const formDescription = ref("")
const formParameterSchema = ref("{}")
const formVersion = ref(0)
const templateTitleInputRef = ref<HTMLInputElement | null>(null)
const { t } = useAdminLanguage()
const copy = computed(() => t.value.messagesAdmin)

const recipientSearch = ref("")

const filteredUsers = computed(() => {
  const keyword = recipientSearch.value.trim().toLowerCase()
  if (!keyword) return users.value
  return users.value.filter((user) => {
    const haystack = [userLabel(user), userId(user), String(user.email || ""), String(user.phone || "")]
    return haystack.some((text) => text.toLowerCase().includes(keyword))
  })
})

const sentStatusChips = computed(() => [
  { value: "", label: copy.value.sent.allStatus },
  { value: "1", label: copy.value.sent.unread },
  { value: "2", label: copy.value.sent.read },
])

const tabs = computed(() => [
  { key: "send" as const, label: copy.value.tabs.send, icon: Send, count: selectedUserIds.value.length },
  { key: "sent" as const, label: copy.value.tabs.sent, icon: List, count: total.value },
  { key: "templates" as const, label: copy.value.tabs.templates, icon: FileText, count: totalTemplates.value || templates.value.length },
])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const templateTotalPages = computed(() => Math.max(1, Math.ceil((totalTemplates.value || templates.value.length) / templatePageSize)))
const selectedTemplateFields = computed(() => selectedTemplate.value || {})
const selectedMessageFields = computed(() => selectedMessage.value || {})

function messageFieldLabel(key: string) {
  const labels = copy.value.sent.fieldLabels as Record<string, string>
  return labels[key] || key
}

function templateFieldLabel(key: string) {
  const labels = copy.value.templates.fieldLabels as Record<string, string>
  return labels[key] || key
}

function templateFieldValue(key: string, value: unknown) {
  if (key.endsWith("_at")) return formatDate(value) || String(value ?? "-")
  return value
}

function isMonoFieldKey(key: string) {
  return key === "id" || key === "path" || /(_id|_ulid|_path|_schema)$/.test(key)
}

function fieldIsMono(key: string, value: unknown) {
  return Array.isArray(value) || (!!value && typeof value === "object") || isMonoFieldKey(key)
}

function userId(user: JsonRecord) {
  return String(pickFirst(user, ["id", "user_id", "candidate_ulid", "ulid"]) || "")
}

function userLabel(user: JsonRecord) {
  return String(pickFirst(user, ["name", "nickname", "email", "phone", "id"]) || userId(user) || copy.value.defaults.user)
}

function userInitial(user: JsonRecord) {
  return userLabel(user).trim().charAt(0).toUpperCase() || "-"
}

function selectAllVisible() {
  const merged = new Set(selectedUserIds.value)
  for (const user of filteredUsers.value) {
    const id = userId(user)
    if (id) merged.add(id)
  }
  selectedUserIds.value = Array.from(merged)
}

function pathOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["path", "template_path", "template_id"]) || "")
}

function titleOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["title_tpl", "title", "name", "template_name", "path"]) || pathOf(template) || copy.value.defaults.template)
}

function messageId(message: JsonRecord | null | undefined) {
  return String(pickFirst(message || {}, ["message_id", "msg_id", "id"]) || "")
}

function messageUserId(message: JsonRecord | null | undefined) {
  return String(pickFirst(message || {}, ["user_id", "user_ulid"]) || "")
}

function messageStatusValue(message: JsonRecord | null | undefined) {
  const value = pickFirst(message || {}, ["status", "raw_status"])
  return value === undefined ? "" : String(value)
}

function messageStatusLabel(status: string) {
  if (status === "1") return copy.value.sent.unread
  if (status === "2") return copy.value.sent.read
  return status || "-"
}

function extractPayloadTemplate(...texts: unknown[]) {
  const vars = new Set<string>()
  const regex = /\{\{([^}]+)\}\}/g
  for (const text of texts) {
    if (typeof text !== "string") continue
    let match: RegExpExecArray | null
    while ((match = regex.exec(text))) {
      vars.add(match[1].trim().replace(/^\./, ""))
    }
  }
  if (!vars.size) return "{\n}"
  const result: Record<string, string> = {}
  for (const key of vars) result[key] = ""
  return JSON.stringify(result, null, 2)
}

function normalizeParameterSchema(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return "{}"
  JSON.parse(trimmed)
  return trimmed
}

function validatePayload() {
  if (!payload.value.trim()) return true
  try {
    const parsed = JSON.parse(payload.value)
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      toast.error(copy.value.toasts.payloadObjectRequired)
      return false
    }
    return true
  } catch (err) {
    toast.error(copy.value.toasts.payloadInvalid(err instanceof Error ? err.message : String(err)))
    return false
  }
}

async function loadUsers() {
  usersLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/user/list")
    const list = Array.isArray(data.users) ? data.users : []
    users.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.usersLoadFailed))
  } finally {
    usersLoading.value = false
  }
}

async function loadTemplates() {
  templatesLoading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(templatePageSize),
    })
    const cursor = templateCursorStack.value[templatePage.value - 1] || ""
    if (cursor) params.set("cursor", cursor)
    const data = await apiClient<JsonRecord>(`/api/messages/templates?${params}`)
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    totalTemplates.value = Number(data.total || data.total_count || templates.value.length)
    templateHasMore.value = Boolean(data.has_more)
    templateNextCursor.value = String(data.next_cursor || "")
    templateCursorStack.value = templateCursorStack.value.slice(0, templatePage.value)
    templateCursorStack.value[templatePage.value] = templateNextCursor.value
    if (!selectedTemplate.value || !templates.value.some((item) => pathOf(item) === pathOf(selectedTemplate.value))) {
      selectedTemplate.value = templates.value[0] || null
    }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templatesLoadFailed))
  } finally {
    templatesLoading.value = false
  }
}

async function loadTemplateDetail(path: string) {
  if (!path) return null
  return apiClient<JsonRecord>(`/api/messages/templates/detail?path=${encodeURIComponent(path)}`)
}

async function selectTemplate(template: JsonRecord) {
  selectedTemplate.value = template
  const path = pathOf(template)
  try {
    const detail = await loadTemplateDetail(path)
    if (detail) selectedTemplate.value = { ...template, ...detail }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
  }
}

async function loadSentMessages() {
  messagesLoading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(pageSize),
    })
    const cursor = messageCursorStack.value[messagePage.value - 1] || ""
    if (cursor) params.set("cursor", cursor)
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/messages/sent?${params}`)
    const list = Array.isArray(data.messages) ? data.messages : []
    messages.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || messages.value.length)
    messageHasMore.value = Boolean(data.has_more)
    messageNextCursor.value = String(data.next_cursor || "")
    messageCursorStack.value = messageCursorStack.value.slice(0, messagePage.value)
    messageCursorStack.value[messagePage.value] = messageNextCursor.value
    if (!selectedMessage.value || !messages.value.some((item) => messageId(item) === messageId(selectedMessage.value))) {
      openMessage(messages.value[0] || null, messageDetailOpen.value)
    }
    if (!messages.value.length) messageDetailOpen.value = false
  } catch (err) {
    console.error(err)
    messages.value = []
    selectedMessage.value = null
    messageDetailOpen.value = false
    toast.error(apiErrorMessage(err, copy.value.toasts.sentLoadFailed))
  } finally {
    messagesLoading.value = false
  }
}

function openMessage(message: JsonRecord | null, open = true) {
  selectedMessage.value = message
  messageDetailOpen.value = open && !!message
}

function closeMessageDetail() {
  messageDetailOpen.value = false
}

async function openTemplateDetail(template: JsonRecord) {
  await selectTemplate(template)
  templateDetailOpen.value = !!selectedTemplate.value
}

function closeTemplateDetail() {
  templateDetailOpen.value = false
}

function startTemplateCreate() {
  resetTemplateForm()
  templateEditOpen.value = true
  void nextTick(() => {
    templateTitleInputRef.value?.focus()
  })
}

async function startTemplateEdit(template: JsonRecord | null = selectedTemplate.value) {
  await editTemplate(template)
  templateEditOpen.value = !!template
  await nextTick()
  templateTitleInputRef.value?.focus()
}

function closeTemplateEdit() {
  templateEditOpen.value = false
  resetTemplateForm()
}

async function sendMessage() {
  if (!selectedUserIds.value.length) {
    toast.error(copy.value.toasts.recipientsRequired)
    return
  }
  if (!templatePath.value) {
    toast.error(copy.value.toasts.templateRequired)
    return
  }
  if (!validatePayload()) return

  sending.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/messages/send", {
      method: "POST",
      body: JSON.stringify({
        user_ids: selectedUserIds.value,
        template_path: templatePath.value,
        payload: payload.value,
        msg_type: msgType.value,
      }),
    })
    toast.success(copy.value.toasts.sendSuccess(data.count))
    selectedUserIds.value = []
    templatePath.value = ""
    payload.value = "{\n}"
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.sendFailed))
  } finally {
    sending.value = false
  }
}

async function editTemplate(template: JsonRecord | null = selectedTemplate.value) {
  if (!template) {
    resetTemplateForm()
    return
  }
  const path = pathOf(template)
  editingTemplatePath.value = path
  formPath.value = path
  formTitle.value = titleOf(template)
  formDescription.value = String(template.description || "")
  formParameterSchema.value = String(template.parameter_schema || "{}")
  formVersion.value = Number(template.version || 0)
  formContent.value = ""

  try {
    const detail = await loadTemplateDetail(path)
    if (detail) {
      selectedTemplate.value = { ...template, ...detail }
      formTitle.value = String(detail.title_tpl || formTitle.value)
      formContent.value = String(detail.content_tpl || "")
      formDescription.value = String(detail.description || formDescription.value)
      formParameterSchema.value = String(detail.parameter_schema || "{}")
      formVersion.value = Number(detail.version || formVersion.value)
    }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
  }
}

function resetTemplateForm() {
  editingTemplatePath.value = ""
  formPath.value = ""
  formTitle.value = ""
  formContent.value = ""
  formDescription.value = ""
  formParameterSchema.value = "{}"
  formVersion.value = 0
}

async function saveTemplate() {
  const path = editingTemplatePath.value || formPath.value.trim()
  if (!path || !formTitle.value.trim() || !formContent.value.trim()) {
    toast.error(copy.value.toasts.templateFieldsRequired)
    return
  }
  let parameterSchema = "{}"
  try {
    parameterSchema = normalizeParameterSchema(formParameterSchema.value)
  } catch {
    toast.error(copy.value.toasts.parameterSchemaInvalid)
    return
  }

  templateSaving.value = true
  try {
    await apiClient("/api/messages/templates", {
      method: editingTemplatePath.value ? "PUT" : "POST",
      body: JSON.stringify({
        path,
        title_tpl: formTitle.value,
        content_tpl: formContent.value,
        description: formDescription.value,
        parameter_schema: parameterSchema,
        current_version: formVersion.value,
      }),
    })
    toast.success(editingTemplatePath.value ? copy.value.toasts.templateUpdated : copy.value.toasts.templateCreated)
    const savedPath = path
    templateEditOpen.value = false
    resetTemplateForm()
    await loadTemplates()
    const savedTemplate = templates.value.find((item) => pathOf(item) === savedPath)
    if (savedTemplate) await selectTemplate(savedTemplate)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateSaveFailed))
  } finally {
    templateSaving.value = false
  }
}

watch(templatePath, async (path) => {
  if (!path) {
    payload.value = "{\n}"
    return
  }
  const template = templates.value.find((item) => pathOf(item) === path)
  payload.value = extractPayloadTemplate(template?.title_tpl, template?.content_tpl)
  try {
    const detail = await loadTemplateDetail(path)
    payload.value = extractPayloadTemplate(detail?.title_tpl, detail?.content_tpl)
  } catch {
    // Keep the best-effort payload generated from the list response.
  }
})

watch(statusFilter, () => {
  messagePage.value = 1
  messageCursorStack.value = [""]
  messageNextCursor.value = ""
  messageHasMore.value = false
})

watch([activeTab, messagePage, statusFilter], () => {
  if (activeTab.value === "sent") void loadSentMessages()
})

watch(templatePage, () => {
  if (activeTab.value === "templates") void loadTemplates()
})

onMounted(async () => {
  await Promise.all([loadUsers(), loadTemplates()])
})
</script>

<template>
  <main class="ui-page">
    <!-- Page header -->
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="ui-btn ui-btn-secondary" type="button" @click="loadUsers">
          <RefreshCw class="h-4 w-4 text-slate-400" :class="usersLoading ? 'animate-spin' : ''" />
          {{ copy.refreshUsers }}
        </button>
        <button class="ui-btn ui-btn-secondary" type="button" @click="loadTemplates">
          <RefreshCw class="h-4 w-4 text-slate-400" :class="templatesLoading ? 'animate-spin' : ''" />
          {{ copy.refreshTemplates }}
        </button>
      </div>
    </header>

    <!-- Tabs -->
    <div class="mt-6 flex gap-5 overflow-x-auto border-b border-hairline">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="ui-tab"
        :class="activeTab === tab.key ? 'ui-tab-active' : ''"
        type="button"
        @click="activeTab = tab.key"
      >
        <component :is="tab.icon" class="h-4 w-4" />
        {{ tab.label }}
        <span v-if="tab.count" class="tnum ui-badge bg-slate-100 text-slate-600">{{ tab.count }}</span>
      </button>
    </div>

    <!-- Send message -->
    <section v-if="activeTab === 'send'" class="ui-card mt-4 overflow-hidden">
      <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <h2 class="text-base font-semibold text-ink">{{ copy.send.title }}</h2>
      </div>
      <div class="grid gap-6 p-4 md:p-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <!-- Recipient picker -->
        <div>
          <div class="mb-1.5 flex items-center justify-between gap-3">
            <span class="ui-label">{{ copy.send.recipients }}</span>
            <div class="flex gap-4 text-sm">
              <button class="font-medium text-primary-700 transition-colors hover:text-primary-600" type="button" @click="selectAllVisible">{{ copy.send.selectAll }}</button>
              <button class="font-medium text-slate-500 transition-colors hover:text-slate-700" type="button" @click="selectedUserIds = []">{{ copy.send.clear }}</button>
            </div>
          </div>
          <div class="max-h-[400px] overflow-y-auto rounded-[10px] border border-slate-200 lg:max-h-[560px]">
            <div class="sticky top-0 z-10 border-b border-hairline bg-white/95 p-2.5 backdrop-blur">
              <div class="relative">
                <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
                <input v-model="recipientSearch" class="ui-input pl-9 pr-10" :placeholder="t.common.searchCandidate" :aria-label="t.common.searchCandidate" />
                <button
                  v-if="recipientSearch"
                  class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                  type="button"
                  :aria-label="copy.send.clear"
                  :title="copy.send.clear"
                  @click="recipientSearch = ''"
                >
                  <X class="h-4 w-4" />
                </button>
              </div>
              <div class="mt-2 flex items-center justify-between gap-3 px-0.5">
                <span class="tnum ui-badge bg-primary-50 text-primary-700">{{ copy.send.selectedUsers(selectedUserIds.length) }}</span>
                <span class="tnum text-xs text-slate-400">{{ filteredUsers.length }}/{{ users.length }}</span>
              </div>
            </div>
            <div v-if="usersLoading" class="space-y-2 p-2">
              <div v-for="i in 6" :key="`user-sk-${i}`" class="skeleton h-10 w-full" />
            </div>
            <div v-else-if="filteredUsers.length" class="p-2">
              <label
                v-for="user in filteredUsers"
                :key="userId(user)"
                class="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 transition-colors"
                :class="selectedUserIds.includes(userId(user)) ? 'bg-primary-50/60' : 'hover:bg-slate-50'"
              >
                <input v-model="selectedUserIds" class="h-4 w-4 shrink-0 accent-primary-600" type="checkbox" :value="userId(user)" />
                <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-50 text-xs font-medium text-primary-700" aria-hidden="true">{{ userInitial(user) }}</span>
                <span class="min-w-0 flex-1 truncate text-sm font-medium text-ink">{{ userLabel(user) }}</span>
                <UlidChip :value="userId(user)" />
              </label>
            </div>
            <div v-else-if="users.length" class="ui-empty">
              <div class="ui-empty-icon">
                <Search class="h-6 w-6" />
              </div>
              <p class="ui-empty-title">{{ t.common.noResults }}</p>
            </div>
            <div v-else class="ui-empty">
              <div class="ui-empty-icon">
                <Users class="h-6 w-6" />
              </div>
              <p class="ui-empty-title">{{ copy.send.noUsers }}</p>
            </div>
          </div>
        </div>

        <!-- Compose -->
        <div class="space-y-4">
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.send.template }}</span>
            <select v-model="templatePath" class="ui-input">
              <option value="">{{ copy.send.selectTemplate }}</option>
              <option v-for="template in templates" :key="pathOf(template)" :value="pathOf(template)">
                {{ titleOf(template) }} ({{ pathOf(template) }})
              </option>
            </select>
          </label>

          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.send.messageType }}</span>
            <select v-model.number="msgType" class="ui-input">
              <option :value="1">{{ copy.send.messageTypes.system }}</option>
              <option :value="2">{{ copy.send.messageTypes.announcement }}</option>
              <option :value="3">{{ copy.send.messageTypes.marketing }}</option>
              <option :value="4">{{ copy.send.messageTypes.payment }}</option>
              <option :value="5">{{ copy.send.messageTypes.other }}</option>
            </select>
          </label>

          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.send.payload }}</span>
            <textarea v-model="payload" class="min-h-52 w-full rounded-[10px] border border-slate-200 bg-white p-3 font-mono text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
          </label>

          <span class="inline-flex w-full sm:w-auto" :title="!selectedUserIds.length ? copy.toasts.recipientsRequired : undefined">
            <button
              class="ui-btn ui-btn-primary w-full"
              :disabled="sending || !selectedUserIds.length"
              :aria-label="!selectedUserIds.length ? copy.toasts.recipientsRequired : undefined"
              type="button"
              @click="sendMessage"
            >
              <Loader2 v-if="sending" class="h-4 w-4 animate-spin" />
              <Send v-else class="h-4 w-4" />
              {{ sending ? copy.send.sending : copy.send.sendButton }}
            </button>
          </span>
        </div>
      </div>
    </section>

    <!-- Sent messages -->
    <section v-else-if="activeTab === 'sent'" class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.sent.title }}</h2>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.sent.totalText(total) }}</span>
      </div>
      <div class="flex flex-wrap items-center gap-2 border-b border-hairline px-4 py-3 md:px-6">
        <button
          v-for="chip in sentStatusChips"
          :key="`msg-status-${chip.value}`"
          type="button"
          class="inline-flex h-8 shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 text-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20"
          :class="statusFilter === chip.value
            ? 'bg-primary-600 text-white shadow-sm'
            : 'border border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-900'"
          :aria-pressed="statusFilter === chip.value"
          @click="statusFilter = chip.value"
        >
          {{ chip.label }}
        </button>
      </div>
      <div class="hidden grid-cols-[minmax(0,1fr)_170px_110px_170px_120px] gap-5 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid">
        <span>{{ copy.sent.columns.message }}</span>
        <span>{{ copy.sent.columns.user }}</span>
        <span class="text-center">{{ copy.sent.columns.status }}</span>
        <span class="text-right">{{ copy.sent.columns.time }}</span>
        <span class="text-right">{{ copy.sent.columns.action }}</span>
      </div>
      <div v-if="messagesLoading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`msg-sk-${i}`" class="px-4 py-3.5 md:grid md:grid-cols-[minmax(0,1fr)_170px_110px_170px_120px] md:items-center md:gap-5">
          <div>
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton mt-2 h-3 w-3/5" />
          </div>
          <div class="skeleton mt-3 h-4 w-24 md:mt-0" />
          <div class="skeleton mt-3 h-5 w-14 rounded-full md:mt-0 md:justify-self-center" />
          <div class="skeleton mt-3 h-4 w-24 md:mt-0 md:justify-self-end" />
          <div class="skeleton mt-3 h-4 w-16 md:mt-0 md:justify-self-end" />
        </div>
      </div>
      <div v-else-if="messages.length" class="divide-y divide-hairline">
        <div
          v-for="message in messages"
          :key="messageId(message)"
          class="ui-table-row flex cursor-pointer flex-col px-4 py-4 md:grid md:grid-cols-[minmax(0,1fr)_170px_110px_170px_120px] md:items-center md:gap-5 md:py-3.5"
          :class="messageId(selectedMessage) === messageId(message) ? 'bg-primary-50/60' : ''"
          role="button"
          tabindex="0"
          :aria-label="copy.sent.viewDetails"
          @click="openMessage(message)"
          @keydown.enter.self.prevent="openMessage(message)"
          @keydown.space.self.prevent="openMessage(message)"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-ink md:truncate">{{ pickFirst(message, ["title", "subject", "template_path", "message_id"]) || copy.defaults.message }}</div>
            <div class="mt-1"><UlidChip :value="messageId(message)" /></div>
          </div>
          <div class="mt-3 flex min-w-0 items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.sent.columns.user }}</span>
            <UlidChip :value="messageUserId(message)" />
          </div>
          <div class="mt-2.5 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-center">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.sent.columns.status }}</span>
            <span v-if="messageStatusValue(message)" class="ui-badge" :class="badgeClass(messageStatusValue(message))">{{ messageStatusLabel(messageStatusValue(message)) }}</span>
            <span v-else class="text-sm text-slate-400">-</span>
          </div>
          <div class="mt-2.5 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-right">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.sent.columns.time }}</span>
            <span class="tnum text-sm text-slate-600">{{ formatDate(String(pickFirst(message, ["created_at", "sent_at", "updated_at"]) || "")) }}</span>
          </div>
          <div class="mt-3 flex justify-end md:mt-0">
            <span class="inline-flex items-center gap-1 text-sm font-medium text-primary-700 transition-colors hover:text-primary-600">
              {{ copy.sent.viewDetails }}
              <ChevronRight class="h-4 w-4" />
            </span>
          </div>
        </div>
      </div>
      <div v-else class="ui-empty">
        <div class="ui-empty-icon">
          <MessageSquare class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.sent.empty }}</p>
      </div>
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.sent.pageText(messagePage, totalPages) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="messagePage <= 1" @click="messagePage--">{{ copy.sent.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!messageHasMore" @click="messagePage++">{{ copy.sent.next }}</button>
        </div>
      </div>
    </section>

    <!-- Templates -->
    <section v-else class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.templates.listTitle }}</h2>
        </div>
        <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
          <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.templates.totalText(totalTemplates || templates.length) }}</span>
          <button class="ui-btn ui-btn-primary flex-1 sm:flex-none" type="button" @click="startTemplateCreate">
            <Plus class="h-4 w-4" />
            {{ copy.templates.createTitle }}
          </button>
        </div>
      </div>
      <div class="hidden grid-cols-[minmax(0,1fr)_140px_180px_160px] gap-5 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid">
        <span>{{ copy.templates.columns.template }}</span>
        <span class="text-center">{{ copy.templates.columns.version }}</span>
        <span class="text-right">{{ copy.templates.columns.updatedAt }}</span>
        <span class="text-right">{{ copy.templates.columns.action }}</span>
      </div>
      <div v-if="templatesLoading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`tpl-sk-${i}`" class="px-4 py-3.5 md:grid md:grid-cols-[minmax(0,1fr)_140px_180px_160px] md:items-center md:gap-5">
          <div>
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton mt-2 h-3 w-3/5" />
          </div>
          <div class="skeleton mt-3 h-5 w-12 rounded-full md:mt-0 md:justify-self-center" />
          <div class="skeleton mt-3 h-4 w-24 md:mt-0 md:justify-self-end" />
          <div class="skeleton mt-3 h-4 w-20 md:mt-0 md:justify-self-end" />
        </div>
      </div>
      <div v-else-if="templates.length" class="divide-y divide-hairline">
        <div
          v-for="template in templates"
          :key="pathOf(template)"
          class="ui-table-row flex cursor-pointer flex-col px-4 py-4 md:grid md:grid-cols-[minmax(0,1fr)_140px_180px_160px] md:items-center md:gap-5 md:py-3.5"
          :class="pathOf(selectedTemplate) === pathOf(template) ? 'bg-primary-50/60' : ''"
          role="button"
          tabindex="0"
          :aria-label="copy.templates.viewDetails"
          @click="openTemplateDetail(template)"
          @keydown.enter.self.prevent="openTemplateDetail(template)"
          @keydown.space.self.prevent="openTemplateDetail(template)"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-ink md:truncate">{{ titleOf(template) }}</div>
            <div class="mt-1"><UlidChip :value="pathOf(template)" :truncate="false" /></div>
            <div v-if="template.description" class="mt-1 break-words text-xs text-slate-400 md:truncate">{{ template.description }}</div>
          </div>
          <div class="mt-3 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-center">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.templates.columns.version }}</span>
            <span class="tnum ui-badge bg-slate-100 text-slate-600">v{{ String(template.version || "-") }}</span>
          </div>
          <div class="mt-2.5 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-right">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.templates.columns.updatedAt }}</span>
            <span class="tnum text-sm text-slate-600">{{ formatDate(String(pickFirst(template, ["updated_at", "created_at"]) || "")) }}</span>
          </div>
          <div class="mt-3 flex items-center justify-end gap-2 md:mt-0">
            <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click.stop="startTemplateEdit(template)">{{ copy.templates.edit }}</button>
            <ChevronRight class="h-4 w-4 shrink-0 text-slate-400" />
          </div>
        </div>
      </div>
      <div v-else class="ui-empty">
        <div class="ui-empty-icon">
          <FileText class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.templates.empty }}</p>
      </div>
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.sent.pageText(templatePage, templateTotalPages) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="templatePage <= 1 || templatesLoading" @click="templatePage--">{{ copy.sent.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!templateHasMore || templatesLoading" @click="templatePage++">{{ copy.sent.next }}</button>
        </div>
      </div>
    </section>

    <!-- Message detail dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="messageDetailOpen && selectedMessage" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1120px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ copy.sent.detailTitle }}</h2>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.sent.close"
                @click="closeMessageDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
              <div class="grid gap-4 md:grid-cols-2">
                <ReadonlyField
                  v-for="(value, key) in selectedMessageFields"
                  :key="key"
                  :label="messageFieldLabel(String(key))"
                  :value="value"
                  :mono="fieldIsMono(String(key), value)"
                  :max-height="Array.isArray(value) || (!!value && typeof value === 'object') ? '180px' : undefined"
                />
              </div>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <!-- Template detail dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="templateDetailOpen && selectedTemplate" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1120px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ copy.templates.detailTitle }}</h2>
                <p class="tnum mt-1 break-all text-sm text-slate-500">{{ pathOf(selectedTemplate) }}</p>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.sent.close"
                @click="closeTemplateDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
              <div class="grid gap-4 md:grid-cols-2">
                <ReadonlyField
                  v-for="(value, key) in selectedTemplateFields"
                  :key="key"
                  :label="templateFieldLabel(String(key))"
                  :value="templateFieldValue(String(key), value)"
                  :mono="fieldIsMono(String(key), value)"
                  :max-height="Array.isArray(value) || (!!value && typeof value === 'object') ? '180px' : undefined"
                />
              </div>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <!-- Template edit dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="templateEditOpen" class="ui-modal-mask">
          <form class="ui-modal-panel max-w-[920px]" @submit.prevent="saveTemplate">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ editingTemplatePath ? copy.templates.editTitle : copy.templates.createTitle }}</h2>
                <p v-if="editingTemplatePath" class="tnum mt-1 break-all text-sm text-slate-500">{{ formPath }}</p>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.sent.close"
                @click="closeTemplateEdit"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 space-y-4 overflow-y-auto p-4 md:p-6">
              <label class="grid gap-1.5">
                <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.templates.path }}</span>
                <input v-model="formPath" class="ui-input" :disabled="!!editingTemplatePath" />
              </label>
              <label class="grid gap-1.5">
                <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.templates.titleTemplate }}</span>
                <input ref="templateTitleInputRef" v-model="formTitle" class="ui-input" />
              </label>
              <label class="grid gap-1.5">
                <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.templates.contentTemplate }}</span>
                <textarea v-model="formContent" class="min-h-56 w-full rounded-[10px] border border-slate-200 bg-white p-3 text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
              </label>
              <label class="grid gap-1.5">
                <span class="ui-label">{{ copy.templates.description }}</span>
                <textarea v-model="formDescription" class="min-h-24 w-full rounded-[10px] border border-slate-200 bg-white p-3 text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
              </label>
              <label class="grid gap-1.5">
                <span class="ui-label">{{ copy.templates.parameterSchema }}</span>
                <textarea v-model="formParameterSchema" class="min-h-32 w-full rounded-[10px] border border-slate-200 bg-white p-3 font-mono text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
              </label>
            </div>

            <div class="flex flex-col items-stretch justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
              <button class="ui-btn ui-btn-secondary" type="button" @click="closeTemplateEdit">{{ copy.templates.cancel }}</button>
              <button class="ui-btn ui-btn-primary" :disabled="templateSaving" type="submit">
                <Loader2 v-if="templateSaving" class="h-4 w-4 animate-spin" />
                {{ templateSaving ? copy.templates.saving : copy.templates.save }}
              </button>
            </div>
          </form>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
