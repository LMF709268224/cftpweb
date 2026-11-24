<script setup lang="ts">
import { computed, ref } from "vue"
import { ArrowDown, ArrowUp, GripVertical } from "lucide-vue-next"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import { useCourseAuthoringCopy } from "@/lib/courseAuthoringCopy"
import type { JsonRecord } from "@/lib/display"

const props = defineProps<{
  courseId: string
  complete: JsonRecord | null
  editable: boolean
  selectedId?: string
  disabled?: boolean
  reload: () => Promise<void>
}>()
const emit = defineEmits<{ select: [lesson: JsonRecord]; chapter: [chapter: JsonRecord] }>()
const copy = useCourseAuthoringCopy()
const busy = ref(false)
const failed = ref(false)
const dragged = ref<{ parent: string; id: string } | null>(null)
function record(value: unknown): value is JsonRecord { return !!value && typeof value === "object" && !Array.isArray(value) }
function positive(value: unknown): value is number { return typeof value === "number" && Number.isSafeInteger(value) && value > 0 }
type Sortable<K extends string> = JsonRecord & Record<K, string> & { sort_order: number; version: number }
type ChapterNode = { chapter: Sortable<"chapter_ulid">; lessons: unknown }
function sortable<K extends string>(value: unknown, key: K): value is Sortable<K> {
  return record(value) && typeof value[key] === "string" && !!value[key] && positive(value.sort_order) && positive(value.version)
}
const outline = computed(() => {
  if (!Array.isArray(props.complete?.chapters)) return null
  const rows: ChapterNode[] = []
  for (const item of props.complete.chapters) {
    if (!record(item) || !sortable(item.chapter, "chapter_ulid")) return null
    rows.push({ chapter: item.chapter, lessons: item.lessons })
  }
  if (new Set(rows.map((item) => item.chapter.chapter_ulid)).size !== rows.length) return null
  return rows.sort((a, b) => a.chapter.sort_order - b.chapter.sort_order)
})
function lessons(chapter: ChapterNode): Sortable<"lesson_ulid">[] | null {
  if (!Array.isArray(chapter.lessons)) return null
  const rows: Sortable<"lesson_ulid">[] = []
  for (const item of chapter.lessons) {
    if (!record(item) || !sortable(item.lesson, "lesson_ulid")) return null
    rows.push(item.lesson)
  }
  if (new Set(rows.map((item) => item.lesson_ulid)).size !== rows.length) return null
  return rows.sort((a, b) => a.sort_order - b.sort_order)
}
const locked = computed(() => busy.value || failed.value || props.disabled)
async function reload() {
  busy.value = true
  try { await props.reload(); failed.value = false } finally { busy.value = false }
}
async function move(parent: string, id: string, targetId: string) {
  if (!props.editable || locked.value || !outline.value || id === targetId) return
  const group = parent ? outline.value.find((item) => item.chapter.chapter_ulid === parent) : null
  const rows = parent ? (group ? lessons(group) : null) : outline.value.map((item) => item.chapter)
  if (!rows) return
  const key = parent ? "lesson_ulid" : "chapter_ulid"
  const from = rows.findIndex((row) => row[key] === id)
  const to = rows.findIndex((row) => row[key] === targetId)
  if (from < 0 || to < 0) return
  const ordered = [...rows]
  ordered.splice(to, 0, ordered.splice(from, 1)[0]!)
  busy.value = true
  try {
    const endpoint = parent ? `/api/lms/chapters/${encodeURIComponent(parent)}/lessons/reorder` : `/api/lms/courses/${encodeURIComponent(props.courseId)}/chapters/reorder`
    const response = await apiClient<{ success?: boolean }>(endpoint, { method: "POST", body: JSON.stringify({ items: ordered.map((row, index) => ({ entity_ulid: row[key], sort_order: index + 1, version: row.version })) }) })
    if (response?.success !== true) throw new Error(copy.value.reorderFailed)
    await props.reload()
    toast.success(copy.value.reordered)
  } catch (error) {
    failed.value = true
    toast.error(apiErrorMessage(error, copy.value.reorderFailed))
  } finally { busy.value = false; dragged.value = null }
}
function start(event: DragEvent, parent: string, id: string) {
  if (!props.editable || locked.value) { event.preventDefault(); return }
  dragged.value = { parent, id }
  event.dataTransfer?.setData("text/plain", id)
  if (event.dataTransfer) event.dataTransfer.effectAllowed = "move"
}
function drop(parent: string, id: string) {
  if (dragged.value?.parent === parent) void move(parent, dragged.value.id, id)
  dragged.value = null
}
</script>

