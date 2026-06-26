<script setup lang="ts">
import { Loader2, Plus, RefreshCw, Trash2 } from "lucide-vue-next"
import { onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { type JsonRecord } from "@/lib/display"
import { pickFirst } from "@/lib/status"

type FileConstraint = {
  name: string
  type: number
  is_required: boolean
}

const definitions = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const creating = ref(false)
const showCreate = ref(false)
const name = ref("")
const category = ref("")
const description = ref("")
const constraints = ref<FileConstraint[]>([])

const fileTypes = [
  { value: 0, label: "不限" },
  { value: 1, label: "ͼƬ" },
  { value: 2, label: "PDF" },
  { value: 4, label: "视频" },
  { value: 8, label: "文本" },
]

function definitionUlid(definition: JsonRecord) {
  return String(pickFirst(definition, ["cred_def_ulid", "cred_def_id", "qual_ulid"]) || "")
}

function definitionName(definition: JsonRecord) {
  return String(pickFirst(definition, ["name", "name_hint", "title"]) || "未命名资格")
}

function fileConstraints(definition: JsonRecord) {
  const value = definition.file_constraints
  return Array.isArray(value) ? value : []
}

function resetForm() {
  name.value = ""
  category.value = ""
  description.value = ""
  constraints.value = []
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/credentials/definitions")
    const list = Array.isArray(data.definitions) ? data.definitions : []
    definitions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    selected.value = definitions.value[0] || null
  } catch (err) {
    console.error(err)
    toast.error("资格定义加载失败")
  } finally {
    loading.value = false
  }
}

function addConstraint() {
  constraints.value.push({ name: "", type: 2, is_required: true })
}

async function createDefinition() {
  if (!name.value.trim() || !category.value.trim()) {
    toast.error("名称和分类必填")
    return
  }
  creating.value = true
  try {
    await apiClient("/api/credentials/definitions", {
      method: "POST",
      body: JSON.stringify({
        name: name.value.trim(),
        category: category.value.trim(),
        description: description.value.trim(),
        file_constraints: constraints.value,
      }),
    })
    toast.success("资格定义已创建")
    showCreate.value = false
    resetForm()
    await load()
  } catch (err) {
    console.error(err)
    toast.error("创建失败")
  } finally {
    creating.value = false
  }
}

onMounted(load)
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">资格定义</h1>
        <p class="mt-2 text-slate-600">维护认证资格、免考资格和最终证书所需材料。</p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="showCreate = true">
          <Plus class="h-4 w-4" />
          新建资格
        </button>
      </div>
    </header>

    <div class="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-200 p-5">
          <h2 class="text-xl font-black">资格列表</h2>
        </div>
        <div v-if="loading" class="p-12 text-center text-slate-500">
          <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
          正在加载...
        </div>
        <button
          v-for="definition in definitions"
          v-else
          :key="definitionUlid(definition)"
          class="w-full border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="selected === definition ? 'bg-sky-50' : ''"
          type="button"
          @click="selected = definition"
        >
          <div class="font-black">{{ definitionName(definition) }}</div>
          <div class="mt-1 text-sm text-slate-500">{{ definition.category || "-" }}</div>
        </button>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm">
        <div v-if="!selected" class="p-10 text-center text-slate-500">请选择一个资格定义</div>
        <template v-else>
          <h2 class="text-2xl font-black">{{ definitionName(selected) }}</h2>
          <p class="mt-2 text-slate-600">{{ selected.description || "暂无描述" }}</p>

          <div class="mt-6 rounded-2xl bg-slate-50 p-4">
            <div class="mb-3 text-sm font-black">所需文件</div>
            <div v-if="!fileConstraints(selected).length" class="text-sm text-slate-500">暂无文件约束</div>
            <div v-for="constraint in fileConstraints(selected)" v-else :key="String(constraint.name)" class="mb-2 rounded-xl border border-slate-200 bg-white p-4">
              <div class="font-bold">{{ constraint.name || "未命名文件" }}</div>
              <div class="mt-1 text-sm text-slate-500">
                类型：{{ fileTypes.find((item) => item.value === Number(constraint.type))?.label || constraint.type }}
                · {{ constraint.is_required ? "必填" : "选填" }}
              </div>
            </div>
          </div>

          <pre class="mt-5 max-h-[520px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
        </template>
      </section>
    </div>

    <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-6">
      <div class="w-full max-w-3xl rounded-3xl bg-white p-6 shadow-2xl">
        <div class="mb-5 flex items-center justify-between">
          <h2 class="text-2xl font-black">新建资格定义</h2>
          <button class="rounded-xl border px-4 py-2 font-bold" type="button" @click="showCreate = false">关闭</button>
        </div>
        <div class="grid gap-4">
          <input v-model="name" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="资格名称" />
          <input v-model="category" class="rounded-xl border border-slate-200 px-4 py-3" placeholder="分类，例如 Certification / Exemption / Qualification" />
          <textarea v-model="description" class="min-h-24 rounded-xl border border-slate-200 px-4 py-3" placeholder="描述" />
          <div class="rounded-2xl border border-slate-200 p-4">
            <div class="mb-3 flex items-center justify-between">
              <div class="font-black">文件约束</div>
              <button class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold" type="button" @click="addConstraint">
                <Plus class="h-4 w-4" />
                添加文件
              </button>
            </div>
            <div v-for="(constraint, index) in constraints" :key="index" class="mb-3 grid gap-3 rounded-xl bg-slate-50 p-3 md:grid-cols-[1fr_140px_100px_auto]">
              <input v-model="constraint.name" class="rounded-lg border border-slate-200 px-3 py-2" placeholder="文件名" />
              <select v-model.number="constraint.type" class="rounded-lg border border-slate-200 px-3 py-2">
                <option v-for="type in fileTypes" :key="type.value" :value="type.value">{{ type.label }}</option>
              </select>
              <label class="flex items-center gap-2 text-sm font-bold">
                <input v-model="constraint.is_required" type="checkbox" />
                必填
              </label>
              <button class="rounded-lg border px-3 py-2 text-red-600" type="button" @click="constraints.splice(index, 1)">
                <Trash2 class="h-4 w-4" />
              </button>
            </div>
            <div v-if="!constraints.length" class="text-sm text-slate-500">暂未添加文件约束</div>
          </div>
        </div>
        <div class="mt-6 flex justify-end gap-3">
          <button class="rounded-xl border px-5 py-3 font-bold" type="button" @click="showCreate = false">取消</button>
          <button class="rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="creating" @click="createDefinition">
            {{ creating ? "创建中..." : "创建" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
