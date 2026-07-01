<script setup lang="ts">
import { ChevronLeft, ChevronRight, FileBadge, Loader2, RefreshCw } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, getDisplayTitle, humanizeKey, isPrimitive, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const PAGE_SIZE = 10

const requests = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
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

function selectRequest(request: JsonRecord) {
  selected.value = request
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
  } catch (err) {
    console.error(err)
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

    <div class="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">流水列表</h2>
            <p class="mt-1 text-sm text-slate-500">每页 10 条；左侧选择流水，右侧查看完整字段。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">共 {{ total }} 条</span>
        </div>

        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <div v-else-if="!requests.length" class="p-12 text-center text-slate-500">暂无流水</div>
        <button
          v-for="request in requests"
          v-else
          :key="requestUlid(request)"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="selected === request ? 'bg-sky-50' : ''"
          type="button"
          @click="selectRequest(request)"
        >
          <div class="min-w-0">
            <div class="truncate text-base font-black">{{ getDisplayTitle(request) }}</div>
            <div class="mt-1 truncate text-sm font-bold text-blue-700">{{ requestUlid(request) || "-" }}</div>
            <div class="mt-1 text-xs text-slate-500">{{ formatDate(request.created_at) || "无创建时间" }}</div>
          </div>
          <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(request.status))">
            {{ statusLabel(request.status) }}
          </span>
        </button>

        <div class="flex items-center justify-end gap-3 border-t border-slate-100 px-5 py-4">
          <button
            class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold disabled:cursor-not-allowed disabled:opacity-40"
            type="button"
            :disabled="page <= 1 || loading"
            @click="goPage(page - 1)"
          >
            <ChevronLeft class="h-4 w-4" />
            上一页
          </button>
          <span class="text-sm font-bold text-slate-500">第 {{ page }} / {{ totalPages }} 页</span>
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
      </section>

      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">流水详情</h2>
            <p class="mt-1 text-sm text-slate-500">这些字段由列表接口返回，当前页面不提供修改操作。</p>
          </div>
          <FileBadge class="h-5 w-5 text-blue-600" />
        </div>

        <div v-if="!selected" class="p-12 text-center text-slate-500">请选择一条流水</div>
        <div v-else class="space-y-5 p-5">
          <div class="rounded-2xl bg-blue-50 p-4">
            <div class="text-sm font-black text-blue-700">当前流水</div>
            <div class="mt-1 break-all text-lg font-black text-slate-950">{{ requestUlid(selected) || "-" }}</div>
            <div class="mt-2 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(selected.status))">
              {{ statusLabel(selected.status) }}
            </div>
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <label v-for="field in detailFields" :key="field.key" class="rounded-2xl bg-slate-50 p-4">
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
  </section>
</template>
