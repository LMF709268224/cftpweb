<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue"
import { EditorContent, useEditor } from "@tiptap/vue-3"
import StarterKit from "@tiptap/starter-kit"
import Image from "@tiptap/extension-image"
import { TableKit } from "@tiptap/extension-table"
import { useCourseAuthoringCopy } from "@/lib/courseAuthoringCopy"
import { courseContentHtml, supportsVisualCourseContent } from "@/lib/courseContentHtml"

const props = defineProps<{ modelValue: string; disabled?: boolean }>()
const emit = defineEmits<{ "update:modelValue": [value: string] }>()
const copy = useCourseAuthoringCopy()
const mode = ref<"visual" | "source">("visual")
const insertion = ref<"link" | "image" | null>(null)
const url = ref("")
const alt = ref("")
const urlError = ref(false)
const invalidMarkup = ref(!supportsVisualCourseContent(props.modelValue))
const preview = computed(() => courseContentHtml(props.modelValue))
const editor = useEditor({
  extensions: [StarterKit.configure({ strike: false, link: { openOnClick: false, autolink: false } }), Image, TableKit],
  content: preview.value,
  enableContentCheck: true,
  onContentError: () => { invalidMarkup.value = true },
  editorProps: { attributes: { role: "textbox", "aria-multiline": "true", "aria-label": copy.value.editorLabel } },
  onUpdate: ({ editor: current }) => { emit("update:modelValue", current.getHTML()) },
})
watch(() => props.modelValue, (value) => {
  if (!editor.value || value === editor.value.getHTML()) return
  try {
    editor.value.commands.setContent(courseContentHtml(value), { emitUpdate: false, errorOnInvalidContent: true })
    invalidMarkup.value = !supportsVisualCourseContent(value)
  } catch { invalidMarkup.value = true }
})
watch([editor, () => props.disabled, invalidMarkup], () => editor.value?.setEditable(!props.disabled && !invalidMarkup.value))
watch(copy, () => editor.value?.setOptions({ editorProps: { attributes: { role: "textbox", "aria-multiline": "true", "aria-label": copy.value.editorLabel } } }))
onBeforeUnmount(() => editor.value?.destroy())

function beginInsert(kind: "link" | "image") {
  insertion.value = kind
  url.value = ""
  alt.value = ""
  urlError.value = false
}
function insert() {
  if (props.disabled || invalidMarkup.value) return
  try {
    const parsed = new URL(url.value.trim())
    if (parsed.protocol !== "https:" || parsed.username || parsed.password) throw new Error("Invalid URL")
    if (insertion.value === "image") editor.value?.chain().focus().setImage({ src: parsed.href, alt: alt.value }).run()
    else if (editor.value?.state.selection.empty) editor.value.chain().focus().insertContent({ type: "text", text: parsed.href, marks: [{ type: "link", attrs: { href: parsed.href } }] }).run()
    else editor.value?.chain().focus().setLink({ href: parsed.href }).run()
    insertion.value = null
  } catch { urlError.value = true }
}
</script>

