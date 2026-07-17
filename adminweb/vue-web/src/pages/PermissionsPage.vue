<script setup lang="ts">
import { FileWarning, Inbox, Loader2, RefreshCw, Search, ShieldCheck, ShieldOff, UserX, X } from "lucide-vue-next"
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
const clearInputLabel = computed(() => copy.value.clearInput || (isZh.value ? "清除输入" : "Clear input"))

const actions = computed(() => [
  {
    key: "grant" as const,
    title: copy.value.actions.grant.title,
    desc: copy.value.actions.grant.desc,
    endpoint: "/api/permissions/grant",
    icon: ShieldCheck,
    tone: "bg-emerald-50 text-emerald-700",
  },
  {
    key: "revoke-upload" as const,
    title: copy.value.actions.revokeUpload.title,
    desc: copy.value.actions.revokeUpload.desc,
    endpoint: "/api/permissions/revoke",
    icon: ShieldOff,
    tone: "bg-amber-50 text-amber-700",
  },
  {
    key: "mark-expired" as const,
    title: copy.value.actions.markExpired.title,
    desc: copy.value.actions.markExpired.desc,
    endpoint: "/api/permissions/mark-expired",
    icon: FileWarning,
    tone: "bg-amber-50 text-amber-700",
  },
  {
    key: "revoke-credential" as const,
    title: copy.value.actions.revokeCredential.title,
    desc: copy.value.actions.revokeCredential.desc,
    endpoint: "/api/permissions/revoke-credential",
    icon: UserX,
    tone: "bg-red-50 text-red-700",
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
  <main class="ui-page">
    <!-- Page header -->
    <header class="flex flex-wrap items-start justify-between gap-4 md:items-center">
      <div class="min-w-0">
        <h1 class="ui-page-title">{{ copy.title }}</h1>
        <p class="ui-page-subtitle">{{ copy.subtitle }}</p>
      </div>
      <button class="ui-btn ui-btn-secondary" type="button" @click="loadDefinitions">
        <RefreshCw class="h-4 w-4 text-slate-400" :class="definitionsLoading ? 'animate-spin' : ''" />
        {{ copy.refreshDefinitions }}
      </button>
    </header>

    <!-- Query target -->
    <section class="ui-card mt-6 p-4 md:p-6">
      <h2 class="text-base font-semibold text-ink">{{ copy.targetTitle }}</h2>
      <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,1fr)_220px] lg:items-end">
        <label class="grid gap-1.5">
          <span class="ui-label">{{ copy.candidateUlid }}</span>
          <span class="relative block">
            <input v-model="candidateUlid" class="tnum ui-input pr-10" maxlength="64" :placeholder="copy.candidatePlaceholder" />
            <button
              v-if="candidateUlid"
              class="absolute right-2 top-1/2 inline-flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
              type="button"
              :aria-label="clearInputLabel"
              :title="clearInputLabel"
              @click="clearCandidate"
            >
              <X class="h-4 w-4" />
            </button>
          </span>
        </label>
        <button class="ui-btn ui-btn-primary" type="button" :disabled="loading || !canCheck" @click="check">
          <Loader2 v-if="loading && !activeAction" class="h-4 w-4 animate-spin" />
          <Search v-else class="h-4 w-4" />
          {{ copy.checkPermission }}
        </button>
      </div>
    </section>

    <!-- Credential definitions -->
    <section class="ui-card mt-4 overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-ink">{{ copy.definitionsTitle }}</h2>
          <p class="mt-1 text-sm text-slate-500">{{ copy.definitionsDescription }}</p>
        </div>
        <span class="tnum ui-badge shrink-0 bg-slate-100 text-slate-600">{{ definitions.length }}</span>
      </div>
      <div v-if="definitionsLoading" class="divide-y divide-hairline">
        <div v-for="i in 4" :key="`sk-${i}`" class="space-y-2.5 px-4 py-4 md:px-6">
          <div class="skeleton h-4 w-2/5" />
          <div class="skeleton h-3.5 w-3/5" />
          <div class="skeleton h-3 w-4/5" />
        </div>
      </div>
      <div v-else-if="!definitions.length" class="ui-empty">
        <div class="ui-empty-icon">
          <Inbox class="h-6 w-6" />
        </div>
        <p class="ui-empty-title">{{ copy.emptyDefinitions }}</p>
      </div>
      <div v-else class="divide-y divide-hairline">
        <div
          v-for="definition in definitions"
          :key="definitionUlid(definition)"
          class="grid cursor-pointer gap-4 px-4 py-4 transition-colors hover:bg-primary-50/40 md:grid-cols-[minmax(0,1fr)_auto] md:items-center md:px-6"
          :class="definitionUlid(selectedDefinition) === definitionUlid(definition) ? 'bg-primary-50/60' : ''"
          @click="selectDefinition(definition)"
        >
          <div class="min-w-0">
            <div class="break-words text-sm font-medium text-ink">{{ definitionName(definition) }}</div>
            <div class="mt-1 text-sm text-slate-500">{{ definition.category || "-" }}</div>
            <div class="tnum mt-1 break-all text-xs text-slate-400">ID: {{ definitionUlid(definition) || "-" }}</div>
          </div>
          <button class="ui-btn ui-btn-secondary ui-btn-sm md:justify-self-end" type="button" @click.stop="openDetail(definition)">
            {{ viewDetailLabel }}
          </button>
        </div>
      </div>
    </section>

    <!-- Detail modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="detailOpen" class="ui-modal-mask">
          <section class="ui-modal-panel max-w-[1180px]">
            <div class="flex items-start justify-between gap-4 border-b border-hairline px-4 py-4 md:px-6 md:py-5">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-ink md:text-lg">{{ detailDialogTitle }}</h2>
                <p class="mt-1 break-all text-sm text-slate-500">{{ definitionName(selectedDefinition) }}</p>
              </div>
              <button
                class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600"
                type="button"
                :aria-label="closeLabel"
                @click="detailOpen = false"
              >
                <X class="h-5 w-5" />
              </button>
            </div>

            <main class="min-h-0 flex-1 overflow-y-auto p-4 md:p-6">
              <div class="space-y-6">
                <!-- Current target -->
                <section class="rounded-2xl border border-edge p-4 md:p-5">
                  <div class="flex flex-wrap items-start justify-between gap-4">
                    <div class="min-w-0">
                      <h2 class="text-base font-semibold text-ink">{{ copy.currentTarget }}</h2>
                      <p class="mt-1 text-sm text-slate-500">{{ copy.currentTargetDescription }}</p>
                    </div>
                    <span v-if="checkResult" class="ui-badge shrink-0" :class="badgeClass(resultStatus())">
                      {{ resultStatus() || "UNKNOWN" }}
                    </span>
                  </div>
                  <div class="mt-5 grid gap-4 md:grid-cols-2">
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.candidate }}</div>
                      <div class="tnum mt-2 break-all text-sm font-medium text-ink">{{ candidateUlid || "-" }}</div>
                    </div>
                    <div class="rounded-2xl bg-slate-50 p-4">
                      <div class="text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy.credentialDefinition }}</div>
                      <div class="mt-2 break-all text-sm font-medium text-ink">{{ definitionName(selectedDefinition) }}</div>
                      <div class="tnum mt-1 break-all text-xs text-slate-400">{{ credDefUlid || "-" }}</div>
                    </div>
                  </div>
                </section>

                <section class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
                  <!-- Check result -->
                  <div class="rounded-2xl border border-edge p-4 md:p-5">
                    <h2 class="text-base font-semibold text-ink">{{ copy.resultTitle }}</h2>
                    <div v-if="!checkResult" class="ui-empty">
                      <div class="ui-empty-icon">
                        <Search class="h-6 w-6" />
                      </div>
                      <p class="ui-empty-title">{{ copy.emptyResult }}</p>
                    </div>
                    <div v-else class="mt-4 space-y-5">
                      <div class="grid gap-4 md:grid-cols-2">
                        <label v-for="(value, key) in resultFields" :key="key" class="grid gap-1.5">
                          <span class="ui-label break-all">{{ key }}</span>
                          <textarea
                            v-if="Array.isArray(value) || (value && typeof value === 'object')"
                            class="ui-input h-auto min-h-24 py-2.5"
                            disabled
                            :value="JSON.stringify(value, null, 2)"
                          />
                          <input
                            v-else
                            class="ui-input"
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

                  <!-- Admin actions -->
                  <aside class="rounded-2xl border border-edge p-4 md:p-5">
                    <h2 class="text-base font-semibold text-ink">{{ copy.actionTitle }}</h2>
                    <p class="mt-1 text-sm text-slate-500">{{ copy.actionDescription }}</p>
                    <label class="mt-4 grid gap-1.5">
                      <span class="ui-label">{{ copy.reason }}</span>
                      <textarea v-model="reason" class="ui-input h-auto min-h-24 py-2.5" maxlength="500" :placeholder="copy.reasonPlaceholder" />
                    </label>
                    <div class="mt-4 grid gap-3">
                      <button
                        v-for="item in actions"
                        :key="item.key"
                        class="w-full rounded-[10px] border border-slate-200 bg-white px-4 py-3 text-left shadow-sm transition-all duration-150 hover:border-slate-300 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary-500/20 disabled:pointer-events-none disabled:opacity-50"
                        type="button"
                        :disabled="loading || !canCheck"
                        @click="runAction(item)"
                      >
                        <div class="flex items-start gap-3">
                          <span class="flex h-8 w-8 shrink-0 items-center justify-center rounded-[10px]" :class="item.tone">
                            <Loader2 v-if="activeAction === item.key" class="h-4 w-4 animate-spin" />
                            <component :is="item.icon" v-else class="h-4 w-4" />
                          </span>
                          <span class="min-w-0">
                            <span class="block text-sm font-medium text-ink">{{ item.title }}</span>
                            <span class="mt-0.5 block text-xs text-slate-500">{{ item.desc }}</span>
                          </span>
                        </div>
                      </button>
                    </div>
                  </aside>
                </section>
              </div>
            </main>
          </section>
        </div>
      </Transition>
    </Teleport>
  </main>
</template>
