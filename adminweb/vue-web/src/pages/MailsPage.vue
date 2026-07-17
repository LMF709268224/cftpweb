<script setup lang="ts">
import { FileText, List, Loader2, Mail, Plus, RefreshCw, Send, Users, X, XCircle } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import ReadonlyField from "@/components/ReadonlyField.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type TabKey = "send" | "sent" | "templates"
type TemplateMode = "detail" | "create" | "edit"

const pageSize = 20
const templatePageSize = 10
const activeTab = ref<TabKey>("send")
const templateMode = ref<TemplateMode>("detail")
const users = ref<JsonRecord[]>([])
const templates = ref<JsonRecord[]>([])
const selectedTemplate = ref<JsonRecord | null>(null)
const templateDialogOpen = ref(false)
const selectedUserIds = ref<string[]>([])
const templatePath = ref("")
const subject = ref("")
const payload = ref("{\n}")
const sending = ref(false)

const mails = ref<JsonRecord[]>([])
const mailsLoading = ref(false)
const mailPage = ref(1)
const statusFilter = ref("")
const total = ref(0)
const selectedMail = ref<JsonRecord | null>(null)
const mailDetail = ref<JsonRecord | null>(null)
const mailStatusDetail = ref<JsonRecord | null>(null)
const mailDetailLoading = ref(false)
const mailDetailOpen = ref(false)
const canceling = ref(false)
const stats = ref<JsonRecord | null>(null)

const usersLoading = ref(false)
const templatesLoading = ref(false)
const templateSaving = ref(false)
const templatePage = ref(1)
const totalTemplates = ref(0)
const editingTemplatePath = ref("")

const mailCursorStack = ref<string[]>([""])
const mailNextCursor = ref("")
const mailHasMore = ref(false)

const templateCursorStack = ref<string[]>([""])
const templateNextCursor = ref("")
const templateHasMore = ref(false)
const formPath = ref("")
const formName = ref("")
const formSubject = ref("")
const formContent = ref("")
const formDescription = ref("")
const formParameterSchema = ref("{}")
const { t } = useAdminLanguage()
const copy = computed(() => t.value.mailsAdmin)

const tabs = computed(() => [
  { key: "send" as const, label: copy.value.tabs.send, icon: Send, count: selectedUserIds.value.length },
  { key: "sent" as const, label: copy.value.tabs.sent, icon: List, count: total.value },
  { key: "templates" as const, label: copy.value.tabs.templates, icon: FileText, count: totalTemplates.value || templates.value.length },
])
const statusOptions = computed(() => [
  { value: "", label: copy.value.statusOptions.all },
  { value: "SCHEDULING", label: copy.value.statusOptions.scheduling },
  { value: "SENT", label: copy.value.statusOptions.sent },
  { value: "FAILED", label: copy.value.statusOptions.failed },
  { value: "CANCELLED", label: copy.value.statusOptions.cancelled },
])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const templateTotalPages = computed(() => Math.max(1, Math.ceil((totalTemplates.value || templates.value.length) / templatePageSize)))
const selectedUsers = computed(() => users.value.filter((user) => selectedUserIds.value.includes(userId(user))))
const selectedMailHtml = computed(() => String(mailDetail.value?.html_body || mailDetail.value?.plain_body || selectedMail.value?.html_body || selectedMail.value?.plain_body || ""))
const selectedMailRecord = computed(() => mailDetail.value || selectedMail.value || {})
const selectedMailCanCancel = computed(() => isSchedulingMail(mailStatusDetail.value) || isSchedulingMail(mailDetail.value) || isSchedulingMail(selectedMail.value))
const selectedTemplateFields = computed(() => selectedTemplate.value || {})
const statsCards = computed(() => {
  const counts = stats.value?.status_counts
  if (!counts || typeof counts !== "object" || Array.isArray(counts)) return []

  const entries = Object.entries(counts)
    .map(([key, value]) => ({
      key,
      label: mailStatusLabel(key),
      value: Number(value || 0),
    }))
    .filter((item) => Number.isFinite(item.value))
    .sort((a, b) => a.label.localeCompare(b.label, "zh-CN"))

  const totalCount = entries.reduce((sum, item) => sum + item.value, 0)
  return [{ key: "TOTAL", label: copy.value.totalEmails, value: totalCount }, ...entries]
})

