<script setup lang="ts">
import { ArrowLeft, CheckCircle2, ClipboardList, Loader2, PlayCircle, RefreshCw, Search } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient, ApiError } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const pageSize = 10

const exams = ref<JsonRecord[]>([])
const selectedSummary = ref<JsonRecord | null>(null)
const detail = ref<JsonRecord | null>(null)
const result = ref<JsonRecord | null>(null)
const transitions = ref<JsonRecord[]>([])
const loading = ref(false)
const detailLoading = ref(false)
const actionLoading = ref(false)
const page = ref(1)
const total = ref(0)

const statusFilter = ref("")
const resultStatusFilter = ref("")
const candidateFilter = ref("")
const confirmationFilter = ref("")
const courseUnitFilter = ref("")

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))
const canPrev = computed(() => page.value > 1)
const canNext = computed(() => page.value < totalPages.value)
const selectedExamUlid = computed(() => examUlid(detail.value || selectedSummary.value))
const candidateName = computed(() => {
  const source = detail.value || selectedSummary.value || {}
  return [source.candidate_first_name, source.candidate_middle_name, source.candidate_last_name].filter(Boolean).join(" ") || "-"
})

const statusOptions = [
  { value: "", label: "全部流程状态" },
  { value: "OPEN", label: "开放 / Open" },
  { value: "SCHEDULED", label: "已预约 / Scheduled" },
  { value: "DONE", label: "已完成 / Done" },
  { value: "CANCELLED", label: "已取消 / Cancelled" },
]

const resultOptions = [
  { value: "", label: "全部成绩状态" },
  { value: "PENDING", label: "待出分 / Pending" },
  { value: "PASS", label: "通过 / Pass" },
  { value: "FAIL", label: "未通过 / Fail" },
]

function asArray(value: unknown): JsonRecord[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    : []
}

function examUlid(item: JsonRecord | null | undefined) {
  return String(pickFirst(item || {}, ["exam_ulid", "exam_id"]) || "")
}

function label(value: unknown) {
  return value === undefined || value === null || value === "" ? "-" : String(value)
}

function examStatusLabel(value: unknown) {
  const status = String(value || "").toUpperCase()
  if (status === "OPEN") return "开放 / Open"
  if (status === "SCHEDULED") return "已预约 / Scheduled"
  if (status === "DONE") return "已完成 / Done"
  if (status === "CANCELLED") return "已取消 / Cancelled"
  return status || "-"
}

function resultStatusLabel(value: unknown, passed?: unknown) {
  const status = String(value || "").toUpperCase()
  if (status === "PASS" || passed === true) return "通过 / Pass"
  if (status === "FAIL" || passed === false && status) return "未通过 / Fail"
  if (status === "PENDING") return "待出分 / Pending"
  return status || "-"
}

function candidateDisplay(item: JsonRecord) {
  const name = [item.candidate_first_name, item.candidate_middle_name, item.candidate_last_name].filter(Boolean).join(" ")
  return name || String(item.candidate_email || item.candidate_ulid || "考生")
}

function scoreDetails(source: JsonRecord | null) {
  const raw = String(source?.score_details_json || "")
  if (!raw) return "-"
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

async function loadExams(targetPage = page.value) {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(targetPage),
      page_size: String(pageSize),
    })
    if (statusFilter.value) params.set("status", statusFilter.value)
    if (resultStatusFilter.value) params.set("result_status", resultStatusFilter.value)
    if (candidateFilter.value.trim()) params.set("candidate_ulid", candidateFilter.value.trim())
    if (confirmationFilter.value.trim()) params.set("confirmation_number", confirmationFilter.value.trim())
    if (courseUnitFilter.value.trim()) params.set("course_unit_ulid", courseUnitFilter.value.trim())

    const data = await apiClient<JsonRecord>(`/api/exams?${params}`)
    exams.value = asArray(data.exams)
    total.value = Number(data.total || exams.value.length || 0)
    page.value = targetPage
    if (selectedExamUlid.value && !exams.value.some((item) => examUlid(item) === selectedExamUlid.value)) {
      clearSelection()
    }
  } catch (err) {
    console.error(err)
    exams.value = []
    total.value = 0
    toast.error("考试列表加载失败")
  } finally {
    loading.value = false
  }
}

