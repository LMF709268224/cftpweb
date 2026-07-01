<script setup lang="ts">
import { FileText, List, Loader2, Mail, RefreshCw, Send, XCircle } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

type TabKey = "send" | "sent" | "templates"

const pageSize = 20
const activeTab = ref<TabKey>("send")
const users = ref<JsonRecord[]>([])
const templates = ref<JsonRecord[]>([])
const selectedTemplate = ref<JsonRecord | null>(null)
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
const canceling = ref(false)
const stats = ref<JsonRecord | null>(null)

const usersLoading = ref(false)
const templatesLoading = ref(false)
const templateSaving = ref(false)
const editingTemplatePath = ref("")
const formPath = ref("")
const formName = ref("")
const formSubject = ref("")
const formContent = ref("")
const formDescription = ref("")

const tabs = computed(() => [
  { key: "send" as const, label: "发送邮件", icon: Send, count: selectedUserIds.value.length },
  { key: "sent" as const, label: "发送记录", icon: List, count: total.value },
  { key: "templates" as const, label: "模板管理", icon: FileText, count: templates.value.length },
])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const selectedUsers = computed(() => users.value.filter((user) => selectedUserIds.value.includes(userId(user))))
const selectedMailHtml = computed(() => String(mailDetail.value?.html_body || mailDetail.value?.plain_body || selectedMail.value?.html_body || selectedMail.value?.plain_body || ""))
const selectedMailRecord = computed(() => mailDetail.value || selectedMail.value || {})
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
  return [{ key: "TOTAL", label: "总邮件数", value: totalCount }, ...entries]
})

function userId(user: JsonRecord) {
  return String(pickFirst(user, ["id", "user_id", "candidate_ulid", "ulid"]) || "")
}

function userEmail(user: JsonRecord) {
  return String(user.email || "")
}

function userName(user: JsonRecord) {
  return String(pickFirst(user, ["name", "nickname", "email", "id"]) || userEmail(user) || "用户")
}

function pathOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["path", "template_path", "template_id"]) || "")
}

function nameOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["name", "template_name", "subject_template", "path"]) || pathOf(template) || "模板")
}

function bodyOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["html_body", "template_body", "plain_body"]) || "")
}

function mailId(mail: JsonRecord | null | undefined) {
  return String(pickFirst(mail || {}, ["mail_id", "mail_ulid", "id"]) || "")
}

function mailStatus(mail: JsonRecord | null | undefined) {
  return pickFirst(mail || {}, ["status", "raw_status"]) || "-"
}

function mailStatusLabel(status: unknown) {
  const value = String(status || "").toUpperCase()
  if (value === "SENT") return "已发送"
  if (value === "FAILED") return "失败"
  if (value === "CANCELLED") return "已取消"
  if (value === "SCHEDULING") return "调度中"
  if (value === "PENDING") return "待处理"
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

function validateTemplatePayload() {
  if (!templatePath.value || !payload.value.trim()) return true
  try {
    const parsed = JSON.parse(payload.value)
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      toast.error("模板 Payload 必须是 JSON 对象")
      return false
    }
    return true
  } catch (err) {
    toast.error(`Payload JSON 格式错误：${err instanceof Error ? err.message : String(err)}`)
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
    toast.error("用户列表加载失败")
  } finally {
    usersLoading.value = false
  }
}

async function loadTemplates() {
  templatesLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/mails/templates")
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selectedTemplate.value || !templates.value.some((item) => pathOf(item) === pathOf(selectedTemplate.value))) {
      selectedTemplate.value = templates.value[0] || null
    }
  } catch (err) {
    console.error(err)
    toast.error("邮件模板加载失败")
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
  try {
    const detail = await loadTemplateDetail(pathOf(template))
    if (detail) selectedTemplate.value = { ...template, ...detail }
  } catch (err) {
    console.error(err)
    toast.error("模板详情加载失败")
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
    const params = new URLSearchParams({ page: String(mailPage.value), page_size: String(pageSize) })
    if (statusFilter.value) params.set("status", statusFilter.value)
    const data = await apiClient<JsonRecord>(`/api/mails/sent?${params}`)
    const list = Array.isArray(data.mails) ? data.mails : []
    mails.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || mails.value.length)
    if (!selectedMail.value || !mails.value.some((item) => mailId(item) === mailId(selectedMail.value))) {
      await openMail(mails.value[0] || null)
    }
  } catch (err) {
    console.error(err)
    toast.error("邮件发送记录加载失败")
  } finally {
    mailsLoading.value = false
  }
}

