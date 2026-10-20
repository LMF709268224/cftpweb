<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useAdminLanguage } from "@/lib/language"
import { objectValue, parameterFields } from "@/lib/notificationTemplate"

const props = defineProps<{ modelValue: string; schema: string; texts: string[]; rawLabel: string; disabled?: boolean }>()
const emit = defineEmits<{ "update:modelValue": [value: string] }>()
const { isZh } = useAdminLanguage()
const error = ref("")
const state = computed(() => {
  try {
    const fields = parameterFields(props.schema, props.texts)
    const value: unknown = JSON.parse(props.modelValue.trim() || "{}")
    if (!objectValue(value)) return null
    const schema: unknown = JSON.parse(props.schema || "{}")
    const complex = objectValue(schema) && ((schema.type !== undefined && schema.type !== "object") || Object.keys(schema).some(key => !["$schema", "$id", "title", "description", "type", "properties", "required"].includes(key)))
    return { fields, value, complex }
  } catch { return null }
})
const fields = computed(() => state.value?.fields.filter(field => field.supported) || [])
const advanced = computed(() => !state.value || state.value.complex || state.value.fields.some(field => !field.supported))
watch(() => [props.schema, props.modelValue], () => { error.value = "" })

function update(name: string, value: string, type: string) {
  if (!state.value) return
  const values = { ...state.value.value }
  if (value === "" && type !== "string") delete values[name]
  const nextValue = type === "boolean" ? value === "true" : type === "number" || type === "integer" ? Number(value) : value
  emit("update:modelValue", JSON.stringify(value === "" && type !== "string" ? values : { ...values, [name]: nextValue }, null, 2))
  error.value = ""
}

function validate() {
  error.value = ""
  if (!state.value) {
    error.value = isZh.value ? "请检查参数 JSON 和模板参数规则。" : "Check the parameter JSON and template schema."
    return false
  }
  for (const field of fields.value) {
    const value = state.value.value[field.name]
    const rule = field.rule
    const fail = () => {
      error.value = isZh.value ? `请检查参数 ${field.name}：${field.required ? "必填；" : ""}值须符合模板规定的类型和格式。` : `Check ${field.name}: ${field.required ? "required; " : ""}the value must match the template type and format.`
      return false
    }
    if (value === undefined || value === "") {
      if (field.required || (value === "" && Number(rule.minLength) > 0)) return fail()
      if (value === undefined) continue
    }
    if (rule.type !== undefined && typeof value !== (field.type === "integer" ? "number" : field.type)) return fail()
    if (Array.isArray(rule.enum) && !rule.enum.includes(value)) return fail()
    if (typeof value === "number") {
      if (!Number.isFinite(value) || (field.type === "integer" && !Number.isInteger(value))) return fail()
      if (typeof rule.minimum === "number" && value < rule.minimum) return fail()
      if (typeof rule.maximum === "number" && value > rule.maximum) return fail()
    }
    if (typeof value === "string") {
      if (typeof rule.minLength === "number" && [...value].length < rule.minLength) return fail()
      if (typeof rule.maxLength === "number" && [...value].length > rule.maxLength) return fail()
      if (typeof rule.pattern === "string") {
        try { if (!new RegExp(rule.pattern).test(value)) return fail() } catch { return fail() }
      }
      if (value && rule.format === "email" && !/^[^\s@]+@[^\s@]+$/.test(value)) return fail()
      if (value && rule.format === "uri") {
        try { new URL(value) } catch { return fail() }
      }
      if (value && rule.format === "date" && (!/^\d{4}-\d{2}-\d{2}$/.test(value) || Number.isNaN(Date.parse(value)) || new Date(value).toISOString().slice(0, 10) !== value)) return fail()
      if (value && rule.format === "date-time" && (!/^\d{4}-\d{2}-\d{2}T.*(?:Z|[+-]\d{2}:\d{2})$/i.test(value) || Number.isNaN(Date.parse(value)))) return fail()
    }
  }
  return true
}
defineExpose({ validate })
</script>

<template>
  <div class="space-y-3">
    <fieldset v-if="fields.length" :disabled="disabled" class="space-y-3">
      <legend class="mb-2 font-bold">{{ isZh ? '模板参数' : 'Template parameters' }}</legend>
      <label v-for="field in fields" :key="field.name" class="block">
        <span class="text-sm font-semibold">{{ field.rule.title || field.name }}<span v-if="field.required" class="ml-1 text-red-600">*</span></span>
        <select v-if="field.type === 'boolean' || Array.isArray(field.rule.enum)" :aria-label="field.name" :aria-required="field.required" :value="String(state?.value[field.name] ?? '')" class="mt-1 h-11 w-full rounded-xl border border-slate-200 px-3" @change="update(field.name, ($event.target as HTMLSelectElement).value, field.type)">
          <option value="">{{ isZh ? '请选择' : 'Select' }}</option>
          <option v-for="option in (Array.isArray(field.rule.enum) ? field.rule.enum : [true, false])" :key="String(option)" :value="String(option)">{{ String(option) }}</option>
        </select>
        <input v-else :aria-label="field.name" :aria-required="field.required" :value="state?.value[field.name] ?? ''" :type="field.type === 'number' || field.type === 'integer' ? 'number' : field.rule.format === 'email' ? 'email' : 'text'" :step="field.type === 'integer' ? 1 : 'any'" class="mt-1 h-11 w-full rounded-xl border border-slate-200 px-3" @input="update(field.name, ($event.target as HTMLInputElement).value, field.type)" />
        <span v-if="field.rule.description" class="mt-1 block text-xs text-slate-500">{{ field.rule.description }}</span>
      </label>
    </fieldset>
    <p v-if="advanced" class="text-sm text-amber-700">{{ isZh ? '部分参数规则无法通过简单表单编辑，请使用 JSON；复杂规则仍由服务端校验。' : 'Some rules require JSON editing; complex rules are validated by the server.' }}</p>
    <details :open="advanced || !fields.length">
      <summary class="cursor-pointer text-sm font-semibold text-slate-600">{{ isZh ? 'JSON 编辑' : 'Edit JSON' }}</summary>
      <label class="mt-2 block">
        <span class="font-bold">{{ rawLabel }}</span>
        <textarea :value="modelValue" :disabled="disabled" class="mt-2 min-h-52 w-full rounded-xl border border-slate-200 p-4 font-mono text-sm" @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value); error = ''" />
      </label>
    </details>
    <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }}</p>
  </div>
</template>
