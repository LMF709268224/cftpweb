<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { ChevronDown, ChevronLeft, ChevronRight } from "lucide-vue-next"

const props = withDefaults(defineProps<{
  page: number
  pageSize: number
  total: number
  totalPages?: number
  pageSizeOptions?: number[]
  disabled?: boolean
  locale?: "zh" | "en"
}>(), {
  totalPages: 0,
  pageSizeOptions: () => [30, 50, 100],
  disabled: false,
  locale: "zh",
})

const emit = defineEmits<{
  "update:page": [value: number]
  "update:pageSize": [value: number]
  "page-change": []
}>()

const pageMenuOpen = ref(false)
const pageJumpInput = ref(String(props.page || 1))
const paginationRef = ref<HTMLElement | null>(null)
const pageSizeWrapRef = ref<HTMLElement | null>(null)
const menuPlacement = ref<"top" | "bottom">("bottom")
const menuStyle = ref<Record<string, string>>({})

const labels = computed(() => {
  const zh = props.locale === "zh"
  return {
    total: zh ? "共" : "Total",
    goTo: zh ? "跳转到" : "Go to",
    pageSize: zh ? "每页条数" : "Page size",
    pageNumber: zh ? "页码" : "Page number",
    previous: zh ? "上一页" : "Previous page",
    next: zh ? "下一页" : "Next page",
  }
})

function pageSizeLabel(value: number) {
  return props.locale === "zh" ? `${value}条/页` : `${value}/page`
}

const normalizedTotalPages = computed(() => {
  if (props.totalPages && props.totalPages > 0) return props.totalPages
  if (props.total <= 0) return 1
  return Math.max(1, Math.ceil(props.total / props.pageSize))
})

const pageItems = computed<(number | "...")[]>(() => {
  const total = normalizedTotalPages.value
  const current = props.page

  if (total <= 7) {
    return Array.from({ length: total }, (_, index) => index + 1)
  }

  const items: (number | "...")[] = [1]
  const start = Math.max(2, current - 2)
  const end = Math.min(total - 1, current + 2)

  if (start > 2) items.push("...")
  for (let item = start; item <= end; item += 1) {
    items.push(item)
  }
  if (end < total - 1) items.push("...")
  items.push(total)

  return items
})

const mobilePageItems = computed<(number | "...")[]>(() => {
  const total = normalizedTotalPages.value
  const current = props.page

  if (total <= 5) {
    return Array.from({ length: total }, (_, index) => index + 1)
  }

  if (current <= 3) {
    return [1, 2, 3, "...", total]
  }

  if (current >= total - 2) {
    return [1, "...", total - 2, total - 1, total]
  }

  return [1, "...", current, "...", total]
})

watch(() => props.page, (value) => {
  pageJumpInput.value = String(value || 1)
})

function handleDocumentPointerDown(event: PointerEvent) {
  if (!pageMenuOpen.value) return
  const target = event.target
  if (!(target instanceof Node)) return
  if (paginationRef.value?.contains(target)) return
  pageMenuOpen.value = false
}

onMounted(() => {
  document.addEventListener("pointerdown", handleDocumentPointerDown)
})

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", handleDocumentPointerDown)
})

function requestPage(nextPage: number) {
  if (props.disabled) return
  const maxPage = normalizedTotalPages.value
  if (nextPage < 1 || nextPage > maxPage || nextPage === props.page) return
  emit("update:page", nextPage)
  emit("page-change")
}

function updateMenuPlacement() {
  const triggerRect = pageSizeWrapRef.value?.getBoundingClientRect()
  if (!triggerRect) return
  const estimatedMenuHeight = Math.max(props.pageSizeOptions.length * 42 + 16, 120)
  const spaceAbove = triggerRect.top
  const spaceBelow = window.innerHeight - triggerRect.bottom
  const nextPlacement = spaceBelow >= estimatedMenuHeight || spaceBelow >= spaceAbove ? "bottom" : "top"
  const menuWidth = 120
  const left = Math.min(Math.max(triggerRect.left, 8), window.innerWidth - menuWidth - 8)
  const top = nextPlacement === "bottom"
    ? triggerRect.bottom + 12
    : Math.max(8, triggerRect.top - estimatedMenuHeight - 12)
  menuPlacement.value = nextPlacement
  menuStyle.value = {
    left: `${left}px`,
    top: `${top}px`,
  }
}

function togglePageMenu() {
  if (props.disabled) return
  if (!pageMenuOpen.value) {
    updateMenuPlacement()
  }
  pageMenuOpen.value = !pageMenuOpen.value
}

