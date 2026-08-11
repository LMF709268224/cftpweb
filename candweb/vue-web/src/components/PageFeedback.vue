<script setup lang="ts">
import { computed } from "vue"
import { AlertCircle, Inbox, Loader2, RefreshCw } from "lucide-vue-next"

const props = withDefaults(defineProps<{
  kind: "loading" | "error" | "empty"
  title?: string
  description?: string
  loadingLabel?: string
  actionLabel?: string
  minHeightClass?: string
}>(), {
  title: "",
  description: "",
  loadingLabel: "",
  actionLabel: "",
  minHeightClass: "min-h-[220px]",
})

const emit = defineEmits<{
  action: []
}>()

const stateRole = computed(() => (props.kind === "error" ? "alert" : "status"))
</script>

<template>
  <section
    :class="['flex flex-col items-center justify-center rounded-xl border border-slate-200 bg-white px-4 py-10 text-center', minHeightClass]"
    :role="stateRole"
    aria-live="polite"
    :aria-busy="kind === 'loading'"
  >
    <div v-if="kind === 'loading'" class="flex items-center justify-center gap-2 text-muted-foreground">
      <Loader2 class="h-5 w-5 animate-spin text-primary" />
      <span>{{ loadingLabel }}</span>
    </div>

    <template v-else>
      <div :class="['mb-4 flex h-16 w-16 items-center justify-center rounded-xl', kind === 'error' ? 'bg-red-50 text-red-600' : 'bg-primary/10 text-primary']">
        <slot name="icon">
          <AlertCircle v-if="kind === 'error'" class="h-8 w-8" />
          <Inbox v-else class="h-8 w-8" />
        </slot>
      </div>
      <h2 v-if="title" class="text-lg font-semibold text-foreground">{{ title }}</h2>
      <p v-if="description" class="mt-2 max-w-md text-sm leading-6 text-muted-foreground">{{ description }}</p>
      <slot name="action">
        <button v-if="actionLabel" type="button" class="btn btn-primary mt-5 min-w-32 justify-center" @click="emit('action')">
          <RefreshCw class="h-4 w-4" />
          {{ actionLabel }}
        </button>
      </slot>
    </template>
  </section>
</template>
