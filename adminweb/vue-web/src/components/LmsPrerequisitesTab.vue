<script setup lang="ts">
import { Loader2, Plus, Trash2 } from "lucide-vue-next"
import { computed, onMounted, ref, watch } from "vue"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { apiErrorMessage } from "@/lib/apiErrorMessage"
import type { JsonRecord } from "@/lib/display"

const props = defineProps<{
  targetEntityType: 1 | 2 | 3 // 1: LESSON, 2: QUIZ, 3: CHAPTER
  targetEntityId: string
  course: JsonRecord | null // The CompleteCourse
  copy: any
}>()

const loading = ref(false)
const prerequisites = ref<JsonRecord[]>([])

const adding = ref(false)
const formRequiredEntity = ref("")

// Flatten all available entities from the course tree
const availableEntities = computed(() => {
  if (!props.course) return []
  
  const entities: { id: string; type: number; title: string; result: number }[] = []
  
  const chapters = (props.course.chapters || []) as any[]
  chapters.forEach((chapterDetail: any) => {
    const chapter = chapterDetail.chapter
    const chapterIdStr = String(chapter?.chapter_id || chapter?.chapter_ulid || "")
    if (chapter && chapterIdStr !== props.targetEntityId) {
      entities.push({
        id: chapterIdStr,
        type: 3, // CHAPTER
        title: `[${props.copy.chapter || '章节'}] ${chapter.title}`,
        result: 1 // COMPLETED
      })
    }
    
    const lessons = (chapterDetail.lessons || []) as any[]
    lessons.forEach((lessonDetail: any) => {
      const lesson = lessonDetail.lesson
      const lessonIdStr = String(lesson?.lesson_id || lesson?.lesson_ulid || "")
      if (lesson && lessonIdStr !== props.targetEntityId) {
        entities.push({
          id: lessonIdStr,
          type: 1, // LESSON
          title: `[${props.copy.lesson || '课时'}] ${lesson.title}`,
          result: 1 // COMPLETED
        })
      }
      
      const quizzes = (lessonDetail.quizzes || []) as any[]
      quizzes.forEach((quizDetail: any) => {
        const quiz = quizDetail.quiz
        const quizIdStr = String(quiz?.quiz_id || quiz?.quiz_ulid || "")
        if (quiz && quizIdStr !== props.targetEntityId) {
          entities.push({
            id: quizIdStr,
            type: 2, // QUIZ
            title: `[${props.copy.quiz || '测验'}] ${quiz.title}`,
            result: 2 // PASSED
          })
        }
      })
    })
    
    const chapterQuizzes = (chapterDetail.quizzes || []) as any[]
    chapterQuizzes.forEach((quizDetail: any) => {
      const quiz = quizDetail.quiz
      const quizIdStr = String(quiz?.quiz_id || quiz?.quiz_ulid || "")
      if (quiz && quizIdStr !== props.targetEntityId) {
        entities.push({
          id: quizIdStr,
          type: 2, // QUIZ
          title: `[${props.copy.quiz || '测验'}] ${quiz.title}`,
          result: 2 // PASSED
        })
      }
    })
  })
  
  const courseQuizzes = (props.course.quizzes || []) as any[]
  courseQuizzes.forEach((quizDetail: any) => {
    const quiz = quizDetail.quiz
    const quizIdStr = String(quiz?.quiz_id || quiz?.quiz_ulid || "")
    if (quiz && quizIdStr !== props.targetEntityId) {
      entities.push({
        id: quizIdStr,
        type: 2, // QUIZ
        title: `[${props.copy.quiz || '测验'}] ${quiz.title}`,
        result: 2 // PASSED
      })
    }
  })
  
  return entities
})

async function loadPrerequisites() {
  if (!props.targetEntityId) return
  
  loading.value = true
  try {
    const res = await apiClient<any>(`/api/lms/prerequisites?target_entity_type=${props.targetEntityType}&target_entity_id=${props.targetEntityId}`, {
      method: 'GET'
    })
    prerequisites.value = res?.prerequisites || []
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, props.copy.loadFailed || 'Load failed'))
  } finally {
    loading.value = false
  }
}

