<script setup lang="ts">
import { Eye, FileWarning, Loader2, RefreshCw, UserX, X } from "lucide-vue-next"
import { computed, onMounted, ref, type Component } from "vue"
import { toast } from "vue-sonner"
import JsonPreview from "@/components/JsonPreview.vue"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { apiClient } from "@/lib/apiClient"
import { type JsonRecord } from "@/lib/display"
import { useAdminLanguage } from "@/lib/language"
import { badgeClass } from "@/lib/status"

type PermissionAction = "mark-expired" | "revoke-credential"
type PermissionActionItem = { key: PermissionAction; title: string; desc: string; endpoint: string; icon: Component }
type PendingPermissionAction = PermissionActionItem & {
  candidateUlid: string
  credDefUlid: string
  credentialUlid: string
  reason: string
}

const credentials = ref<JsonRecord[]>([])
const selectedCredential = ref<JsonRecord | null>(null)
const total = ref(0)
const nextCursor = ref("")
const hasMore = ref(false)
const listLoading = ref(false)
const loadingMore = ref(false)
const detailLoading = ref(false)
const detailOpen = ref(false)
const reason = ref("")
const pendingAction = ref<PendingPermissionAction | null>(null)
const activeAction = ref<PermissionAction | null>(null)
let credentialsRequestSeq = 0
let detailRequestSeq = 0

const { t, isZh } = useAdminLanguage()
const copy = computed(() => t.value.permissions)
const selectedStatus = computed(() => credentialStatus(selectedCredential.value))
const canOperate = computed(() => Boolean(selectedCredential.value && selectedStatus.value === "active"))
const actions = computed<PermissionActionItem[]>(() => [
  {
    key: "mark-expired",
    title: copy.value.actions.markExpired.title,
    desc: copy.value.actions.markExpired.desc,
    endpoint: "/api/permissions/mark-expired",
    icon: FileWarning,
  },
  {
    key: "revoke-credential",
    title: copy.value.actions.revokeCredential.title,
    desc: copy.value.actions.revokeCredential.desc,
    endpoint: "/api/permissions/revoke-credential",
    icon: UserX,
  },
])

function stringValue(record: JsonRecord | null | undefined, keys: string[]) {
  if (!record) return ""
  for (const key of keys) {
    const value = record[key]
    if (value !== undefined && value !== null && String(value).trim()) return String(value).trim()
  }
  return ""
}

function credentialUlid(credential: JsonRecord | null | undefined) {
  return stringValue(credential, ["cred_ulid", "credential_ulid", "cred_id"])
}

function candidateUlid(credential: JsonRecord | null | undefined) {
  return stringValue(credential, ["candidate_ulid", "candidate_id"])
}

function candidateName(credential: JsonRecord | null | undefined) {
  return stringValue(credential, ["candidate_name", "candidate_display_name", "user_name"])
}

function credentialDefinitionUlid(credential: JsonRecord | null | undefined) {
  return stringValue(credential, ["cred_def_ulid", "cred_def_id"])
}

function credentialDefinitionName(credential: JsonRecord | null | undefined) {
  const inlineName = stringValue(credential, ["cred_def_name", "credential_name", "name"])
  return inlineName || copy.value.unnamedDefinition
}

function credentialStatus(credential: JsonRecord | null | undefined) {
  const raw = stringValue(credential, ["status", "credential_status", "state"]).toUpperCase()
  if (raw === "1" || raw.includes("ACTIVE")) return "active"
  if (raw === "2" || raw.includes("REVOKED") || raw.includes("REVOKE")) return "revoked"
  if (raw === "3" || raw.includes("EXPIRED") || raw.includes("EXPIRE")) return "expired"
  return "unknown"
}

function credentialStatusLabel(credential: JsonRecord | null | undefined) {
  return copy.value.status[credentialStatus(credential)]
}

