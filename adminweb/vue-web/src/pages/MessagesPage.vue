<script setup lang="ts">
import { FileText, List, Loader2, RefreshCw, Send, X } from "lucide-vue-next"
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type TabKey = "send" | "sent" | "templates"

const pageSize = 20

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
const editingTemplatePath = ref("")
const formPath = ref("")
const formTitle = ref("")
const formContent = ref("")
const formDescription = ref("")
const formVersion = ref(0)
const templateTitleInputRef = ref<HTMLInputElement | null>(null)
const { t } = useAdminLanguage()
const copy = computed(() => t.value.messagesAdmin)

const tabs = computed(() => [
  { key: "send" as const, label: copy.value.tabs.send, icon: Send, count: selectedUserIds.value.length },
  { key: "sent" as const, label: copy.value.tabs.sent, icon: List, count: total.value },
  { key: "templates" as const, label: copy.value.tabs.templates, icon: FileText, count: templates.value.length },
])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
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

function messageStatus(message: JsonRecord | null | undefined) {
  return pickFirst(message || {}, ["status", "raw_status"]) || "-"
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
    toast.error(copy.value.toasts.usersLoadFailed)
  } finally {
    usersLoading.value = false
  }
}

async function loadTemplates() {
  templatesLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/messages/templates")
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selectedTemplate.value || !templates.value.some((item) => pathOf(item) === pathOf(selectedTemplate.value))) {
      selectedTemplate.value = templates.value[0] || null
    }
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.templatesLoadFailed)
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
    toast.error(copy.value.toasts.templateDetailLoadFailed)
  }
}

