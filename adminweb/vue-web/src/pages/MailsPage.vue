<script setup lang="ts">
import { FileText, List, Loader2, Mail, Plus, RefreshCw, Send, X, XCircle } from "lucide-vue-next"
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
let templateDetailRequestId = 0
let templateEditRequestId = 0
let templatePayloadRequestId = 0
let mailDetailRequestId = 0
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

let usersRequestId = 0
let templatesListRequestId = 0
let sentMailsRequestId = 0

async function loadUsers() {
  const requestId = ++usersRequestId
  usersLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/user/list")
    if (requestId !== usersRequestId) return
    const list = Array.isArray(data.users) ? data.users : []
    users.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item) && !!item.email)
  } catch (err) {
    if (requestId !== usersRequestId) return
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.usersLoadFailed))
  } finally {
    if (requestId === usersRequestId) usersLoading.value = false
  }
}

async function loadTemplates() {
  const requestId = ++templatesListRequestId
  templatesLoading.value = true
  try {
    const params = new URLSearchParams({
      page_size: String(templatePageSize),
    })
    const cursor = templateCursorStack.value[templatePage.value - 1] || ""
    if (cursor) params.set("cursor", cursor)
    const data = await apiClient<JsonRecord>(`/api/mails/templates?${params}`)
    if (requestId !== templatesListRequestId) return
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
    if (requestId !== templatesListRequestId) return
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templatesLoadFailed))
  } finally {
    if (requestId === templatesListRequestId) templatesLoading.value = false
  }
}

async function loadTemplateDetail(path: string) {
  if (!path) return null
  return apiClient<JsonRecord>(`/api/mails/templates/detail?path=${encodeURIComponent(path)}`)
}

async function selectTemplate(template: JsonRecord) {
  const path = pathOf(template)
  if (!path) return false
  const requestId = ++templateDetailRequestId
  selectedTemplate.value = template
  templateMode.value = "detail"
  try {
    const detail = await loadTemplateDetail(path)
    if (requestId !== templateDetailRequestId || pathOf(selectedTemplate.value) !== path || templateMode.value !== "detail") return false
    if (detail) selectedTemplate.value = { ...template, ...detail }
    return true
  } catch (err) {
    if (requestId !== templateDetailRequestId || pathOf(selectedTemplate.value) !== path || templateMode.value !== "detail") return false
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
    return true
  }
}

async function openTemplateDetail(template: JsonRecord) {
  templateEditRequestId += 1
  templateDialogOpen.value = false
  resetTemplateForm()
  const current = await selectTemplate(template)
  if (!current) return
  templateDialogOpen.value = true
}

async function openTemplateEdit(template: JsonRecord | null = selectedTemplate.value) {
  templateDetailRequestId += 1
  templateDialogOpen.value = false
  const current = await editTemplate(template)
  if (!current) return
  templateDialogOpen.value = true
}

function closeTemplateDialog() {
  templateDetailRequestId += 1
  templateEditRequestId += 1
  templateDialogOpen.value = false
  selectedTemplate.value = null
  templateMode.value = "detail"
  resetTemplateForm()
}

async function loadStats() {
  try {
    stats.value = await apiClient<JsonRecord>("/api/mails/stats")
  } catch {
    stats.value = null
  }
}

