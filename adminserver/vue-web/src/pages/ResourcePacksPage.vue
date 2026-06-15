<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue"
import { toast } from "vue-sonner"
import { Archive, FilePlus2, Pencil, Plus, RefreshCw, Trash2 } from "lucide-vue-next"
import AdminShell from "@/components/AdminShell.vue"
import { apiClient, toQuery } from "@/lib/apiClient"

type ResourcePack = {
  pack_id?: string
  title?: string
  description?: string
  thumbnail_object_key?: string
  thumbnail_file_hash?: string
  respath?: string
  status?: string
  version?: number
  created_at?: string
  updated_at?: string
}

type ResourcePackFile = {
  file_id?: string
  pack_id?: string
  title?: string
  description?: string
  thumbnail_object_key?: string
  thumbnail_file_hash?: string
  file_type?: number
  file_name?: string
  file_size?: number
  file_hash?: string
  file_object_key?: string
  video_stream_uid?: string
  sort_order?: number
  version?: number
}

const loading = ref(false)
const saving = ref(false)
const packs = ref<ResourcePack[]>([])
const files = ref<ResourcePackFile[]>([])
const selectedPackId = ref("")
const statusFilter = ref("")

const packForm = reactive<ResourcePack>({
  title: "",
  description: "",
  thumbnail_object_key: "",
  thumbnail_file_hash: "",
  respath: "",
  status: "Draft",
})

const fileForm = reactive<ResourcePackFile>({
  title: "",
  description: "",
  file_type: 2,
  file_name: "",
  file_size: 0,
  file_hash: "",
  file_object_key: "",
  video_stream_uid: "",
  sort_order: 0,
})

const selectedPack = computed(() => packs.value.find((pack) => pack.pack_id === selectedPackId.value))
const sortedFiles = computed(() => files.value.slice().sort((a, b) => Number(a.sort_order || 0) - Number(b.sort_order || 0)))

function resetPackForm() {
  selectedPackId.value = ""
  Object.assign(packForm, {
    pack_id: "",
    title: "",
    description: "",
    thumbnail_object_key: "",
    thumbnail_file_hash: "",
    respath: "",
    status: "Draft",
    version: undefined,
  })
  files.value = []
}

function resetFileForm() {
  Object.assign(fileForm, {
    title: "",
    description: "",
    thumbnail_object_key: "",
    thumbnail_file_hash: "",
    file_type: 2,
    file_name: "",
    file_size: 0,
    file_hash: "",
    file_object_key: "",
    video_stream_uid: "",
    sort_order: sortedFiles.value.length + 1,
  })
}

function selectPack(pack: ResourcePack) {
  selectedPackId.value = pack.pack_id || ""
  Object.assign(packForm, { ...pack })
  void loadFiles()
}

async function loadPacks() {
  loading.value = true
  try {
    const resp = await apiClient(`/api/lms/resource-packs${toQuery({ page_size: 100, status: statusFilter.value })}`)
    packs.value = Array.isArray(resp?.packs) ? resp.packs : []
    if (selectedPackId.value) {
      const fresh = packs.value.find((pack) => pack.pack_id === selectedPackId.value)
      if (fresh) Object.assign(packForm, fresh)
    }
  } finally {
    loading.value = false
  }
}

async function loadFiles() {
  if (!selectedPackId.value) return
  const resp = await apiClient(`/api/lms/resource-packs/${encodeURIComponent(selectedPackId.value)}/files?page_size=100`)
  files.value = Array.isArray(resp?.files) ? resp.files : []
  resetFileForm()
}

async function savePack() {
  if (!packForm.title?.trim()) {
    toast.error("title is required")
    return
  }
  saving.value = true
  try {
    const payload = { ...packForm }
    if (selectedPackId.value) {
      const updated = await apiClient(`/api/lms/resource-packs/${encodeURIComponent(selectedPackId.value)}`, {
        method: "PUT",
        body: JSON.stringify(payload),
      })
      toast.success("资源包已更新")
      Object.assign(packForm, updated)
      await loadPacks()
      return
    }
    const created = await apiClient("/api/lms/resource-packs", {
      method: "POST",
      body: JSON.stringify(payload),
    })
    toast.success("资源包已创建")
    await loadPacks()
    selectPack(created)
  } finally {
    saving.value = false
  }
}

