<script setup lang="ts">
import { FileWarning, Loader2, RefreshCw, Search, ShieldCheck, ShieldOff, UserX, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type PermissionAction = "grant" | "revoke-upload" | "mark-expired" | "revoke-credential"

const definitions = ref<JsonRecord[]>([])
const selectedDefinition = ref<JsonRecord | null>(null)
const candidateUlid = ref("")
const reason = ref("")
const checkResult = ref<JsonRecord | null>(null)
const definitionsLoading = ref(false)
const loading = ref(false)
const activeAction = ref<PermissionAction | null>(null)
const detailOpen = ref(false)
const { t, isZh } = useAdminLanguage()
const copy = computed(() => t.value.permissions)

const credDefUlid = computed(() => definitionUlid(selectedDefinition.value))
const canCheck = computed(() => Boolean(candidateUlid.value.trim() && credDefUlid.value))
const resultFields = computed(() => checkResult.value || {})
const viewDetailLabel = computed(() => copy.value.viewDetail || (isZh.value ? "查看详情" : "View Details"))
const detailDialogTitle = computed(() => copy.value.detailDialogTitle || (isZh.value ? "权限详情" : "Permission Details"))
const closeLabel = computed(() => copy.value.close || (isZh.value ? "关闭" : "Close"))

const actions = computed(() => [
  {
    key: "grant" as const,
    title: copy.value.actions.grant.title,
    desc: copy.value.actions.grant.desc,
    endpoint: "/api/permissions/grant",
    icon: ShieldCheck,
    tone: "bg-emerald-600 text-white",
  },
  {
    key: "revoke-upload" as const,
    title: copy.value.actions.revokeUpload.title,
    desc: copy.value.actions.revokeUpload.desc,
    endpoint: "/api/permissions/revoke",
    icon: ShieldOff,
    tone: "bg-amber-500 text-white",
  },
  {
    key: "mark-expired" as const,
    title: copy.value.actions.markExpired.title,
    desc: copy.value.actions.markExpired.desc,
    endpoint: "/api/permissions/mark-expired",
    icon: FileWarning,
    tone: "bg-orange-600 text-white",
  },
  {
    key: "revoke-credential" as const,
    title: copy.value.actions.revokeCredential.title,
    desc: copy.value.actions.revokeCredential.desc,
    endpoint: "/api/permissions/revoke-credential",
    icon: UserX,
    tone: "bg-red-600 text-white",
  },
])

function definitionUlid(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["cred_def_ulid", "cred_def_id", "qual_ulid"]) || "")
}

function definitionName(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["name", "name_hint", "title"]) || definitionUlid(definition) || copy.value.unnamedDefinition)
}

function selectDefinition(definition: JsonRecord) {
  const changed = definitionUlid(selectedDefinition.value) !== definitionUlid(definition)
  selectedDefinition.value = definition
  if (changed) checkResult.value = null
}

function openDetail(definition: JsonRecord) {
  selectDefinition(definition)
  detailOpen.value = true
}

function resultStatus() {
  return pickFirst(checkResult.value || {}, ["credential_status", "status", "state", "eligible"])
}

async function loadDefinitions() {
  definitionsLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/credentials/definitions")
    const list = Array.isArray(data.definitions) ? data.definitions : []
    definitions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selectedDefinition.value || !definitions.value.some((item) => definitionUlid(item) === definitionUlid(selectedDefinition.value))) {
      selectedDefinition.value = definitions.value[0] || null
    }
  } catch (err) {
    console.error(err)
    toast.error(copy.value.toasts.definitionsLoadFailed)
  } finally {
    definitionsLoading.value = false
  }
}

async function check() {
  if (!canCheck.value) {
    toast.error(copy.value.toasts.checkRequired)
    return
  }
  loading.value = true
  try {
    checkResult.value = await apiClient<JsonRecord>(
      `/api/permissions/check?candidate_ulid=${encodeURIComponent(candidateUlid.value.trim())}&cred_def_ulid=${encodeURIComponent(credDefUlid.value)}`,
    )
    detailOpen.value = true
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.checkFailed))
  } finally {
    loading.value = false
  }
}

async function runAction(action: { key: PermissionAction; endpoint: string }) {
  if (!canCheck.value) {
    toast.error(copy.value.toasts.actionRequired)
    return
  }
  if (!reason.value.trim()) {
    toast.error(copy.value.toasts.reasonRequired)
    return
  }
  activeAction.value = action.key
  loading.value = true
  try {
    await apiClient(action.endpoint, {
      method: "POST",
      body: JSON.stringify({
        candidate_ulid: candidateUlid.value.trim(),
        cred_def_ulid: credDefUlid.value,
        reason: reason.value.trim(),
      }),
    })
    toast.success(copy.value.toasts.actionSuccess)
    reason.value = ""
    await check()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.actionFailed))
  } finally {
    activeAction.value = null
    loading.value = false
  }
}

