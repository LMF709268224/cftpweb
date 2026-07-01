<script setup lang="ts">
import { ChevronLeft, ChevronRight, FileBadge, Loader2, RefreshCw, X } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, getDisplayTitle, humanizeKey, isPrimitive, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const PAGE_SIZE = 10

const requests = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailOpen = ref(false)
const page = ref(1)
const total = ref(0)

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

const detailFields = computed(() => {
  if (!selected.value) return []
  return Object.entries(selected.value)
    .filter(([, value]) => isPrimitive(value))
    .map(([key, value]) => ({
      key,
      label: fieldLabel(key),
      value: formatFieldValue(key, value),
    }))
})

function fieldLabel(key: string) {
  const labels: Record<string, string> = {
    request_ulid: "请求 ID",
    request_id: "请求 ID",
    id: "ID",
    degree_no: "证书编号",
    status: "状态",
    candidate_id: "考生 ID",
    candidate_ulid: "考生 ID",
    pipeline_cc: "管线实例",
    pipeline_ulid: "管线实例",
    cert_ulid: "证书定义",
    qual_ulid: "资格定义",
    pdf_template_ulid: "PDF 模板",
    file_object_key: "文件 Object Key",
    file_hash: "文件 Hash",
    error_message: "失败原因",
    created_at: "创建时间",
    updated_at: "更新时间",
  }
  return labels[key] || humanizeKey(key)
}

function requestUlid(request: JsonRecord) {
  return String(pickFirst(request, ["request_ulid", "request_id", "id", "pdf_request_ulid"]) || "")
}

function statusLabel(value: unknown) {
  const status = Number(value)
  if (status === 1) return "待处理"
  if (status === 2) return "生成中"
  if (status === 3) return "成功"
  if (status === 4) return "失败"
  return String(value || "未知")
}

function formatFieldValue(key: string, value: unknown) {
  if (key.endsWith("_at")) return formatDate(value)
  if (value === null || value === undefined || value === "") return "-"
  return String(value)
}

function openRequest(request: JsonRecord | null, open = true) {
  selected.value = request
  detailOpen.value = !!request && open
}

function closeDetail() {
  detailOpen.value = false
}

async function load(nextPage = page.value) {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>(`/api/pdf-requests?page=${nextPage}&page_size=${PAGE_SIZE}`)
    const list = Array.isArray(data.requests) ? data.requests : []
    const selectedId = selected.value ? requestUlid(selected.value) : ""

    page.value = Number(data.page || nextPage)
    total.value = Number(data.total || list.length)
    requests.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    selected.value = requests.value.find((item) => requestUlid(item) === selectedId) || requests.value[0] || null
    if (!selected.value) detailOpen.value = false
  } catch (err) {
    console.error(err)
    requests.value = []
    selected.value = null
    detailOpen.value = false
    toast.error("证书生成流水加载失败")
  } finally {
    loading.value = false
  }
}

function goPage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value || nextPage === page.value) return
  load(nextPage)
}