function changePageSize(nextPageSize: number) {
  if (props.disabled || nextPageSize === props.pageSize) {
    pageMenuOpen.value = false
    return
  }
  emit("update:pageSize", nextPageSize)
  emit("update:page", 1)
  pageJumpInput.value = "1"
  pageMenuOpen.value = false
  emit("page-change")
}

function submitPageJump() {
  const parsed = Number(pageJumpInput.value)
  if (!Number.isInteger(parsed)) {
    pageJumpInput.value = String(props.page || 1)
    return
  }

  const maxPage = normalizedTotalPages.value
  const nextPage = Math.min(Math.max(parsed, 1), maxPage)
  if (nextPage === props.page) {
    pageJumpInput.value = String(props.page || 1)
    return
  }
  requestPage(nextPage)
}
</script>

<template>
  <div ref="paginationRef" class="app-pagination">
    <div class="pagination-total">
      <span>{{ labels.total }} {{ total }}</span>
      <div ref="pageSizeWrapRef" class="page-size-wrap">
        <button
          type="button"
          class="page-size-trigger"
          :disabled="disabled"
          :aria-expanded="pageMenuOpen"
          :aria-label="labels.pageSize"
          @click="togglePageMenu"
        >
          <span>{{ pageSizeLabel(pageSize) }}</span>
          <ChevronDown class="h-4 w-4 transition-transform" :class="{ 'rotate-180': pageMenuOpen }" />
        </button>
        <div v-if="pageMenuOpen" class="page-size-menu" :class="`is-${menuPlacement}`" :style="menuStyle">
          <button
            v-for="option in pageSizeOptions"
            :key="option"
            type="button"
            class="page-size-option"
            :class="{ 'is-selected': option === pageSize }"
            @click="changePageSize(option)"
          >
            {{ pageSizeLabel(option) }}
          </button>
        </div>
      </div>
    </div>

    <div class="pagination-pages">
      <button
        type="button"
        class="page-arrow"
        :disabled="disabled || page <= 1"
        :aria-label="labels.previous"
        @click="requestPage(page - 1)"
      >
        <ChevronLeft class="h-4 w-4" />
      </button>
      <template v-for="(item, index) in pageItems" :key="`desktop-${item}-${index}`">
        <span v-if="item === '...'" class="page-ellipsis desktop-page-item">...</span>
        <button
          v-else
          type="button"
          class="page-number desktop-page-item"
          :class="{ 'is-active': item === page }"
          :disabled="disabled || item === page"
          @click="requestPage(item)"
        >
          {{ item }}
        </button>
      </template>
      <template v-for="(item, index) in mobilePageItems" :key="`mobile-${item}-${index}`">
        <span v-if="item === '...'" class="page-ellipsis mobile-page-item">...</span>
        <button
          v-else
          type="button"
          class="page-number mobile-page-item"
          :class="{ 'is-active': item === page }"
          :disabled="disabled || item === page"
          @click="requestPage(item)"
        >
          {{ item }}
        </button>
      </template>
      <button
        type="button"
        class="page-arrow"
        :disabled="disabled || page >= normalizedTotalPages"
        :aria-label="labels.next"
        @click="requestPage(page + 1)"
      >
        <ChevronRight class="h-4 w-4" />
      </button>
    </div>

    <div class="pagination-jump">
      <span>{{ labels.goTo }}</span>
      <label class="sr-only" for="pagination-page-jump">{{ labels.pageNumber }}</label>
      <input
        id="pagination-page-jump"
        v-model="pageJumpInput"
        class="page-jump-input"
        inputmode="numeric"
        :disabled="disabled || total <= 0"
        @blur="submitPageJump"
        @keyup.enter="submitPageJump"
      />
    </div>
  </div>
</template>

<style scoped>
.app-pagination {
  position: relative;
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(220px, auto) auto;
  align-items: center;
  gap: 1rem;
  min-height: 4.75rem;
  border: 1px solid rgba(226, 232, 240, 0.95);
  border-radius: 0 0 0.875rem 0.875rem;
  background: rgba(248, 250, 252, 0.78);
  padding: 0 1.5rem;
  color: #475569;
}

.pagination-total,
.pagination-pages,
.pagination-jump {
  display: flex;
  align-items: center;
}

.pagination-total {
  gap: 1.5rem;
  font-size: 1rem;
}

.page-size-wrap {
  position: relative;
}

.page-size-trigger {
  display: inline-flex;
  min-width: 7.5rem;
  height: 2.25rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border: 1px solid rgba(203, 213, 225, 0.9);
  border-radius: 0.45rem;
  background: rgba(255, 255, 255, 0.9);
  padding: 0 0.9rem;
  color: #334155;
  transition:
    border-color 0.18s ease,
    background-color 0.18s ease,
    box-shadow 0.18s ease;
}

