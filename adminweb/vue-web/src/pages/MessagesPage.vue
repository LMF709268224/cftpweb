<script setup lang="ts">
import { FileText, List, Loader2, RefreshCw, Send } from "lucide-vue-next"
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
const payload = ref("{\n}")
const msgType = ref(1)
const sending = ref(false)

const messages = ref<JsonRecord[]>([])
const messagesLoading = ref(false)
const messagePage = ref(1)
const statusFilter = ref("")
const total = ref(0)
const selectedMessage = ref<JsonRecord | null>(null)

const usersLoading = ref(false)
const templatesLoading = ref(false)
const templateSaving = ref(false)
const editingTemplatePath = ref("")
const formPath = ref("")
const formTitle = ref("")
const formContent = ref("")
const formDescription = ref("")
const formVersion = ref(0)

const tabs = [
  { key: "send" as const, label: "发送站内信", icon: Send, count: selectedUserIds.value.length },
  { key: "sent" as const, label: "发送记录", icon: List, count: total.value },
  { key: "templates" as const, label: "模板管理", icon: FileText, count: templates.value.length },
]

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const selectedTemplateFields = computed(() => selectedTemplate.value || {})
const selectedMessageFields = computed(() => selectedMessage.value || {})

function userId(user: JsonRecord) {
  return String(pickFirst(user, ["id", "user_id", "candidate_ulid", "ulid"]) || "")
}

function userLabel(user: JsonRecord) {
  return String(pickFirst(user, ["name", "nickname", "email", "phone", "id"]) || userId(user) || "用户")
}

function pathOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["path", "template_path", "template_id"]) || "")
}

function titleOf(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["title_tpl", "title", "name", "template_name", "path"]) || pathOf(template) || "模板")
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
      toast.error("Payload 必须是 JSON 对象")
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
    users.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
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
    const data = await apiClient<JsonRecord>("/api/messages/templates")
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selectedTemplate.value || !templates.value.some((item) => pathOf(item) === pathOf(selectedTemplate.value))) {
      selectedTemplate.value = templates.value[0] || null
    }
  } catch (err) {
    console.error(err)
    toast.error("站内信模板加载失败")
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
    toast.error("模板详情加载失败")
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
      selectedMessage.value = messages.value[0] || null
    }
  } catch (err) {
    console.error(err)
    toast.error("发送记录加载失败")
  } finally {
    messagesLoading.value = false
  }
}

async function sendMessage() {
  if (!selectedUserIds.value.length) {
    toast.error("请先选择收件用户")
    return
  }
  if (!templatePath.value) {
    toast.error("请先选择站内信模板")
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
    toast.success(`站内信已发送${data.count ? `：${data.count} 条` : ""}`)
    selectedUserIds.value = []
    templatePath.value = ""
    payload.value = "{\n}"
  } catch (err) {
    console.error(err)
    toast.error("站内信发送失败")
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
    toast.error("模板详情加载失败")
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
    toast.error("请填写模板路径、标题和内容")
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
        <h1 class="text-4xl font-black tracking-tight">站内信</h1>
        <p class="mt-2 text-slate-600">发送站内通知、维护模板并查看发送记录。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：send/list sent/list templates/get template/create template/update template。delete 路由未实现，页面不提供删除按钮。
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
          <h2 class="text-xl font-black">新建站内信</h2>
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
                  <span class="font-semibold">{{ userLabel(user) }}</span>
                  <span class="break-all text-xs text-slate-400">{{ userId(user) }}</span>
                </label>
                <div v-if="!users.length" class="p-6 text-center text-slate-500">暂无用户</div>
              </div>
              <p class="mt-2 text-sm text-slate-500">已选择 {{ selectedUserIds.length }} 个用户。</p>
            </div>

            <div class="space-y-4">
              <label class="block">
                <span class="font-bold">模板</span>
                <select v-model="templatePath" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                  <option value="">请选择模板</option>
                  <option v-for="template in templates" :key="pathOf(template)" :value="pathOf(template)">
                    {{ titleOf(template) }} ({{ pathOf(template) }})
                  </option>
                </select>
              </label>

              <label class="block">
                <span class="font-bold">消息类型</span>
                <select v-model.number="msgType" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                  <option :value="1">系统通知</option>
                  <option :value="2">公告</option>
                  <option :value="3">营销</option>
                  <option :value="4">支付</option>
                  <option :value="5">其他</option>
                </select>
              </label>

              <label class="block">
                <span class="font-bold">Payload JSON</span>
                <textarea v-model="payload" class="mt-2 min-h-52 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" />
              </label>

              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="sending" type="button" @click="sendMessage">
                <Loader2 v-if="sending" class="h-4 w-4 animate-spin" />
                <Send v-else class="h-4 w-4" />
                {{ sending ? "发送中..." : "发送站内信" }}
              </button>
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'sent'" class="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
          <div class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
            <div class="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">发送记录</h2>
                <p class="mt-1 text-sm text-slate-500">每页 {{ pageSize }} 条，总计 {{ total }} 条。</p>
              </div>
              <select v-model="statusFilter" class="rounded-xl border border-slate-200 px-4 py-2">
                <option value="">全部状态</option>
                <option value="1">未读</option>
                <option value="2">已读</option>
              </select>
            </div>
            <div v-if="messagesLoading" class="p-12 text-center text-slate-500">
              <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
              正在加载...
            </div>
            <button
              v-for="message in messages"
              v-else
              :key="messageId(message)"
              class="block w-full border-b border-slate-100 p-5 text-left last:border-b-0 hover:bg-sky-50"
              :class="messageId(selectedMessage) === messageId(message) ? 'bg-sky-50' : ''"
              type="button"
              @click="selectedMessage = message"
            >
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate font-black">{{ pickFirst(message, ["title", "subject", "template_path", "message_id"]) || "站内信" }}</div>
                  <div class="mt-1 break-all text-sm text-slate-500">{{ pickFirst(message, ["user_name", "user_id", "candidate_ulid"]) || "-" }}</div>
                </div>
                <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(messageStatus(message))">{{ messageStatus(message) }}</span>
              </div>
              <div class="mt-2 text-sm text-slate-500">{{ formatDate(String(pickFirst(message, ["created_at", "sent_at", "updated_at"]) || "")) }}</div>
            </button>
            <div v-if="!messagesLoading && !messages.length" class="p-12 text-center text-slate-500">暂无发送记录</div>
            <div class="flex items-center justify-end gap-3 border-t border-slate-200 p-5">
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="messagePage <= 1" @click="messagePage--">上一页</button>
              <span class="text-sm font-bold">{{ messagePage }} / {{ totalPages }}</span>
              <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="messagePage >= totalPages" @click="messagePage++">下一页</button>
            </div>
          </div>

          <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
            <h2 class="text-xl font-black">站内信详情</h2>
            <div v-if="!selectedMessage" class="p-10 text-center text-slate-500">请选择一条发送记录</div>
            <div v-else class="mt-4 space-y-5">
              <div class="grid gap-4 md:grid-cols-2">
                <label v-for="(value, key) in selectedMessageFields" :key="key" class="grid gap-2 text-sm font-bold">
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
              <pre class="max-h-[420px] overflow-auto rounded-2xl bg-slate-950 p-4 text-xs text-slate-100">{{ JSON.stringify(selectedMessage, null, 2) }}</pre>
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
              <div class="font-black">{{ titleOf(template) }}</div>
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
                <span class="text-sm font-bold">标题模板</span>
                <input v-model="formTitle" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
              </label>
              <label class="mt-4 block">
                <span class="text-sm font-bold">内容模板</span>
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