onMounted(() => load(1))
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">证书生成流水</h1>
        <p class="mt-2 text-slate-600">查看证书 PDF 生成任务及失败原因；当前后台只暴露列表查询，所以详情为只读展示。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load()">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
      <div class="flex items-center justify-between border-b border-slate-200 p-5">
        <div>
          <h2 class="text-xl font-black">流水列表</h2>
          <p class="mt-1 text-sm text-slate-500">每页 10 条；点击查看详情后在弹框中查看完整字段。</p>
        </div>
        <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">共 {{ total }} 条</span>
      </div>
      <div class="grid grid-cols-[minmax(0,1fr)_160px_180px_112px] gap-5 border-b border-slate-200 bg-slate-50 px-5 py-3 text-xs font-black text-slate-500">
        <span>流水</span>
        <span class="text-center">状态</span>
        <span class="text-right">创建时间</span>
        <span class="text-right">操作</span>
      </div>

      <div v-if="loading" class="p-12 text-center text-slate-500">
        <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
        正在加载...
      </div>
      <div v-else-if="!requests.length" class="p-12 text-center text-slate-500">暂无流水</div>
      <div v-else class="divide-y divide-slate-100">
        <div
          v-for="request in requests"
          :key="requestUlid(request)"
          class="grid cursor-pointer grid-cols-[minmax(0,1fr)_160px_180px_112px] items-center gap-5 px-5 py-4 transition hover:bg-sky-50"
          :class="requestUlid(selected || {}) === requestUlid(request) ? 'bg-sky-50' : ''"
          role="button"
          tabindex="0"
          @click="openRequest(request)"
          @keydown.enter.prevent="openRequest(request)"
          @keydown.space.prevent="openRequest(request)"
        >
          <div class="min-w-0">
            <div class="truncate text-base font-black">{{ getDisplayTitle(request) }}</div>
            <div class="mt-1 truncate text-sm font-bold text-blue-700">{{ requestUlid(request) || "-" }}</div>
          </div>
          <div class="min-w-0 text-center">
            <span class="inline-flex max-w-full truncate rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(request.status))">
              {{ statusLabel(request.status) }}
            </span>
          </div>
          <div class="text-right text-sm font-semibold text-slate-500">{{ formatDate(request.created_at) || "无创建时间" }}</div>
          <div class="text-right">
            <button
              class="rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-black text-[#0b4ea2] shadow-sm transition hover:border-sky-200 hover:bg-sky-50"
              type="button"
              @click.stop="openRequest(request)"
            >
              查看详情
            </button>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3 border-t border-slate-200 p-5">
        <span class="text-sm font-bold text-slate-500">第 {{ page }} / {{ totalPages }} 页</span>
        <div class="flex gap-3">
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page <= 1 || loading"
            @click="goPage(page - 1)"
          >
            <ChevronLeft class="h-4 w-4" />
            上一页
          </button>
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page >= totalPages || loading"
            @click="goPage(page + 1)"
          >
            下一页
            <ChevronRight class="h-4 w-4" />
          </button>
        </div>
      </div>
    </section>

    <Teleport to="body">
      <div v-if="detailOpen && selected" class="fixed inset-0 z-40 flex items-center justify-center bg-slate-950/50 p-6">
        <section class="flex max-h-[88vh] w-full max-w-[1120px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl">
          <div class="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-5">
            <div class="flex min-w-0 items-start gap-3">
              <span class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-blue-50 text-blue-700">
                <FileBadge class="h-5 w-5" />
              </span>
              <div class="min-w-0">
                <h2 class="text-2xl font-black text-slate-950">流水详情</h2>
              </div>
            </div>
            <button
              class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full border border-slate-200 bg-white text-slate-500 shadow-sm transition hover:bg-slate-50 hover:text-slate-900"
              type="button"
              aria-label="关闭"
              @click="closeDetail"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="min-h-0 flex-1 space-y-5 overflow-y-auto p-5">
            <div class="rounded-2xl border border-blue-100 bg-blue-50 p-4">
              <div class="text-sm font-black text-blue-700">当前流水</div>
              <div class="mt-1 break-all text-lg font-black text-slate-950">{{ requestUlid(selected) || "-" }}</div>
              <div class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(selected.status))">
                {{ statusLabel(selected.status) }}
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <label v-for="field in detailFields" :key="field.key" class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <span class="text-xs font-black uppercase tracking-wide text-slate-400">{{ field.label }}</span>
                <input class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-3 py-2 text-sm font-bold text-slate-700" :value="field.value" disabled />
              </label>
            </div>

            <div class="rounded-2xl border border-slate-200">
              <div class="border-b border-slate-100 px-4 py-3 text-sm font-black">完整原始字段</div>
              <pre class="max-h-[460px] overflow-auto rounded-b-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
            </div>
          </div>
        </section>
      </div>
    </Teleport>
  </section>
</template>
