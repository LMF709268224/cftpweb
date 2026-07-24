<script setup lang="ts">
import { FileText, List, Loader2, Plus, RefreshCw, Send, X } from "lucide-vue-next"
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import ReadonlyField from "@/components/ReadonlyField.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

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
let templateDetailRequestId = 0
let templateEditRequestId = 0
let templatePayloadRequestId = 0
const { t } = useAdminLanguage()
const copy = computed(() => t.value.messagesAdmin)

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

function userId(user: JsonRecord) {
  return String(pickFirst(user, ["id", "user_id", "candidate_ulid", "ulid"]) || "")
}

function userLabel(user: JsonRecord) {
  return String(pickFirst(user, ["name", "nickname", "email", "phone", "id"]) || userId(user) || copy.value.defaults.user)
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
  const path = pathOf(template)
  if (!path) return false
  const requestId = ++templateDetailRequestId
  selectedTemplate.value = template
  try {
    const detail = await loadTemplateDetail(path)
    if (requestId !== templateDetailRequestId || pathOf(selectedTemplate.value) !== path) return false
    if (detail) selectedTemplate.value = { ...template, ...detail }
    return true
  } catch (err) {
    if (requestId !== templateDetailRequestId || pathOf(selectedTemplate.value) !== path) return false
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
    return true
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
  selectedMessage.value = null
}

async function openTemplateDetail(template: JsonRecord) {
  templateEditRequestId += 1
  templateEditOpen.value = false
  resetTemplateForm()
  const current = await selectTemplate(template)
  if (!current) return
  templateDetailOpen.value = true
}

function closeTemplateDetail() {
  templateDetailRequestId += 1
  templateDetailOpen.value = false
  selectedTemplate.value = null
}

function startTemplateCreate() {
  templateDetailRequestId += 1
  templateEditRequestId += 1
  templateDetailOpen.value = false
  selectedTemplate.value = null
  resetTemplateForm()
  templateEditOpen.value = true
  void nextTick(() => {
    templateTitleInputRef.value?.focus()
  })
}

async function startTemplateEdit(template: JsonRecord | null = selectedTemplate.value) {
  templateDetailRequestId += 1
  templateDetailOpen.value = false
  const current = await editTemplate(template)
  if (!current) return
  templateEditOpen.value = true
  await nextTick()
  templateTitleInputRef.value?.focus()
}

function closeTemplateEdit() {
  templateEditRequestId += 1
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
    return false
  }
  const path = pathOf(template)
  if (!path) {
    resetTemplateForm()
    return false
  }
  const requestId = ++templateEditRequestId
  editingTemplatePath.value = path
  formPath.value = path
  formTitle.value = titleOf(template)
  formDescription.value = String(template.description || "")
  formParameterSchema.value = String(template.parameter_schema || "{}")
  formVersion.value = Number(template.version || 0)
  formContent.value = ""

  try {
    const detail = await loadTemplateDetail(path)
    if (requestId !== templateEditRequestId || editingTemplatePath.value !== path) return false
    if (detail) {
      selectedTemplate.value = { ...template, ...detail }
      formTitle.value = String(detail.title_tpl || formTitle.value)
      formContent.value = String(detail.content_tpl || "")
      formDescription.value = String(detail.description || formDescription.value)
      formParameterSchema.value = String(detail.parameter_schema || "{}")
      formVersion.value = Number(detail.version || formVersion.value)
    }
    return true
  } catch (err) {
    if (requestId !== templateEditRequestId || editingTemplatePath.value !== path) return false
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
    return true
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
    closeTemplateEdit()
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
  const requestId = ++templatePayloadRequestId
  if (!path) {
    payload.value = "{\n}"
    return
  }
  const template = templates.value.find((item) => pathOf(item) === path)
  payload.value = extractPayloadTemplate(template?.title_tpl, template?.content_tpl)
  try {
    const detail = await loadTemplateDetail(path)
    if (requestId !== templatePayloadRequestId || templatePath.value !== path) return
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex h-10 items-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm" type="button" @click="loadUsers">
          <RefreshCw class="h-4 w-4" :class="usersLoading ? 'animate-spin' : ''" />
          {{ copy.refreshUsers }}
        </button>
        <button class="inline-flex h-10 items-center gap-2 rounded-xl border bg-white px-4 text-sm font-bold shadow-sm" type="button" @click="loadTemplates">
          <RefreshCw class="h-4 w-4" :class="templatesLoading ? 'animate-spin' : ''" />
          {{ copy.refreshTemplates }}
        </button>
      </div>
    </header>

    <section class="rounded-2xl border border-slate-200 bg-white p-3 shadow-sm md:rounded-3xl">
      <div class="grid grid-cols-3 gap-2 md:flex md:flex-wrap md:gap-3">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="min-h-16 rounded-2xl border px-2 py-2 text-center transition md:min-h-12 md:flex-none md:min-w-48 md:px-4 md:py-3 md:text-left"
          :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50 shadow-sm' : 'border-slate-100 hover:bg-slate-50'"
          type="button"
          @click="activeTab = tab.key"
        >
          <div class="flex h-full items-center justify-center md:justify-between">
            <span class="inline-flex flex-col items-center justify-center gap-1 whitespace-nowrap text-xs font-black leading-4 md:flex-row md:gap-2 md:text-base">
              <component :is="tab.icon" class="h-4 w-4" />
              {{ tab.label }}
            </span>
            
          </div>
        </button>
      </div>
    </section>

    <div>

      <main class="min-w-0">
        <section v-if="activeTab === 'send'" class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-6">
          <h2 class="text-xl font-black">{{ copy.send.title }}</h2>
          <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_1fr]">
            <div>
              <div class="mb-2 flex items-center justify-between">
                <label class="font-bold">{{ copy.send.recipients }}</label>
                <div class="flex gap-3 text-sm">
                  <button class="font-bold text-blue-700" type="button" @click="selectedUserIds = users.map(userId).filter(Boolean)">{{ copy.send.selectAll }}</button>
                  <button class="font-bold text-slate-500" type="button" @click="selectedUserIds = []">{{ copy.send.clear }}</button>
                </div>
              </div>
              <div class="max-h-[360px] overflow-y-auto rounded-2xl border border-slate-200 p-3 md:max-h-[520px]">
                <label v-for="user in users" :key="userId(user)" class="flex cursor-pointer items-center gap-3 rounded-xl px-3 py-2 hover:bg-slate-50">
                  <input v-model="selectedUserIds" class="h-4 w-4" type="checkbox" :value="userId(user)" />
                  <span class="font-semibold">{{ userLabel(user) }}</span>
                  <span class="break-all text-xs text-slate-400">{{ userId(user) }}</span>
                </label>
                <div v-if="!users.length" class="p-6 text-center text-slate-500">{{ copy.send.noUsers }}</div>
              </div>
              <p class="mt-2 text-sm text-slate-500">{{ copy.send.selectedUsers(selectedUserIds.length) }}</p>
            </div>

            <div class="space-y-4">
              <label class="block">
                <span class="font-bold">{{ copy.send.template }}</span>
                <select v-model="templatePath" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4">
                  <option value="">{{ copy.send.selectTemplate }}</option>
                  <option v-for="template in templates" :key="pathOf(template)" :value="pathOf(template)">
                    {{ titleOf(template) }} ({{ pathOf(template) }})
                  </option>
                </select>
              </label>

              <label class="block">
                <span class="font-bold">{{ copy.send.messageType }}</span>
                <select v-model.number="msgType" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4">
                  <option :value="1">{{ copy.send.messageTypes.system }}</option>
                  <option :value="2">{{ copy.send.messageTypes.announcement }}</option>
                  <option :value="3">{{ copy.send.messageTypes.marketing }}</option>
                  <option :value="4">{{ copy.send.messageTypes.payment }}</option>
                  <option :value="5">{{ copy.send.messageTypes.other }}</option>
                </select>
              </label>

              <label class="block">
                <span class="font-bold">{{ copy.send.payload }}</span>
                <textarea v-model="payload" class="mt-2 min-h-52 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" />
              </label>

              <button class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50 sm:w-auto" :disabled="sending" type="button" @click="sendMessage">
                <Loader2 v-if="sending" class="h-4 w-4 animate-spin" />
                <Send v-else class="h-4 w-4" />
                {{ sending ? copy.send.sending : copy.send.sendButton }}
              </button>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'sent'" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-4 md:p-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black">{{ copy.sent.title }}</h2>
              
            </div>
            <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
              <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.sent.totalText(total) }}</span>
              <select v-model="statusFilter" class="h-11 min-w-0 flex-1 rounded-xl border border-slate-200 px-4 text-sm sm:flex-none">
              <option value="">{{ copy.sent.allStatus }}</option>
              <option value="1">{{ copy.sent.unread }}</option>
              <option value="2">{{ copy.sent.read }}</option>
            </select>
            </div>
          </div>
          <div class="hidden grid-cols-[minmax(0,1fr)_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500 md:grid">
            <span>{{ copy.sent.columns.message }}</span>
            <span class="text-right">{{ copy.sent.columns.time }}</span>
            <span class="text-right">{{ copy.sent.columns.action }}</span>
          </div>
          <div v-if="messagesLoading" class="p-8 text-center text-slate-500 md:p-12">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.sent.loading }}
          </div>
          <div v-else-if="messages.length" class="divide-y divide-slate-100">
            <div
              v-for="message in messages"
              :key="messageId(message)"
              class="flex cursor-pointer flex-col gap-3 px-4 py-4 transition hover:bg-sky-50 md:grid md:grid-cols-[minmax(0,1fr)_180px_112px] md:items-center md:gap-5 md:px-5"
              :class="messageId(selectedMessage) === messageId(message) ? 'bg-sky-50' : ''"
              role="button"
              tabindex="0"
              @click="openMessage(message)"
              @keydown.enter.prevent="openMessage(message)"
              @keydown.space.prevent="openMessage(message)"
            >
              <div class="min-w-0">
                <div class="break-words font-black text-slate-950 md:truncate">{{ pickFirst(message, ["title", "subject", "template_path", "message_id"]) || copy.defaults.message }}</div>
                <div class="mt-1 break-all text-xs font-semibold text-slate-500">{{ copy.sent.idPrefix }}{{ messageId(message) || "-" }}</div>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.sent.columns.time }}</span>
                <span class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(pickFirst(message, ["created_at", "sent_at", "updated_at"]) || "")) }}</span>
              </div>
              <div class="text-right">
                <button
                  class="inline-flex w-full items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline md:w-auto md:border-0 md:bg-transparent md:px-0 md:py-0"
                  type="button"
                  @click.stop="openMessage(message)"
                >
                  {{ copy.sent.viewDetails }}
                </button>
              </div>
            </div>
          </div>
          <div v-else class="p-8 text-center text-slate-500 md:p-12">{{ copy.sent.empty }}</div>
          <div class="flex flex-col items-stretch justify-between gap-3 border-t border-slate-200 p-4 sm:flex-row sm:items-center md:p-5">
            <span class="text-center text-sm font-bold text-slate-500 sm:text-left">{{ copy.sent.pageText(messagePage, totalPages) }}</span>
            <div class="flex flex-col gap-3 sm:flex-row">
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="messagePage <= 1" @click="messagePage--">{{ copy.sent.prev }}</button>
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!messageHasMore" @click="messagePage++">{{ copy.sent.next }}</button>
            </div>
          </div>
        </section>

        <section v-else class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
          <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-4 md:p-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black">{{ copy.templates.listTitle }}</h2>
              
            </div>
            <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
              <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.templates.totalText(totalTemplates || templates.length) }}</span>
              <button class="inline-flex h-10 flex-1 items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 text-sm font-bold text-white shadow-sm sm:flex-none" type="button" @click="startTemplateCreate">
              <Plus class="h-4 w-4" />
              {{ copy.templates.createTitle }}
            </button>
          </div>
          </div>
          <div class="hidden grid-cols-[minmax(0,1fr)_140px_180px_160px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500 md:grid">
            <span>{{ copy.templates.columns.template }}</span>
            <span class="text-center">{{ copy.templates.columns.version }}</span>
            <span class="text-right">{{ copy.templates.columns.updatedAt }}</span>
            <span class="text-right">{{ copy.templates.columns.action }}</span>
          </div>
          <div v-if="templatesLoading" class="p-8 text-center text-slate-500 md:p-12">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.templates.loading }}
          </div>
          <div v-else-if="templates.length" class="divide-y divide-slate-100">
            <div
              v-for="template in templates"
              :key="pathOf(template)"
              class="flex flex-col gap-3 px-4 py-4 transition hover:bg-slate-50 md:grid md:grid-cols-[minmax(0,1fr)_140px_180px_160px] md:items-center md:gap-5 md:px-5"
              :class="pathOf(selectedTemplate) === pathOf(template) ? 'bg-sky-50' : ''"
            >
              <div class="min-w-0">
                <div class="break-words font-black text-slate-950 md:truncate">{{ titleOf(template) }}</div>
                <div class="mt-1 break-all text-sm font-semibold text-slate-500">{{ pathOf(template) }}</div>
                <div v-if="template.description" class="mt-1 break-words text-xs font-semibold text-slate-400 md:truncate">{{ template.description }}</div>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0 md:text-center">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.templates.columns.version }}</span>
                <span class="inline-flex rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">v{{ String(template.version || "-") }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.templates.columns.updatedAt }}</span>
                <span class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(pickFirst(template, ["updated_at", "created_at"]) || "")) }}</span>
              </div>
              <div class="flex flex-col justify-end gap-3 sm:flex-row">
                <button class="inline-flex items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline md:border-0 md:bg-transparent md:px-0 md:py-0" type="button" @click="openTemplateDetail(template)">{{ copy.templates.viewDetails }}</button>
                <button class="inline-flex items-center justify-center rounded-xl border border-amber-100 bg-amber-50 px-3 py-2 text-sm font-bold text-amber-600 transition hover:underline md:border-0 md:bg-transparent md:px-0 md:py-0" type="button" @click="startTemplateEdit(template)">{{ copy.templates.edit }}</button>
              </div>
            </div>
          </div>
          <div v-else class="p-8 text-center text-slate-500 md:p-12">{{ copy.templates.empty }}</div>
          <div class="flex flex-col items-stretch justify-between gap-3 border-t border-slate-200 p-4 sm:flex-row sm:items-center md:p-5">
            <span class="text-center text-sm font-bold text-slate-500 sm:text-left">{{ copy.sent.pageText(templatePage, templateTotalPages) }}</span>
            <div class="flex flex-col gap-3 sm:flex-row">
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="templatePage <= 1 || templatesLoading" @click="templatePage--">{{ copy.sent.prev }}</button>
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!templateHasMore || templatesLoading" @click="templatePage++">{{ copy.sent.next }}</button>
            </div>
          </div>
        </section>
      </main>
    </div>

    <Teleport to="body">
      <div v-if="messageDetailOpen && selectedMessage" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <section class="flex h-full max-h-none w-full max-w-[1120px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black text-slate-950 md:truncate md:text-2xl">{{ copy.sent.detailTitle }}</h2>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              type="button"
              :aria-label="copy.sent.close"
              @click="closeMessageDetail"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-5">
            <div class="grid gap-4 md:grid-cols-2">
              <ReadonlyField
                v-for="(value, key) in selectedMessageFields"
                :key="key"
                :label="messageFieldLabel(String(key))"
                :value="value"
                :mono="Array.isArray(value) || (!!value && typeof value === 'object')"
                :max-height="Array.isArray(value) || (!!value && typeof value === 'object') ? '180px' : undefined"
              />
            </div>
          </div>
        </section>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="templateDetailOpen && selectedTemplate" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <section class="flex h-full max-h-none w-full max-w-[1120px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black text-slate-950 md:truncate md:text-2xl">{{ copy.templates.detailTitle }}</h2>
              <p class="mt-1 break-all text-sm font-semibold text-slate-500">{{ pathOf(selectedTemplate) }}</p>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              type="button"
              :aria-label="copy.sent.close"
              @click="closeTemplateDetail"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-5">
            <div class="grid gap-4 md:grid-cols-2">
              <ReadonlyField
                v-for="(value, key) in selectedTemplateFields"
                :key="key"
                :label="templateFieldLabel(String(key))"
                :value="templateFieldValue(String(key), value)"
                :mono="Array.isArray(value) || (!!value && typeof value === 'object')"
                :max-height="Array.isArray(value) || (!!value && typeof value === 'object') ? '180px' : undefined"
              />
            </div>
          </div>
        </section>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="templateEditOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <form class="flex h-full max-h-none w-full max-w-[920px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl" @submit.prevent="saveTemplate">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black text-slate-950 md:truncate md:text-2xl">{{ editingTemplatePath ? copy.templates.editTitle : copy.templates.createTitle }}</h2>
              <p v-if="editingTemplatePath" class="mt-1 break-all text-sm font-semibold text-slate-500">{{ formPath }}</p>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              type="button"
              :aria-label="copy.sent.close"
              @click="closeTemplateEdit"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="min-h-0 flex-1 space-y-4 overflow-y-auto p-4 md:p-5">
            <label class="block">
              <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.templates.path }}</span>
              <input v-model="formPath" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4 disabled:bg-slate-100" :disabled="!!editingTemplatePath" />
            </label>
            <label class="block">
              <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.templates.titleTemplate }}</span>
              <input ref="templateTitleInputRef" v-model="formTitle" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4" />
            </label>
            <label class="block">
              <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.templates.contentTemplate }}</span>
              <textarea v-model="formContent" class="mt-2 min-h-56 w-full rounded-xl border border-slate-200 p-4" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.templates.description }}</span>
              <textarea v-model="formDescription" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.templates.parameterSchema }}</span>
              <textarea v-model="formParameterSchema" class="mt-2 min-h-32 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" />
            </label>
          </div>

          <div class="flex flex-col items-stretch justify-end gap-3 border-t border-slate-200 px-4 py-4 sm:flex-row sm:items-center md:px-6 md:py-5">
            <button class="inline-flex h-11 min-w-[96px] items-center justify-center rounded-xl border px-5 font-bold" type="button" @click="closeTemplateEdit">{{ copy.templates.cancel }}</button>
            <button class="inline-flex h-11 min-w-[120px] items-center justify-center rounded-xl bg-blue-700 px-5 font-bold text-white shadow-sm disabled:opacity-50" :disabled="templateSaving" type="submit">
              {{ templateSaving ? copy.templates.saving : copy.templates.save }}
            </button>
          </div>
        </form>
      </div>
    </Teleport>
  </section>
</template>
