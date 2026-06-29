<script setup lang="ts">
import { CheckCircle2, Eye, Loader2, RefreshCw, RotateCcw, XCircle } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { applicationStatusLabel, applicationStatusOptions, badgeClass, pickFirst } from "@/lib/status"

const applications = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const auditing = ref(false)
const page = ref(1)
const total = ref(0)
const statusFilter = ref("0")
const auditRemark = ref("")
const pageSize = 20

const canPrev = computed(() => page.value > 1)
const canNext = computed(() => applications.value.length >= pageSize)

function appUlid(app: JsonRecord) {
  return String(pickFirst(app, ["app_ulid", "app_id", "application_ulid", "application_id"]) || "")
}

function candidate(app: JsonRecord) {
  return String(pickFirst(app, ["candidate_name", "candidate_email", "candidate_ulid", "candidate_id"]) || "-")
}

function credential(app: JsonRecord) {
  return String(pickFirst(app, ["cred_def_name", "credential_name", "cred_def_ulid", "cred_def_id"]) || "-")
}

function status(app: JsonRecord) {
  return pickFirst(app, ["status", "application_status"])
}

function files(app: JsonRecord) {
  const value = app.files
  return Array.isArray(value) ? value : []
}

async function load(targetPage = page.value) {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/applications?page_number=${targetPage}&page_size=${pageSize}&status=${statusFilter.value}`)
    const list = Array.isArray(data.applications) ? data.applications : []
    applications.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    total.value = Number(data.total || applications.value.length) || 0
    selected.value = applications.value[0] || null
    page.value = targetPage
  } catch (err) {
    console.error(err)
    applications.value = []
    selected.value = null
    total.value = 0
    toast.error("申请列表加载失败")
  } finally {
    loading.value = false
  }
}

async function audit(action: "approve" | "reject" | "resubmit") {
  if (!selected.value) return
  if ((action === "reject" || action === "resubmit") && !auditRemark.value.trim()) {
    toast.error("拒绝或要求补交时需要填写审核备注")
    return
  }

  auditing.value = true
  try {
    await apiClient("/api/applications/audit", {
      method: "POST",
      body: JSON.stringify({
        application_id: appUlid(selected.value),
        approved: action === "approve",
        reject_reason: auditRemark.value,
        require_resubmit: action === "resubmit",
      }),
    })
    toast.success("审核已提交")
    auditRemark.value = ""
    await load(page.value)
  } catch (err) {
    console.error(err)
    toast.error("审核提交失败")
  } finally {
    auditing.value = false
  }
}

watch(statusFilter, () => load(1))
onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">审核中心</h1>
        <p class="mt-2 text-slate-600">审核考生提交的资格申请材料。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load(page)">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <div class="flex items-center gap-4 rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
      <select v-model="statusFilter" class="rounded-xl border border-slate-200 px-4 py-3">
        <option v-for="option in applicationStatusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <span class="text-sm font-bold text-slate-500">共 {{ total }} 条</span>
    </div>

    <div class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-200 p-5">
          <h2 class="text-xl font-black">申请列表</h2>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <div v-else-if="!applications.length" class="p-12 text-center text-slate-500">暂无申请</div>
        <button
          v-for="app in applications"
          v-else
          :key="appUlid(app)"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="selected === app ? 'bg-sky-50' : ''"
          type="button"
          @click="selected = app"
        >
          <div class="min-w-0">
            <div class="font-black text-slate-950">{{ credential(app) }}</div>
            <div class="mt-1 text-sm text-slate-500">{{ candidate(app) }}</div>
            <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(app.created_at || "")) }}</div>
          </div>
          <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(app))">
            {{ applicationStatusLabel(status(app)) }}
          </span>
        </button>
        <div class="flex justify-end gap-3 border-t border-slate-200 p-5">
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="load(page - 1)">上一页</button>
          <button class="rounded-xl border px-4 py-2 font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="load(page + 1)">下一页</button>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div v-if="!selected" class="p-10 text-center text-slate-500">请选择一条申请</div>
        <template v-else>
          <div class="mb-5 flex items-start justify-between gap-4">
            <div>
              <h2 class="text-xl font-black">{{ credential(selected) }}</h2>
              <p class="mt-1 text-sm text-slate-500">{{ candidate(selected) }}</p>
            </div>
            <span class="rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(status(selected))">
              {{ applicationStatusLabel(status(selected)) }}
            </span>
          </div>

          <div class="mb-5 rounded-2xl bg-slate-50 p-4">
            <div class="mb-3 text-sm font-black">申请材料</div>
            <div v-if="!files(selected).length" class="text-sm text-slate-500">暂无文件</div>
            <a
              v-for="file in files(selected)"
              v-else
              :key="String(file.file_name || file.name || file.file_hash)"
              class="mb-2 flex items-center justify-between rounded-xl border border-slate-200 bg-white px-4 py-3 text-sm font-bold text-slate-700 hover:bg-slate-50"
              :href="String(file.view_url || file.url || '#')"
              target="_blank"
            >
              {{ file.file_name || file.name || "文件" }}
              <Eye class="h-4 w-4" />
            </a>
          </div>

          <textarea
            v-model="auditRemark"
            class="mb-4 min-h-28 w-full rounded-2xl border border-slate-200 p-4 text-sm"
            placeholder="审核备注：打回重提或最终拒绝时必填。需要用户重新提交材料时请选择“打回重提”。"
          />
          <div class="grid gap-3 md:grid-cols-3">
            <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('approve')">
              <CheckCircle2 class="h-4 w-4" />
              通过
            </button>
            <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-amber-500 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('resubmit')">
              <RotateCcw class="h-4 w-4" />
              打回重提（允许再次提交）
            </button>
            <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="auditing" @click="audit('reject')">
              <XCircle class="h-4 w-4" />
              最终拒绝
            </button>
          </div>

          <pre class="mt-5 max-h-[420px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
        </template>
      </section>
    </div>
  </section>
</template>