async function sendMail() {
  if (!selectedUsers.value.length) {
    toast.error("请先选择收件用户")
    return
  }
  if (!templatePath.value && !subject.value.trim()) {
    toast.error("未选择模板时必须填写邮件主题")
    return
  }
  if (!templatePath.value && !payload.value.trim()) {
    toast.error("未选择模板时必须填写邮件正文")
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
    toast.success(`邮件已发送：${count} 封`)
    selectedUserIds.value = []
    templatePath.value = ""
    subject.value = ""
    payload.value = "{\n}"
  } catch (err) {
    console.error(err)
    toast.error("邮件发送失败")
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
  formName.value = nameOf(template)
  formSubject.value = String(template.subject_template || "")
  formContent.value = bodyOf(template)
  formDescription.value = String(template.description || "")

  try {
    const detail = await loadTemplateDetail(path)
    if (detail) {
      selectedTemplate.value = { ...template, ...detail }
      formName.value = String(detail.name || formName.value)
      formSubject.value = String(detail.subject_template || "")
      formContent.value = String(detail.html_body || detail.template_body || detail.plain_body || "")
      formDescription.value = String(detail.description || formDescription.value)
    }
  } catch (err) {
    console.error(err)
    toast.error("模板详情加载失败")
  }
}

function resetTemplateForm() {
  editingTemplatePath.value = ""
  formPath.value = ""
  formName.value = ""
  formSubject.value = ""
  formContent.value = ""
  formDescription.value = ""
}

async function saveTemplate() {
  const path = editingTemplatePath.value || formPath.value.trim()
  if (!path || !formName.value.trim() || !formSubject.value.trim() || !formContent.value.trim()) {
    toast.error("请填写模板路径、名称、主题和正文")
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
      }),
    })
    toast.success(editingTemplatePath.value ? "模板已更新" : "模板已创建")
    resetTemplateForm()
    await loadTemplates()
  } catch (err) {
    console.error(err)
    toast.error("模板保存失败")
  } finally {
    templateSaving.value = false
  }
}

async function openMail(mail: JsonRecord | null) {
  selectedMail.value = mail
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
    toast.error("邮件详情加载失败")
  } finally {
    mailDetailLoading.value = false
  }
}

async function cancelMail() {
  const id = mailId(selectedMail.value)
  if (!id) return
  canceling.value = true
  try {
    await apiClient("/api/mails/cancel", { method: "POST", body: JSON.stringify({ mail_id: id }) })
    toast.success("邮件已取消")
    await loadSentMails()
  } catch (err) {
    console.error(err)
    toast.error("取消失败")
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
  payload.value = extractPayloadTemplate(template?.subject_template, bodyOf(template || {}))
  try {
    const detail = await loadTemplateDetail(path)
    payload.value = extractPayloadTemplate(detail?.subject_template, detail?.html_body, detail?.template_body, detail?.plain_body)
  } catch {
    // Keep the best-effort payload generated from the list response.
  }
})

watch([activeTab, mailPage, statusFilter], () => {
  if (activeTab.value === "sent") void loadSentMails()
})