.page-size-trigger:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--primary) 35%, transparent);
  background: color-mix(in srgb, var(--primary) 6%, white);
  box-shadow: 0 10px 22px color-mix(in srgb, var(--primary) 10%, transparent);
}

.page-size-menu {
  position: fixed;
  z-index: 60;
  width: 7.5rem;
  border: 1px solid rgba(203, 213, 225, 0.9);
  border-radius: 0.35rem;
  background: #fff;
  padding: 0.5rem 0;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.14);
}

.page-size-menu::after {
  position: absolute;
  left: 50%;
  width: 0.7rem;
  height: 0.7rem;
  content: "";
  transform: translateX(-50%) rotate(45deg);
  background: #fff;
}

.page-size-menu.is-bottom::after {
  top: -0.35rem;
  border-left: 1px solid rgba(203, 213, 225, 0.9);
  border-top: 1px solid rgba(203, 213, 225, 0.9);
}

.page-size-menu.is-top::after {
  bottom: -0.35rem;
  border-right: 1px solid rgba(203, 213, 225, 0.9);
  border-bottom: 1px solid rgba(203, 213, 225, 0.9);
}

.page-size-option {
  position: relative;
  z-index: 1;
  display: block;
  width: 100%;
  padding: 0.6rem 1.25rem;
  text-align: left;
  color: #475569;
  font-weight: 500;
  transition:
    color 0.16s ease,
    background-color 0.16s ease;
}

.page-size-option:hover,
.page-size-option.is-selected {
  background: color-mix(in srgb, var(--primary) 6%, white);
  color: var(--primary);
  font-weight: 650;
}

.pagination-pages {
  justify-content: flex-end;
  gap: 0.55rem;
}

.page-arrow,
.page-number {
  display: inline-flex;
  height: 2rem;
  min-width: 2rem;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 0.2rem;
  color: #334155;
  font-weight: 600;
  transition:
    color 0.16s ease,
    border-color 0.16s ease,
    background-color 0.16s ease,
    box-shadow 0.16s ease;
}

.page-arrow {
  min-width: 1.75rem;
  color: #64748b;
}

.page-number:hover:not(:disabled),
.page-arrow:hover:not(:disabled) {
  border-color: color-mix(in srgb, var(--primary) 26%, transparent);
  background: color-mix(in srgb, var(--primary) 8%, white);
  color: var(--primary);
}

.page-number.is-active {
  border-color: var(--primary);
  background: var(--primary);
  color: var(--primary-foreground);
  font-weight: 700;
  box-shadow: 0 10px 18px color-mix(in srgb, var(--primary) 24%, transparent);
}

.page-arrow:disabled,
.page-number:disabled,
.page-size-trigger:disabled,
.page-jump-input:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.page-number.is-active:disabled {
  cursor: default;
  opacity: 1;
}

.page-ellipsis {
  color: #64748b;
  font-weight: 600;
}

.mobile-page-item {
  display: none;
}

.pagination-jump {
  justify-content: flex-end;
  gap: 0.75rem;
  color: #475569;
}

.page-jump-input {
  width: 3.5rem;
  height: 2.25rem;
  border: 1px solid rgba(203, 213, 225, 0.9);
  border-radius: 0.45rem;
  background: rgba(255, 255, 255, 0.88);
  text-align: center;
  color: #334155;
  outline: none;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.page-jump-input:focus {
  border-color: color-mix(in srgb, var(--primary) 55%, transparent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 12%, transparent);
}

@media (max-width: 1024px) {
  .app-pagination {
    grid-template-columns: 1fr;
    gap: 0.75rem;
    justify-items: center;
    padding: 1rem;
  }

  .pagination-total,
  .pagination-jump {
    justify-content: center;
  }
}

@media (max-width: 640px) {
  .app-pagination {
    min-height: auto;
    padding: 0.875rem 0.75rem;
  }

  .pagination-pages {
    gap: 0.35rem;
    flex-wrap: nowrap;
  }

  .desktop-page-item {
    display: none;
  }

  .mobile-page-item {
    display: inline-flex;
  }

  .pagination-total {
    gap: 0.75rem;
  }

  .pagination-jump {
    gap: 0.5rem;
  }

  .page-size-trigger {
    min-width: 7rem;
  }

  .page-arrow,
  .page-number {
    height: 1.9rem;
    min-width: 1.9rem;
  }
}
</style>