async function deletePack(pack: ResourcePack) {
  if (!pack.pack_id || !pack.version) return
  if (!window.confirm(`确定删除资源包 ${pack.title || pack.pack_id} 吗？`)) return
  await apiClient(`/api/lms/resource-packs/${encodeURIComponent(pack.pack_id)}?version=${pack.version}`, { method: "DELETE" })
  toast.success("资源包已删除")
  if (selectedPackId.value === pack.pack_id) resetPackForm()
  await loadPacks()
}

async function createFile() {
  if (!selectedPackId.value) {
    toast.error("请先选择资源包")
    return
  }
  if (!fileForm.title?.trim()) {
    toast.error("file title is required")
    return
  }
  await apiClient(`/api/lms/resource-packs/${encodeURIComponent(selectedPackId.value)}/files`, {
    method: "POST",
    body: JSON.stringify(fileForm),
  })
  toast.success("文件已新增")
  await loadFiles()
}

async function deleteFile(file: ResourcePackFile) {
  if (!file.file_id || !file.version) return
  if (!window.confirm(`确定删除文件 ${file.title || file.file_id} 吗？`)) return
  await apiClient(`/api/lms/resource-pack-files/${encodeURIComponent(file.file_id)}?version=${file.version}`, { method: "DELETE" })
  toast.success("文件已删除")
  await loadFiles()
}

function fileTypeName(type?: number) {
  if (Number(type) === 1) return "Video"
  if (Number(type) === 2) return "PDF"
  if (Number(type) === 3) return "ZIP"
  return "Unknown"
}

onMounted(() => {
  void loadPacks()
})
</script>