async function openExam(item: JsonRecord) {
  selectedSummary.value = item
  detail.value = null
  result.value = null
  transitions.value = []
  await loadExamDetail(examUlid(item))
}

async function loadExamDetail(id: string) {
  if (!id) return
  detailLoading.value = true
  try {
    const [detailData, resultData, transitionsData] = await Promise.all([
      apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(id)}`),
      loadExamResult(id),
      apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(id)}/transitions`),
    ])
    detail.value = detailData
    result.value = resultData
    transitions.value = asArray(transitionsData.transitions)
  } catch (err) {
    console.error(err)
    toast.error("考试详情加载失败")
  } finally {
    detailLoading.value = false
  }
}

async function loadExamResult(id: string) {
  try {
    return await apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(id)}/result`)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    console.error(err)
    return null
  }
}

async function syncExamResult() {
  if (!selectedExamUlid.value) return
  actionLoading.value = true
  try {
    result.value = await apiClient<JsonRecord>(`/api/exams/${encodeURIComponent(selectedExamUlid.value)}/sync-result`, { method: "POST" })
    toast.success("已触发考试结果同步")
    await loadExamDetail(selectedExamUlid.value)
    await loadExams(page.value)
  } catch (err) {
    console.error(err)
    toast.error("考试结果同步失败")
  } finally {
    actionLoading.value = false
  }
}

async function search() {
  clearSelection()
  await loadExams(1)
}

async function refreshAll() {
  await loadExams(page.value)
  if (selectedExamUlid.value) await loadExamDetail(selectedExamUlid.value)
}

function clearSelection() {
  selectedSummary.value = null
  detail.value = null
  result.value = null
  transitions.value = []
}

function changePage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value) return
  loadExams(nextPage)
}

function field(source: JsonRecord | null, keys: string[]) {
  return label(pickFirst(source || {}, keys))
}

onMounted(() => loadExams(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1600px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">考试管理</h1>
        <p class="mt-2 text-slate-600">查看 gexam 考试实例、预约信息、成绩和状态流转；可触发结果同步。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：list/get detail/get result/get transitions/sync result。没有接入删除或清空类操作。
        </p>
      </div>
      <div class="flex flex-wrap gap-3">
        <button v-if="selectedSummary" class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="clearSelection">
          <ArrowLeft class="h-4 w-4" />
          返回列表
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="refreshAll">
          <RefreshCw class="h-4 w-4" :class="loading || detailLoading ? 'animate-spin' : ''" />
          刷新
        </button>
      </div>
    </header>

    <section class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div class="grid gap-4 xl:grid-cols-[1fr_1fr_1.2fr_1.2fr_1.2fr_auto]">
        <label class="grid gap-2 text-sm font-bold">
          流程状态
          <select v-model="statusFilter" class="h-11 rounded-xl border border-slate-200 px-3">
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          成绩状态
          <select v-model="resultStatusFilter" class="h-11 rounded-xl border border-slate-200 px-3">
            <option v-for="option in resultOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
          </select>
        </label>
        <label class="grid gap-2 text-sm font-bold">
          Candidate ULID
          <input v-model="candidateFilter" class="h-11 rounded-xl border border-slate-200 px-3" placeholder="按考生筛选" />
        </label>
        <label class="grid gap-2 text-sm font-bold">
          确认号
          <input v-model="confirmationFilter" class="h-11 rounded-xl border border-slate-200 px-3" placeholder="Confirmation Number" />
        </label>
        <label class="grid gap-2 text-sm font-bold">
          课程单元 ULID
          <input v-model="courseUnitFilter" class="h-11 rounded-xl border border-slate-200 px-3" placeholder="Course Unit ULID" />
        </label>
        <button class="mt-7 inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-5 text-sm font-black text-white shadow-sm" type="button" @click="search">
          <Search class="h-4 w-4" />
          查询
        </button>
      </div>
    </section>

    <div class="grid gap-6 xl:grid-cols-[540px_minmax(0,1fr)]">
      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">考试列表</h2>
            <p class="mt-1 text-sm text-slate-500">每页 10 条；左侧选择考试，右侧查看详情和状态流转。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">共 {{ total }} 条</span>
        </div>

        <div v-if="loading" class="px-6 py-14 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载考试...
        </div>
        <div v-else-if="!exams.length" class="px-6 py-14 text-center text-slate-500">暂无考试记录</div>
        <div v-else>
          <button
            v-for="exam in exams"
            :key="examUlid(exam)"
            class="grid w-full gap-3 border-b border-slate-100 px-5 py-4 text-left transition last:border-b-0 hover:bg-slate-50"
            :class="examUlid(exam) === selectedExamUlid ? 'bg-sky-50' : ''"
            type="button"
            @click="openExam(exam)"
          >
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0">
                <h3 class="truncate text-lg font-black">{{ field(exam, ["exam_code", "program_code", "exam_ulid"]) }}</h3>
                <p class="mt-1 truncate text-sm text-slate-500">{{ candidateDisplay(exam) }}</p>
              </div>
              <span class="shrink-0 rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(exam.exam_status)">
                {{ examStatusLabel(exam.exam_status) }}
              </span>
            </div>
            <div class="flex flex-wrap gap-2 text-xs font-bold text-slate-500">
              <span class="rounded-full bg-slate-100 px-2.5 py-1">成绩：{{ resultStatusLabel(exam.result_status, exam.is_passed) }}</span>
              <span class="rounded-full bg-slate-100 px-2.5 py-1">确认号：{{ label(exam.confirmation_number) }}</span>
              <span class="rounded-full bg-slate-100 px-2.5 py-1">预约：{{ formatDate(String(exam.appointment_start_time || "")) || "-" }}</span>
            </div>
            <p class="truncate font-mono text-xs font-bold text-blue-700">{{ examUlid(exam) }}</p>
          </button>
        </div>

        <div class="flex items-center justify-end gap-3 border-t border-slate-200 px-5 py-4">
          <button class="rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="!canPrev" @click="changePage(page - 1)">上一页</button>
          <span class="text-sm font-bold text-slate-600">第 {{ page }} / {{ totalPages }} 页</span>
          <button class="rounded-xl border px-4 py-2 text-sm font-bold disabled:opacity-40" type="button" :disabled="!canNext" @click="changePage(page + 1)">下一页</button>
        </div>
      </section>

      <section class="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">考试详情</h2>
            <p class="mt-1 text-sm text-slate-500">展示考试归属、预约、成绩和状态流转。</p>
          </div>
          <button
            v-if="selectedExamUlid"
            class="inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-4 py-2.5 text-sm font-black text-white shadow-sm disabled:opacity-50"
            type="button"
            :disabled="actionLoading"
            @click="syncExamResult"
          >
            <PlayCircle class="h-4 w-4" :class="actionLoading ? 'animate-spin' : ''" />
            同步/模拟考试结果
          </button>
        </div>

        <div v-if="detailLoading" class="px-6 py-16 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载详情...
        </div>
        <div v-else-if="!selectedSummary" class="px-6 py-16 text-center text-slate-500">
          <ClipboardList class="mx-auto mb-3 h-10 w-10 text-slate-300" />
          请选择左侧一条考试记录
        </div>
        <div v-else class="space-y-5 p-5">
          <div class="rounded-2xl bg-blue-50 p-5">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p class="text-xs font-black uppercase text-blue-600">当前考试</p>
                <h3 class="mt-1 break-all text-2xl font-black">{{ field(detail || selectedSummary, ["exam_code", "exam_ulid"]) }}</h3>
                <p class="mt-1 break-all font-mono text-sm font-bold text-blue-800">{{ selectedExamUlid }}</p>
              </div>
              <span class="rounded-full border px-3 py-1 text-sm font-black" :class="badgeClass((detail || selectedSummary)?.exam_status)">
                {{ examStatusLabel((detail || selectedSummary)?.exam_status) }}
              </span>
            </div>
          </div>

          <div class="grid gap-4 2xl:grid-cols-2">
            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 text-lg font-black">归属关系</h4>
              <div class="grid gap-3">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">管线实例 / Pipeline</div>
                  <div class="mt-1 break-all font-mono text-sm font-bold">{{ field(detail, ["pipeline_ulid"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">课程单元 / Course Unit</div>
                  <div class="mt-1 break-all font-mono text-sm font-bold">{{ field(detail, ["course_unit_ulid"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">认证项目 / Certification</div>
                  <div class="mt-1 font-bold">{{ field(detail, ["certification_name"]) }}</div>
                </div>
              </div>
            </article>

            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 text-lg font-black">考生信息</h4>
              <div class="grid gap-3 sm:grid-cols-2">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">姓名</div>
                  <div class="mt-1 font-bold">{{ candidateName }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">邮箱</div>
                  <div class="mt-1 break-all font-bold">{{ field(detail || selectedSummary, ["candidate_email"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3 sm:col-span-2">
                  <div class="text-xs font-black uppercase text-slate-400">Candidate ULID</div>
                  <div class="mt-1 break-all font-mono text-sm font-bold">{{ field(detail || selectedSummary, ["candidate_ulid"]) }}</div>
                </div>
              </div>
            </article>
          </div>

          <div class="grid gap-4 2xl:grid-cols-2">
            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 text-lg font-black">预约信息</h4>
              <div class="grid gap-3 sm:grid-cols-2">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">确认号</div>
                  <div class="mt-1 break-all font-bold">{{ field(detail || selectedSummary, ["confirmation_number"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">考点</div>
                  <div class="mt-1 font-bold">{{ field(detail || selectedSummary, ["site_name"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">开始时间</div>
                  <div class="mt-1 font-bold">{{ formatDate(String((detail || selectedSummary)?.appointment_start_time || "")) || "-" }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">结束时间</div>
                  <div class="mt-1 font-bold">{{ formatDate(String((detail || selectedSummary)?.appointment_end_time || "")) || "-" }}</div>
                </div>
              </div>
            </article>

            <article class="rounded-2xl border border-slate-200 p-4">
              <h4 class="mb-4 flex items-center gap-2 text-lg font-black">
                成绩信息
                <CheckCircle2 v-if="(result || detail)?.is_passed === true" class="h-5 w-5 text-emerald-500" />
              </h4>
              <div class="grid gap-3 sm:grid-cols-3">
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">成绩状态</div>
                  <div class="mt-1 font-bold">{{ resultStatusLabel((result || detail || selectedSummary)?.result_status, (result || detail || selectedSummary)?.is_passed) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">总分</div>
                  <div class="mt-1 font-bold">{{ field(result || detail, ["total_score"]) }}</div>
                </div>
                <div class="rounded-xl bg-slate-50 p-3">
                  <div class="text-xs font-black uppercase text-slate-400">是否通过</div>
                  <div class="mt-1 font-bold">{{ (result || detail)?.is_passed === true ? "是 / Yes" : (result || detail)?.is_passed === false ? "否 / No" : "-" }}</div>
                </div>
              </div>
              <pre class="mt-3 max-h-52 overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-100">{{ scoreDetails(result || detail) }}</pre>
            </article>
          </div>

          <article class="rounded-2xl border border-slate-200">
            <div class="border-b border-slate-200 px-4 py-3">
              <h4 class="text-lg font-black">状态流转</h4>
              <p class="mt-1 text-sm text-slate-500">来自 gexam 状态变更记录。</p>
            </div>
            <div v-if="!transitions.length" class="px-4 py-8 text-center text-slate-500">暂无状态流转记录</div>
            <div v-else class="divide-y divide-slate-100">
              <div v-for="transition in transitions" :key="String(transition.msg_fp || transition.transitioned_at)" class="grid gap-3 px-4 py-4 lg:grid-cols-[160px_minmax(0,1fr)_180px]">
                <div>
                  <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-600">{{ label(transition.status_type) }}</span>
                  <p class="mt-2 font-bold">{{ label(transition.event_type) }}</p>
                </div>
                <div class="min-w-0">
                  <p class="font-bold">{{ label(transition.from_status) }} → {{ label(transition.to_status) }}</p>
                  <p class="mt-1 break-all font-mono text-xs text-blue-700">{{ label(transition.msg_fp) }}</p>
                </div>
                <div class="text-sm font-bold text-slate-500">{{ formatDate(String(transition.transitioned_at || transition.created_at || "")) || "-" }}</div>
              </div>
            </div>
          </article>

          <details class="rounded-2xl border border-slate-200 p-4">
            <summary class="cursor-pointer text-sm font-black text-slate-700">完整字段（排查用）</summary>
            <pre class="mt-4 max-h-96 overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-100">{{ JSON.stringify({ detail: detail || selectedSummary, result, transitions }, null, 2) }}</pre>
          </details>
        </div>
      </section>
    </div>
  </section>
</template>
