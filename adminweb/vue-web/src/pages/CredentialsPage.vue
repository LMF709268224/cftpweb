<script setup lang="ts">
import { Loader2, Plus, RefreshCw, Trash2 } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { type JsonRecord } from "@/lib/display"
import { pickFirst } from "@/lib/status"

type FileConstraint = {
  name: string
  type: number
  is_required: boolean
}

type DetailMode = "detail" | "create"

const definitions = ref<JsonRecord[]>([])
const selected = ref<JsonRecord | null>(null)
const loading = ref(false)
const detailLoading = ref(false)
const creating = ref(false)
const mode = ref<DetailMode>("detail")
const name = ref("")
const category = ref("")
const description = ref("")
const respath = ref("")
const acquisitionMethod = ref("")
const constraints = ref<FileConstraint[]>([])

const fileTypes = [
  { value: 0, label: "不限 / Any" },
  { value: 1, label: "图片 / Image" },
  { value: 2, label: "PDF 文档 / PDF" },
  { value: 4, label: "视频 / Video" },
  { value: 8, label: "文本 / Text" },
]

const selectedFields = computed(() => selected.value || {})

function definitionUlid(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["cred_def_ulid", "cred_def_id", "qual_ulid"]) || "")
}

function definitionName(definition: JsonRecord | null | undefined) {
  return String(pickFirst(definition || {}, ["name", "name_hint", "title"]) || "未命名资格")
}

function fileConstraints(definition: JsonRecord | null | undefined) {
  const value = definition?.file_constraints
  return Array.isArray(value) ? value.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item)) : []
}

function fileTypeLabel(type: unknown) {
  return fileTypes.find((item) => item.value === Number(type))?.label || String(type || "-")
}

function resetForm() {
  name.value = ""
  category.value = ""
  description.value = ""
  respath.value = ""
  acquisitionMethod.value = ""
  constraints.value = []
}

function startCreate() {
  mode.value = "create"
  resetForm()
}

async function loadDefinitionDetail(definition: JsonRecord) {
  const id = definitionUlid(definition)
  if (!id) {
    selected.value = definition
    return
  }

  detailLoading.value = true
  try {
    const detail = await apiClient<JsonRecord>(`/api/credentials/definitions/${encodeURIComponent(id)}`)
    const merged = { ...definition, ...detail }
    const index = definitions.value.findIndex((item) => definitionUlid(item) === id)
    if (index >= 0) definitions.value.splice(index, 1, merged)
    selected.value = merged
  } catch (err) {
    console.error(err)
    toast.error("资格定义详情加载失败")
    selected.value = definition
  } finally {
    detailLoading.value = false
  }
}

async function selectDefinition(definition: JsonRecord) {
  selected.value = definition
  mode.value = "detail"
  await loadDefinitionDetail(definition)
}