function credentialStatusClass(credential: JsonRecord | null | undefined) {
  const status = credentialStatus(credential)
  if (status === "active") return "border-emerald-200 bg-emerald-50 text-emerald-700"
  if (status === "revoked") return "border-red-200 bg-red-50 text-red-700"
  if (status === "expired") return "border-amber-200 bg-amber-50 text-amber-700"
  return badgeClass(stringValue(credential, ["status", "credential_status", "state"]))
}

function formatDate(value: unknown) {
  const raw = String(value || "").trim()
  if (!raw) return "-"
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return new Intl.DateTimeFormat(isZh.value ? "zh-CN" : "en-US", { dateStyle: "medium", timeStyle: "short" }).format(date)
}

function auditRemark(credential: JsonRecord | null | undefined) {
  return stringValue(credential, ["audit_remark", "revocation_reason", "reason"]) || "-"
}

function selectCredential(credential: JsonRecord) {
  selectedCredential.value = credential
  reason.value = ""
}

async function openDetail(credential: JsonRecord) {
  selectCredential(credential)
  detailOpen.value = true
  const id = credentialUlid(credential)
  const requestSeq = ++detailRequestSeq
  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/credentials/${encodeURIComponent(id)}`)
    if (requestSeq === detailRequestSeq && credentialUlid(selectedCredential.value) === id) {
      selectedCredential.value = { ...credential, ...detail }
    }
  } catch (err) {
    if (requestSeq === detailRequestSeq) {
      console.error(err)
      toast.error(apiErrorMessage(err, copy.value.toasts.detailLoadFailed))
    }
  } finally {
    if (requestSeq === detailRequestSeq) detailLoading.value = false
  }
}

function closeDetail() {
  if (!activeAction.value) {
    detailRequestSeq += 1
    detailLoading.value = false
    detailOpen.value = false
  }
}

async function loadCredentials(reset = true) {
  if (reset) {
    credentialsRequestSeq += 1
    listLoading.value = true
  } else {
    loadingMore.value = true
  }
  const requestSeq = credentialsRequestSeq
  try {
    const params = new URLSearchParams({ is_current: "true", page_size: "50" })
    if (!reset && nextCursor.value) params.set("cursor", nextCursor.value)
    const data = await apiClient<JsonRecord>(`/api/credentials?${params.toString()}`)
    if (reset && requestSeq !== credentialsRequestSeq) return
    const items = Array.isArray(data.credentials)
      ? data.credentials.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
      : []
    credentials.value = reset ? items : [...credentials.value, ...items]
    total.value = Number(data.total || 0)
    nextCursor.value = String(data.next_cursor || "")
    hasMore.value = Boolean(data.has_more) && Boolean(nextCursor.value)
    if (selectedCredential.value) {
      const selectedCandidate = candidateUlid(selectedCredential.value)
      const selectedDefinition = credentialDefinitionUlid(selectedCredential.value)
      selectedCredential.value = credentials.value.find((item) => candidateUlid(item) === selectedCandidate && credentialDefinitionUlid(item) === selectedDefinition) || null
    }
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.credentialsLoadFailed))
  } finally {
    if (reset) listLoading.value = false
    else loadingMore.value = false
  }
}

function refreshPage() {
  void loadCredentials()
}

function createPendingAction(action: PermissionActionItem) {
  const credential = selectedCredential.value
  if (!credential || !canOperate.value) {
    toast.error(copy.value.toasts.actionUnavailable)
    return null
  }
  if (!reason.value.trim()) {
    toast.error(copy.value.toasts.reasonRequired)
    return null
  }
  return {
    ...action,
    candidateUlid: candidateUlid(credential),
    credDefUlid: credentialDefinitionUlid(credential),
    credentialUlid: credentialUlid(credential),
    reason: reason.value.trim(),
  } satisfies PendingPermissionAction
}

function requestAction(action: PermissionActionItem) {
  if (activeAction.value) return
  const pending = createPendingAction(action)
  if (pending) pendingAction.value = pending
}

function closeActionConfirm() {
  if (!activeAction.value) pendingAction.value = null
}

async function executeAction(action: PendingPermissionAction) {
  if (activeAction.value) return
  const current = selectedCredential.value
  if (!current || credentialUlid(current) !== action.credentialUlid) {
    pendingAction.value = null
    toast.error(copy.value.toasts.actionTargetChanged)
    return
  }
  activeAction.value = action.key
  try {
    await apiClient(action.endpoint, {
      method: "POST",
      body: JSON.stringify({ candidate_ulid: action.candidateUlid, cred_def_ulid: action.credDefUlid, reason: action.reason }),
    })
    toast.success(copy.value.toasts.actionSuccess)
    pendingAction.value = null
    reason.value = ""
    await loadCredentials(true)
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, copy.value.toasts.actionFailed))
  } finally {
    activeAction.value = null
  }
}

onMounted(() => {
  refreshPage()
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-5 px-4 py-5 md:gap-6 md:px-8 md:py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-3xl font-black tracking-tight md:text-4xl">{{ copy.title }}</h1>
        <p class="mt-2 text-slate-600">{{ copy.subtitle }}</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm disabled:cursor-not-allowed disabled:opacity-50" type="button" :disabled="listLoading" @click="refreshPage">
        <RefreshCw class="h-4 w-4" :class="listLoading ? 'animate-spin' : ''" />
        {{ copy.refresh }}
      </button>
    </header>

    <section class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-5">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 class="text-xl font-black">{{ copy.listTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.listDescription }}</p>
        </div>
        <div class="rounded-full bg-slate-100 px-3 py-1 text-sm font-bold text-slate-600">{{ copy.total }}: {{ total }}</div>
      </div>

      <div v-if="listLoading" class="px-4 py-12 text-center text-slate-500"><Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />{{ copy.loading }}</div>
      <div v-else-if="!credentials.length" class="px-4 py-12 text-center text-slate-500">{{ copy.emptyCredentials }}</div>
      <div v-else class="mt-5 overflow-x-auto rounded-2xl border border-slate-200">
        <table class="w-full min-w-[850px] text-left text-sm">
          <thead class="bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
            <tr><th class="px-4 py-3">{{ copy.columns.candidate }}</th><th class="px-4 py-3">{{ copy.columns.credential }}</th><th class="px-4 py-3">{{ copy.columns.status }}</th><th class="px-4 py-3">{{ copy.columns.version }}</th><th class="px-4 py-3">{{ copy.columns.auditTime }}</th><th class="px-4 py-3 text-right">{{ copy.columns.actions }}</th></tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="credential in credentials" :key="credentialUlid(credential)" class="cursor-pointer transition hover:bg-sky-50" :class="credentialUlid(selectedCredential) === credentialUlid(credential) ? 'bg-sky-50' : ''" @click="selectCredential(credential)">
              <td class="max-w-[220px] px-4 py-4"><div class="font-bold">{{ candidateName(credential) || copy.unknownCandidate }}</div><div class="mt-1 break-all text-xs text-slate-500">{{ candidateUlid(credential) || "-" }}</div></td>
              <td class="max-w-[220px] px-4 py-4"><div class="font-bold">{{ credentialDefinitionName(credential) }}</div><div class="mt-1 break-all text-xs text-slate-500">{{ credentialDefinitionUlid(credential) || "-" }}</div></td>
              <td class="px-4 py-4"><span class="inline-flex rounded-full border px-2.5 py-1 text-xs font-black" :class="credentialStatusClass(credential)">{{ credentialStatusLabel(credential) }}</span><div v-if="auditRemark(credential) !== '-'" class="mt-2 max-w-[210px] truncate text-xs text-slate-500" :title="auditRemark(credential)">{{ auditRemark(credential) }}</div></td>
              <td class="px-4 py-4 font-bold text-slate-700">v{{ credential.version || "-" }}</td>
              <td class="whitespace-nowrap px-4 py-4 text-slate-600">{{ formatDate(credential.audit_time || credential.created_at) }}</td>
              <td class="px-4 py-4 text-right"><button class="inline-flex items-center gap-1 rounded-lg border border-blue-100 bg-blue-50 px-3 py-2 font-bold text-blue-700 hover:bg-blue-100" type="button" @click.stop="openDetail(credential)"><Eye class="h-4 w-4" />{{ copy.viewDetail }}</button></td>
            </tr>
          </tbody>
        </table>
      </div>
      <button v-if="hasMore" class="mx-auto mt-4 flex items-center gap-2 rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-50" type="button" :disabled="loadingMore" @click="void loadCredentials(false)"><Loader2 v-if="loadingMore" class="h-4 w-4 animate-spin" />{{ copy.loadMore }}</button>
    </section>

    <section v-if="selectedCredential" class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm md:rounded-3xl md:p-5">
      <div class="flex flex-wrap items-start justify-between gap-4"><div><h2 class="text-xl font-black">{{ copy.detailTitle }}</h2><p class="mt-1 break-all text-sm text-slate-500">{{ credentialUlid(selectedCredential) }}</p></div><span class="inline-flex rounded-full border px-3 py-1 text-sm font-black" :class="credentialStatusClass(selectedCredential)">{{ credentialStatusLabel(selectedCredential) }}</span></div>
      <div class="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-xl bg-slate-50 p-4"><div class="text-xs font-black text-slate-400">{{ copy.fields.candidate }}</div><div class="mt-2 break-all font-bold">{{ candidateName(selectedCredential) || copy.unknownCandidate }}</div><div class="mt-1 break-all text-xs text-slate-500">{{ candidateUlid(selectedCredential) }}</div></div>
        <div class="rounded-xl bg-slate-50 p-4"><div class="text-xs font-black text-slate-400">{{ copy.fields.credential }}</div><div class="mt-2 font-bold">{{ credentialDefinitionName(selectedCredential) }}</div><div class="mt-1 break-all text-xs text-slate-500">{{ credentialDefinitionUlid(selectedCredential) }}</div></div>
        <div class="rounded-xl bg-slate-50 p-4"><div class="text-xs font-black text-slate-400">{{ copy.fields.auditTime }}</div><div class="mt-2 font-bold">{{ formatDate(selectedCredential.audit_time || selectedCredential.created_at) }}</div></div>
        <div class="rounded-xl bg-slate-50 p-4"><div class="text-xs font-black text-slate-400">{{ copy.fields.current }}</div><div class="mt-2 font-bold">{{ selectedCredential.is_current === false ? copy.no : copy.yes }}</div></div>
      </div>
      <div class="mt-4 rounded-xl border border-slate-200 p-4"><div class="text-xs font-black text-slate-400">{{ copy.fields.reason }}</div><div class="mt-2 whitespace-pre-wrap text-sm text-slate-700">{{ auditRemark(selectedCredential) }}</div></div>

      <div class="mt-5 grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
        <JsonPreview :title="copy.rawJson" :value="selectedCredential" :copy-label="copy.copyJson" :copied-label="copy.copiedJson" :copied-message="copy.toasts.jsonCopied" :copy-error-message="copy.toasts.jsonCopyFailed" />
        <aside class="rounded-2xl border border-slate-200 p-4">
          <h3 class="text-lg font-black">{{ copy.actionTitle }}</h3><p class="mt-1 text-sm text-slate-500">{{ canOperate ? copy.actionDescription : copy.operationUnavailable }}</p>
          <label class="mt-4 grid gap-2 text-sm font-bold">{{ copy.reason }}<textarea v-model="reason" class="min-h-24 rounded-xl border border-slate-200 px-4 py-3 disabled:bg-slate-100" :disabled="!canOperate" maxlength="500" :placeholder="copy.reasonPlaceholder" /></label>
          <div class="mt-4 grid gap-3"><button v-for="action in actions" :key="action.key" class="flex items-start gap-3 rounded-xl px-4 py-3 text-left font-bold text-white disabled:cursor-not-allowed disabled:opacity-50" :class="action.key === 'revoke-credential' ? 'bg-red-600' : 'bg-orange-600'" type="button" :disabled="!canOperate || !!activeAction" @click="requestAction(action)"><component :is="action.icon" class="mt-0.5 h-5 w-5 shrink-0" /><span><span class="block">{{ action.title }}</span><span class="mt-1 block text-xs font-normal text-white/80">{{ action.desc }}</span></span></button></div>
        </aside>
      </div>
    </section>

    <div v-if="detailOpen && selectedCredential" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4" @click.self="closeDetail">
      <div class="flex max-h-[90vh] w-full max-w-4xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl">
        <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4"><div><h2 class="text-xl font-black">{{ copy.detailTitle }}</h2><p class="mt-1 break-all text-xs text-slate-500">{{ credentialUlid(selectedCredential) }}</p></div><button class="inline-flex h-9 w-9 items-center justify-center rounded-full border border-slate-200" type="button" :aria-label="copy.close" @click="closeDetail"><X class="h-5 w-5" /></button></div>
        <div v-if="detailLoading" class="p-12 text-center text-slate-500"><Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />{{ copy.loading }}</div>
        <div v-else class="overflow-y-auto p-5"><div class="mb-5 text-lg font-black">{{ credentialDefinitionName(selectedCredential) }} <span class="ml-2 inline-flex rounded-full border px-2 py-1 text-xs" :class="credentialStatusClass(selectedCredential)">{{ credentialStatusLabel(selectedCredential) }}</span></div><div class="grid gap-3 md:grid-cols-2"><div><span class="text-xs text-slate-400">{{ copy.fields.candidate }}</span><div class="mt-1 break-all font-bold">{{ candidateUlid(selectedCredential) }}</div></div><div><span class="text-xs text-slate-400">{{ copy.fields.reason }}</span><div class="mt-1 whitespace-pre-wrap">{{ auditRemark(selectedCredential) }}</div></div></div><div class="mt-5"><JsonPreview :title="copy.rawJson" :value="selectedCredential" :copy-label="copy.copyJson" :copied-label="copy.copiedJson" :copied-message="copy.toasts.jsonCopied" :copy-error-message="copy.toasts.jsonCopyFailed" /></div></div>
      </div>
    </div>

    <div v-if="pendingAction" class="fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/50 p-4">
      <div class="w-full max-w-lg rounded-2xl bg-white p-5 shadow-2xl">
        <h2 class="text-xl font-black">{{ copy.actionConfirm.title(pendingAction.title) }}</h2><p class="mt-2 text-sm text-slate-600">{{ copy.actionConfirm.description }}</p>
        <dl class="mt-4 grid gap-3 rounded-xl bg-slate-50 p-4 text-sm"><div><dt class="text-xs text-slate-400">{{ copy.actionConfirm.action }}</dt><dd class="mt-1 font-bold">{{ pendingAction.title }}</dd></div><div><dt class="text-xs text-slate-400">{{ copy.actionConfirm.credentialId }}</dt><dd class="mt-1 break-all font-bold">{{ pendingAction.credentialUlid }}</dd></div><div><dt class="text-xs text-slate-400">{{ copy.reason }}</dt><dd class="mt-1 whitespace-pre-wrap">{{ pendingAction.reason }}</dd></div></dl>
        <div class="mt-5 flex justify-end gap-3"><button class="rounded-xl border px-4 py-2 font-bold" type="button" :disabled="!!activeAction" @click="closeActionConfirm">{{ copy.actionConfirm.cancel }}</button><button class="inline-flex items-center gap-2 rounded-xl bg-blue-700 px-4 py-2 font-bold text-white disabled:opacity-50" type="button" :disabled="!!activeAction" @click="void executeAction(pendingAction)"><Loader2 v-if="activeAction" class="h-4 w-4 animate-spin" />{{ activeAction ? copy.actionConfirm.processing : copy.actionConfirm.confirm }}</button></div>
      </div>
    </div>
  </section>
</template>