async function addPrerequisite() {
  if (!formRequiredEntity.value) return
  
  const entity = availableEntities.value.find(e => e.id === formRequiredEntity.value)
  if (!entity) return
  
  adding.value = true
  try {
    await apiClient("/api/lms/prerequisites", {
      method: "POST",
      body: JSON.stringify({
        required_entity_type: entity.type,
        required_entity_ulid: entity.id,
        required_result: entity.result,
        target_entity_type: props.targetEntityType,
        target_entity_ulid: props.targetEntityId
      })
    })
    toast.success(props.copy.addSuccess || 'Added successfully')
    formRequiredEntity.value = ""
    await loadPrerequisites()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, props.copy.addFailed || 'Add failed'))
  } finally {
    adding.value = false
  }
}

async function deletePrerequisite(prerequisite: JsonRecord) {
  if (!confirm(props.copy.deleteConfirm || 'Are you sure?')) return
  
  try {
    const pId = prerequisite.prerequisite_id || prerequisite.prerequisite_ulid
    const version = prerequisite.version || 1
    await apiClient(`/api/lms/prerequisites/${pId}?version=${version}`, { method: "DELETE" })
    toast.success(props.copy.deleteSuccess || 'Deleted successfully')
    await loadPrerequisites()
  } catch (err) {
    console.error(err)
    toast.error(apiErrorMessage(err, props.copy.deleteFailed || 'Delete failed'))
  }
}

function entityName(id: string) {
  const entity = availableEntities.value.find(e => e.id === id)
  return entity ? entity.title : id
}

watch(() => props.targetEntityId, (newId) => {
  if (newId) {
    loadPrerequisites()
  } else {
    prerequisites.value = []
  }
})

onMounted(() => {
  loadPrerequisites()
})
</script>

<template>
  <div class="space-y-6 rounded-2xl border border-slate-200 p-5">
    <div>
      <h3 class="text-base font-black">{{ copy.prerequisites || '前置条件' }}</h3>
      <p class="mt-1 text-sm text-slate-500">{{ copy.prerequisitesDesc || '添加必须完成的课时或测验' }}</p>
    </div>
    
    <div v-if="loading" class="flex justify-center p-8 text-slate-400">
      <Loader2 class="h-6 w-6 animate-spin" />
    </div>
    <div v-else>
      <div v-if="prerequisites.length === 0" class="rounded-xl border border-dashed border-slate-200 p-8 text-center text-sm font-semibold text-slate-500">
        {{ copy.noPrerequisites || '尚未添加前置条件' }}
      </div>
      <div v-else class="space-y-3">
        <div v-for="item in prerequisites" :key="String(item.prerequisite_id || item.prerequisite_ulid)" class="flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3">
          <div>
            <div class="font-bold text-slate-700">{{ entityName(String(item.required_entity_id || item.required_entity_ulid)) }}</div>
            <div class="mt-0.5 text-xs text-slate-500">
              {{ item.required_result === 2 || item.required_result === 'PREREQUISITE_RESULT_PASSED' ? (copy.mustPass || '必须通过') : (copy.mustComplete || '必须完成') }}
            </div>
          </div>
          <button class="shrink-0 rounded-full p-2 text-slate-400 transition hover:bg-slate-200 hover:text-red-500" :aria-label="copy.delete" @click="deletePrerequisite(item)">
            <Trash2 class="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
    
    <form class="flex items-end gap-3 border-t border-slate-200 pt-5" @submit.prevent="addPrerequisite">
      <label class="flex-1 space-y-2 text-sm font-bold">
        <span>{{ copy.requireEntity || '要求完成项' }}</span>
        <select v-model="formRequiredEntity" class="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 font-semibold outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500" required>
          <option value="" disabled>{{ copy.selectRequireEntity || '请选择一个课时/章节/测验' }}</option>
          <option v-for="entity in availableEntities" :key="entity.id" :value="entity.id">
            {{ entity.title }}
          </option>
        </select>
      </label>
      <button class="inline-flex h-10 shrink-0 items-center gap-1.5 rounded-xl bg-blue-700 px-4 font-bold text-white shadow-sm hover:bg-blue-800 disabled:opacity-50" type="submit" :disabled="!formRequiredEntity || adding">
        <Loader2 v-if="adding" class="h-4 w-4 animate-spin" />
        <Plus v-else class="h-4 w-4" />
        {{ copy.add || '添加' }}
      </button>
    </form>
  </div>
</template>
