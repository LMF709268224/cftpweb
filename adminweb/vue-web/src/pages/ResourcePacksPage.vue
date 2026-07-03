<script setup lang="ts">
import { FileBox, Loader2, Plus, RefreshCw, Save, Trash2, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, humanizeKey, type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"

const pageSize = 10

const packs = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const saving = ref(false)
const detailOpen = ref(false)
const mode = ref<"create" | "edit">("edit")
const pageToken = ref("")
const nextPageToken = ref("")
const previousTokens = ref<string[]>([])
const currentPage = ref(1)
let detailRequestId = 0
const { t } = useAdminLanguage()
const copy = computed(() => t.value.resourcePacksAdmin)

const form = ref({
  pack_id: "",
  title: "",
  description: "",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
  respath: "",
  status: "Active",
  version: 0,
  icon: "",
  category: "",
})

const selectedEntries = computed(() => Object.entries(selected.value || {}))
const canPrevious = computed(() => previousTokens.value.length > 0)
const canNext = computed(() => !!nextPageToken.value)

function asRecordList(value: unknown) {
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function packId(pack: JsonRecord | null) {
  return String(pack?.pack_id || "")
}

function packTitle(pack: JsonRecord | null) {
  return String(pack?.title || pack?.pack_id || copy.value.unnamed)
}

function packVersion(pack: JsonRecord | null) {
  return Number(pack?.version || 0)
}

function fillForm(pack: JsonRecord | null) {
  form.value = {
    pack_id: String(pack?.pack_id || ""),
    title: String(pack?.title || ""),
    description: String(pack?.description || ""),
    thumbnail_object_key: String(pack?.thumbnail_object_key || ""),
    thumbnail_file_hash: String(pack?.thumbnail_file_hash || ""),
    respath: String(pack?.respath || ""),
    status: String(pack?.status || "Active"),
    version: Number(pack?.version || 0),
    icon: String(pack?.icon || ""),
    category: String(pack?.category || ""),
  }
}

function mergePackDetail(id: string, detail: JsonRecord) {
  const index = packs.value.findIndex((pack) => packId(pack) === id)
  const base = index >= 0 ? packs.value[index] : selected.value || {}
  const merged = { ...base, ...detail }
  if (index >= 0) {
    packs.value.splice(index, 1, merged)
  }
  if (selected.value && packId(selected.value) === id) {
    selected.value = merged
    fillForm(merged)
  }
}

async function loadPackDetail(pack: JsonRecord | null) {
  if (!pack) return
  const id = packId(pack)
  if (!id) return

  const requestId = ++detailRequestId
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/lms/resource-packs/${encodeURIComponent(id)}`)
    if (requestId !== detailRequestId) return
    if (detail && typeof detail === "object" && !Array.isArray(detail)) {
      mergePackDetail(id, detail)
    }
  } catch (err) {
    console.error(err)
    if (requestId === detailRequestId) {
      toast.error(copy.value.toasts.detailLoadFailed)
    }
  } finally {
    if (requestId === detailRequestId) {
      detailLoading.value = false
    }
  }
}

async function load() {
  loading.value = true
  try {
    const url = new URL("/api/lms/resource-packs", window.location.origin)
    url.searchParams.set("page_size", String(pageSize))
    if (pageToken.value) url.searchParams.set("page_token", pageToken.value)

    const data = await apiClient<JsonRecord>(`${url.pathname}${url.search}`)
    packs.value = asRecordList(data.packs || data.items)
    nextPageToken.value = String(data.next_page_token || "")
    const nextSelected = packs.value.find((pack) => packId(pack) === form.value.pack_id) || packs.value[0] || null
    selectPack(nextSelected)
  } catch (err) {
    console.error(err)
    packs.value = []
    selected.value = null
    fillForm(null)
    toast.error(copy.value.toasts.listLoadFailed)
  } finally {
    loading.value = false
  }
}

function selectPack(pack: JsonRecord | null, openDetail = false) {
  selected.value = pack
  mode.value = pack ? "edit" : "create"
  fillForm(pack)
  void loadPackDetail(pack)
  if (openDetail) detailOpen.value = true
}

function openPackDetail(pack: JsonRecord) {
  selectPack(pack, true)
}

function closePackDetail() {
  detailOpen.value = false
}

function startCreate() {
  detailRequestId += 1
  detailLoading.value = false
  selected.value = null
  mode.value = "create"
  fillForm(null)
  detailOpen.value = true
}

async function savePack() {
  if (!form.value.title.trim()) {
    toast.error(copy.value.toasts.titleRequired)
    return
  }
  if (mode.value === "edit" && (!form.value.pack_id || form.value.version <= 0)) {
    toast.error(copy.value.toasts.updateRequiresVersion)
    return
  }

  saving.value = true
  try {
    const body: JsonRecord = {
      title: form.value.title.trim(),
      description: form.value.description,
      thumbnail_object_key: form.value.thumbnail_object_key,
      thumbnail_file_hash: form.value.thumbnail_file_hash,
      respath: form.value.respath,
      icon: form.value.icon,
      category: form.value.category,
    }

    if (mode.value === "create") {
      if (form.value.pack_id.trim()) body.pack_id = form.value.pack_id.trim()
      await apiClient("/api/lms/resource-packs", {
        method: "POST",
        body: JSON.stringify(body),
      })
      toast.success(copy.value.toasts.created)
    } else {
      body.status = form.value.status
      body.version = form.value.version
      await apiClient(`/api/lms/resource-packs/${encodeURIComponent(form.value.pack_id)}`, {
        method: "PUT",
        body: JSON.stringify(body),
      })
      toast.success(copy.value.toasts.saved)
    }

    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.saveFailed)
  } finally {
    saving.value = false
  }
}

async function deletePack() {
  if (!selected.value) return
  const id = packId(selected.value)
  const version = packVersion(selected.value)
  if (!id || version <= 0) {
    toast.error(copy.value.toasts.deleteRequiresVersion)
    return
  }
  if (!window.confirm(copy.value.confirmDelete(packTitle(selected.value)))) return

  saving.value = true
  try {
    await apiClient(`/api/lms/resource-packs/${encodeURIComponent(id)}?version=${version}`, {
      method: "DELETE",
    })
    toast.success(copy.value.toasts.deleted)
    await load()
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.deleteFailed)
  } finally {
    saving.value = false
  }
}

function previousPage() {
  if (!canPrevious.value) return
  pageToken.value = previousTokens.value.pop() || ""
  currentPage.value = Math.max(1, currentPage.value - 1)
  void load()
}

function nextPage() {
  if (!canNext.value) return
  previousTokens.value.push(pageToken.value)
  pageToken.value = nextPageToken.value
  currentPage.value += 1
  void load()
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight text-slate-950">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" :disabled="loading" @click="load">
          <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
          <RefreshCw v-else class="h-4 w-4" />
          {{ copy.refresh }}
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          {{ copy.newPack }}
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
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-500">{{ copy.pageSizeText(packs.length, pageSize) }}</span>
        </div>

        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else-if="!packs.length" class="p-12 text-center text-slate-500">{{ copy.empty }}</div>
        <div v-else>
          <div class="hidden grid-cols-[minmax(0,1fr)_84px_96px_110px] gap-4 border-b border-slate-100 bg-slate-50 px-5 py-3 text-xs font-black uppercase tracking-wide text-slate-400 lg:grid">
            <span>{{ copy.columns.pack }}</span>
            <span>{{ copy.columns.version }}</span>
            <span class="text-right">{{ copy.columns.status }}</span>
            <span class="text-right">{{ copy.columns.action }}</span>
          </div>
          <button
            v-for="pack in packs"
            :key="packId(pack)"
            class="block w-full border-b border-slate-100 px-5 py-3 text-left transition last:border-b-0 hover:bg-slate-50"
            :class="packId(selected) === packId(pack) ? 'bg-sky-50/70' : ''"
            type="button"
            @click="openPackDetail(pack)"
          >
            <div class="grid gap-3 lg:grid-cols-[minmax(0,1fr)_84px_96px_110px] lg:items-center lg:gap-4">
              <div class="min-w-0">
                <div class="truncate text-base font-black text-slate-950">{{ packTitle(pack) }}</div>
                <div class="mt-1 line-clamp-1 text-sm text-slate-500">{{ pack.description || "-" }}</div>
                <div class="mt-1 truncate font-mono text-xs text-slate-500">{{ copy.fields.idPrefix }}{{ pack.pack_id }}</div>
              </div>
              <div class="text-sm font-bold text-slate-700">
                <span class="mr-2 text-xs font-bold text-slate-400 lg:hidden">{{ copy.fields.version }}</span>{{ pack.version || 0 }}
              </div>
              <span class="justify-self-start rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-black text-emerald-700 lg:justify-self-end">
                {{ pack.status || "Active" }}
              </span>
              <span class="inline-flex h-9 items-center justify-self-start rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-blue-700 shadow-sm transition hover:border-blue-200 hover:bg-blue-50 lg:justify-self-end">{{ copy.viewDetails }}</span>
            </div>
          </button>
        </div>

        <div class="flex items-center justify-end gap-3 border-t border-slate-200 p-5">
          <span class="mr-auto text-sm font-bold text-slate-500">{{ copy.pageText(currentPage) }}</span>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrevious || loading" @click="previousPage">{{ copy.prev }}</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext || loading" @click="nextPage">{{ copy.next }}</button>
        </div>
      </section>

      <section v-if="detailOpen" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <div class="flex max-h-[88vh] w-full max-w-[1100px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div class="flex items-start justify-between gap-4 border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ mode === "create" ? copy.createTitle : copy.detailTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.readonlyHint }}</p>
            <p v-if="detailLoading" class="mt-1 inline-flex items-center gap-2 text-xs font-bold text-blue-600">
              <Loader2 class="h-3.5 w-3.5 animate-spin" />
              {{ copy.detailLoading }}
            </p>
          </div>
          <button class="inline-flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="copy.close" @click="closePackDetail">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="flex-1 space-y-5 overflow-y-auto p-5">
          <div>
            <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.basic }}</div>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="text-sm font-bold">
                {{ copy.fields.title }}
                <input v-model="form.title" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.title" />
              </label>
              <label class="text-sm font-bold">
                {{ copy.fields.category }}
                <input v-model="form.category" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.category" />
              </label>
              <label class="text-sm font-bold">
                {{ copy.fields.icon }}
                <input v-model="form.icon" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.icon" />
              </label>
              <label class="md:col-span-2 text-sm font-bold">
                {{ copy.fields.description }}
                <textarea v-model="form.description" class="mt-2 min-h-24 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.description" />
              </label>
            </div>
          </div>

          <div class="border-t border-slate-100 pt-5">
            <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.pathThumbnail }}</div>
            <div class="grid gap-4 md:grid-cols-2">
              <label class="text-sm font-bold">
                {{ copy.fields.respath }}
                <input v-model="form.respath" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.respath" />
              </label>
              <label class="text-sm font-bold">
                {{ copy.fields.thumbnailObjectKey }}
                <input v-model="form.thumbnail_object_key" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.thumbnailObjectKey" />
              </label>
              <label class="md:col-span-2 text-sm font-bold">
                {{ copy.fields.thumbnailFileHash }}
                <input v-model="form.thumbnail_file_hash" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :placeholder="copy.placeholders.thumbnailFileHash" />
              </label>
            </div>
          </div>

          <div class="border-t border-slate-100 pt-5">
            <div class="mb-3 text-sm font-black text-slate-950">{{ copy.sections.system }}</div>
            <div class="grid gap-4 md:grid-cols-3">
              <label class="text-sm font-bold">
                {{ copy.fields.packId }}
                <input v-model="form.pack_id" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2 disabled:bg-slate-100" :disabled="mode === 'edit'" :placeholder="copy.placeholders.packId" />
              </label>
              <label class="text-sm font-bold">
                {{ copy.fields.status }}
                <input v-model="form.status" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2" :disabled="mode === 'create'" :placeholder="copy.placeholders.status" />
              </label>
              <label class="text-sm font-bold">
                {{ copy.fields.version }}
                <input v-model.number="form.version" class="mt-2 w-full rounded-xl border border-slate-200 px-3 py-2 disabled:bg-slate-100" type="number" min="0" :disabled="mode === 'create'" />
              </label>
            </div>
          </div>

          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 pt-5">
            <button
              v-if="mode === 'edit'"
              class="inline-flex h-10 items-center gap-2 rounded-xl border border-red-200 bg-red-50 px-4 text-sm font-bold text-red-700 disabled:opacity-50"
              type="button"
              :disabled="saving"
              @click="deletePack"
            >
              <Trash2 class="h-4 w-4" />
              {{ copy.deletePack }}
            </button>
            <span v-else class="hidden sm:block"></span>
            <button class="inline-flex h-10 min-w-[180px] items-center justify-center gap-2 rounded-xl bg-blue-700 px-4 font-bold text-white disabled:opacity-50" type="button" :disabled="saving" @click="savePack">
              <Loader2 v-if="saving" class="h-4 w-4 animate-spin" />
              <Save v-else class="h-4 w-4" />
              {{ mode === "create" ? copy.createPack : copy.savePack }}
            </button>
          </div>

          <div v-if="selected" class="rounded-2xl border border-slate-200 bg-slate-50">
            <div class="flex items-center gap-2 border-b border-slate-200 px-4 py-3 text-sm font-black">
              <FileBox class="h-4 w-4 text-blue-700" />
              {{ copy.sections.raw }}
            </div>
            <div class="divide-y divide-slate-200 px-4">
              <div v-for="[key, value] in selectedEntries" :key="key" class="grid gap-2 py-2.5 text-sm md:grid-cols-[170px_1fr]">
                <div class="text-[11px] font-black uppercase text-slate-400">{{ humanizeKey(key) }}</div>
                <div v-if="typeof value === 'string' && key.endsWith('_at')" class="break-words font-semibold text-slate-700">{{ formatDate(value) }}</div>
                <div v-else class="break-words font-semibold text-slate-700">{{ value ?? "-" }}</div>
              </div>
            </div>
          </div>
        </div>
        </div>
      </section>
    </div>
  </section>
</template>
