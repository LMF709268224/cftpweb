<script setup lang="ts">
import { computed, ref } from "vue"
import { VueDatePicker } from "@vuepic/vue-datepicker"
import "@vuepic/vue-datepicker/dist/main.css"
import { enGB, zhCN } from "date-fns/locale"
import { CalendarDays } from "lucide-vue-next"

const props = withDefaults(defineProps<{
  modelValue: string
  language?: "zh" | "en"
  placeholder?: string
  ariaLabel?: string
  disabled?: boolean
  min?: string
  max?: string
}>(), {
  language: "zh",
  placeholder: "DD/MM/YYYY",
  ariaLabel: "",
  disabled: false,
  min: undefined,
  max: undefined,
})

const emit = defineEmits<{
  "update:modelValue": [value: string]
}>()

const datePickerRef = ref<InstanceType<typeof VueDatePicker> | null>(null)
const dateLocale = computed(() => props.language === "zh" ? zhCN : enGB)
const calendarIconLabel = computed(() => props.language === "zh" ? "打开日期选择器" : "Open date picker")

function handleUpdate(value: string | null) {
  emit("update:modelValue", value || "")
}

function openDatePicker() {
  datePickerRef.value?.openMenu()
}
</script>

<template>
  <VueDatePicker
    ref="datePickerRef"
    class="localized-date-picker"
    :model-value="modelValue || null"
    :locale="dateLocale"
    model-type="yyyy-MM-dd"
    :formats="{ input: 'dd/MM/yyyy' }"
    :time-config="{ enableTimePicker: false }"
    :input-attrs="{ clearable: false, hideInputIcon: false, autocomplete: 'bday' }"
    :aria-labels="{ input: ariaLabel, calendarIcon: calendarIconLabel }"
    :placeholder="placeholder"
    :disabled="disabled"
    :min-date="min"
    :max-date="max"
    auto-apply
    @update:model-value="handleUpdate"
  >
    <template #input-icon>
      <button
        type="button"
        class="date-picker-trigger"
        :aria-label="calendarIconLabel"
        @click.stop="openDatePicker"
      >
        <CalendarDays class="h-4 w-4" aria-hidden="true" />
      </button>
    </template>
  </VueDatePicker>
</template>

<style scoped>
.localized-date-picker {
  --dp-font-family: inherit;
  --dp-font-size: 0.875rem;
  --dp-background-color: #ffffff;
  --dp-text-color: #002a66;
  --dp-hover-color: #e6efff;
  --dp-hover-text-color: #002a66;
  --dp-primary-color: #0957f9;
  --dp-primary-text-color: #ffffff;
  --dp-secondary-color: #60708d;
  --dp-border-color: rgba(0, 42, 102, 0.2);
  --dp-border-color-hover: rgba(0, 42, 102, 0.38);
  --dp-border-color-focus: #0957f9;
  --dp-menu-border-color: rgba(0, 42, 102, 0.16);
  --dp-icon-color: #60708d;
  --dp-border-radius: 0.5rem;
  --dp-cell-border-radius: 0.375rem;
}

.localized-date-picker :deep(.dp__input) {
  height: 2.5rem;
  border-radius: 0.5rem;
  color: #002a66;
  box-shadow: none;
}

.localized-date-picker :deep(.dp--input-icon) {
  inset-inline-start: auto;
  inset-inline-end: 0.75rem;
}

.date-picker-trigger {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  padding: 0;
  color: inherit;
  cursor: pointer;
  background: transparent;
  border: 0;
  place-items: center;
}

.localized-date-picker :deep(.dp--input-icon-pad) {
  padding-inline-start: 0.75rem !important;
  padding-inline-end: 2.5rem !important;
}

.localized-date-picker :deep(.dp__input:focus),
.localized-date-picker :deep(.dp__input_focus) {
  border-color: #0957f9;
  box-shadow: 0 0 0 3px rgba(9, 87, 249, 0.12);
}

.localized-date-picker :deep(.dp__menu) {
  box-shadow: 0 16px 36px rgba(0, 42, 102, 0.14);
}
</style>
