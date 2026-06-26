<script setup lang="ts">
import { Loader2, RefreshCw } from "lucide-vue-next"
import { onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { formatDate, type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const requests = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)

function requestUlid(request: JsonRecord) {
  return String(pickFirst(request, ["request_ulid", "request_id", "id"]) || "")
}

function statusLabel(value: unknown) {
  const status = Number(value)
  if (status === 1) return "待处理"
  if (status === 2) return "生成中"
  if (status === 3) return "成功"
  if (status === 4) return "失败"
  return String(value || "δ֪")
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/pdf-requests")
    const list = Array.isArray(data.requests) ? data.requests : []
    requests.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    selected.value = requests.value[0] || null
  } catch (err) {
    console.error(err)
    toast.error("证书生成流水加载失败")
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">证书生成流水</h1>
        <p class="mt-2 text-slate-600">查看证书 PDF 生成任务及失败原因。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
        <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
        刷新
      </button>
    </header>

    <div class="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <button
          v-for="request in requests"
          v-else
          :key="requestUlid(request)"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="selected === request ? 'bg-sky-50' : ''"
          type="button"
          @click="selected = request"
        >
          <div>
            <div class="font-black">{{ request.degree_no || requestUlid(request) || "生成请求" }}</div>
            <div class="mt-1 text-sm text-slate-500">{{ request.candidate_id || request.candidate_ulid || "-" }}</div>
            <div class="mt-1 text-xs text-slate-400">{{ formatDate(String(request.created_at || "")) }}</div>
          </div>
          <span class="h-fit rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(statusLabel(request.status))">
            {{ statusLabel(request.status) }}
          </span>
        </button>
        <div v-if="!loading && !requests.length" class="p-12 text-center text-slate-500">暂无流水</div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div v-if="!selected" class="p-10 text-center text-slate-500">请选择一条流水</div>
        <pre v-else class="max-h-[720px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
      </section>
    </div>
  </section>
</template>