function userId(user: JsonRecord) {
  return String(pickFirst(user, ["id", "user_id", "candidate_ulid", "ulid"]) || "")
}

function userEmail(user: JsonRecord) {
  return String(user.email || "")
}

function userName(user: JsonRecord) {
  return String(pickFirst(user, ["name", "nickname", "email", "id"]) || userEmail(user) || copy.value.defaults.user)
}

function resolvedTemplate(template: JsonRecord | null | undefined) {
  if (!template || typeof template !== "object" || Array.isArray(template)) return {}
  const nested = template.template
  if (nested && typeof nested === "object" && !Array.isArray(nested)) {
    return { ...template, ...(nested as JsonRecord) }
  }
  return template
}

function pathOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(resolvedTemplate(template), ["path", "template_path", "template_id"]) || "")
}

function nameOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(resolvedTemplate(template), ["name", "template_name", "templateName", "subject_template", "subjectTemplate", "path"]) || pathOf(template) || copy.value.defaults.template)
}

function subjectOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(resolvedTemplate(template), ["subject_template", "subjectTemplate"]) || "")
}

function bodyOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(resolvedTemplate(template), ["html_body", "htmlBody", "template_body", "templateBody", "plain_body", "plainBody"]) || "")
}

function descriptionOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(resolvedTemplate(template), ["description"]) || "")
}

function templateFieldValue(key: string, value: unknown) {
  if (key.endsWith("_at")) return formatDate(value) || String(value ?? "-")
  return value
}

function templateFieldLabel(key: string) {
  return copy.value.templateFieldLabels[key as keyof typeof copy.value.templateFieldLabels] || key
}

function mailId(mail: JsonRecord | null | undefined) {
  return String(pickFirst(mail || {}, ["mail_id", "mail_ulid", "id"]) || "")
}

function mailStatus(mail: JsonRecord | null | undefined) {
  return pickFirst(mail || {}, ["status", "raw_status"]) || "-"
}

function isSchedulingMail(mail: JsonRecord | null | undefined) {
  return String(mailStatus(mail)).toUpperCase() === "SCHEDULING"
}

function mailStatusLabel(status: unknown) {
  const value = String(status || "").toUpperCase()
  if (value === "SENT") return copy.value.statusLabels.sent
  if (value === "FAILED") return copy.value.statusLabels.failed
  if (value === "CANCELLED") return copy.value.statusLabels.cancelled
  if (value === "SCHEDULING") return copy.value.statusLabels.scheduling
  if (value === "PENDING") return copy.value.statusLabels.pending
  return value || "-"
}

function stripHtml(value: string) {
  return value.replace(/<[^>]+>/g, "")
}

function extractPayloadTemplate(...texts: unknown[]) {
  const vars = new Set<string>()
  const regex = /\{\{([^}]+)\}\}/g
  for (const text of texts) {
    if (typeof text !== "string") continue
    let match: RegExpExecArray | null
    while ((match = regex.exec(text))) vars.add(match[1].trim().replace(/^\./, ""))
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

function validateTemplatePayload() {
  if (!templatePath.value || !payload.value.trim()) return true
  try {
    const parsed = JSON.parse(payload.value)
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      toast.error(copy.value.toasts.payloadObjectRequired)
      return false
    }
    return true
  } catch (err) {
    toast.error(copy.value.toasts.payloadJsonInvalid(err instanceof Error ? err.message : String(err)))
    return false
  }
}

