<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { templateVariables } from "@/lib/notificationTemplate"

const props = defineProps<{
  modelValue: string
  schema: string
  texts: string[]
  target: HTMLInputElement | HTMLTextAreaElement | null
  label: string
  emptyLabel: string
  syntax: "message" | "mail"
}>()
const emit = defineEmits<{ "update:modelValue": [value: string] }>()
const selected = ref("")
const variables = computed(() => templateVariables(props.schema, props.texts))

async function insertVariable() {
  const variable = variables.value.find((item) => item.name === selected.value)
  const target = props.target
  selected.value = ""
  if (!variable || !target) return

  const token = props.syntax === "mail" ? `{{.${variable.name}}}` : `{{${variable.name}}}`
  const start = target.selectionStart ?? props.modelValue.length
  const end = target.selectionEnd ?? start
  emit("update:modelValue", props.modelValue.slice(0, start) + token + props.modelValue.slice(end))
  await nextTick()
  target.focus()
  target.setSelectionRange(start + token.length, start + token.length)
}
</script>

<template>
  <label class="mt-2 block w-full max-w-64 text-xs font-semibold text-slate-600">
    {{ label }}
    <select v-model="selected" class="mt-1 h-9 w-full rounded-md border border-slate-200 bg-white px-2 text-sm text-slate-800 disabled:bg-slate-100 disabled:text-slate-500" :disabled="!variables.length || !target" @change="insertVariable">
      <option value="" disabled>{{ variables.length ? label : emptyLabel }}</option>
      <option v-for="variable in variables" :key="variable.name" :value="variable.name">{{ variable.name }}</option>
    </select>
  </label>
</template>