async function load() {
  loading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/credentials/definitions")
    const list = Array.isArray(data.definitions) ? data.definitions : []
    definitions.value = list.filter((item): item is JsonRecord => !!item && typeof item === "object" && !Array.isArray(item))
    if (!selected.value || !definitions.value.some((item) => definitionUlid(item) === definitionUlid(selected.value))) {
      selected.value = definitions.value[0] || null
    }
    if (!selected.value) mode.value = "create"
    if (selected.value && mode.value === "detail") await loadDefinitionDetail(selected.value)
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
        respath: respath.value.trim(),
        acquisition_method: acquisitionMethod.value.trim(),
        file_constraints: constraints.value.map((constraint) => ({
          name: constraint.name.trim(),
          type: Number(constraint.type),
          is_required: Boolean(constraint.is_required),
        })),
      }),
    })
    toast.success("资格定义已创建")
    resetForm()
    mode.value = "detail"
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
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">资格定义</h1>
        <p class="mt-2 text-slate-600">维护认证资格、免考资格和最终证书所需材料。</p>
        <p class="mt-2 text-xs font-semibold text-slate-500">
          已确认接口：list/create/get detail。当前 gcreds 未提供 update/delete，所以已有定义只读展示。
        </p>
      </div>
      <div class="flex gap-3">
        <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="load">
          <RefreshCw class="h-4 w-4" :class="loading ? 'animate-spin' : ''" />
          刷新
        </button>
        <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b7bdc] px-4 py-3 text-sm font-bold text-white shadow-sm" type="button" @click="startCreate">
          <Plus class="h-4 w-4" />
          新建资格
        </button>
      </div>
    </header>

    <div class="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-200 p-5">
          <div>
            <h2 class="text-xl font-black">资格列表</h2>
            <p class="mt-1 text-sm text-slate-500">来自 `/api/credentials/definitions`。</p>
          </div>
          <span class="rounded-full bg-slate-100 px-3 py-1 text-sm font-black text-slate-600">{{ definitions.length }}</span>
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
          :class="mode === 'detail' && definitionUlid(selected) === definitionUlid(definition) ? 'bg-sky-50' : ''"
          type="button"
          @click="selectDefinition(definition)"
        >
          <div class="font-black">{{ definitionName(definition) }}</div>
          <div class="mt-1 text-sm text-slate-500">{{ definition.category || "-" }}</div>
          <div class="mt-2 break-all text-xs font-semibold text-slate-500">ID: {{ definitionUlid(definition) || "-" }}</div>
        </button>
        <div v-if="!loading && !definitions.length" class="p-12 text-center text-slate-500">暂无资格定义</div>
      </section>

      <section class="rounded-3xl border border-slate-200 bg-white shadow-sm">
        <template v-if="mode === 'create'">
          <div class="border-b border-slate-200 p-5">
            <h2 class="text-2xl font-black">新建资格定义</h2>
            <p class="mt-1 text-sm text-slate-500">保存后会调用 create 接口，列表刷新后可在左侧选择查看。</p>
          </div>
          <div class="grid gap-5 p-5">
            <div class="grid gap-4 md:grid-cols-2">
              <label class="grid gap-2 text-sm font-bold">
                名称
                <input v-model="name" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="120" placeholder="资格名称" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                分类
                <input v-model="category" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="80" placeholder="Certification / Exemption / Qualification" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                资源路径
                <input v-model="respath" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="240" placeholder="可选，取决于微服务是否使用" />
              </label>
              <label class="grid gap-2 text-sm font-bold">
                获取方式说明
                <input v-model="acquisitionMethod" class="rounded-xl border border-slate-200 px-4 py-3" maxlength="240" placeholder="可选" />
              </label>
            </div>
            <label class="grid gap-2 text-sm font-bold">
              描述
              <textarea v-model="description" class="min-h-28 rounded-xl border border-slate-200 px-4 py-3" maxlength="1000" placeholder="描述" />
            </label>
            <div class="rounded-2xl border border-slate-200 p-4">
              <div class="mb-3 flex items-center justify-between">
                <div>
                  <div class="font-black">文件约束</div>
                  <div class="mt-1 text-xs text-slate-500">用于告诉考生申请该资格时需要上传哪些材料。</div>
                </div>
                <button class="inline-flex items-center gap-2 rounded-xl border px-3 py-2 text-sm font-bold" type="button" @click="addConstraint">
                  <Plus class="h-4 w-4" />
                  添加文件
                </button>
              </div>
              <div v-for="(constraint, index) in constraints" :key="index" class="mb-3 grid gap-3 rounded-xl bg-slate-50 p-3 md:grid-cols-[1fr_170px_100px_auto]">
                <input v-model="constraint.name" class="rounded-lg border border-slate-200 px-3 py-2" maxlength="120" placeholder="文件用途，例如 Employment Certificate" />
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
            <div class="flex justify-end gap-3">
              <button class="rounded-xl border px-5 py-3 font-bold" type="button" @click="mode = 'detail'">取消</button>
              <button class="rounded-xl bg-[#0b7bdc] px-5 py-3 font-bold text-white disabled:opacity-50" type="button" :disabled="creating" @click="createDefinition">
                {{ creating ? "创建中..." : "创建" }}
              </button>
            </div>
          </div>
        </template>

        <template v-else>
          <div v-if="!selected" class="p-10 text-center text-slate-500">请选择一个资格定义</div>
          <template v-else>
            <div class="border-b border-slate-200 p-5">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <h2 class="text-2xl font-black">{{ definitionName(selected) }}</h2>
                  <p class="mt-2 break-all text-sm text-slate-500">{{ definitionUlid(selected) }}</p>
                </div>
                <div class="flex items-center gap-2">
                  <Loader2 v-if="detailLoading" class="h-4 w-4 animate-spin text-slate-400" />
                  <span class="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-black text-slate-600">只读 / Readonly</span>
                </div>
              </div>
            </div>

            <div class="space-y-5 p-5">
              <div class="grid gap-4 md:grid-cols-2">
                <label v-for="(value, key) in selectedFields" :key="key" class="grid gap-2 text-sm font-bold">
                  {{ key }}
                  <textarea
                    v-if="Array.isArray(value) || (value && typeof value === 'object')"
                    class="min-h-24 rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                    disabled
                    :value="JSON.stringify(value, null, 2)"
                  />
                  <input
                    v-else
                    class="rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-600"
                    disabled
                    :value="key === 'name' ? definitionName(selected) : String(value ?? '-')"
                  />
                </label>
              </div>

              <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div class="mb-3 text-sm font-black">所需文件</div>
                <div v-if="!fileConstraints(selected).length" class="text-sm text-slate-500">暂无文件约束</div>
                <div v-for="constraint in fileConstraints(selected)" v-else :key="String(constraint.name)" class="mb-2 rounded-xl border border-slate-200 bg-white p-4">
                  <div class="font-bold">{{ constraint.name || "未命名文件" }}</div>
                  <div class="mt-1 text-sm text-slate-500">
                    类型：{{ fileTypeLabel(constraint.type) }}
                    · {{ constraint.is_required ? "必填" : "选填" }}
                  </div>
                </div>
              </div>

              <pre class="max-h-[460px] overflow-auto rounded-2xl bg-slate-950 p-5 text-xs leading-6 text-slate-100">{{ JSON.stringify(selected, null, 2) }}</pre>
            </div>
          </template>
        </template>
      </section>
    </div>
  </section>
</template>