async function loadUsers() {
  usersLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/user/list")
    const list = Array.isArray(data.users) ? data.users : []
    users.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item) && !!item.email)
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
    const data = await apiClient<JsonRecord>(`/api/mails/templates?${params}`)
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    totalTemplates.value = Number(data.total || templates.value.length)
    templateHasMore.value = Boolean(data.has_more)
    templateNextCursor.value = String(data.next_cursor || "")
    templateCursorStack.value = templateCursorStack.value.slice(0, templatePage.value)
    templateCursorStack.value[templatePage.value] = templateNextCursor.value
    if (!selectedTemplate.value || !templates.value.some((item) => pathOf(item) === pathOf(selectedTemplate.value))) {
      selectedTemplate.value = templates.value[0] || null
      templateMode.value = selectedTemplate.value ? "detail" : "create"
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
  return apiClient<JsonRecord>(`/api/mails/templates/detail?path=${encodeURIComponent(path)}`)
}

async function selectTemplate(template: JsonRecord) {
  selectedTemplate.value = template
  templateMode.value = "detail"
  try {
    const detail = await loadTemplateDetail(pathOf(template))
    if (detail) selectedTemplate.value = { ...template, ...detail }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
  }
}

async function openTemplateDetail(template: JsonRecord) {
  await selectTemplate(template)
  templateDialogOpen.value = !!selectedTemplate.value
}

async function openTemplateEdit(template: JsonRecord | null = selectedTemplate.value) {
  await editTemplate(template)
  templateDialogOpen.value = !!template
}

function closeTemplateDialog() {
  templateDialogOpen.value = false
  if (templateMode.value !== "detail") {
    templateMode.value = "detail"
    resetTemplateForm()
  }
}

async function loadStats() {
  try {
    stats.value = await apiClient<JsonRecord>("/api/mails/stats")
  } catch {
    stats.value = null
  }
}

