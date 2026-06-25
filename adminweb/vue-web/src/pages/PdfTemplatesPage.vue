<script setup lang="ts">
import { Edit, Loader2, Plus, RefreshCw } from "lucide-vue-next"
import { onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { pickFirst } from "@/lib/status"

const templates = ref<JsonRecord[]>([])
const loading = ref(false)
const saving = ref(false)
const showEditor = ref(false)
const editing = ref(false)
const form = ref({
  template_id: "",
  name: "",
  description: "",
  html_template: "",
})

function templateUlid(template: JsonRecord) {
  return String(pickFirst(template, ["template_ulid", "template_id", "id"]) || "")
}

function openCreate() {
  editing.value = false
  form.value = { template_id: "", name: "", description: "", html_template: "" }
  showEditor.value = true
}

function openEdit(template: JsonRecord) {
  editing.value = true
  form.value = {
    template_id: templateUlid(template),
    name: String(template.name || ""),
    description: String(template.description || ""),
    html_template: String(template.html_template || ""),
  }
  showEditor.value = true
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pdf-templates")
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
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
      method: editing.value ? "PUT" : "POST",
      body: JSON.stringify(editing.value ? form.value : {
        name: form.value.name,
        description: form.value.description,
        html_template: form.value.html_template,
      }),
    })
    toast.success("模板已保存")
    showEditor.value = false
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
    <header class="flex items-start justify-between gap-4">
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

    <div v-if="loading" class="rounded-3xl border border-slate-200 bg-white p-12 text-center text-slate-500">
      <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
      正在加载...
    </div>
    <div v-else class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
      <article v-for="template in templates" :key="templateUlid(template)" class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div class="mb-4 flex items-start justify-between gap-4">
          <div>
            <h2 class="text-xl font-black">{{ template.name || "未命名模板" }}</h2>
            <p class="mt-2 line-clamp-2 text-sm text-slate-500">{{ template.description || "暂无描述" }}</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">v{{ template.version || 1 }}</span>
        </div>
        <div class="mb-4 text-xs text-slate-500">创建时间：{{ formatDate(String(template.created_at || "")) }}</div>
        <button class="inline-flex w-full items-center justify-center gap-2 rounded-xl border px-4 py-3 font-bold hover:bg-slate-50" type="button" @click="openEdit(template)">
          <Edit class="h-4 w-4" />
          编辑模板
        </button>
      </article>
      <div v-if="!templates.length" class="col-span-full rounded-3xl border border-dashed border-slate-200 bg-white p-12 text-center text-slate-500">暂无模板</div>
    </div>

    <div v-if="showEditor" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="flex max-h-[88vh] w-full max-w-5xl flex-col rounded-3xl bg-white p-6 shadow-2xl">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-2xl font-black">{{ editing ? "编辑模板" : "新建模板" }}</h2>
          <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="showEditor = false">关闭</button>
        </div>
        <div class="grid flex-1 gap-4 overflow-y-auto">
          <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="模板名称" />
          <input v-model="form.description" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="描述" />
          <textarea v-model="form.html_template" class="min-h-[420px] rounded-xl border border-slate-200 p-4 font-mono text-sm" placeholder="HTML 模板内容" />
        </div>
        <div class="mt-5 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold" type="button" @click="showEditor = false">取消</button>
          <button class="rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="save">
            {{ saving ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