<template>
  <nav class="min-w-0 space-y-3 p-4" :aria-label="copy.outline" data-testid="course-outline">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h3 class="font-black">{{ copy.outline }}</h3>
      <button type="button" class="text-xs font-bold text-blue-700 disabled:opacity-40" :disabled="busy || disabled" @click="reload">{{ copy.reload }}</button>
    </div>
    <p v-if="editable" class="text-xs text-slate-500">{{ copy.reorder }}</p>
    <p v-if="!outline" role="alert" class="text-sm text-amber-800">{{ copy.outlineUnavailable }}</p>
    <p v-else-if="!outline.length" class="text-sm text-slate-500">{{ copy.empty }}</p>
    <p v-if="failed" role="alert" class="text-sm text-red-700">{{ copy.reorderFailed }}</p>
    <div v-for="(group, index) in outline" :key="group.chapter.chapter_ulid" class="rounded-lg border border-slate-200 bg-white p-2" @dragover.prevent @drop.stop.prevent="drop('', group.chapter.chapter_ulid)">
      <div class="flex items-center gap-1">
        <button v-if="editable" type="button" :draggable="!locked" :disabled="locked" :aria-label="copy.reorder" class="cursor-grab" @dragstart.stop="start($event, '', group.chapter.chapter_ulid)" @dragend="dragged = null"><GripVertical class="h-4 w-4" /></button>
        <button type="button" class="min-w-0 flex-1 break-words text-left text-sm font-bold" :disabled="locked" @click="emit('chapter', group.chapter)">{{ typeof group.chapter.title === 'string' ? group.chapter.title : copy.unavailable }}</button>
        <template v-if="editable && outline">
          <button type="button" :aria-label="copy.up" :disabled="locked || index === 0" class="disabled:opacity-25" @click="move('', group.chapter.chapter_ulid, outline[index - 1]!.chapter.chapter_ulid)"><ArrowUp class="h-4 w-4" /></button>
          <button type="button" :aria-label="copy.down" :disabled="locked || index === outline.length - 1" class="disabled:opacity-25" @click="move('', group.chapter.chapter_ulid, outline[index + 1]!.chapter.chapter_ulid)"><ArrowDown class="h-4 w-4" /></button>
        </template>
      </div>
      <p v-if="lessons(group) === null" class="mt-2 text-xs text-amber-800">{{ copy.outlineUnavailable }}</p>
      <p v-else-if="lessons(group)?.length === 0" class="mt-2 text-xs text-slate-500">{{ copy.noLessons }}</p>
      <div v-for="(lesson, lessonIndex) in lessons(group)" :key="lesson.lesson_ulid" class="mt-2 flex items-center gap-1 rounded p-2" :class="lesson.lesson_ulid === selectedId ? 'bg-blue-100 text-blue-900' : 'bg-slate-50'" @dragover.stop.prevent @drop.stop.prevent="drop(group.chapter.chapter_ulid, lesson.lesson_ulid)">
        <button v-if="editable" type="button" :draggable="!locked" :disabled="locked" :aria-label="copy.reorder" class="cursor-grab" @dragstart.stop="start($event, group.chapter.chapter_ulid, lesson.lesson_ulid)" @dragend="dragged = null"><GripVertical class="h-4 w-4" /></button>
        <button type="button" class="min-w-0 flex-1 break-words text-left text-sm" :aria-current="selectedId === lesson.lesson_ulid ? 'true' : undefined" :disabled="locked" @click="emit('select', lesson)">{{ typeof lesson.title === 'string' ? lesson.title : copy.unavailable }}</button>
        <template v-if="editable">
          <button type="button" :aria-label="copy.up" :disabled="locked || lessonIndex === 0" class="disabled:opacity-25" @click="move(group.chapter.chapter_ulid, lesson.lesson_ulid, lessons(group)![lessonIndex - 1]!.lesson_ulid)"><ArrowUp class="h-3 w-3" /></button>
          <button type="button" :aria-label="copy.down" :disabled="locked || lessonIndex === lessons(group)!.length - 1" class="disabled:opacity-25" @click="move(group.chapter.chapter_ulid, lesson.lesson_ulid, lessons(group)![lessonIndex + 1]!.lesson_ulid)"><ArrowDown class="h-3 w-3" /></button>
        </template>
      </div>
    </div>
  </nav>
</template>
