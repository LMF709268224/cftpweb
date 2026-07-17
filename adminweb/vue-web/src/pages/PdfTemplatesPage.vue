<script setup lang="ts">
import { FileText, Loader2, Plus, RefreshCw, Save, X } from "lucide-vue-next"
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

const hiddenRawFieldKeys = new Set(["detail", "summary", "template_id", "template_ulid", "name", "description", "html_template"])
const selectedFields = computed(() => {
  const entries = Object.entries(selected.value || {}).filter(([key]) => !hiddenRawFieldKeys.has(key))
  return Object.fromEntries(entries)
})
const previewHtml = computed(() => form.value.html_template || `<p style="color:#64748b">${copy.value.previewEmpty}</p>`)
const readonlyMode = computed(() => mode.value === "detail")

let detailRequestController: AbortController | null = null
let detailRequestSeq = 0

function fieldLabel(key: string) {
  return copy.value.fieldLabels?.[key as keyof typeof copy.value.fieldLabels] || key.replaceAll("_", " ")
}

function fieldValue(key: string, value: unknown) {
  return key.endsWith("_at") ? formatDate(value) : value
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

function isAbortError(err: unknown) {
  return Boolean(err && typeof err === "object" && "name" in err && err.name === "AbortError")
}

function cancelTemplateDetailRequest() {
  detailRequestSeq += 1
  detailRequestController?.abort()
  detailRequestController = null
  detailLoading.value = false
}

async function loadTemplateDetail(template: JsonRecord, signal?: AbortSignal) {
  const id = templateUlid(template)
  if (!id) return template
  const detail = await apiClient<JsonRecord>(`/api/pdf-templates/detail?template_id=${encodeURIComponent(id)}`, { signal })
  return { ...template, ...detail }
}

function openCreate() {
  cancelTemplateDetailRequest()
  mode.value = "create"
  selected.value = null
  form.value = { template_id: "", name: "", description: "", html_template: "" }
  dialogOpen.value = true
}

async function openTemplate(template: JsonRecord, nextMode: Exclude<Mode, "create">) {
  cancelTemplateDetailRequest()
  const requestSeq = detailRequestSeq
  const controller = new AbortController()
  detailRequestController = controller
  selected.value = template
  mode.value = nextMode
  form.value = formFromTemplate(template)
  dialogOpen.value = true
  detailLoading.value = true
  try {
    const detail = await loadTemplateDetail(template, controller.signal)
    if (requestSeq !== detailRequestSeq || controller.signal.aborted) return
    selected.value = detail
    form.value = formFromTemplate(selected.value)
  } catch (err) {
    if (isAbortError(err) || requestSeq !== detailRequestSeq) return
    console.error(err)
    toast.error(copy.value.toasts.loadFailed)
  } finally {
    if (requestSeq === detailRequestSeq) {
      detailLoading.value = false
      if (detailRequestController === controller) {
        detailRequestController = null
      }
    }
  }
}

function openTemplateDetail(template: JsonRecord) {
  void openTemplate(template, "detail")
}

function openTemplateEditor(template: JsonRecord) {
  void openTemplate(template, "edit")
}

function closeDialog() {
  if (saving.value) return
  if (detailLoading.value) {
    cancelTemplateDetailRequest()
  }
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
  if (!form.value.html_template.trim()) {
    toast.error(copy.value.toasts.htmlTemplateRequired)
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
  <main class="ui-page">
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button class="ui-btn ui-btn-secondary" type="button" @click="load">
          <RefreshCw class="h-4 w-4 text-slate-400" :class="loading ? 'animate-spin' : ''" />
          {{ copy.refresh }}
        </button>
        <button class="ui-btn ui-btn-primary" type="button" @click="openCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newTemplate }}
        </button>
      </div>
    </header>

    <section class="ui-card mt-6 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
      </div>
      <div v-if="loading" class="divide-y divide-hairline">
        <div v-for="i in 5" :key="`sk-${i}`" class="grid gap-3 px-4 py-3.5 lg:grid-cols-[minmax(0,1fr)_96px_150px_150px] lg:items-center lg:gap-6">
          <div class="space-y-2">
            <div class="skeleton h-4 w-2/5" />
            <div class="skeleton h-3 w-4/5" />
            <div class="skeleton h-3 w-3/5" />
          </div>
          <div class="skeleton h-4 w-10" />
          <div class="skeleton h-4 w-20" />
          <div class="skeleton h-8 w-24 rounded-lg lg:justify-self-center" />
        </div>
      </div>
      <div v-else-if="!templates.length" class="ui-empty">
        <div class="ui-empty-icon">
          <FileText class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.empty }}</p>
      </div>
      <div v-else>
        <div class="hidden grid-cols-[minmax(0,1fr)_96px_150px_150px] gap-6 border-b border-edge bg-slate-50/60 px-4 py-3 text-xs font-medium uppercase tracking-wider text-slate-400 lg:grid">
          <span>{{ copy.columns.template }}</span>
          <span>{{ copy.columns.version }}</span>
          <span>{{ copy.columns.createdAt }}</span>
          <span class="text-center">{{ copy.columns.action }}</span>
        </div>
        <div class="divide-y divide-hairline">
          <div
            v-for="template in templates"
            :key="templateUlid(template)"
            class="ui-table-row grid w-full gap-3 px-4 py-3.5 text-left lg:grid-cols-[minmax(0,1fr)_96px_150px_150px] lg:items-center lg:gap-6"
            :class="templateUlid(selected) === templateUlid(template) ? 'bg-primary-50/60' : ''"
          >
            <div class="min-w-0">
              <div class="break-words text-sm font-medium text-ink lg:truncate">{{ templateName(template) }}</div>
              <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ template.description || copy.noDescription }}</div>
              <div class="tnum mt-1 break-all text-xs text-slate-400 lg:truncate">ID: {{ templateUlid(template) || "-" }}</div>
            </div>
            <div class="flex items-center justify-between gap-4 lg:block">
              <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.columns.version }}</span>
              <span class="tnum text-sm text-slate-600">v{{ template.version || 1 }}</span>
            </div>
            <div class="flex items-center justify-between gap-4 lg:block">
              <span class="text-xs font-medium text-slate-400 lg:hidden">{{ copy.columns.createdAt }}</span>
              <span class="tnum text-sm text-slate-600">{{ formatDate(String(template.created_at || "")) }}</span>
            </div>
            <div class="grid grid-cols-2 gap-3 sm:flex sm:items-center sm:justify-start lg:justify-center">
              <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click="openTemplateDetail(template)">
                {{ copy.viewDetails }}
              </button>
              <button class="ui-btn ui-btn-secondary ui-btn-sm" type="button" @click="openTemplateEditor(template)">
                {{ copy.edit }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="dialogOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1100px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-ink md:text-lg">
                  {{ mode === "create" ? copy.newTemplate : mode === "edit" ? copy.editTitle : copy.detailTitle }}
                </h2>
                <p class="tnum mt-1 break-all text-sm text-slate-500">{{ mode === "create" ? copy.createDescription : templateUlid(selected) }}</p>
              </div>
              <button class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 disabled:cursor-not-allowed disabled:opacity-50" type="button" :aria-label="copy.close" :disabled="saving" @click="closeDialog">
                <X class="h-5 w-5" />
              </button>
            </div>
            <div class="relative min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <div v-if="detailLoading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/80 backdrop-blur-[2px]">
                <div class="ui-card flex flex-col items-center gap-3 px-6 py-5 text-sm font-medium text-slate-600">
                  <Loader2 class="h-8 w-8 animate-spin text-primary-600" />
                  {{ copy.loading }}
                </div>
              </div>
              <div class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                <form id="pdf-template-form" class="space-y-4" @submit.prevent="save">
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.templateId" :value="form.template_id" />
                  <label v-else-if="mode !== 'create'" class="grid gap-1.5">
                    <span class="ui-label">{{ copy.fields.templateId }}</span>
                    <input v-model="form.template_id" disabled class="ui-input tnum" />
                  </label>
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.name" :value="form.name" />
                  <label v-else class="grid gap-1.5">
                    <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.name }}</span>
                    <input v-model="form.name" class="ui-input" maxlength="160" required />
                  </label>
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.description" :value="form.description" />
                  <label v-else class="grid gap-1.5">
                    <span class="ui-label">{{ copy.fields.description }}</span>
                    <input v-model="form.description" class="ui-input" maxlength="500" />
                  </label>
                  <ReadonlyField v-if="readonlyMode" :label="copy.fields.htmlTemplate" :value="form.html_template" mono min-height="460px" max-height="460px" />
                  <label v-else class="grid gap-1.5">
                    <span class="ui-label"><span class="mr-1 text-red-500" aria-hidden="true">*</span>{{ copy.fields.htmlTemplate }}</span>
                    <textarea v-model="form.html_template" class="ui-input min-h-80 py-2.5 font-mono leading-6 md:min-h-[460px]" required />
                  </label>
                </form>

                <div class="space-y-4">
                  <div v-if="mode === 'detail'" class="rounded-2xl bg-slate-50 p-4">
                    <h3 class="text-sm font-semibold text-ink">{{ copy.rawFields }}</h3>
                    <p class="mt-1 text-sm text-slate-500">{{ copy.rawHint }}</p>
                    <div class="mt-4 grid gap-3">
                      <ReadonlyField
                        v-for="(value, key) in selectedFields"
                        :key="key"
                        :label="fieldLabel(String(key))"
                        :value="fieldValue(String(key), value)"
                        :mono="Array.isArray(value) || (!!value && typeof value === 'object')"
                        :max-height="Array.isArray(value) || (!!value && typeof value === 'object') ? '180px' : undefined"
                      />
                    </div>
                  </div>
                  <div class="rounded-2xl border border-edge p-4">
                    <h3 class="text-sm font-semibold text-ink">{{ copy.preview }}</h3>
                    <iframe class="mt-4 h-80 w-full rounded-[10px] border border-edge bg-white md:h-[520px]" sandbox="allow-same-origin" :srcdoc="previewHtml" />
                  </div>
                </div>
              </div>
            </div>
            <div v-if="!readonlyMode" class="flex shrink-0 flex-col justify-end gap-3 border-t border-hairline bg-white px-4 py-4 sm:flex-row md:px-6">
              <button class="ui-btn ui-btn-primary w-full sm:w-auto sm:min-w-[180px]" type="submit" form="pdf-template-form" :disabled="saving || detailLoading">
                <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
                <Save v-else class="h-4 w-4" />
                {{ saving ? copy.saving : copy.saveTemplate }}
              </button>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>