onMounted(loadDefinitions)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadDefinitions">
        <RefreshCw class="h-4 w-4" :class="definitionsLoading ? 'animate-spin' : ''" />
        {{ copy.refreshDefinitions }}
      </button>
    </header>

    <div class="grid gap-6">
      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 class="text-xl font-black">{{ copy.targetTitle }}</h2>
        <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px] lg:items-end">
          <label class="grid gap-2 text-sm font-bold">
            {{ copy.candidateUlid }}
            <input v-model="candidateUlid" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="64" :placeholder="copy.candidatePlaceholder" />
          </label>
          <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="loading || !canCheck" @click="check">
            <Loader2 v-if="loading && !activeAction" class="h-4 w-4 animate-spin" />
            <Search v-else class="h-4 w-4" />
            {{ copy.checkPermission }}
          </button>
        </div>
      </section>

      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">{{ copy.definitionsTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.definitionsDescription }}</p>
          </div>
        </div>
        <div v-if="definitionsLoading" class="p-10 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else class="divide-y divide-slate-100">
          <div
            v-for="definition in definitions"
            :key="definitionUlid(definition)"
            class="grid cursor-pointer gap-4 px-5 py-4 hover:bg-sky-50 md:grid-cols-[minmax(0,1fr)_120px] md:items-center"
            :class="definitionUlid(selectedDefinition) === definitionUlid(definition) ? 'bg-sky-50' : ''"
            @click="selectDefinition(definition)"
          >
            <div class="min-w-0">
              <div class="font-black">{{ definitionName(definition) }}</div>
              <div class="mt-1 text-sm text-slate-500">{{ definition.category || "-" }}</div>
              <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ definitionUlid(definition) || "-" }}</div>
            </div>
            <button class="justify-self-start text-sm font-bold text-blue-700 hover:text-blue-900 md:justify-self-end" type="button" @click.stop="openDetail(definition)">
              {{ viewDetailLabel }}
            </button>
          </div>
        </div>
        <div v-if="!definitionsLoading && !definitions.length" class="p-10 text-center text-slate-500">{{ copy.emptyDefinitions }}</div>
      </section>
    </div>

    <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="flex max-h-[88vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
        <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-6 py-4">
          <div class="min-w-0">
            <h2 class="truncate text-2xl font-black">{{ detailDialogTitle }}</h2>
            <p class="mt-1 break-all text-sm text-slate-500">{{ definitionName(selectedDefinition) }}</p>
          </div>
          <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="closeLabel" @click="detailOpen = false">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-5">
          <div class="space-y-6">
            <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <h2 class="text-2xl font-black">{{ copy.currentTarget }}</h2>
                  <p class="mt-1 text-sm text-slate-500">{{ copy.currentTargetDescription }}</p>
                </div>
                <span v-if="checkResult" class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(resultStatus())">
                  {{ resultStatus() || "UNKNOWN" }}
                </span>
              </div>
              <div class="mt-5 grid gap-4 md:grid-cols-2">
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.candidate }}</div>
                  <div class="mt-2 break-all text-sm font-bold">{{ candidateUlid || "-" }}</div>
                </div>
                <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                  <div class="text-xs font-black uppercase text-slate-400">{{ copy.credentialDefinition }}</div>
                  <div class="mt-2 break-all text-sm font-bold">{{ definitionName(selectedDefinition) }}</div>
                  <div class="mt-1 break-all text-xs text-slate-500">{{ credDefUlid || "-" }}</div>
                </div>
              </div>
            </section>

            <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
              <div class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
                <h2 class="mb-4 text-xl font-black">{{ copy.resultTitle }}</h2>
                <div v-if="!checkResult" class="p-12 text-center text-slate-500">{{ copy.emptyResult }}</div>
                <div v-else class="space-y-5">
                  <div class="grid gap-4 md:grid-cols-2">
                    <label v-for="(value, key) in resultFields" :key="key" class="grid gap-2 text-sm font-bold">
                      {{ key }}
                      <textarea
                        v-if="Array.isArray(value) || (value && typeof value === 'object')"
                        class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                        disabled
                        :value="JSON.stringify(value, null, 2)"
                      />
                      <input
                        v-else
                        class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                        disabled
                        :value="String(value ?? '-')"
                      />
                    </label>
                  </div>
                  <JsonPreview
                    :title="copy.rawJson"
                    :value="checkResult"
                    :copy-label="copy.copyJson"
                    :copied-label="copy.copiedJson"
                    :copied-message="copy.toasts.jsonCopied"
                    :copy-error-message="copy.toasts.jsonCopyFailed"
                  />
                </div>
              </div>

              <aside class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
                <h2 class="text-xl font-black">{{ copy.actionTitle }}</h2>
                <p class="mt-1 text-sm text-slate-500">{{ copy.actionDescription }}</p>
                <label class="mt-4 grid gap-2 text-sm font-bold">
                  {{ copy.reason }}
                  <textarea v-model="reason" class="min-h-24 rounded-xl border border-slate-200 px-4 py-3" maxlength="500" :placeholder="copy.reasonPlaceholder" />
                </label>
                <div class="mt-4 grid gap-3">
                  <button
                    v-for="item in actions"
                    :key="item.key"
                    class="rounded-2xl px-4 py-3 text-left font-bold disabled:opacity-50"
                    :class="item.tone"
                    type="button"
                    :disabled="loading || !canCheck"
                    @click="runAction(item)"
                  >
                    <div class="flex items-center gap-2">
                      <Loader2 v-if="activeAction === item.key" class="h-4 w-4 animate-spin" />
                      <component :is="item.icon" v-else class="h-4 w-4" />
                      {{ item.title }}
                    </div>
                    <div class="mt-1 text-xs font-semibold opacity-80">{{ item.desc }}</div>
                  </button>
                </div>
              </aside>
            </section>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
