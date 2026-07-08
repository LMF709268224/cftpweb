<script setup lang="ts">
import { Loader2, Plus, RefreshCw, Save, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import ReadonlyField from "@/components/ReadonlyField.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { pickFirst } from "@/lib/status"

type Mode = "detail" | "edit" | "create"

const templates = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const saving = ref(false)
const detailLoading = ref(false)
const dialogOpen = ref(false)
const mode = ref<Mode>("detail")
const form = ref({
  template_id: "",
  name: "",
  description: "",
  html_template: "",
})
const { t } = useAdminLanguage()
const copy = computed(() => t.value.pdfTemplatesAdmin)

const selectedFields = computed(() => selected.value || {})
const previewHtml = computed(() => form.value.html_template || `<p style="color:#64748b">${copy.value.previewEmpty}</p>`)
const readonlyMode = computed(() => mode.value === "detail")

function fieldLabel(key: string) {
  return copy.value.fieldLabels?.[key as keyof typeof copy.value.fieldLabels] || key.replaceAll("_", " ")
}

function templateUlid(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["template_ulid", "template_id", "id"]) || "")
}

function templateName(template: JsonRecord | null | undefined) {
  return String(pickFirst(template || {}, ["name", "title"]) || copy.value.unnamed)
}

function formFromTemplate(template: JsonRecord | null) {
  return {
    template_id: templateUlid(template),
    name: String(template?.name || ""),
    description: String(template?.description || ""),
    html_template: String(template?.html_template || ""),
  }
}

async function loadTemplateDetail(template: JsonRecord) {
  const id = templateUlid(template)
  if (!id) return template
  const detail = await apiClient<JsonRecord>(`/api/pdf-templates/detail?template_id=${encodeURIComponent(id)}`)
  return { ...template, ...detail }
}

function openCreate() {
  mode.value = "create"
  selected.value = null
  form.value = { template_id: "", name: "", description: "", html_template: "" }
  dialogOpen.value = true
}

async function openTemplate(template: JsonRecord, nextMode: Exclude<Mode, "create">) {
  selected.value = template
  mode.value = nextMode
  form.value = formFromTemplate(template)
  dialogOpen.value = true
  detailLoading.value = true
  try {
    selected.value = await loadTemplateDetail(template)
    form.value = formFromTemplate(selected.value)
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    detailLoading.value = false
  }
}

function openTemplateDetail(template: JsonRecord) {
  void openTemplate(template, "detail")
}

function openTemplateEditor(template: JsonRecord) {
  void openTemplate(template, "edit")
}