<template>
  <AdminShell>
    <section class="mb-6 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
      <div>
        <p class="text-xs font-black uppercase tracking-[0.24em] text-slate-500">Resource Packs</p>
        <h1 class="mt-2 text-4xl font-black">资源包配置</h1>
        <p class="mt-2 max-w-2xl text-sm leading-6 text-slate-500">配置资源包、访问路径 respath、封面和文件。发布 Active 前后端会做完整校验。</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <select v-model="statusFilter" class="select w-36" @change="loadPacks">
          <option value="">全部状态</option>
          <option value="Draft">Draft</option>
          <option value="Active">Active</option>
          <option value="Archived">Archived</option>
        </select>
        <button class="btn btn-outline" :disabled="loading" @click="loadPacks">
          <RefreshCw :class="['h-4 w-4', loading ? 'animate-spin' : '']" />刷新
        </button>
        <button class="btn btn-primary" @click="resetPackForm"><Plus class="h-4 w-4" />新建资源包</button>
      </div>
    </section>

    <section class="grid gap-5 xl:grid-cols-[0.9fr_1.25fr]">
      <div class="card overflow-hidden p-4">
        <div v-if="packs.length" class="space-y-3">
          <article
            v-for="pack in packs"
            :key="pack.pack_id"
            :class="['rounded-[22px] border p-4 transition', selectedPackId === pack.pack_id ? 'border-[var(--clay)] bg-[rgba(201,120,72,0.08)]' : 'border-[var(--line)] bg-white/58 hover:bg-white']"
          >
            <button class="w-full text-left" @click="selectPack(pack)">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <h2 class="font-black">{{ pack.title || pack.pack_id }}</h2>
                  <p class="mt-1 text-xs text-slate-500">{{ pack.pack_id }}</p>
                </div>
                <span class="rounded-full bg-[var(--ink)] px-2.5 py-1 text-xs font-black text-white">{{ pack.status || "Draft" }}</span>
              </div>
              <p class="mt-3 line-clamp-2 text-sm leading-6 text-slate-500">{{ pack.description || "暂无说明" }}</p>
              <p class="mt-3 text-xs font-bold text-slate-500">respath: {{ pack.respath || "-" }}</p>
            </button>
            <button class="btn btn-ghost mt-3 text-red-700" @click="deletePack(pack)"><Trash2 class="h-4 w-4" />删除</button>
          </article>
        </div>
        <div v-else class="p-10 text-center text-slate-500">
          <Archive class="mx-auto mb-4 h-10 w-10 opacity-40" />
          暂无资源包
        </div>
      </div>

      <div class="space-y-5">
        <form class="card p-5" @submit.prevent="savePack">
          <div class="mb-5 flex items-center justify-between gap-3">
            <div>
              <p class="text-xs font-black uppercase tracking-[0.18em] text-slate-500">{{ selectedPackId ? "Edit Pack" : "Create Pack" }}</p>
              <h2 class="mt-1 text-2xl font-black">{{ selectedPackId ? "编辑资源包" : "新建资源包" }}</h2>
            </div>
            <button class="btn btn-primary" :disabled="saving" type="submit"><Pencil class="h-4 w-4" />保存</button>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <label><span class="label">标题 *</span><input v-model="packForm.title" class="input" /></label>
            <label><span class="label">状态</span><select v-model="packForm.status" class="select"><option>Draft</option><option>Active</option><option>Archived</option></select></label>
            <label><span class="label">respath</span><input v-model="packForm.respath" class="input" placeholder="/certifications/cftp" /></label>
            <label><span class="label">版本</span><input :value="packForm.version || '-'" class="input" disabled /></label>
            <label><span class="label">封面 object key</span><input v-model="packForm.thumbnail_object_key" class="input" /></label>
            <label><span class="label">封面 SHA256</span><input v-model="packForm.thumbnail_file_hash" class="input mono" /></label>
            <label class="md:col-span-2"><span class="label">说明</span><textarea v-model="packForm.description" class="textarea" /></label>
          </div>
        </form>

        <form class="card p-5" @submit.prevent="createFile">
          <div class="mb-5 flex items-center justify-between gap-3">
            <div>
              <p class="text-xs font-black uppercase tracking-[0.18em] text-slate-500">Pack Files</p>
              <h2 class="mt-1 text-2xl font-black">新增文件</h2>
            </div>
            <button class="btn btn-clay" :disabled="!selectedPackId" type="submit"><FilePlus2 class="h-4 w-4" />新增文件</button>
          </div>
          <div class="grid gap-4 md:grid-cols-3">
            <label><span class="label">文件标题 *</span><input v-model="fileForm.title" class="input" /></label>
            <label><span class="label">类型 *</span><select v-model.number="fileForm.file_type" class="select"><option :value="1">Video</option><option :value="2">PDF</option><option :value="3">ZIP</option></select></label>
            <label><span class="label">排序</span><input v-model.number="fileForm.sort_order" class="input" type="number" min="0" /></label>
            <label><span class="label">文件名</span><input v-model="fileForm.file_name" class="input" /></label>
            <label><span class="label">文件大小 bytes</span><input v-model.number="fileForm.file_size" class="input" type="number" min="0" /></label>
            <label><span class="label">视频 Stream UID</span><input v-model="fileForm.video_stream_uid" class="input" /></label>
            <label><span class="label">文件 object key</span><input v-model="fileForm.file_object_key" class="input" /></label>
            <label><span class="label">文件 SHA256</span><input v-model="fileForm.file_hash" class="input mono" /></label>
            <label><span class="label">封面 object key</span><input v-model="fileForm.thumbnail_object_key" class="input" /></label>
            <label class="md:col-span-3"><span class="label">说明</span><textarea v-model="fileForm.description" class="textarea" /></label>
          </div>
        </form>

        <div class="card p-5">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="text-2xl font-black">文件列表</h2>
            <span class="text-sm font-black text-slate-500">{{ selectedPack?.title || "未选择资源包" }}</span>
          </div>
          <div v-if="sortedFiles.length" class="table-wrap">
            <table>
              <thead>
                <tr><th>标题</th><th>类型</th><th>文件名</th><th>排序</th><th>版本</th><th>操作</th></tr>
              </thead>
              <tbody>
                <tr v-for="file in sortedFiles" :key="file.file_id">
                  <td><strong>{{ file.title }}</strong><br /><span class="mono text-xs text-slate-500">{{ file.file_id }}</span></td>
                  <td>{{ fileTypeName(file.file_type) }}</td>
                  <td>{{ file.file_name || "-" }}</td>
                  <td>{{ file.sort_order || 0 }}</td>
                  <td>{{ file.version || "-" }}</td>
                  <td><button class="btn btn-ghost text-red-700" @click="deleteFile(file)"><Trash2 class="h-4 w-4" />删除</button></td>
                </tr>
              </tbody>
            </table>
          </div>
          <p v-else class="rounded-[22px] border border-dashed border-[var(--line)] p-8 text-center text-sm text-slate-500">暂无文件</p>
        </div>
      </div>
    </section>
  </AdminShell>
</template>
