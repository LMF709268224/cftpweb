<script setup lang="ts">
import { Loader2, Search, ShieldAlert, ShieldCheck, UserX } from "lucide-vue-next"
import { onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { type JsonRecord } from "@/lib/display"
import { badgeClass, pickFirst } from "@/lib/status"

const definitions = ref<JsonRecord[]>([])
const candidateUlid = ref("")
const credDefUlid = ref("")
const reason = ref("")
const checkResult = ref<JsonRecord | null>(null)
const loading = ref(false)

function definitionUlid(definition: JsonRecord) {
  return String(pickFirst(definition, ["cred_def_ulid", "cred_def_id", "qual_ulid"]) || "")
}

function definitionName(definition: JsonRecord) {
  return String(pickFirst(definition, ["name", "name_hint", "title"]) || definitionUlid(definition))
}

async function loadDefinitions() {
  try {
    const data = await apiClient<JsonRecord>("/api/credentials/definitions")
    const list = Array.isArray(data.definitions) ? data.definitions : []
    definitions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
  } catch (err) {
    console.error(err)
    toast.error("资格定义加载失败")
  }
}

async function check() {
  if (!candidateUlid.value.trim() || !credDefUlid.value) {
    toast.error("请填写候选人 ULID 并选择资格定义")
    return
  }
  loading.value = true
  try {
    checkResult.value = await apiClient<JsonRecord>(
      `/api/permissions/check?candidate_id=${encodeURIComponent(candidateUlid.value.trim())}&cred_def_id=${encodeURIComponent(credDefUlid.value)}`,
    )
  } catch (err) {
    console.error(err)
    toast.error("权限检查失败")
  } finally {
    loading.value = false
  }
}

async function action(endpoint: string) {
  if (!reason.value.trim()) {
    toast.error("请填写操作原因")
    return
  }
  loading.value = true
  try {
    await apiClient(endpoint, {
      method: "POST",
      body: JSON.stringify({
        candidate_id: candidateUlid.value.trim(),
        cred_def_id: credDefUlid.value,
        reason: reason.value.trim(),
      }),
    })
    toast.success("操作成功")
    reason.value = ""
    await check()
  } catch (err) {
    console.error(err)
    toast.error("操作失败")
  } finally {
    loading.value = false
  }
}

onMounted(loadDefinitions)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header>
      <h1 class="text-4xl font-black tracking-tight">考生权限管理</h1>
      <p class="mt-2 text-slate-600">检查和调整考生资格权限。</p>
    </header>

    <div class="grid gap-6 xl:grid-cols-[0.8fr_1.2fr]">
      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 class="mb-4 text-xl font-black">查询目标</h2>
        <div class="grid gap-4">
          <input v-model="candidateUlid" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="Candidate ULID" />
          <select v-model="credDefUlid" class="rounded-xl border border-slate-200 px-4 py-3">
            <option value="">选择资格定义</option>
            <option v-for="definition in definitions" :key="definitionUlid(definition)" :value="definitionUlid(definition)">
              {{ definitionName(definition) }}
            </option>
          </select>
          <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="loading" @click="check">
            <Loader2 v-if="loading" class="h-4 w-4 animate-spin" />
            <Search v-else class="h-4 w-4" />
            检查权限
          </button>
        </div>

        <div v-if="checkResult" class="mt-6 rounded-2xl border border-slate-200 bg-slate-50 p-4">
          <h3 class="mb-3 font-black">管理操作</h3>
          <input v-model="reason" class="mb-3 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="操作原因" />
          <div class="grid gap-3">
            <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-3 font-bold text-white" type="button" @click="action('/api/permissions/grant')">
              <ShieldCheck class="h-4 w-4" />
              授予权限
            </button>
            <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-amber-500 px-4 py-3 font-bold text-white" type="button" @click="action('/api/permissions/suspend')">
              <ShieldAlert class="h-4 w-4" />
              暂停权限
            </button>
            <button class="inline-flex items-center justify-center gap-2 rounded-xl bg-red-600 px-4 py-3 font-bold text-white" type="button" @click="action('/api/permissions/revoke')">
              <UserX class="h-4 w-4" />
              撤销权限
            </button>
          </div>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <h2 class="mb-4 text-xl font-black">检查结果</h2>
        <div v-if="!checkResult" class="p-12 text-center text-slate-500">暂无结果</div>
        <template v-else>
          <span class="mb-4 inline-flex rounded-full border px-3 py-1 text-xs font-black" :class="badgeClass(checkResult.status || checkResult.state)">
            {{ checkResult.status || checkResult.state || "UNKNOWN" }}
          </span>
          <pre class="max-h-[720px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(checkResult, null, 2) }}</pre>
        </template>
      </section>
    </div>
  </section>
</template>
