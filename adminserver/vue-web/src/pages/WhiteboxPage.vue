<script setup lang="ts">
import { computed, ref } from "vue"
import { Database, Play, RefreshCw, Search } from "lucide-vue-next"
import AdminShell from "@/components/AdminShell.vue"
import { apiClient } from "@/lib/apiClient"

type Endpoint = {
  key: string
  service: string
  name: string
  path: string
  hint: string
}

const endpoints: Endpoint[] = [
  { key: "gcc-pipelines", service: "GCC", name: "Pipeline 配置列表", path: "/api/pipelines/", hint: "课程/认证管线配置" },
  { key: "gprog-pipelines", service: "GPROG", name: "Pipeline 运行列表", path: "/api/prog/pipelines/", hint: "candidate 运行态管线" },
  { key: "glms-courses", service: "GLMS", name: "课程列表", path: "/api/lms/courses/", hint: "page_size, page_token, category_tips" },
  { key: "glms-enrollments", service: "GLMS", name: "选课列表", path: "/api/lms/enrollments", hint: "candidate_id 可选" },
  { key: "glms-assets", service: "GLMS", name: "课程资源资产", path: "/api/lms/assets", hint: "course_id/material_id 等筛选" },
  { key: "glms-objects", service: "GLMS", name: "对象存储列表", path: "/api/lms/objects", hint: "bucket/prefix/page_size" },
  { key: "glms-broken-assets", service: "GLMS", name: "异常资产", path: "/api/lms/broken-assets", hint: "排查资产缺失" },
  { key: "glms-resource-packs", service: "GLMS", name: "资源包列表", path: "/api/lms/resource-packs/", hint: "status/page_size" },
  { key: "glms-quizzes", service: "GLMS", name: "测验列表", path: "/api/lms/quizzes/", hint: "quizzable_type/quizzable_id" },
  { key: "gmsg-templates", service: "GMSG", name: "消息模板", path: "/api/messages/templates", hint: "站内消息模板" },
  { key: "gmsg-sent", service: "GMSG", name: "已发消息", path: "/api/messages/sent", hint: "candidate_id/page_size" },
  { key: "gmail-templates", service: "GMAIL", name: "邮件模板", path: "/api/mails/templates/", hint: "邮件模板列表" },
  { key: "gmail-sent", service: "GMAIL", name: "已发邮件", path: "/api/mails/sent", hint: "recipient/status/page_size" },
  { key: "gcreds-definitions", service: "GCREDS", name: "资质定义", path: "/api/credentials/definitions", hint: "证书/资质定义" },
  { key: "gcreds-applications", service: "GCREDS", name: "资质申请", path: "/api/applications/", hint: "candidate_id/status" },
  { key: "pdf-templates", service: "PDF", name: "PDF 模板", path: "/api/pdf-templates/", hint: "证书/PDF 模板" },
  { key: "pdf-requests", service: "PDF", name: "PDF 请求", path: "/api/pdf-requests/", hint: "生成请求列表" },
  { key: "gmall-orders", service: "GMALL", name: "订单列表", path: "/api/mall/orders", hint: "candidate_id/status" },
  { key: "gmall-stage-orders", service: "GMALL", name: "阶段订单", path: "/api/mall/stage-orders", hint: "stage_ulid/candidate_id" },
  { key: "gexam-webhooks", service: "GEXAM", name: "考试 Webhook", path: "/api/audit/webhooks/", hint: "第三方考试回调审计" },
  { key: "permissions", service: "GMID/GCREDS", name: "权限检查", path: "/api/permissions/check", hint: "按 query 检查 candidate 资格" },
]

const serviceFilter = ref("全部")
const keyword = ref("")
const selectedKey = ref(endpoints[0].key)
const queryText = ref("page_size=50")
const loading = ref(false)
const result = ref<any>(null)
const lastUrl = ref("")

const services = computed(() => ["全部", ...Array.from(new Set(endpoints.map((item) => item.service)))])
const filteredEndpoints = computed(() => endpoints.filter((endpoint) => {
  const matchesService = serviceFilter.value === "全部" || endpoint.service === serviceFilter.value
  const text = `${endpoint.service} ${endpoint.name} ${endpoint.path} ${endpoint.hint}`.toLowerCase()
  return matchesService && text.includes(keyword.value.trim().toLowerCase())
}))
const selectedEndpoint = computed(() => endpoints.find((endpoint) => endpoint.key === selectedKey.value) || endpoints[0])
const rows = computed(() => flattenRows(result.value))

function selectEndpoint(endpoint: Endpoint) {
  selectedKey.value = endpoint.key
  queryText.value = defaultQuery(endpoint)
  result.value = null
}

function defaultQuery(endpoint: Endpoint) {
  if (endpoint.path.includes("resource-packs")) return "page_size=50"
  if (endpoint.path.includes("objects")) return "page_size=50"
  if (endpoint.path.includes("check")) return "candidate_id="
  return "page_size=50"
}

function buildUrl() {
  const query = queryText.value.trim().replace(/^\?/, "")
  return query ? `${selectedEndpoint.value.path}?${query}` : selectedEndpoint.value.path
}

async function runQuery() {
  loading.value = true
  try {
    const url = buildUrl()
    lastUrl.value = url
    result.value = await apiClient(url)
  } finally {
    loading.value = false
  }
}