async function loadSentMails() {
  const requestId = ++sentMailsRequestId
  mailsLoading.value = true
  try {
    const params = new URLSearchParams({ page_size: String(pageSize) })
    const cursor = mailCursorStack.value[mailPage.value - 1] || ""
    if (cursor) params.set("cursor", cursor)
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/mails/sent?${params}`)
    if (requestId !== sentMailsRequestId) return
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
    if (requestId !== sentMailsRequestId) return
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.sentLoadFailed))
  } finally {
    if (requestId === sentMailsRequestId) mailsLoading.value = false
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
    resetTemplateForm()
    return false
  }
  const path = pathOf(template)
  if (!path) {
    resetTemplateForm()
    return false
  }
  const requestId = ++templateEditRequestId
  templateMode.value = "edit"
  editingTemplatePath.value = path
  formPath.value = path
  formName.value = nameOf(template)
  formSubject.value = subjectOf(template)
  formContent.value = bodyOf(template)
  formDescription.value = descriptionOf(template)
  formParameterSchema.value = String(template.parameter_schema || "{}")

  try {
    const detail = await loadTemplateDetail(path)
    if (requestId !== templateEditRequestId || editingTemplatePath.value !== path || templateMode.value !== "edit") return false
    if (detail) {
      const merged = resolvedTemplate({ ...template, ...detail })
      selectedTemplate.value = merged
      formName.value = nameOf(merged)
      formSubject.value = subjectOf(merged)
      formContent.value = bodyOf(merged)
      formDescription.value = descriptionOf(merged)
      formParameterSchema.value = String(merged.parameter_schema || "{}")
    }
    return true
  } catch (err) {
    if (requestId !== templateEditRequestId || editingTemplatePath.value !== path || templateMode.value !== "edit") return false
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.templateDetailLoadFailed))
    return true
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
  templateDetailRequestId += 1
  templateEditRequestId += 1
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
    closeTemplateDialog()
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
  const requestId = ++mailDetailRequestId
  selectedMail.value = mail
  mailDetailOpen.value = open && !!mail
  mailDetail.value = null
  mailStatusDetail.value = null
  const id = mailId(mail)
  if (!id) {
    mailDetailLoading.value = false
    return
  }

  mailDetailLoading.value = true
  try {
    const [detailResult, statusResult] = await Promise.allSettled([
      apiClient<JsonRecord>(`/api/mails?mail_id=${encodeURIComponent(id)}`),
      apiClient<JsonRecord>(`/api/mails/status?mail_id=${encodeURIComponent(id)}`),
    ])
    if (requestId !== mailDetailRequestId || mailId(selectedMail.value) !== id) return

    if (detailResult.status === "fulfilled") mailDetail.value = detailResult.value
    if (statusResult.status === "fulfilled") mailStatusDetail.value = statusResult.value

    const failedResult = [detailResult, statusResult].find((result) => result.status === "rejected")
    if (failedResult?.status === "rejected") {
      console.error(failedResult.reason)
      toast.error(apiErrorMessage(failedResult.reason, copy.value.toasts.mailDetailLoadFailed))
    }
  } finally {
    if (requestId === mailDetailRequestId && mailId(selectedMail.value) === id) mailDetailLoading.value = false
  }
}

function closeMailDetail() {
  mailDetailRequestId += 1
  mailDetailOpen.value = false
  selectedMail.value = null
  mailDetail.value = null
  mailStatusDetail.value = null
  mailDetailLoading.value = false
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
  const requestId = ++templatePayloadRequestId
  if (!path) {
    payload.value = "{\n}"
    return
  }
  const template = templates.value.find((item) => pathOf(item) === path)
  payload.value = extractPayloadTemplate(subjectOf(template), bodyOf(template || {}))
  try {
    const detail = await loadTemplateDetail(path)
    if (requestId !== templatePayloadRequestId || templatePath.value !== path) return
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

    <div v-if="statsCards.length" class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 class="text-xl font-black">{{ copy.statsTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.statsDescription }}</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">{{ copy.statsBadge }}</span>
      </div>
      <div class="mt-4 grid gap-3 md:grid-cols-4">
        <div v-for="card in statsCards" :key="card.key" class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
          <div class="text-xs font-black uppercase tracking-wide text-slate-400">{{ card.key }}</div>
          <div class="mt-2 text-sm font-bold text-slate-600">{{ card.label }}</div>
          <div class="mt-1 text-3xl font-black text-slate-950">{{ card.value }}</div>
        </div>
      </div>
    </div>

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
          <h2 class="text-xl font-black">{{ copy.sendTitle }}</h2>
          <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_1fr]">
            <div>
              <div class="mb-2 flex items-center justify-between">
                <label class="font-bold">{{ copy.recipients }}</label>
                <div class="flex gap-3 text-sm">
                  <button class="font-bold text-blue-700" type="button" @click="selectedUserIds = users.map(userId).filter(Boolean)">{{ copy.selectAll }}</button>
                  <button class="font-bold text-slate-500" type="button" @click="selectedUserIds = []">{{ copy.clear }}</button>
                </div>
              </div>
              <div class="max-h-[360px] overflow-y-auto rounded-2xl border border-slate-200 p-3 md:max-h-[520px]">
                <label v-for="user in users" :key="userId(user)" class="flex cursor-pointer items-center gap-3 rounded-xl px-3 py-2 hover:bg-slate-50">
                  <input v-model="selectedUserIds" class="h-4 w-4" type="checkbox" :value="userId(user)" />
                  <span class="font-semibold">{{ userName(user) }}</span>
                  <span class="break-all text-xs text-slate-400">{{ userEmail(user) }}</span>
                </label>
                <div v-if="!users.length" class="p-6 text-center text-slate-500">{{ copy.noUsers }}</div>
              </div>
              <p class="mt-2 text-sm text-slate-500">{{ copy.selectedUsers(selectedUserIds.length) }}</p>
            </div>

            <div class="space-y-4">
              <label class="block">
                <span class="font-bold">{{ copy.template }}</span>
                <select v-model="templatePath" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4">
                  <option value="">{{ copy.noTemplateOption }}</option>
                  <option v-for="template in templates" :key="pathOf(template)" :value="pathOf(template)">
                    {{ nameOf(template) }} ({{ pathOf(template) }})
                  </option>
                </select>
              </label>
              <label class="block">
                <span class="font-bold">{{ copy.subject }}</span>
                <input v-model="subject" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4" :placeholder="copy.subjectPlaceholder" />
              </label>
              <label class="block">
                <span class="font-bold">{{ templatePath ? copy.payloadLabel : copy.bodyLabel }}</span>
                <textarea v-model="payload" class="mt-2 min-h-52 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" />
              </label>
              <button class="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50 sm:w-auto" :disabled="sending" type="button" @click="sendMail">
                <Loader2 v-if="sending" class="h-4 w-4 animate-spin" />
                <Mail v-else class="h-4 w-4" />
                {{ sending ? copy.sending : copy.sendMail }}
              </button>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'sent'" class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
          <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-4 md:p-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black">{{ copy.sentTitle }}</h2>
              
            </div>
            <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
              <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalText(total) }}</span>
              <select v-model="statusFilter" class="h-11 min-w-0 flex-1 rounded-xl border border-slate-200 px-4 text-sm sm:flex-none">
              <option v-for="option in statusOptions" :key="option.value || 'all'" :value="option.value">{{ option.label }}</option>
            </select>
            </div>
          </div>
          <div class="hidden grid-cols-[minmax(0,1fr)_220px_150px_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500 md:grid">
            <span>{{ copy.columns.mail }}</span>
            <span>{{ copy.columns.recipient }}</span>
            <span class="text-center">{{ copy.columns.status }}</span>
            <span class="text-right">{{ copy.columns.time }}</span>
            <span class="text-right">{{ copy.columns.action }}</span>
          </div>
          <div v-if="mailsLoading" class="p-8 text-center text-slate-500 md:p-12">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.loading }}
          </div>
          <div v-else-if="mails.length" class="divide-y divide-slate-100">
            <div
              v-for="mail in mails"
              :key="mailId(mail)"
              class="flex cursor-pointer flex-col gap-3 px-4 py-4 transition hover:bg-sky-50 md:grid md:grid-cols-[minmax(0,1fr)_220px_150px_180px_112px] md:items-center md:gap-5 md:px-5"
              :class="mailId(selectedMail) === mailId(mail) ? 'bg-sky-50' : ''"
              role="button"
              tabindex="0"
              @click="openMail(mail)"
              @keydown.enter.prevent="openMail(mail)"
              @keydown.space.prevent="openMail(mail)"
            >
              <div class="min-w-0">
                <div class="break-words font-black text-slate-950 md:truncate">{{ pickFirst(mail, ["subject", "template_path", "mail_id"]) || copy.defaults.mail }}</div>
                <div class="mt-1 break-all text-xs font-semibold text-slate-500">{{ copy.mailIdPrefix }}{{ mailId(mail) || "-" }}</div>
              </div>
              <div class="flex min-w-0 items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.recipient }}</span>
                <span class="break-all text-right text-sm font-semibold text-slate-500 md:text-left">{{ pickFirst(mail, ["to_email", "recipient_email", "user_email"]) || "-" }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0 md:text-center">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.status }}</span>
                <span class="inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(mailStatus(mail))">{{ mailStatusLabel(mailStatus(mail)) }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.columns.time }}</span>
                <span class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(pickFirst(mail, ["created_at", "sent_at", "updated_at"]) || "")) }}</span>
              </div>
              <div class="text-right">
                <button
                  class="inline-flex w-full items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline md:w-auto md:border-0 md:bg-transparent md:px-0 md:py-0"
                  type="button"
                  @click.stop="openMail(mail)"
                >
                  {{ copy.viewDetails }}
                </button>
              </div>
            </div>
          </div>
          <div v-else class="p-8 text-center text-slate-500 md:p-12">{{ copy.emptySent }}</div>
          <div class="flex flex-col items-stretch justify-between gap-3 border-t border-slate-200 p-4 sm:flex-row sm:items-center md:p-5">
            <span class="text-center text-sm font-bold text-slate-500 sm:text-left">{{ copy.pageText(mailPage, totalPages) }}</span>
            <div class="flex flex-col gap-3 sm:flex-row">
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="mailPage <= 1" @click="mailPage--">{{ copy.prev }}</button>
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!mailHasMore" @click="mailPage++">{{ copy.next }}</button>
            </div>
          </div>
        </section>

        <section v-else class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
          <div class="flex flex-wrap items-start justify-between gap-3 border-b border-slate-200 p-4 md:p-5">
            <div class="min-w-0">
              <h2 class="text-xl font-black">{{ copy.templateListTitle }}</h2>
              
            </div>
            <div class="flex w-full flex-wrap items-center gap-3 sm:w-auto">
              <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ copy.totalText(totalTemplates || templates.length) }}</span>
              <button class="inline-flex h-10 flex-1 items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 text-sm font-bold text-white shadow-sm sm:flex-none" type="button" @click="startCreateTemplate">
              <Plus class="h-4 w-4" />
              {{ copy.createTemplate }}
            </button>
          </div>
          </div>
          <div class="hidden grid-cols-[minmax(0,1fr)_140px_180px_160px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500 md:grid">
            <span>{{ copy.template }}</span>
            <span class="text-center">{{ copy.templateColumns.version }}</span>
            <span class="text-right">{{ copy.templateColumns.updatedAt }}</span>
            <span class="text-right">{{ copy.templateColumns.action }}</span>
          </div>
          <div v-if="templatesLoading" class="p-8 text-center text-slate-500 md:p-12">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            {{ copy.loading }}
          </div>
          <div v-else-if="templates.length" class="divide-y divide-slate-100">
            <div
              v-for="template in templates"
              :key="pathOf(template)"
              class="flex flex-col gap-3 px-4 py-4 transition hover:bg-slate-50 md:grid md:grid-cols-[minmax(0,1fr)_140px_180px_160px] md:items-center md:gap-5 md:px-5"
              :class="pathOf(selectedTemplate) === pathOf(template) ? 'bg-sky-50' : ''"
            >
              <div class="min-w-0">
                <div class="break-words font-black text-slate-950 md:truncate">{{ nameOf(template) }}</div>
                <div class="mt-1 break-all text-sm font-semibold text-slate-500">{{ pathOf(template) }}</div>
                <div v-if="template.description" class="mt-1 break-words text-xs font-semibold text-slate-400 md:truncate">{{ template.description }}</div>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0 md:text-center">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.templateColumns.version }}</span>
                <span class="inline-flex rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">v{{ String(template.version || "-") }}</span>
              </div>
              <div class="flex items-center justify-between gap-3 rounded-2xl bg-slate-50 px-3 py-2 md:block md:rounded-none md:bg-transparent md:p-0">
                <span class="text-xs font-black text-slate-400 md:hidden">{{ copy.templateColumns.updatedAt }}</span>
                <span class="text-right text-sm font-semibold text-slate-500">{{ formatDate(String(pickFirst(template, ["updated_at", "created_at"]) || "")) }}</span>
              </div>
              <div class="flex flex-col justify-end gap-3 sm:flex-row">
                <button class="inline-flex items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 transition hover:underline md:border-0 md:bg-transparent md:px-0 md:py-0" type="button" @click="openTemplateDetail(template)">{{ copy.viewDetails }}</button>
                <button class="inline-flex items-center justify-center rounded-xl border border-amber-100 bg-amber-50 px-3 py-2 text-sm font-bold text-amber-600 transition hover:underline md:border-0 md:bg-transparent md:px-0 md:py-0" type="button" @click="openTemplateEdit(template)">{{ copy.editTemplate }}</button>
              </div>
            </div>
          </div>
          <div v-else class="p-8 text-center text-slate-500 md:p-12">{{ copy.emptyTemplates }}</div>
          <div class="flex flex-col items-stretch justify-between gap-3 border-t border-slate-200 p-4 sm:flex-row sm:items-center md:p-5">
            <span class="text-center text-sm font-bold text-slate-500 sm:text-left">{{ copy.pageText(templatePage, templateTotalPages) }}</span>
            <div class="flex flex-col gap-3 sm:flex-row">
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="templatePage <= 1 || templatesLoading" @click="templatePage--">{{ copy.prev }}</button>
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!templateHasMore || templatesLoading" @click="templatePage++">{{ copy.next }}</button>
            </div>
          </div>
        </section>
      </main>
    </div>

    <Teleport to="body">
      <div v-if="templateDialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <section class="flex h-full max-h-none w-full max-w-[1120px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black text-slate-950 md:truncate md:text-2xl">
                {{ templateMode === "detail" ? copy.templateDetails : editingTemplatePath ? copy.editTemplate : copy.createTemplate }}
              </h2>
              <p v-if="templateMode === 'detail' && selectedTemplate" class="mt-1 break-all text-sm font-semibold text-slate-500">{{ pathOf(selectedTemplate) }}</p>
              <p v-else-if="editingTemplatePath" class="mt-1 break-all text-sm font-semibold text-slate-500">{{ formPath }}</p>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              type="button"
              :aria-label="copy.close"
              @click="closeTemplateDialog"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
            <template v-if="templateMode === 'detail'">
              <div v-if="!selectedTemplate" class="p-10 text-center text-slate-500">{{ copy.selectTemplate }}</div>
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
              <label class="block">
                <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.path }}</span>
                <input v-model="formPath" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4 disabled:bg-slate-100" :disabled="!!editingTemplatePath" />
              </label>
              <label class="block">
                <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.name }}</span>
                <input v-model="formName" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4" />
              </label>
              <label class="block">
                <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.subjectTemplate }}</span>
                <input v-model="formSubject" class="mt-2 h-11 w-full rounded-xl border border-slate-200 px-4" />
              </label>
              <label class="block">
                <span class="text-sm font-bold"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.htmlBody }}</span>
                <textarea v-model="formContent" class="mt-2 min-h-56 w-full rounded-xl border border-slate-200 p-4" />
              </label>
              <label class="block">
                <span class="text-sm font-bold">{{ copy.fields.description }}</span>
                <textarea v-model="formDescription" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" />
              </label>
              <label class="block">
                <span class="text-sm font-bold">{{ copy.fields.parameterSchema }}</span>
                <textarea v-model="formParameterSchema" class="mt-2 min-h-32 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" />
              </label>
            </form>
          </div>

          <div v-if="templateMode !== 'detail'" class="flex flex-col items-stretch justify-end gap-3 border-t border-slate-200 px-4 py-4 sm:flex-row sm:items-center md:px-6 md:py-5">
            <button class="inline-flex h-11 min-w-[96px] items-center justify-center rounded-xl border px-5 font-bold" type="button" @click="closeTemplateDialog">{{ copy.reset }}</button>
            <button class="inline-flex h-11 min-w-[120px] items-center justify-center rounded-xl bg-blue-700 px-5 font-bold text-white shadow-sm disabled:opacity-50" :disabled="templateSaving" type="submit" form="mail-template-form">
              {{ templateSaving ? copy.saving : copy.saveTemplate }}
            </button>
          </div>
        </section>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="mailDetailOpen && selectedMail" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
        <section class="flex h-full max-h-none w-full max-w-[1280px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
          <div class="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6 md:py-5">
            <div class="min-w-0">
              <h2 class="break-words text-xl font-black text-slate-950 md:truncate md:text-2xl">{{ copy.detailTitle }}</h2>
            </div>
            <div class="flex w-full shrink-0 flex-wrap items-center gap-3 sm:w-auto">
              <button
                v-if="selectedMailCanCancel"
                class="inline-flex flex-1 items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-2 text-sm font-bold text-white disabled:opacity-50 sm:flex-none"
                type="button"
                :disabled="!selectedMail || canceling"
                @click="cancelMail"
              >
                <XCircle class="h-4 w-4" />
                {{ copy.cancelMail }}
              </button>
              <button
                class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
                type="button"
                :aria-label="copy.close"
                @click="closeMailDetail"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
          </div>

          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-4 md:p-5">
            <div v-if="mailDetailLoading" class="p-10 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              {{ copy.loading }}
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
              <iframe v-if="selectedMailHtml" class="h-[320px] w-full rounded-2xl border border-slate-200 bg-white md:h-[440px]" sandbox="allow-same-origin" :srcdoc="selectedMailHtml" />
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
    </Teleport>
  </section>
</template>
