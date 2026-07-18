<script setup lang="ts">
import { Loader2, Search, X } from "lucide-vue-next"
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { apiClient } from "@/lib/apiClient"
import { useAdminLanguage } from "@/lib/language"
import UlidChip from "@/components/UlidChip.vue"

type CandidateOption = {
  candidate_ulid: string
  name: string
  email: string
}

type OpsUser = {
  id?: string
  candidate_ulid?: string
  name?: string
  email?: string
}

type OpsResponse = {
  users?: OpsUser[]
}

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  (e: "update:modelValue", value: string): void
}>()

const { t } = useAdminLanguage()

const rootEl = ref<HTMLElement | null>(null)
const keyword = ref("")
const options = ref<CandidateOption[]>([])
const open = ref(false)
const searching = ref(false)
const searched = ref(false)
const activeIndex = ref(-1)
const selected = ref<CandidateOption | null>(null)
let debounceTimer: number | undefined
let requestSeq = 0

const ULID_PATTERN = /^[0-9A-HJKMNP-TV-Z]{26}$/i

const pastedUlid = computed(() => {
  const raw = keyword.value.trim()
  return ULID_PATTERN.test(raw) ? raw.toUpperCase() : ""
})

watch(
  () => props.modelValue,
  (value) => {
    if (!value) {
      selected.value = null
      return
    }
    if (selected.value?.candidate_ulid !== value) {
      selected.value = { candidate_ulid: value, name: "", email: "" }
    }
  },
  { immediate: true },
)

watch(keyword, () => {
  if (debounceTimer) window.clearTimeout(debounceTimer)
  const raw = keyword.value.trim()
  if (!raw) {
    options.value = []
    searched.value = false
    open.value = false
    activeIndex.value = -1
    return
  }
  debounceTimer = window.setTimeout(() => {
    void search(raw)
  }, 250)
})

async function search(raw: string) {
  const seq = ++requestSeq
  searching.value = true
  try {
    const params = new URLSearchParams({
      user_keyword: raw,
      user_page: "1",
      user_page_size: "8",
    })
    const data = await apiClient<OpsResponse>(`/api/dashboard/ops?${params}`)
    if (seq !== requestSeq) return
    options.value = (data?.users || [])
      .filter((user) => Boolean(user.candidate_ulid))
      .map((user) => ({
        candidate_ulid: String(user.candidate_ulid),
        name: user.name || "",
        email: user.email || "",
      }))
    searched.value = true
    open.value = true
    activeIndex.value = options.value.length ? 0 : -1
  } catch (err) {
    if (seq === requestSeq) {
      console.error(err)
      searched.value = true
      options.value = []
    }
  } finally {
    if (seq === requestSeq) searching.value = false
  }
}

function select(option: CandidateOption) {
  selected.value = option
  emit("update:modelValue", option.candidate_ulid)
  keyword.value = ""
  options.value = []
  searched.value = false
  open.value = false
  activeIndex.value = -1
}

function clearSelection() {
  selected.value = null
  emit("update:modelValue", "")
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === "ArrowDown") {
    event.preventDefault()
    if (!open.value && (options.value.length || searched.value)) open.value = true
    if (options.value.length) {
      activeIndex.value = (activeIndex.value + 1) % options.value.length
    }
    return
  }
  if (event.key === "ArrowUp") {
    event.preventDefault()
    if (options.value.length) {
      activeIndex.value = activeIndex.value <= 0 ? options.value.length - 1 : activeIndex.value - 1
    }
    return
  }
  if (event.key === "Enter") {
    event.preventDefault()
    if (open.value && activeIndex.value >= 0 && options.value[activeIndex.value]) {
      select(options.value[activeIndex.value])
      return
    }
    if (pastedUlid.value) {
      select({ candidate_ulid: pastedUlid.value, name: "", email: "" })
    }
    return
  }
  if (event.key === "Escape") {
    open.value = false
    activeIndex.value = -1
  }
}

function onFocus() {
  if (keyword.value.trim() && (options.value.length || searched.value)) {
    open.value = true
  }
}

function onPointerDown(event: PointerEvent) {
  if (rootEl.value && event.target instanceof Node && !rootEl.value.contains(event.target)) {
    open.value = false
  }
}

onMounted(() => {
  document.addEventListener("pointerdown", onPointerDown)
})

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", onPointerDown)
  if (debounceTimer) window.clearTimeout(debounceTimer)
})
</script>

<template>
  <div ref="rootEl" class="relative">
    <!-- Selected candidate card -->
    <div
      v-if="selected"
      class="flex h-10 items-center justify-between gap-3 rounded-[10px] border border-edge bg-slate-50/70 px-3"
    >
      <div class="flex min-w-0 items-center gap-2">
        <span v-if="selected.name" class="truncate text-sm font-medium text-ink">{{ selected.name }}</span>
        <UlidChip :value="selected.candidate_ulid" />
      </div>
      <button
        type="button"
        class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30"
        :aria-label="t.common.clearSelection"
        :title="t.common.clearSelection"
        @click.stop="clearSelection"
      >
        <X class="h-4 w-4" />
      </button>
    </div>

    <!-- Search input -->
    <div v-else class="relative">
      <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
      <input
        v-model="keyword"
        class="ui-input pl-9 pr-9"
        type="text"
        :placeholder="t.common.searchCandidate"
        role="combobox"
        :aria-expanded="open"
        autocomplete="off"
        spellcheck="false"
        @keydown="onKeydown"
        @focus="onFocus"
      />
      <Loader2
        v-if="searching"
        class="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-slate-400"
      />
    </div>

    <!-- Results popover -->
    <Transition name="menu">
      <div
        v-if="!selected && open"
        class="absolute left-0 right-0 z-30 mt-1.5 overflow-hidden rounded-xl border border-edge bg-white py-1 shadow-lg shadow-slate-900/5"
        role="listbox"
      >
        <button
          v-for="(option, index) in options"
          :key="option.candidate_ulid"
          type="button"
          role="option"
          :aria-selected="index === activeIndex"
          class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left transition-colors"
          :class="index === activeIndex ? 'bg-primary-50/60' : 'hover:bg-slate-50'"
          @mouseenter="activeIndex = index"
          @click.stop="select(option)"
        >
          <span class="min-w-0">
            <span class="block truncate text-sm font-medium text-ink">{{ option.name || "-" }}</span>
            <span class="block truncate text-xs text-slate-400">{{ option.email || "-" }}</span>
          </span>
          <span class="tnum shrink-0 font-mono text-xs text-slate-400">
            {{ option.candidate_ulid.length > 14 ? `${option.candidate_ulid.slice(0, 6)}…${option.candidate_ulid.slice(-5)}` : option.candidate_ulid }}
          </span>
        </button>
        <div v-if="searched && !options.length && !searching" class="px-3 py-3 text-center text-xs text-slate-400">
          {{ t.common.noResults }}
        </div>
      </div>
    </Transition>
  </div>
</template>