function flattenRows(value: any): any[] {
  if (!value) return []
  if (Array.isArray(value)) return value
  const preferredKeys = [
    "list",
    "items",
    "pipelines",
    "courses",
    "enrollments",
    "assets",
    "objects",
    "packs",
    "files",
    "templates",
    "messages",
    "mails",
    "definitions",
    "applications",
    "requests",
    "orders",
    "stage_orders",
    "webhooks",
    "logs",
  ]
  for (const key of preferredKeys) {
    if (Array.isArray(value[key])) return value[key]
  }
  const firstArray = Object.values(value).find(Array.isArray)
  return Array.isArray(firstArray) ? firstArray : [value]
}

function visibleColumns(row: any) {
  return Object.keys(row || {}).slice(0, 8)
}

function cellText(value: any) {
  if (value === null || value === undefined || value === "") return "-"
  if (typeof value === "object") return JSON.stringify(value)
  return String(value)
}
</script>

<template>
  <AdminShell>
    <section class="mb-6 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <p class="text-xs font-black uppercase tracking-[0.24em] text-slate-500">Whitebox</p>
        <h1 class="mt-2 text-4xl font-black">微服务白盒查看台</h1>
        <p class="mt-2 max-w-3xl text-sm leading-6 text-slate-500">
          把 adminserver 已经接出的 list/detail 查询统一放到一个页面。排查问题时先在这里看服务返回，再决定是否需要深入数据库。
        </p>
      </div>
      <button class="btn btn-primary" :disabled="loading" @click="runQuery">
        <Play class="h-4 w-4" />
        执行查询
      </button>
    </section>

    <section class="grid gap-5 xl:grid-cols-[0.82fr_1.18fr]">
      <aside class="card p-4">
        <div class="mb-4 grid gap-3 sm:grid-cols-[140px_1fr]">
          <select v-model="serviceFilter" class="select">
            <option v-for="service in services" :key="service">{{ service }}</option>
          </select>
          <label class="relative">
            <Search class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
            <input v-model="keyword" class="input pl-10" placeholder="搜索接口、服务或说明" />
          </label>
        </div>
        <div class="max-h-[70vh] space-y-3 overflow-auto pr-1">
          <button
            v-for="endpoint in filteredEndpoints"
            :key="endpoint.key"
            :class="['w-full rounded-[22px] border p-4 text-left transition', selectedKey === endpoint.key ? 'border-[var(--clay)] bg-[rgba(201,120,72,0.10)]' : 'border-[var(--line)] bg-white/60 hover:bg-white']"
            @click="selectEndpoint(endpoint)"
          >
            <span class="rounded-full bg-[var(--ink)] px-2.5 py-1 text-[11px] font-black text-white">{{ endpoint.service }}</span>
            <h2 class="mt-3 font-black">{{ endpoint.name }}</h2>
            <p class="mono mt-2 break-all text-xs text-slate-500">{{ endpoint.path }}</p>
            <p class="mt-2 text-xs leading-5 text-slate-500">{{ endpoint.hint }}</p>
          </button>
        </div>
      </aside>

      <div class="space-y-5">
        <section class="card p-5">
          <div class="mb-4 flex items-start justify-between gap-4">
            <div>
              <div class="mb-2 inline-flex items-center gap-2 rounded-full bg-[rgba(95,111,82,0.12)] px-3 py-1 text-xs font-black text-[var(--moss)]">
                <Database class="h-3.5 w-3.5" />
                {{ selectedEndpoint.service }}
              </div>
              <h2 class="text-2xl font-black">{{ selectedEndpoint.name }}</h2>
              <p class="mono mt-2 break-all text-xs text-slate-500">{{ selectedEndpoint.path }}</p>
            </div>
            <button class="btn btn-outline" :disabled="loading" @click="runQuery">
              <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />
              查询
            </button>
          </div>
          <label>
            <span class="label">Query String</span>
            <input v-model="queryText" class="input mono" placeholder="page_size=50&status=Active" @keyup.enter="runQuery" />
          </label>
          <p v-if="lastUrl" class="mono mt-3 rounded-2xl bg-white/60 p-3 text-xs text-slate-600">{{ lastUrl }}</p>
        </section>

        <section class="card p-5">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-2xl font-black">表格预览</h2>
            <span class="text-sm font-black text-slate-500">{{ rows.length }} rows</span>
          </div>
          <div v-if="rows.length" class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th v-for="column in visibleColumns(rows[0])" :key="column">{{ column }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, index) in rows" :key="index">
                  <td v-for="column in visibleColumns(rows[0])" :key="column" class="max-w-[260px] truncate">{{ cellText(row[column]) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="rounded-[22px] border border-dashed border-[var(--line)] p-8 text-center text-sm text-slate-500">
            还没有查询结果
          </div>
        </section>

        <section class="card p-5">
          <h2 class="text-2xl font-black">原始 JSON</h2>
          <pre class="mono mt-4 max-h-[460px] overflow-auto rounded-[22px] bg-[var(--ink)] p-4 text-xs leading-6 text-[#f7efe1]">{{ JSON.stringify(result, null, 2) }}</pre>
        </section>
      </div>
    </section>
  </AdminShell>
</template>
