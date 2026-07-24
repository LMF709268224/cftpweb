<script setup lang="ts">
import { FileWarning, Loader2, RefreshCw, Search, ShieldCheck, ShieldOff, TriangleAlert, UserX, X } from "lucide-vue-next"
import { computed, onMounted, ref, type Component } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass, pickFirst } from "@/lib/status"

type PermissionAction = "grant" | "revoke-upload" | "mark-expired" | "revoke-credential"
type PermissionActionItem = {
  key: PermissionAction
  title: string
  desc: string
  endpoint: string
  icon: Component
  tone: string
  requiresConfirmation: boolean
}
type PendingPermissionAction = {
  key: PermissionAction
  title: string
  endpoint: string
  candidateUlid: string
  credDefUlid: string
  credDefName: string
  reason: string
}

const definitions = ref<JsonRecord[]>([])
const selectedDefinition = ref<JsonRecord | null>(null)
const candidateUlid = ref("")
const reason = ref("")
const checkResult = ref<JsonRecord | null>(null)
const definitionsLoading = ref(false)
const loading = ref(false)
const activeAction = ref<PermissionAction | null>(null)
const pendingAction = ref<PendingPermissionAction | null>(null)
const detailOpen = ref(false)
const { t, isZh } = useAdminLanguage()
const copy = computed(() => t.value.permissions)

const credDefUlid = computed(() => definitionUlid(selectedDefinition.value))
const canCheck = computed(() => Boolean(candidateUlid.value.trim() && credDefUlid.value))
const resultFields = computed(() => checkResult.value || {})
const viewDetailLabel = computed(() => copy.value.viewDetail || (isZh.value ? "查看详情" : "View Details"))
const detailDialogTitle = computed(() => copy.value.detailDialogTitle || (isZh.value ? "权限详情" : "Permission Details"))
const closeLabel = computed(() => copy.value.close || (isZh.value ? "关闭" : "Close"))
const clearInputLabel = computed(() => copy.value.clearInput || (isZh.value ? "清除输入" : "Clear input"))

