<script setup lang="ts">
import { Loader2, Plus, RefreshCw, Save } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { pickFirst } from "@/lib/status"

type Mode = "detail" | "create"

const templates = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const saving = ref(false)
const mode = ref<Mode>("detail")
const form = ref({
  template_id: "",
  name: "",
  description: "",
  html_template: "",
})

const selectedFields = computed(() => selected.value || {})

function templateUlid(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["template_ulid", "template_id", "id"]) || "")
}

function templateName(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["name", "title"]) || "未命名模板")
}

function formFromTemplate(template: JsonRecord | null) {
  return {
    template_id: templateUlid(template),
    name: String(template?.name || ""),
    description: String(template?.description || ""),
    html_template: String(template?.html_template || ""),
  }
}

function openCreate() {
  mode.value = "create"
  selected.value = null
  form.value = { template_id: "", name: "", description: "", html_template: "" }
}

function openTemplate(template: JsonRecord) {
  selected.value = template
  mode.value = "detail"
  form.value = formFromTemplate(template)
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pdf-templates")
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selected.value || !templates.value.some((item) => templateUlid(item) === templateUlid(selected.value))) {
      selected.value = templates.value[0] || null
      mode.value = selected.value ? "detail" : "create"
      form.value = formFromTemplate(selected.value)
    }
  } catch (err) {
    console.error(err)
    toast.error("PDF 模板加载失败")
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!form.value.name.trim()) {
    toast.error("模板名称必填")
    return
  }
  saving.value = true
  try {
    await apiClient("/api/pdf-templates", {
      method: mode.value === "detail" ? "PUT" : "POST",
      body: JSON.stringify(mode.value === "detail" ? form.value : {
        name: form.value.name,
        description: form.value.description,
        html_template: form.value.html_template,
      }),
    })
    toast.success("模板已保存")
    await load()
  } catch (err) {
    console.error(err)
    toast.error("保存失败")
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">PDF 模板配置</h1>
        <p class="mt-2 text-slate-600">维护证书和证明文件的 HTML 模板。</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="openCreate">
          <Plus class="h-4 w-4" />
          新建模板
        </button>
      </div>
    </header>

    <div class="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">模板列表</h2>
            <p class="mt-1 text-sm text-slate-500">选择模板后查看或编辑。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ templates.length }}</span>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <button
          v-for="template in templates"
          v-else
          :key="templateUlid(template)"
          class="w-full border-b border-slate-100 px-5 py-5 text-left last:border-b-0 hover:bg-sky-50"
          :class="mode === 'detail' && templateUlid(selected) === templateUlid(template) ? 'bg-sky-50' : ''"
          type="button"
          @click="openTemplate(template)"
        >
          <div class="flex items-start justify-between gap-4">
            <div>
              <div class="text-lg font-black">{{ templateName(template) }}</div>
              <div class="mt-2 line-clamp-2 text-sm text-slate-500">{{ template.description || "暂无描述" }}</div>
            </div>
            <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">v{{ template.version || 1 }}</span>
          </div>
          <div class="mt-3 break-all text-xs font-semibold text-slate-500">ID: {{ templateUlid(template) || "-" }}</div>
          <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(template.created_at || "")) }}</div>
        </button>
        <div v-if="!loading && !templates.length" class="p-12 text-center text-slate-500">暂无模板</div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-200 p-5">
          <h2 class="text-2xl font-black">{{ mode === "create" ? "新建模板" : templateName(selected) }}</h2>
          <p class="mt-1 break-all text-sm text-slate-500">{{ mode === "create" ? "保存后会创建新的 PDF 模板。" : templateUlid(selected) }}</p>
        </div>
        <div class="grid gap-6 p-5 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
          <form class="space-y-4" @submit.prevent="save">
            <label v-if="mode === 'detail'" class="grid gap-2 text-sm font-bold">
              Template ID
              <input v-model="form.template_id" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600" />
            </label>
            <label class="grid gap-2 text-sm font-bold">
              模板名称
              <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" />
            </label>
            <label class="grid gap-2 text-sm font-bold">
              描述
              <input v-model="form.description" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="500" />
            </label>
            <label class="grid gap-2 text-sm font-bold">
              HTML 模板
              <textarea v-model="form.html_template" class="min-h-[460px] rounded-xl border border-slate-200 p-4 font-mono text-sm leading-6" />
            </label>
            <div class="flex justify-end">
              <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="submit" :disabled="saving">
                <Save class="h-4 w-4" />
                {{ saving ? "保存中..." : "保存模板" }}
              </button>
            </div>
          </form>

          <div class="space-y-4">
            <div v-if="mode === 'detail'" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
              <h3 class="font-black">完整字段</h3>
              <p class="mt-1 text-sm text-slate-500">系统字段仅用于查看。</p>
              <div class="mt-4 grid gap-3">
                <label v-for="(value, key) in selectedFields" :key="key" class="grid gap-2 text-sm font-bold">
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
            <div class="rounded-2xl border border-slate-200 bg-white p-4">
              <h3 class="font-black">HTML 预览</h3>
              <iframe class="mt-4 h-[520px] w-full rounded-xl border border-slate-200 bg-white" sandbox="allow-same-origin" :srcdoc="form.html_template || '<p style=&quot;color:#64748b&quot;>暂无模板内容</p>'" />
            </div>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>