function closeDialog() {
  if (detailLoading.value || saving.value) return
  dialogOpen.value = false
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pdf-templates")
    const list = Array.isArray(data.templates) ? data.templates : []
    templates.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (selected.value) {
      const refreshed = templates.value.find((item) => templateUlid(item) === templateUlid(selected.value))
      if (refreshed) {
        selected.value = { ...selected.value, ...refreshed }
        if (mode.value !== "create") {
          form.value = formFromTemplate(selected.value)
        }
      } else {
        selected.value = null
        if (mode.value !== "create") {
          dialogOpen.value = false
        }
      }
    }
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!form.value.name.trim()) {
    toast.error(copy.value.toasts.nameRequired)
    return
  }
  saving.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pdf-templates", {
      method: mode.value === "edit" ? "PUT" : "POST",
      body: JSON.stringify(mode.value === "edit" ? form.value : {
        name: form.value.name,
        description: form.value.description,
        html_template: form.value.html_template,
      }),
    })
    toast.success(copy.value.toasts.saved)
    const savedId = templateUlid(data) || form.value.template_id
    await load()
    if (savedId) {
      const refreshed = templates.value.find((item) => templateUlid(item) === savedId)
      selected.value = refreshed || selected.value
    }
    dialogOpen.value = false
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.saveFailed))
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
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="openCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newTemplate }}
        </button>
      </div>
    </header>

    <div class="grid gap-6">
      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ templates.length }}</span>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!templates.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
        <div v-else>
          <div class="hidden grid-cols-[minmax(0,1fr)_96px_150px_150px] gap-6 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
            <span>{{ copy.columns.template }}</span>
            <span>{{ copy.columns.version }}</span>
            <span>{{ copy.columns.createdAt }}</span>
            <span class="text-center">{{ copy.columns.action }}</span>
          </div>
          <div
            v-for="template in templates"
            :key="templateUlid(template)"
            class="grid w-full gap-3 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50 lg:grid-cols-[minmax(0,1fr)_96px_150px_150px] lg:items-center lg:gap-6"
            :class="templateUlid(selected) === templateUlid(template) ? 'bg-sky-50/70' : ''"
          >
            <div class="min-w-0">
              <div class="truncate text-lg font-black text-slate-950">{{ templateName(template) }}</div>
              <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ template.description || copy.noDescription }}</div>
              <div class="mt-1 truncate font-mono text-xs font-semibold text-slate-500">ID: {{ templateUlid(template) || "-" }}</div>
            </div>
            <div class="text-sm font-bold text-slate-700">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.columns.version }}</span>v{{ template.version || 1 }}
            </div>
            <div class="text-sm font-bold text-slate-500">
              <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.columns.createdAt }}</span>{{ formatDate(String(template.created_at || "")) }}
            </div>
            <div class="flex items-center justify-start gap-4 lg:justify-center">
              <button class="text-sm font-bold text-[#1890ff] transition hover:underline" type="button" @click="openTemplateDetail(template)">
                {{ copy.viewDetails }}
              </button>
              <button class="text-sm font-bold text-[#ffba00] transition hover:underline" type="button" @click="openTemplateEditor(template)">
                {{ copy.edit }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <Teleport to="body">
        <section v-if="dialogOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
          <div class="flex max-h-[88vh] w-full max-w-[1100px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
            <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
              <div>
                <h2 class="text-xl font-black">
                  {{ mode === "create" ? copy.newTemplate : mode === "edit" ? copy.editTitle : copy.detailTitle }}
                </h2>
                <p class="mt-1 break-all text-sm text-slate-500">{{ mode === "create" ? copy.createDescription : templateUlid(selected) }}</p>
              </div>
              <button class="rounded-full border border-slate-200 p-2 text-slate-500 shadow-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50" type="button" :aria-label="copy.close" :disabled="detailLoading || saving" @click="closeDialog">
                <X class="h-5 w-5" />
              </button>
            </div>
            <div class="relative flex-1 overflow-y-auto p-5">
              <div v-if="detailLoading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/80 backdrop-blur-[2px]">
                <div class="flex flex-col items-center gap-3 rounded-2xl border border-slate-200 bg-white px-6 py-5 text-sm font-bold text-slate-600 shadow-xl">
                  <Loader2 class="h-8 w-8 animate-spin text-blue-700" />
                  {{ copy.loading }}
                </div>
              </div>
              <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                <form id="pdf-template-form" class="space-y-4" @submit.prevent="save">
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.templateId" :value="form.template_id" />
                  <label v-else-if="mode !== 'create'" class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.templateId }}
                    <input v-model="form.template_id" disabled class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600" />
                  </label>
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.name" :value="form.name" />
                  <label v-else class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.name }}
                    <input v-model="form.name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="160" />
                  </label>
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.description" :value="form.description" />
                  <label v-else class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.description }}
                    <input v-model="form.description" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="500" />
                  </label>
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.htmlTemplate" :value="form.html_template" mono min-height="460px" max-height="460px" />
                  <label v-else class="grid gap-2 text-sm font-bold">
                    {{ copy.fields.htmlTemplate }}
                    <textarea v-model="form.html_template" class="min-h-[460px] rounded-xl border border-slate-200 p-4 font-mono text-sm leading-6" />
                  </label>
                </form>

                <div class="space-y-4">
                  <div v-if="mode === 'detail'" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                    <h3 class="font-black">{{ copy.rawFields }}</h3>
                    <p class="mt-1 text-sm text-slate-500">{{ copy.rawHint }}</p>
                    <div class="mt-4 grid gap-3">
                      <ReadonlyField
                        v-for="(value, key) in selectedFields"
                        :key="key"
                        :label="fieldLabel(String(key))"
                        :value="value"
                        :mono="Array.isArray(value) || (!!value && typeof value === 'object')"
                        :max-height="Array.isArray(value) || (!!value && typeof value === 'object') ? '180px' : undefined"
                      />
                    </div>
                  </div>
                  <div class="rounded-2xl border border-slate-200 bg-white p-4">
                    <h3 class="font-black">{{ copy.preview }}</h3>
                    <iframe class="mt-4 h-[520px] w-full rounded-xl border border-slate-200 bg-white" sandbox="allow-same-origin" :srcdoc="previewHtml" />
                  </div>
                </div>
              </div>
            </div>
            <div v-if="!readonlyMode" class="flex shrink-0 justify-end border-t border-slate-200 bg-white px-5 py-4">
              <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" type="submit" form="pdf-template-form" :disabled="saving || detailLoading">
                <Save class="h-4 w-4" />
                {{ saving ? copy.saving : copy.saveTemplate }}
              </button>
            </div>
          </div>
        </section>
      </Teleport>
    </div>
  </section>
</template>