const actions = computed<PermissionActionItem[]>(() => [
  {
    key: "grant" as const,
    title: copy.value.actions.grant.title,
    desc: copy.value.actions.grant.desc,
    endpoint: "/api/permissions/grant",
    icon: ShieldCheck,
    tone: "bg-emerald-600 text-white",
    requiresConfirmation: false,
  },
  {
    key: "revoke-upload" as const,
    title: copy.value.actions.revokeUpload.title,
    desc: copy.value.actions.revokeUpload.desc,
    endpoint: "/api/permissions/revoke",
    icon: ShieldOff,
    tone: "bg-amber-500 text-white",
    requiresConfirmation: true,
  },
  {
    key: "mark-expired" as const,
    title: copy.value.actions.markExpired.title,
    desc: copy.value.actions.markExpired.desc,
    endpoint: "/api/permissions/mark-expired",
    icon: FileWarning,
    tone: "bg-orange-600 text-white",
    requiresConfirmation: true,
  },
  {
    key: "revoke-credential" as const,
    title: copy.value.actions.revokeCredential.title,
    desc: copy.value.actions.revokeCredential.desc,
    endpoint: "/api/permissions/revoke-credential",
    icon: UserX,
    tone: "bg-red-600 text-white",
    requiresConfirmation: true,
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

function clearCandidate() {
  candidateUlid.value = ""
  checkResult.value = null
  detailOpen.value = false
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
  const targetCandidateUlid = candidateUlid.value.trim()
  const targetCredDefUlid = credDefUlid.value
  loading.value = true
  try {
    const result = await apiClient<JsonRecord>(
      `/api/permissions/check?candidate_ulid=${encodeURIComponent(targetCandidateUlid)}&cred_def_ulid=${encodeURIComponent(targetCredDefUlid)}`,
    )
    if (candidateUlid.value.trim() !== targetCandidateUlid || credDefUlid.value !== targetCredDefUlid) return
    checkResult.value = result
    detailOpen.value = true
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.checkFailed))
  } finally {
    loading.value = false
  }
}

function createPendingAction(action: PermissionActionItem) {
  if (!canCheck.value) {
    toast.error(copy.value.toasts.actionRequired)
    return null
  }
  if (!reason.value.trim()) {
    toast.error(copy.value.toasts.reasonRequired)
    return null
  }

  return {
    key: action.key,
    title: action.title,
    endpoint: action.endpoint,
    candidateUlid: candidateUlid.value.trim(),
    credDefUlid: credDefUlid.value,
    credDefName: definitionName(selectedDefinition.value),
    reason: reason.value.trim(),
  } satisfies PendingPermissionAction
}

function requestAction(action: PermissionActionItem) {
  if (loading.value) return
  const pending = createPendingAction(action)
  if (!pending) return
  if (action.requiresConfirmation) {
    pendingAction.value = pending
    return
  }
  void executeAction(pending)
}

function closeActionConfirm() {
  if (loading.value) return
  pendingAction.value = null
}

function isCurrentActionTarget(action: PendingPermissionAction) {
  return candidateUlid.value.trim() === action.candidateUlid && credDefUlid.value === action.credDefUlid
}

async function executeAction(action: PendingPermissionAction) {
  if (loading.value) return
  if (!isCurrentActionTarget(action)) {
    pendingAction.value = null
    toast.error(copy.value.toasts.actionTargetChanged)
    return
  }

  activeAction.value = action.key
  loading.value = true
  try {
    await apiClient(action.endpoint, {
      method: "POST",
      body: JSON.stringify({
        candidate_ulid: action.candidateUlid,
        cred_def_ulid: action.credDefUlid,
        reason: action.reason,
      }),
    })
    toast.success(copy.value.toasts.actionSuccess)
    pendingAction.value = null
    if (isCurrentActionTarget(action)) {
      reason.value = ""
      await check()
    }
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
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadDefinitions">
        <RefreshCw class="h-4 w-4" :class="definitionsLoading ? 'animate-spin' : ''" />
        {{ copy.refreshDefinitions }}
      </button>
    </header>

    <div class="grid gap-6">
      <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-5">
        <h2 class="text-xl font-black">{{ copy.targetTitle }}</h2>
        <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px] lg:items-end">
          <label class="grid gap-2 text-sm font-bold">
            {{ copy.candidateUlid }}
            <span class="relative block">
              <input v-model="candidateUlid" class="w-full rounded-xl border border-slate-200 px-4 py-3 pr-11" maxlength="64" :placeholder="copy.candidatePlaceholder" />
              <button
                v-if="candidateUlid"
                class="absolute right-2 top-1/2 inline-flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
                type="button"
                :aria-label="clearInputLabel"
                :title="clearInputLabel"
                @click="clearCandidate"
              >
                <X class="h-4 w-4" />
              </button>
            </span>
          </label>
          <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-blue-700 px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="loading || !canCheck" @click="check">
            <Loader2 v-if="loading && !activeAction" class="h-4 w-4 animate-spin" />
            <Search v-else class="h-4 w-4" />
            {{ copy.checkPermission }}
          </button>
        </div>
      </section>

      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm md:rounded-3xl">
        <div class="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-4 md:p-5">
          <div class="min-w-0">
            <h2 class="text-xl font-black">{{ copy.definitionsTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">{{ copy.definitionsDescription }}</p>
          </div>
        </div>
        <div v-if="definitionsLoading" class="px-4 py-10 text-center text-slate-500 md:p-10">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          {{ copy.loading }}
        </div>
        <div v-else class="divide-y divide-slate-100">
          <div
            v-for="definition in definitions"
            :key="definitionUlid(definition)"
            class="grid cursor-pointer gap-4 px-4 py-4 hover:bg-sky-50 md:grid-cols-[minmax(0,1fr)_120px] md:items-center md:px-5"
            :class="definitionUlid(selectedDefinition) === definitionUlid(definition) ? 'bg-sky-50' : ''"
            @click="selectDefinition(definition)"
          >
            <div class="min-w-0">
              <div class="break-words font-black">{{ definitionName(definition) }}</div>
              <div class="mt-1 text-sm text-slate-500">{{ definition.category || "-" }}</div>
              <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ definitionUlid(definition) || "-" }}</div>
            </div>
            <button class="inline-flex items-center justify-center rounded-xl border border-blue-100 bg-blue-50 px-3 py-2 text-sm font-bold text-blue-700 hover:text-blue-900 md:justify-self-end md:border-0 md:bg-transparent md:px-0 md:py-0" type="button" @click.stop="openDetail(definition)">
              {{ viewDetailLabel }}
            </button>
          </div>
        </div>
        <div v-if="!definitionsLoading && !definitions.length" class="px-4 py-10 text-center text-slate-500 md:p-10">{{ copy.emptyDefinitions }}</div>
      </section>
    </div>

    <div v-if="detailOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-0 md:p-6">
      <div class="flex h-full max-h-none w-full max-w-[1180px] flex-col overflow-hidden rounded-none bg-white shadow-2xl md:h-auto md:max-h-[88vh] md:rounded-3xl">
        <div class="flex items-center justify-between gap-4 border-b border-slate-200 px-4 py-4 md:px-6">
          <div class="min-w-0">
            <h2 class="break-words text-xl font-black md:truncate md:text-2xl">{{ detailDialogTitle }}</h2>
            <p class="mt-1 break-all text-sm text-slate-500">{{ definitionName(selectedDefinition) }}</p>
          </div>
          <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm hover:bg-slate-50 hover:text-slate-900" type="button" :aria-label="closeLabel" @click="detailOpen = false">
            <X class="h-5 w-5" />
          </button>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
          <div class="space-y-6">
            <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-5">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div class="min-w-0">
                  <h2 class="text-xl font-black md:text-2xl">{{ copy.currentTarget }}</h2>
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
              <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-5">
                <h2 class="mb-4 text-xl font-black">{{ copy.resultTitle }}</h2>
                <div v-if="!checkResult" class="px-4 py-10 text-center text-slate-500 md:p-12">{{ copy.emptyResult }}</div>
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

              <aside class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-5">
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
                    @click="requestAction(item)"
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

    <Teleport to="body">
      <div v-if="pendingAction" class="fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/60 p-4 md:p-6">
        <section class="flex max-h-[calc(100vh-2rem)] w-full max-w-lg flex-col overflow-hidden rounded-2xl bg-white shadow-2xl md:rounded-3xl" role="dialog" aria-modal="true" :aria-labelledby="`permission-action-confirm-${pendingAction.key}`">
          <header class="flex items-start justify-between gap-4 border-b border-slate-200 px-5 py-5 md:px-6">
            <div class="flex min-w-0 items-start gap-3">
              <span class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-red-50 text-red-600">
                <TriangleAlert class="h-5 w-5" />
              </span>
              <div class="min-w-0">
                <h2 :id="`permission-action-confirm-${pendingAction.key}`" class="break-words text-xl font-black text-slate-950">
                  {{ copy.actionConfirm.title(pendingAction.title) }}
                </h2>
                <p class="mt-1 text-sm leading-6 text-slate-500">{{ copy.actionConfirm.description }}</p>
              </div>
            </div>
            <button class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 text-slate-500 transition hover:bg-slate-50 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-40" type="button" :aria-label="copy.close" :disabled="loading" @click="closeActionConfirm">
              <X class="h-5 w-5" />
            </button>
          </header>

          <div class="min-h-0 flex-1 overflow-y-auto px-5 py-5 md:px-6">
            <dl class="divide-y divide-slate-200 rounded-2xl bg-slate-50 px-4">
              <div class="grid gap-1 py-3 sm:grid-cols-[112px_minmax(0,1fr)] sm:gap-4">
                <dt class="text-xs font-black text-slate-500">{{ copy.actionConfirm.action }}</dt>
                <dd class="break-words text-sm font-black text-slate-950">{{ pendingAction.title }}</dd>
              </div>
              <div class="grid gap-1 py-3 sm:grid-cols-[112px_minmax(0,1fr)] sm:gap-4">
                <dt class="text-xs font-black text-slate-500">{{ copy.candidate }}</dt>
                <dd class="break-all font-mono text-xs font-bold text-blue-700">{{ pendingAction.candidateUlid }}</dd>
              </div>
              <div class="grid gap-1 py-3 sm:grid-cols-[112px_minmax(0,1fr)] sm:gap-4">
                <dt class="text-xs font-black text-slate-500">{{ copy.credentialDefinition }}</dt>
                <dd class="break-words text-sm font-bold text-slate-900">{{ pendingAction.credDefName }}</dd>
              </div>
              <div class="grid gap-1 py-3 sm:grid-cols-[112px_minmax(0,1fr)] sm:gap-4">
                <dt class="text-xs font-black text-slate-500">{{ copy.actionConfirm.credentialId }}</dt>
                <dd class="break-all font-mono text-xs font-bold text-blue-700">{{ pendingAction.credDefUlid }}</dd>
              </div>
              <div class="grid gap-1 py-3 sm:grid-cols-[112px_minmax(0,1fr)] sm:gap-4">
                <dt class="text-xs font-black text-slate-500">{{ copy.reason }}</dt>
                <dd class="whitespace-pre-wrap break-words text-sm font-bold text-slate-900">{{ pendingAction.reason }}</dd>
              </div>
            </dl>
          </div>

          <footer class="flex flex-col-reverse gap-3 border-t border-slate-200 px-5 py-4 sm:flex-row sm:justify-end md:px-6">
            <button class="h-11 rounded-xl border border-slate-300 px-5 text-sm font-black text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40" type="button" :disabled="loading" @click="closeActionConfirm">
              {{ copy.actionConfirm.cancel }}
            </button>
            <button class="inline-flex h-11 items-center justify-center rounded-xl bg-red-600 px-5 text-sm font-black text-white transition hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="loading" @click="executeAction(pendingAction)">
              <Loader2 v-if="loading" class="mr-2 h-4 w-4 animate-spin" />
              <TriangleAlert v-else class="mr-2 h-4 w-4" />
              {{ loading ? copy.actionConfirm.processing : copy.actionConfirm.confirm }}
            </button>
          </footer>
        </section>
      </div>
    </Teleport>
  </section>
</template>