onMounted(async () => {
  await Promise.all([loadUsers(), loadTemplates(), loadStats()])
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1580px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">邮件中心</h1>
        <p class="mt-2 text-slate-600">发送邮件、维护模板并查看投递记录。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：send/list/get/status/cancel/stats/list templates/get/create/update/render/exists/builtin paths。delete 路由未实现，页面不提供删除按钮。
        </p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadUsers">
          <RefreshCw class="h-4 w-4" :class="usersLoading ? 'animate-spin' : ''" />
          刷新用户
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadTemplates">
          <RefreshCw class="h-4 w-4" :class="templatesLoading ? 'animate-spin' : ''" />
          刷新模板
        </button>
      </div>
    </header>

    <div v-if="statsCards.length" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 class="text-xl font-black">邮件统计</h2>
          <p class="mt-1 text-sm text-slate-500">来自 `/api/mails/stats`，按邮件状态汇总。</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">只读统计</span>
      </div>
      <div class="mt-4 grid gap-3 md:grid-cols-4">
        <div v-for="card in statsCards" :key="card.key" class="rounded-2xl border border-slate-100 bg-slate-50 p-4">
          <div class="text-xs font-black uppercase tracking-wide text-slate-400">{{ card.key }}</div>
          <div class="mt-2 text-sm font-bold text-slate-600">{{ card.label }}</div>
          <div class="mt-1 text-3xl font-black text-slate-950">{{ card.value }}</div>
        </div>
      </div>
    </div>

    <div class="grid gap-6 xl:grid-cols-[320px_minmax(0,1fr)]">
      <aside class="rounded-3xl border border-slate-200 bg-white p-4 shadow-sm">
        <h2 class="px-2 text-lg font-black">功能</h2>
        <div class="mt-4 space-y-2">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            class="w-full rounded-2xl border px-4 py-3 text-left"
            :class="activeTab === tab.key ? 'border-sky-200 bg-sky-50' : 'border-slate-100 hover:bg-slate-50'"
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
      </aside>

      <main class="min-w-0">
        <section v-if="activeTab === 'send'" class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 class="text-xl font-black">新建邮件</h2>
          <div class="mt-6 grid gap-6 lg:grid-cols-[1fr_1fr]">
            <div>
              <div class="mb-2 flex items-center justify-between">
                <label class="font-bold">收件用户</label>
                <div class="flex gap-3 text-sm">
                  <button class="font-bold text-[#0b7bdc]" type="button" @click="selectedUserIds = users.map(userId).filter(Boolean)">全选</button>
                  <button class="font-bold text-slate-500" type="button" @click="selectedUserIds = []">清空</button>
                </div>
              </div>
              <div class="max-h-[520px] overflow-y-auto rounded-2xl border border-slate-200 p-3">
                <label v-for="user in users" :key="userId(user)" class="flex cursor-pointer items-center gap-3 rounded-xl px-3 py-2 hover:bg-slate-50">
                  <input v-model="selectedUserIds" class="h-4 w-4" type="checkbox" :value="userId(user)" />
                  <span class="font-semibold">{{ userName(user) }}</span>
                  <span class="break-all text-xs text-slate-400">{{ userEmail(user) }}</span>
                </label>
                <div v-if="!users.length" class="p-6 text-center text-slate-500">暂无带邮箱的用户</div>
              </div>
              <p class="mt-2 text-sm text-slate-500">已选择 {{ selectedUserIds.length }} 个用户。</p>
            </div>

            <div class="space-y-4">
              <label class="block">
                <span class="font-bold">模板</span>
                <select v-model="templatePath" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                  <option value="">不使用模板，直接发送正文</option>
                  <option v-for="template in templates" :key="pathOf(template)" :value="pathOf(template)">
                    {{ nameOf(template) }} ({{ pathOf(template) }})
                  </option>
                </select>
              </label>
              <label class="block">
                <span class="font-bold">邮件主题</span>
                <input v-model="subject" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="使用模板时可留空" />
              </label>
              <label class="block">
                <span class="font-bold">{{ templatePath ? "Payload JSON" : "邮件正文 HTML / Text" }}</span>
                <textarea v-model="payload" class="mt-2 min-h-52 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" />
              </label>
              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="sending" type="button" @click="sendMail">
                <Loader2 v-if="sending" class="h-4 w-4 animate-spin" />
                <Mail v-else class="h-4 w-4" />
                {{ sending ? "发送中..." : "发送邮件" }}
              </button>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'sent'" class="grid gap-6 xl:grid-cols-[430px_minmax(0,1fr)]">
          <div class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
            <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">发送记录</h2>
                <p class="mt-1 text-sm text-slate-500">每页 {{ pageSize }} 条，总计 {{ total }} 条。</p>
              </div>
              <select v-model="statusFilter" class="rounded-xl border border-slate-200 px-4 py-2">
                <option value="">全部状态</option>
                <option value="SCHEDULING">调度中</option>
                <option value="SENT">已发送</option>
                <option value="FAILED">失败</option>
                <option value="CANCELLED">已取消</option>
              </select>
            </div>
            <div v-if="mailsLoading" class="p-12 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <button
              v-for="mail in mails"
              v-else
              :key="mailId(mail)"
              class="block w-full border-b border-slate-100 p-5 text-left last:border-b-0 hover:bg-sky-50"
              :class="mailId(selectedMail) === mailId(mail) ? 'bg-sky-50' : ''"
              type="button"
              @click="openMail(mail)"
            >
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate font-black">{{ pickFirst(mail, ["subject", "template_path", "mail_id"]) || "邮件" }}</div>
                  <div class="mt-1 break-all text-sm text-slate-500">{{ pickFirst(mail, ["to_email", "recipient_email", "user_email"]) || "-" }}</div>
                </div>
                <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(mailStatus(mail))">{{ mailStatus(mail) }}</span>
              </div>
              <div class="mt-2 text-sm text-slate-500">{{ formatDate(String(pickFirst(mail, ["created_at", "sent_at", "updated_at"]) || "")) }}</div>
            </button>
            <div v-if="!mailsLoading && !mails.length" class="p-12 text-center text-slate-500">暂无发送记录</div>
            <div class="flex items-center justify-end gap-3 border-t border-slate-200 p-5">
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="mailPage <= 1" @click="mailPage--">上一页</button>
              <span class="text-sm font-bold">{{ mailPage }} / {{ totalPages }}</span>
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="mailPage >= totalPages" @click="mailPage++">下一页</button>
            </div>
          </div>

          <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <h2 class="text-xl font-black">邮件详情</h2>
              <button class="inline-flex items-center gap-2 rounded-xl bg-red-600 px-4 py-2 text-sm font-bold text-white disabled:opacity-50" type="button" :disabled="!selectedMail || canceling" @click="cancelMail">
                <XCircle class="h-4 w-4" />
                取消邮件
              </button>
            </div>
            <div v-if="mailDetailLoading" class="p-10 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <div v-else-if="!selectedMail" class="p-10 text-center text-slate-500">请选择一封邮件</div>
            <div v-else class="mt-4 space-y-5">
              <pre v-if="mailStatusDetail" class="max-h-36 overflow-auto rounded-2xl bg-slate-100 p-4 text-xs text-slate-700">{{ JSON.stringify(mailStatusDetail, null, 2) }}</pre>
              <iframe v-if="selectedMailHtml" class="h-[360px] w-full rounded-2xl border border-slate-200 bg-white" sandbox="allow-same-origin" :srcdoc="selectedMailHtml" />
              <pre class="max-h-[420px] overflow-auto rounded-2xl bg-slate-950 p-4 text-xs text-slate-100">{{ JSON.stringify(selectedMailRecord, null, 2) }}</pre>
            </div>
          </div>
        </section>

        <section v-else class="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
          <div class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
            <div class="border-b border-slate-200 p-5">
              <h2 class="text-xl font-black">模板列表</h2>
              <p class="mt-1 text-sm text-slate-500">删除接口未实现，因此这里只提供查看、创建和更新。</p>
            </div>
            <div v-if="templatesLoading" class="p-12 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <button
              v-for="template in templates"
              v-else
              :key="pathOf(template)"
              class="w-full border-b border-slate-100 p-5 text-left last:border-b-0 hover:bg-sky-50"
              :class="pathOf(selectedTemplate) === pathOf(template) ? 'bg-sky-50' : ''"
              type="button"
              @click="selectTemplate(template)"
            >
              <div class="font-black">{{ nameOf(template) }}</div>
              <div class="mt-1 break-all text-sm text-slate-500">{{ pathOf(template) }}</div>
            </button>
            <div v-if="!templatesLoading && !templates.length" class="p-12 text-center text-slate-500">暂无模板</div>
          </div>

          <div class="space-y-6">
            <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <h2 class="text-xl font-black">模板详情</h2>
                <button class="rounded-xl border px-4 py-2 text-sm font-bold" type="button" @click="editTemplate(selectedTemplate)">编辑当前模板</button>
              </div>
              <div v-if="!selectedTemplate" class="p-10 text-center text-slate-500">请选择模板</div>
              <div v-else class="mt-4 grid gap-4 md:grid-cols-2">
                <label v-for="(value, key) in selectedTemplateFields" :key="key" class="grid gap-2 text-sm font-bold">
                  {{ key }}
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

            <form class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm" @submit.prevent="saveTemplate">
              <h2 class="text-xl font-black">{{ editingTemplatePath ? "编辑模板" : "创建模板" }}</h2>
              <label class="mt-4 block">
                <span class="text-sm font-bold">路径</span>
                <input v-model="formPath" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100" :disabled="!!editingTemplatePath" />
              </label>
              <label class="mt-4 block">
                <span class="text-sm font-bold">名称</span>
                <input v-model="formName" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="mt-4 block">
                <span class="text-sm font-bold">主题模板</span>
                <input v-model="formSubject" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="mt-4 block">
                <span class="text-sm font-bold">HTML 正文</span>
                <textarea v-model="formContent" class="mt-2 min-h-40 w-full rounded-xl border border-slate-200 p-4" />
              </label>
              <label class="mt-4 block">
                <span class="text-sm font-bold">描述</span>
                <textarea v-model="formDescription" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 p-4" />
              </label>
              <div class="mt-5 flex gap-3">
                <button class="rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="templateSaving" type="submit">
                  {{ templateSaving ? "保存中..." : "保存模板" }}
                </button>
                <button class="rounded-xl border px-5 py-3 font-bold" type="button" @click="resetTemplateForm">清空</button>
              </div>
            </form>
          </div>
        </section>
      </main>
    </div>
  </section>
</template>