async function loadSentMails() {
  mailsLoading.value = true
  try {
    const params = new URLSearchParams({ page_size: String(pageSize) })
    const cursor = mailCursorStack.value[mailPage.value - 1] || ""
    if (cursor) params.set("cursor", cursor)
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/mails/sent?${params}`)
    const list = Array.isArray(data.mails) ? data.mails : []
    mails.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || mails.value.length)
    mailHasMore.value = Boolean(data.has_more)
    mailNextCursor.value = String(data.next_cursor || "")
    mailCursorStack.value = mailCursorStack.value.slice(0, mailPage.value)
    mailCursorStack.value[mailPage.value] = mailNextCursor.value
    if (!selectedMail.value || !mails.value.some((item) => mailId(item) === mailId(selectedMail.value))) {
      await openMail(mails.value[0] || null, mailDetailOpen.value)
    }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.sentLoadFailed))
  } finally {
    mailsLoading.value = false
  }
}

async function sendMail() {
  if (!selectedUsers.value.length) {
    toast.error(copy.value.toasts.recipientsRequired)
    return
  }
  if (!templatePath.value && !subject.value.trim()) {
    toast.error(copy.value.toasts.subjectRequired)
    return
  }
  if (!templatePath.value && !payload.value.trim()) {
    toast.error(copy.value.toasts.bodyRequired)
    return
  }
  if (!validateTemplatePayload()) return

  sending.value = true
  let count = 0
  try {
    for (const user of selectedUsers.value) {
      await apiClient("/api/mails/send", {
        method: "POST",
        body: JSON.stringify({
          to_email: userEmail(user),
          to_name: userName(user),
          subject: subject.value,
          template_path: templatePath.value,
          payload: payload.value,
          html_body: templatePath.value ? "" : payload.value,
          plain_body: templatePath.value ? "" : stripHtml(payload.value),
        }),
      })
      count += 1
    }
    toast.success(copy.value.toasts.sendSuccess(count))
    selectedUserIds.value = []
    templatePath.value = ""
    subject.value = ""
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
    startCreateTemplate()
    return
  }
  templateMode.value = "edit"
  const path = pathOf(template)
  editingTemplatePath.value = path
  formPath.value = path
  formName.value = nameOf(template)
  formSubject.value = subjectOf(template)
  formContent.value = bodyOf(template)
  formDescription.value = descriptionOf(template)
  formParameterSchema.value = String(template.parameter_schema || "{}")

  try {
    const detail = await loadTemplateDetail(path)
    if (detail) {
      const merged = resolvedTemplate({ ...template, ...detail })
      selectedTemplate.value = merged
      formName.value = nameOf(merged)
      formSubject.value = subjectOf(merged)
      formContent.value = bodyOf(merged)
      formDescription.value = descriptionOf(merged)
      formParameterSchema.value = String(merged.parameter_schema || "{}")
    }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
  }
}

function resetTemplateForm() {
  editingTemplatePath.value = ""
  formPath.value = ""
  formName.value = ""
  formSubject.value = ""
  formContent.value = ""
  formDescription.value = ""
  formParameterSchema.value = "{}"
}

function startCreateTemplate() {
  selectedTemplate.value = null
  templateMode.value = "create"
  resetTemplateForm()
  templateDialogOpen.value = true
}

async function saveTemplate() {
  const path = editingTemplatePath.value || formPath.value.trim()
  if (!path || !formName.value.trim() || !formSubject.value.trim() || !formContent.value.trim()) {
    toast.error(copy.value.toasts.templateRequired)
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
    await apiClient("/api/mails/templates", {
      method: editingTemplatePath.value ? "PUT" : "POST",
      body: JSON.stringify({
        path,
        name: formName.value,
        subject_template: formSubject.value,
        html_body: formContent.value,
        plain_body: stripHtml(formContent.value),
        description: formDescription.value,
        parameter_schema: parameterSchema,
      }),
    })
    toast.success(editingTemplatePath.value ? copy.value.toasts.templateUpdated : copy.value.toasts.templateCreated)
    const savedPath = path
    resetTemplateForm()
    templateMode.value = "detail"
    templateDialogOpen.value = false
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

async function openMail(mail: JsonRecord | null, open = true) {
  selectedMail.value = mail
  mailDetailOpen.value = open && !!mail
  mailDetail.value = null
  mailStatusDetail.value = null
  const id = mailId(mail)
  if (!id) return

  mailDetailLoading.value = true
  try {
    mailDetail.value = await apiClient<JsonRecord>(`/api/mails?mail_id=${encodeURIComponent(id)}`)
    mailStatusDetail.value = await apiClient<JsonRecord>(`/api/mails/status?mail_id=${encodeURIComponent(id)}`)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.mailDetailLoadFailed))
  } finally {
    mailDetailLoading.value = false
  }
}

function closeMailDetail() {
  mailDetailOpen.value = false
}

async function cancelMail() {
  const id = mailId(selectedMail.value)
  if (!id) return
  if (!selectedMailCanCancel.value) {
    toast.error(copy.value.toasts.cancelNotAllowed)
    return
  }
  canceling.value = true
  try {
    await apiClient("/api/mails/cancel", { method: "POST", body: JSON.stringify({ mail_id: id }) })
    toast.success(copy.value.toasts.mailCancelled)
    await loadSentMails()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.cancelFailed))
  } finally {
    canceling.value = false
  }
}

watch(templatePath, async (path) => {
  if (!path) {
    payload.value = "{\n}"
    return
  }
  const template = templates.value.find((item) => pathOf(item) === path)
  payload.value = extractPayloadTemplate(subjectOf(template), bodyOf(template || {}))
  try {
    const detail = await loadTemplateDetail(path)
    payload.value = extractPayloadTemplate(subjectOf(detail), bodyOf(detail))
  } catch {
    // Keep the best-effort payload generated from the list response.
  }
})

watch(statusFilter, () => {
  mailPage.value = 1
  mailCursorStack.value = [""]
  mailNextCursor.value = ""
  mailHasMore.value = false
})

watch([activeTab, mailPage, statusFilter], () => {
  if (activeTab.value === "sent") void loadSentMails()
})

watch(templatePage, () => {
  if (activeTab.value === "templates") void loadTemplates()
})

onMounted(async () => {
  await Promise.all([loadUsers(), loadTemplates(), loadStats()])
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

    <!-- Mail stats -->
    <section v-if="statsCards.length" class="ui-card mt-6 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.statsTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.statsDescription }}</p>
        </div>
        <span class="ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.statsBadge }}</span>
      </div>
      <div class="grid gap-3 p-4 md:grid-cols-4 md:gap-4 md:p-5">
        <div v-for="card in statsCards" :key="card.key" class="rounded-2xl bg-slate-50 p-4 md:p-5">
          <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ card.key }}</div>
          <div class="mt-1 text-sm font-medium text-slate-500">{{ card.label }}</div>
          <div class="tnum mt-3 text-3xl font-bold text-ink">{{ card.value }}</div>
        </div>
      </div>
    </section>

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

    <!-- Send mail -->
    <section v-if="activeTab === 'send'" class="ui-card mt-4 overflow-hidden">
      <div class="border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <h2 class="text-base font-semibold text-ink">{{ copy.sendTitle }}</h2>
      </div>
      <div class="grid gap-6 p-4 md:p-6 lg:grid-cols-[1fr_1fr]">
        <div>
          <div class="mb-1.5 flex items-center justify-between gap-3">
            <label class="ui-label">{{ copy.recipients }}</label>
            <div class="flex gap-4 text-sm">
              <button class="font-medium text-primary-700 transition-colors hover:text-primary-600" type="button" @click="selectedUserIds = users.map(userId).filter(Boolean)">{{ copy.selectAll }}</button>
              <button class="font-medium text-slate-500 transition-colors hover:text-slate-700" type="button" @click="selectedUserIds = []">{{ copy.clear }}</button>
            </div>
          </div>
          <div class="max-h-[360px] overflow-y-auto rounded-[10px] border border-slate-200 p-2 md:max-h-[520px]">
            <label v-for="user in users" :key="userId(user)" class="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 transition-colors hover:bg-slate-50">
              <input v-model="selectedUserIds" class="h-4 w-4 accent-primary-600" type="checkbox" :value="userId(user)" />
              <span class="text-sm font-medium text-ink">{{ userName(user) }}</span>
              <span class="break-all text-xs text-slate-400">{{ userEmail(user) }}</span>
            </label>
            <div v-if="!users.length" class="ui-empty">
              <div class="ui-empty-icon">
                <Users class="h-6 w-6" />
              </div>
              <p class="ui-empty-title">{{ copy.noUsers }}</p>
            </div>
          </div>
          <p class="tnum mt-2 text-sm text-slate-500">{{ copy.selectedUsers(selectedUserIds.length) }}</p>
        </div>

        <div class="space-y-4">
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.template }}</span>
            <select v-model="templatePath" class="ui-input">
              <option value="">{{ copy.noTemplateOption }}</option>
              <option v-for="template in templates" :key="pathOf(template)" :value="pathOf(template)">
                {{ nameOf(template) }} ({{ pathOf(template) }})
              </option>
            </select>
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.subject }}</span>
            <input v-model="subject" class="ui-input" :placeholder="copy.subjectPlaceholder" />
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ templatePath ? copy.payloadLabel : copy.bodyLabel }}</span>
            <textarea v-model="payload" class="min-h-52 w-full rounded-[10px] border border-slate-200 bg-white p-3 font-mono text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
          </label>
          <button class="ui-btn ui-btn-primary w-full sm:w-auto" :disabled="sending" type="button" @click="sendMail">
            <Loader2 v-if="sending" class="h-4 w-4 animate-spin" />
            <Mail v-else class="h-4 w-4" />
            {{ sending ? copy.sending : copy.sendMail }}
          </button>
        </div>
      </div>
    </section>

    <!-- Sent mails -->
    <section v-else-if="activeTab === 'sent'" class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.sentTitle }}</h2>
        </div>
        <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
          <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(total) }}</span>
          <select v-model="statusFilter" class="ui-input min-w-0 flex-1 sm:w-44 sm:flex-none">
            <option v-for="option in statusOptions" :key="option.value || 'all'" :value="option.value">{{ option.label }}</option>
          </select>
        </div>
      </div>
      <div class="hidden grid-cols-[minmax(0,1fr)_220px_150px_180px_112px] gap-5 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid">
        <span>{{ copy.columns.mail }}</span>
        <span>{{ copy.columns.recipient }}</span>
        <span class="text-center">{{ copy.columns.status }}</span>
        <span class="text-right">{{ copy.columns.time }}</span>
        <span class="text-right">{{ copy.columns.action }}</span>
      </div>
      <div v-if="mailsLoading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`mail-sk-${i}`" class="px-4 py-3.5 md:grid md:grid-cols-[minmax(0,1fr)_220px_150px_180px_112px] md:items-center md:gap-5">
          <div>
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton mt-2 h-3 w-3/5" />
          </div>
          <div class="skeleton mt-3 h-4 w-4/5 md:mt-0" />
          <div class="skeleton mt-3 h-5 w-16 rounded-full md:mt-0 md:justify-self-center" />
          <div class="skeleton mt-3 h-4 w-24 md:mt-0 md:justify-self-end" />
          <div class="skeleton mt-3 h-4 w-16 md:mt-0 md:justify-self-end" />
        </div>
      </div>
      <div v-else-if="mails.length" class="divide-y divide-hairline">
        <div
          v-for="mail in mails"
          :key="mailId(mail)"
          class="ui-table-row flex cursor-pointer flex-col px-4 py-4 md:grid md:grid-cols-[minmax(0,1fr)_220px_150px_180px_112px] md:items-center md:gap-5 md:py-3.5"
          :class="mailId(selectedMail) === mailId(mail) ? 'bg-primary-50/60' : ''"
          role="button"
          tabindex="0"
          @click="openMail(mail)"
          @keydown.enter.prevent="openMail(mail)"
          @keydown.space.prevent="openMail(mail)"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-ink md:truncate">{{ pickFirst(mail, ["subject", "template_path", "mail_id"]) || copy.defaults.mail }}</div>
            <div class="tnum mt-0.5 break-all text-xs text-slate-400">{{ copy.mailIdPrefix }}{{ mailId(mail) || "-" }}</div>
          </div>
          <div class="mt-3 flex min-w-0 items-start justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0">
            <span class="shrink-0 text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.recipient }}</span>
            <span class="break-all text-right text-sm text-slate-600 md:text-left">{{ pickFirst(mail, ["to_email", "recipient_email", "user_email"]) || "-" }}</span>
          </div>
          <div class="mt-2.5 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-center">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.status }}</span>
            <span class="ui-badge" :class="badgeClass(mailStatus(mail))">{{ mailStatusLabel(mailStatus(mail)) }}</span>
          </div>
          <div class="mt-2.5 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-right">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.columns.time }}</span>
            <span class="tnum text-sm text-slate-600">{{ formatDate(String(pickFirst(mail, ["created_at", "sent_at", "updated_at"]) || "")) }}</span>
          </div>
          <div class="mt-3 flex justify-end md:mt-0 md:block md:text-right">
            <button
              class="text-sm font-medium text-primary-700 transition-colors hover:text-primary-600"
              type="button"
              @click.stop="openMail(mail)"
            >
              {{ copy.viewDetails }}
            </button>
          </div>
        </div>
      </div>
      <div v-else class="ui-empty">
        <div class="ui-empty-icon">
          <Mail class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.emptySent }}</p>
      </div>
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(mailPage, totalPages) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="mailPage <= 1" @click="mailPage--">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!mailHasMore" @click="mailPage++">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <!-- Templates -->
    <section v-else class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.templateListTitle }}</h2>
        </div>
        <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
          <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ copy.totalText(totalTemplates || templates.length) }}</span>
          <button class="ui-btn ui-btn-primary flex-1 sm:flex-none" type="button" @click="startCreateTemplate">
            <Plus class="h-4 w-4" />
            {{ copy.createTemplate }}
          </button>
        </div>
      </div>
      <div class="hidden grid-cols-[minmax(0,1fr)_140px_180px_160px] gap-5 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 md:grid">
        <span>{{ copy.template }}</span>
        <span class="text-center">{{ copy.templateColumns.version }}</span>
        <span class="text-right">{{ copy.templateColumns.updatedAt }}</span>
        <span class="text-right">{{ copy.templateColumns.action }}</span>
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
          class="ui-table-row flex flex-col px-4 py-4 md:grid md:grid-cols-[minmax(0,1fr)_140px_180px_160px] md:items-center md:gap-5 md:py-3.5"
          :class="pathOf(selectedTemplate) === pathOf(template) ? 'bg-primary-50/60' : ''"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-ink md:truncate">{{ nameOf(template) }}</div>
            <div class="tnum mt-0.5 break-all text-xs text-slate-400">{{ pathOf(template) }}</div>
            <div v-if="template.description" class="mt-1 break-words text-xs text-slate-400 md:truncate">{{ template.description }}</div>
          </div>
          <div class="mt-3 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-center">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.templateColumns.version }}</span>
            <span class="tnum ui-badge bg-slate-100 text-slate-600">v{{ String(template.version || "-") }}</span>
          </div>
          <div class="mt-2.5 flex items-center justify-between gap-4 border-t border-hairline pt-2.5 md:mt-0 md:block md:border-0 md:pt-0 md:text-right">
            <span class="text-xs font-medium text-slate-400 md:hidden">{{ copy.templateColumns.updatedAt }}</span>
            <span class="tnum text-sm text-slate-600">{{ formatDate(String(pickFirst(template, ["updated_at", "created_at"]) || "")) }}</span>
          </div>
          <div class="mt-3 flex items-center justify-end gap-4 md:mt-0">
            <button class="text-sm font-medium text-primary-700 transition-colors hover:text-primary-600" type="button" @click="openTemplateDetail(template)">{{ copy.viewDetails }}</button>
            <button class="text-sm font-medium text-primary-700 transition-colors hover:text-primary-600" type="button" @click="openTemplateEdit(template)">{{ copy.editTemplate }}</button>
          </div>
        </div>
      </div>
      <div v-else class="ui-empty">
        <div class="ui-empty-icon">
          <FileText class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.emptyTemplates }}</p>
      </div>
      <div class="flex flex-col items-stretch justify-between gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
        <span class="tnum text-center text-sm text-slate-500 sm:text-left">{{ copy.pageText(templatePage, templateTotalPages) }}</span>
        <div class="flex flex-col gap-2 sm:flex-row">
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="templatePage <= 1 || templatesLoading" @click="templatePage--">{{ copy.prev }}</button>
          <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" :disabled="!templateHasMore || templatesLoading" @click="templatePage++">{{ copy.next }}</button>
        </div>
      </div>
    </section>

    <!-- Template detail / edit dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="templateDialogOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1120px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:text-lg">
                  {{ templateMode === "detail" ? copy.templateDetails : editingTemplatePath ? copy.editTemplate : copy.createTemplate }}
                </h2>
                <p v-if="templateMode === 'detail' && selectedTemplate" class="tnum mt-1 break-all text-sm text-slate-500">{{ pathOf(selectedTemplate) }}</p>
                <p v-else-if="editingTemplatePath" class="tnum mt-1 break-all text-sm text-slate-500">{{ formPath }}</p>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="copy.close"
                @click="closeTemplateDialog"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <template v-if="templateMode === 'detail'">
                <div v-if="!selectedTemplate" class="ui-empty">
                  <div class="ui-empty-icon">
                    <FileText class="h-6 w-6" />
                  </div>
                  <p class="ui-empty-title">{{ copy.selectTemplate }}</p>
                </div>
                <div v-else class="grid gap-4 md:grid-cols-2">
                  <ReadonlyField
                    v-for="(value, key) in selectedTemplateFields"
                    :key="key"
                    :label="templateFieldLabel(String(key))"
                    :value="templateFieldValue(String(key), value)"
                    :mono="Array.isArray(value) || (!!value && typeof value === 'object')"
                    :max-height="Array.isArray(value) || (!!value && typeof value === 'object') ? '180px' : undefined"
                  />
                </div>
              </template>

              <form v-else id="mail-template-form" class="space-y-4" @submit.prevent="saveTemplate">
                <label class="grid gap-1.5">
                  <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.path }}</span>
                  <input v-model="formPath" class="ui-input" :disabled="!!editingTemplatePath" />
                </label>
                <label class="grid gap-1.5">
                  <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.name }}</span>
                  <input v-model="formName" class="ui-input" />
                </label>
                <label class="grid gap-1.5">
                  <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.subjectTemplate }}</span>
                  <input v-model="formSubject" class="ui-input" />
                </label>
                <label class="grid gap-1.5">
                  <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.htmlBody }}</span>
                  <textarea v-model="formContent" class="min-h-56 w-full rounded-[10px] border border-slate-200 bg-white p-3 text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
                </label>
                <label class="grid gap-1.5">
                  <span class="ui-label">{{ copy.fields.description }}</span>
                  <textarea v-model="formDescription" class="min-h-24 w-full rounded-[10px] border border-slate-200 bg-white p-3 text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
                </label>
                <label class="grid gap-1.5">
                  <span class="ui-label">{{ copy.fields.parameterSchema }}</span>
                  <textarea v-model="formParameterSchema" class="min-h-32 w-full rounded-[10px] border border-slate-200 bg-white p-3 font-mono text-sm text-slate-900 transition-colors placeholder:text-slate-400 focus:border-primary-400 focus:outline-none focus:ring-4 focus:ring-primary-500/10" />
                </label>
              </form>
            </div>

            <div v-if="templateMode !== 'detail'" class="flex flex-col items-stretch justify-end gap-3 border-t border-hairline px-4 py-4 sm:flex-row sm:items-center md:px-6">
              <button class="ui-btn ui-btn-secondary" type="button" @click="closeTemplateDialog">{{ copy.reset }}</button>
              <button class="ui-btn ui-btn-primary" :disabled="templateSaving" type="submit" form="mail-template-form">
                <Loader2 v-if="templateSaving" class="h-4 w-4 animate-spin" />
                {{ templateSaving ? copy.saving : copy.saveTemplate }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <!-- Mail detail dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="mailDetailOpen && selectedMail" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1280px]">
            <div class="flex flex-wrap items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="break-words text-base font-semibold text-ink md:text-lg">{{ copy.detailTitle }}</h2>
              </div>
              <div class="flex w-full shrink-0 flex-wrap items-center gap-3 sm:w-auto">
                <button
                  v-if="selectedMailCanCancel"
                  class="ui-btn ui-btn-danger flex-1 sm:flex-none"
                  type="button"
                  :disabled="!selectedMail || canceling"
                  @click="cancelMail"
                >
                  <Loader2 v-if="canceling" class="h-4 w-4 animate-spin" />
                  <XCircle v-else class="h-4 w-4" />
                  {{ copy.cancelMail }}
                </button>
                <button
                  class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                  type="button"
                  :aria-label="copy.close"
                  @click="closeMailDetail"
                >
                  <X class="h-5 w-5" />
                </button>
              </div>
            </div>

            <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-6">
              <div v-if="mailDetailLoading" class="space-y-3">
                <div class="skeleton h-4 w-2/5" />
                <div class="skeleton h-36 w-full" />
                <div class="skeleton h-4 w-3/5" />
                <div class="skeleton h-24 w-full" />
              </div>
              <template v-else>
                <JsonPreview
                  v-if="mailStatusDetail"
                  :title="copy.statusDetail"
                  :value="mailStatusDetail"
                  :copy-label="copy.copyJson"
                  :copied-label="copy.copiedJson"
                  :copied-message="copy.toasts.jsonCopied"
                  :copy-error-message="copy.toasts.jsonCopyFailed"
                  max-height="144px"
                />
                <iframe v-if="selectedMailHtml" class="h-[320px] w-full rounded-2xl border border-edge bg-white md:h-[440px]" sandbox="allow-same-origin" :srcdoc="selectedMailHtml" />
                <JsonPreview
                  :title="copy.rawJson"
                  :value="selectedMailRecord"
                  :copy-label="copy.copyJson"
                  :copied-label="copy.copiedJson"
                  :copied-message="copy.toasts.jsonCopied"
                  :copy-error-message="copy.toasts.jsonCopyFailed"
                />
              </template>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