<template>
  <div class="space-y-3" data-testid="course-body-editor">
    <div class="flex flex-wrap gap-2">
      <button type="button" class="mode" :aria-pressed="mode === 'visual'" @click="mode = 'visual'">{{ copy.visual }}</button>
      <button type="button" class="mode" :aria-pressed="mode === 'source'" @click="mode = 'source'">{{ copy.source }}</button>
    </div>
    <p class="text-xs text-slate-500">{{ copy.markupNotice }}</p>
    <div class="grid gap-4 xl:grid-cols-2">
      <div class="min-w-0 rounded-xl border border-slate-200 bg-white">
        <template v-if="mode === 'visual' && editor">
          <div class="flex flex-wrap gap-1 border-b border-slate-200 p-2" role="toolbar" :aria-label="copy.visual">
            <button type="button" :disabled="disabled || invalidMarkup" :aria-pressed="editor.isActive('bold')" @click="editor.chain().focus().toggleBold().run()">{{ copy.bold }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" :aria-pressed="editor.isActive('italic')" @click="editor.chain().focus().toggleItalic().run()">{{ copy.italic }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" :aria-pressed="editor.isActive('underline')" @click="editor.chain().focus().toggleUnderline().run()">{{ copy.underline }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" @click="editor.chain().focus().toggleHeading({ level: 2 }).run()">{{ copy.heading }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" @click="editor.chain().focus().setParagraph().run()">{{ copy.paragraph }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" @click="editor.chain().focus().toggleBulletList().run()">{{ copy.bullet }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" @click="editor.chain().focus().toggleOrderedList().run()">{{ copy.numbered }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" @click="beginInsert('link')">{{ copy.link }}</button>
            <button type="button" :disabled="disabled || invalidMarkup || !editor.isActive('link')" @click="editor.chain().focus().unsetLink().run()">{{ copy.unlink }}</button>
            <button type="button" :disabled="disabled || invalidMarkup" @click="beginInsert('image')">{{ copy.image }}</button>
            <button type="button" :disabled="disabled || invalidMarkup || !editor.can().undo()" @click="editor.chain().focus().undo().run()">{{ copy.undo }}</button>
            <button type="button" :disabled="disabled || invalidMarkup || !editor.can().redo()" @click="editor.chain().focus().redo().run()">{{ copy.redo }}</button>
          </div>
          <div v-if="insertion" class="space-y-2 border-b bg-slate-50 p-3">
            <label class="grid gap-1 text-sm">{{ copy.url }}<input v-model="url" type="url" class="rounded border bg-white p-2" /></label>
            <label v-if="insertion === 'image'" class="grid gap-1 text-sm">{{ copy.alt }}<input v-model="alt" class="rounded border bg-white p-2" /></label>
            <p v-if="urlError" role="alert" class="text-sm text-red-700">{{ copy.invalidUrl }}</p>
            <button type="button" class="mode" :disabled="disabled || invalidMarkup" @click="insert">{{ copy.insert }}</button>
            <button type="button" class="mode" @click="insertion = null">{{ copy.cancel }}</button>
          </div>
          <p v-if="invalidMarkup" role="alert" class="p-3 text-sm text-amber-800">{{ copy.invalidMarkup }}</p>
          <EditorContent :editor="editor" class="course-content p-4" />
        </template>
        <textarea v-else :value="modelValue" :disabled="disabled" :aria-label="copy.sourceLabel" class="min-h-80 w-full resize-y rounded-xl p-4 font-mono text-sm" @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)" />
      </div>
      <section class="min-w-0 rounded-xl border border-slate-200 bg-slate-50 p-4" :aria-label="copy.preview">
        <h4 class="mb-4 text-sm font-bold text-slate-500">{{ copy.preview }}</h4>
        <div v-if="preview" class="course-content" v-html="preview" @click.prevent />
        <p v-else class="text-sm text-slate-400">{{ copy.previewEmpty }}</p>
      </section>
    </div>
  </div>
</template>

<style scoped>
.mode, [role="toolbar"] button { border: 1px solid #cbd5e1; border-radius: .4rem; padding: .35rem .6rem; font-size: .8rem; background: white; }
button[aria-pressed="true"] { background: #dbeafe; color: #1e40af; }
button:disabled { opacity: .4; cursor: not-allowed; }
.course-content { overflow-wrap: anywhere; line-height: 1.7; }
.course-content :deep(.tiptap) { min-height: 18rem; outline: none; }
.course-content :deep(p) { margin: .6em 0; }
.course-content :deep(h1), .course-content :deep(h2), .course-content :deep(h3) { font-weight: 700; font-size: 1.4em; margin: .8em 0; }
.course-content :deep(ul) { list-style: disc; padding-left: 1.5em; }
.course-content :deep(ol) { list-style: decimal; padding-left: 1.5em; }
.course-content :deep(a) { color: #1d4ed8; text-decoration: underline; }
.course-content :deep(img) { max-width: 100%; height: auto; }
.course-content :deep(blockquote) { border-left: 3px solid #cbd5e1; padding-left: 1em; }
.course-content :deep(pre) { white-space: pre-wrap; background: #f1f5f9; padding: .6em; }
.course-content :deep(table) { width: 100%; border-collapse: collapse; }
.course-content :deep(td), .course-content :deep(th) { border: 1px solid #cbd5e1; padding: .4em; }
</style>