async function loadSentMessages() {
  messagesLoading.value = true
  try {
    const params = new URLSearchParams({
      page: String(messagePage.value),
      page_size: String(pageSize),
    })
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/messages/sent?${params}`)
    const list = Array.isArray(data.messages) ? data.messages : []
    messages.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || messages.value.length)
    if (!selectedMessage.value || !messages.value.some((item) => messageId(item) === messageId(selectedMessage.value))) {
      openMessage(messages.value[0] || null, messageDetailOpen.value)
    }
    if (!messages.value.length) messageDetailOpen.value = false
  } catch (err) {
    console.error(err)
    messages.value = []
    selectedMessage.value = null
    messageDetailOpen.value = false
    toast.error(copy.value.toasts.sentLoadFailed)
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
    toast.error(copy.value.toasts.sendFailed)
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
  formVersion.value = Number(template.version || 0)
  formContent.value = ""

  try {
    const detail = await loadTemplateDetail(path)
    if (detail) {
      selectedTemplate.value = { ...template, ...detail }
      formTitle.value = String(detail.title_tpl || formTitle.value)
      formContent.value = String(detail.content_tpl || "")
      formDescription.value = String(detail.description || formDescription.value)
      formVersion.value = Number(detail.version || formVersion.value)
    }
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.templateDetailLoadFailed)
  }
}

function resetTemplateForm() {
  editingTemplatePath.value = ""
  formPath.value = ""
  formTitle.value = ""
  formContent.value = ""
  formDescription.value = ""
  formVersion.value = 0
}

async function saveTemplate() {
  const path = editingTemplatePath.value || formPath.value.trim()
  if (!path || !formTitle.value.trim() || !formContent.value.trim()) {
    toast.error(copy.value.toasts.templateFieldsRequired)
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
        current_version: formVersion.value,
      }),
    })
    toast.success(editingTemplatePath.value ? copy.value.toasts.templateUpdated : copy.value.toasts.templateCreated)
    templateEditOpen.value = false
    resetTemplateForm()
    await loadTemplates()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.templateSaveFailed)
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

watch([activeTab, messagePage, statusFilter], () => {
  if (activeTab.value === "sent") void loadSentMessages()
})

onMounted(async () => {
  await Promise.all([loadUsers(), loadTemplates()])
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadUsers">
          <RefreshCw class="h-4 w-4" :class="usersLoading ? 'animate-spin' : ''" />
          {{ copy.refreshUsers }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadTemplates">
          <RefreshCw class="h-4 w-4" :class="templatesLoading ? 'animate-spin' : ''" />
          {{ copy.refreshTemplates }}
        </button>
      </div>
    </header>

    <section class="rounded-3xl border border-slate-200 bg-white p-3 shadow-sm">
      <div class="flex flex-wrap gap-3">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="min-h-12 flex-1 rounded-2xl border px-4 py-3 text-left transition md:flex-none md:min-w-48"
          :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50 shadow-sm' : 'border-slate-100 hover:bg-slate-50'"
          type="button"
          @click="activeTab = tab.key"
        >
          <div class="flex items-center justify-between gap-3">
            <span class="inline-flex items-center gap-2 font-black">
              <component :is="tab.icon" class="h-4 w-4" />
              {{ tab.label }}
            </span>
            <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ tab.count }}</span>
          </div>
        </button>
      </div>
    </section>

    <div>

      <main class="min-w-0">
        <section v-if="activeTab === 'send'" class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 class="text-xl font-black">{{ copy.send.title }}</h2>
          <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_1fr]">
            <div>
              <div class="mb-2 flex items-center justify-between">
                <label class="font-bold">{{ copy.send.recipients }}</label>
                <div class="flex gap-3 text-sm">
                  <button class="font-bold text-[#0b7bdc]" type="button" @click="selectedUserIds = users.map(userId).filter(Boolean)">{{ copy.send.selectAll }}</button>
                  <button class="font-bold text-slate-500" type="button" @click="selectedUserIds = []">{{ copy.send.clear }}</button>
                </div>
              </div>
              <div class="max-h-[520px] overflow-y-auto rounded-2xl border border-slate-200 p-3">
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
                <select v-model="templatePath" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                  <option value="">{{ copy.send.selectTemplate }}</option>
                  <option v-for="template in templates" :key="pathOf(template)" :value="pathOf(template)">
                    {{ titleOf(template) }} ({{ pathOf(template) }})
                  </option>
                </select>
              </label>

              <label class="block">
                <span class="font-bold">{{ copy.send.messageType }}</span>
                <select v-model.number="msgType" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
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

              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="sending" type="button" @click="sendMessage">
                <Loader2 v-if="sending" class="h-4 w-4 animate-spin" />
                <Send v-else class="h-4 w-4" />
                {{ sending ? copy.send.sending : copy.send.sendButton }}
              </button>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'sent'" class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
            <div>
              <h2 class="text-xl font-black">{{ copy.sent.title }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ copy.sent.summary(pageSize, total) }}</p>
            </div>
            <select v-model="statusFilter" class="h-10 rounded-xl border border-slate-200 px-4 text-sm">
              <option value="">{{ copy.sent.allStatus }}</option>
              <option value="1">{{ copy.sent.unread }}</option>
              <option value="2">{{ copy.sent.read }}</option>
            </select>
          </div>
          <div class="grid grid-cols-[minmax(0,1fr)_220px_150px_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
            <span>{{ copy.sent.columns.message }}</span>
            <span>{{ copy.sent.columns.user }}</span>
            <span class="text-center">{{ copy.sent.columns.status }}</span>
            <span class="text-right">{{ copy.sent.columns.time }}</span>
            <span class="text-right">{{ copy.sent.columns.action }}</span>
          </div>
          <div v-if="messagesLoading" class="p-12 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.sent.loading }}
          </div>
          <div v-else-if="messages.length" class="divide-y divide-slate-100">
            <div
              v-for="message in messages"
              :key="messageId(message)"
              class="grid cursor-pointer grid-cols-[minmax(0,1fr)_220px_150px_180px_112px] items-center gap-5 px-5 py-4 transition hover:bg-sky-50"
              :class="messageId(selectedMessage) === messageId(message) ? 'bg-sky-50' : ''"
              role="button"
              tabindex="0"
              @click="openMessage(message)"
              @keydown.enter.prevent="openMessage(message)"
              @keydown.space.prevent="openMessage(message)"
            >
              <div class="min-w-0">
                <div class="truncate font-black text-slate-950">{{ pickFirst(message, ["title", "subject", "template_path", "message_id"]) || copy.defaults.message }}</div>
                <div class="mt-1 break-all text-xs font-semibold text-slate-500">{{ copy.sent.idPrefix }}{{ messageId(message) || "-" }}</div>
              </div>
              <div class="min-w-0 break-all text-sm font-semibold text-slate-500">{{ pickFirst(message, ["user_name", "user_id", "candidate_ulid"]) || "-" }}</div>
              <div class="text-center">
                <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(message))">{{ messageStatus(message) }}</span>
              </div>
              <div class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(pickFirst(message, ["created_at", "sent_at", "updated_at"]) || "")) }}</div>
              <div class="text-right">
                <button
                  class="inline-flex h-9 items-center justify-center rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-blue-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50"
                  type="button"
                  @click.stop="openMessage(message)"
                >
                  {{ copy.sent.viewDetails }}
                </button>
              </div>
            </div>
          </div>
          <div v-else class="p-12 text-center text-slate-500">{{ copy.sent.empty }}</div>
          <div class="flex items-center justify-between gap-3 border-t border-slate-200 p-5">
            <span class="text-sm font-bold text-slate-500">{{ copy.sent.pageText(messagePage, totalPages) }}</span>
            <div class="flex gap-3">
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="messagePage <= 1" @click="messagePage--">{{ copy.sent.prev }}</button>
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="messagePage >= totalPages" @click="messagePage++">{{ copy.sent.next }}</button>
            </div>
          </div>
        </section>

        <section v-else class="space-y-6">
          <div class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
            <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">{{ copy.templates.listTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ copy.templates.listDescription }}</p>
              </div>
              <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ templates.length }}</span>
            </div>
            <div v-if="templatesLoading" class="p-12 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              {{ copy.templates.loading }}
            </div>
            <div v-else-if="templates.length" class="overflow-x-auto">
              <div class="min-w-[980px]">
                <div class="grid grid-cols-[minmax(320px,1.35fr)_minmax(240px,1fr)_110px_180px_180px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
                  <span>{{ copy.templates.columns.template }}</span>
                  <span>{{ copy.templates.columns.description }}</span>
                  <span class="text-center">{{ copy.templates.columns.version }}</span>
                  <span>{{ copy.templates.columns.updatedAt }}</span>
                  <span class="text-right">{{ copy.templates.columns.action }}</span>
                </div>
                <div class="divide-y divide-slate-100">
                  <div
                    v-for="template in templates"
                    :key="pathOf(template)"
                    class="grid grid-cols-[minmax(320px,1.35fr)_minmax(240px,1fr)_110px_180px_180px] items-center gap-5 px-5 py-4 transition hover:bg-sky-50"
                    :class="pathOf(selectedTemplate) === pathOf(template) ? 'bg-sky-50' : ''"
                  >
                    <button class="min-w-0 text-left" type="button" @click="selectTemplate(template)">
                      <div class="truncate font-black text-slate-950">{{ titleOf(template) }}</div>
                      <div class="mt-1 break-all text-xs font-semibold text-slate-500">{{ pathOf(template) }}</div>
                    </button>
                    <div class="min-w-0 truncate text-sm font-semibold text-slate-500">{{ String(template.description || "-") }}</div>
                    <div class="text-center text-sm font-black text-slate-700">{{ String(template.version || "-") }}</div>
                    <div class="text-sm font-semibold text-slate-500">{{ formatDate(String(pickFirst(template, ["updated_at", "created_at"]) || "")) }}</div>
                    <div class="flex justify-end gap-2">
                      <button
                        class="inline-flex h-9 items-center justify-center rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-blue-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50"
                        type="button"
                        @click="openTemplateDetail(template)"
                      >
                        {{ copy.templates.viewDetails }}
                      </button>
                      <button
                        class="rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-black text-slate-700 shadow-sm transition hover:border-slate-300 hover:bg-slate-50"
                        type="button"
                        @click="startTemplateEdit(template)"
                      >
                        {{ copy.templates.edit }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="p-12 text-center text-slate-500">{{ copy.templates.empty }}</div>
          </div>

          <form class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm" @submit.prevent="saveTemplate">
            <h2 class="text-xl font-black">{{ copy.templates.createTitle }}</h2>
            <label class="mt-4 block">
              <span class="text-sm font-bold">{{ copy.templates.path }}</span>
              <input v-model="formPath" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100" :disabled="!!editingTemplatePath" />
            </label>
            <label class="mt-4 block">
              <span class="text-sm font-bold">{{ copy.templates.titleTemplate }}</span>
              <input v-model="formTitle" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
            </label>
            <label class="mt-4 block">
              <span class="text-sm font-bold">{{ copy.templates.contentTemplate }}</span>
              <textarea v-model="formContent" class="mt-2 min-h-40 w-full rounded-xl border border-slate-200 p-4" />
            </label>
            <label class="mt-4 block">
              <span class="text-sm font-bold">{{ copy.templates.description }}</span>
              <textarea v-model="formDescription" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" />
            </label>
            <div class="mt-5 flex gap-3">
              <button class="rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="templateSaving" type="submit">
                {{ templateSaving ? copy.templates.saving : copy.templates.save }}
              </button>
              <button class="rounded-xl border px-5 py-3 font-bold" type="button" @click="resetTemplateForm">{{ copy.templates.clear }}</button>
            </div>
          </form>
        </section>
      </main>
    </div>

    <Teleport to="body">
      <div v-if="messageDetailOpen && selectedMessage" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1120px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="truncate text-2xl font-black text-slate-950">{{ copy.sent.detailTitle }}</h2>
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

          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
            <div class="grid gap-4 md:grid-cols-2">
              <label v-for="(value, key) in selectedMessageFields" :key="key" class="grid gap-2 text-sm font-bold">
                {{ messageFieldLabel(String(key)) }}
                <textarea
                  v-if="Array.isArray(value) || (value && typeof value === 'object')"
                  class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                  disabled
                  :value="JSON.stringify(value, null, 2)"
                />
                <input v-else class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600" disabled :value="String(value ?? '-')" />
              </label>
            </div>
            <pre class="max-h-[420px] overflow-auto rounded-2xl bg-slate-950 p-4 text-xs text-slate-100">{{ JSON.stringify(selectedMessage, null, 2) }}</pre>
          </div>
        </section>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="templateDetailOpen && selectedTemplate" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1120px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="truncate text-2xl font-black text-slate-950">{{ copy.templates.detailTitle }}</h2>
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

          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
            <div class="grid gap-4 md:grid-cols-2">
              <label v-for="(value, key) in selectedTemplateFields" :key="key" class="grid gap-2 text-sm font-bold">
                {{ templateFieldLabel(String(key)) }}
                <textarea
                  v-if="Array.isArray(value) || (value && typeof value === 'object')"
                  class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                  disabled
                  :value="JSON.stringify(value, null, 2)"
                />
                <input v-else class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600" disabled :value="String(value ?? '-')" />
              </label>
            </div>
          </div>
        </section>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="templateEditOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <form class="flex max-h-[88vh] w-full max-w-[920px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl" @submit.prevent="saveTemplate">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="min-w-0">
              <h2 class="truncate text-2xl font-black text-slate-950">{{ copy.templates.editTitle }}</h2>
              <p class="mt-1 break-all text-sm font-semibold text-slate-500">{{ formPath || "-" }}</p>
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

          <div class="min-h-0 flex-1 space-y-4 overflow-y-auto p-5">
            <label class="block">
              <span class="text-sm font-bold">{{ copy.templates.path }}</span>
              <input v-model="formPath" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100" disabled />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.templates.titleTemplate }}</span>
              <input ref="templateTitleInputRef" v-model="formTitle" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.templates.contentTemplate }}</span>
              <textarea v-model="formContent" class="mt-2 min-h-56 w-full rounded-xl border border-slate-200 p-4" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">{{ copy.templates.description }}</span>
              <textarea v-model="formDescription" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" />
            </label>
          </div>

          <div class="flex justify-end gap-3 border-t border-slate-200 px-6 py-5">
            <button class="rounded-xl border px-5 py-3 font-bold" type="button" @click="closeTemplateEdit">{{ copy.templates.cancel }}</button>
            <button class="rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="templateSaving" type="submit">
              {{ templateSaving ? copy.templates.saving : copy.templates.save }}
            </button>
          </div>
        </form>
      </div>
    </Teleport>
  </section>
</template>
