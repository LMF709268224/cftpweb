<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { Check, GraduationCap, Loader2, X } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"
import { useBodyScrollLock } from "@/lib/bodyScrollLock"
import { useTranslation } from "@/lib/language"

type ExemptionQualification = string | {
  qual_id?: string
  cred_def_ulid?: string
  name?: string
  name_hint?: string
  eligible?: boolean
  credential_status?: string
}

type StageExemptionUnit = {
  unit_id?: string
  name?: string
  unit_name?: string
  allow_exemption?: boolean
  exemption_quals?: ExemptionQualification[]
}

type StageExemptionStage = {
  stage_id?: string
  stage_cc_ulid?: string
  name?: string
  stage_name?: string
  units?: StageExemptionUnit[]
}

const props = defineProps<{
  open: boolean
  stage: StageExemptionStage | null
  submitting?: boolean
}>()

const emit = defineEmits<{
  "update:open": [value: boolean]
  submit: [selectedUnitIds: string[]]
}>()

const { t } = useTranslation()
const selectedUnitIds = ref<Record<string, boolean>>({})
const eligibleQualificationIds = ref<Set<string>>(new Set())
const eligibilityLoading = ref(false)
const eligibilityLoadFailed = ref(false)

useBodyScrollLock(() => props.open)

function firstString(...values: unknown[]) {
  for (const value of values) {
    const normalized = String(value || "").trim()
    if (normalized) return normalized
  }
  return ""
}

function qualificationId(qualification: ExemptionQualification) {
  if (typeof qualification === "string") return qualification.trim()
  return firstString(qualification.qual_id, qualification.cred_def_ulid)
}

function qualificationIsEligible(qualification: ExemptionQualification) {
  if (typeof qualification === "string") {
    return eligibleQualificationIds.value.has(qualification.trim())
  }
  const status = String(qualification.credential_status || "").trim().toUpperCase()
  return Boolean(
    qualification.eligible
    || status === "CREDENTIAL_STATUS_ACTIVE"
    || eligibleQualificationIds.value.has(qualificationId(qualification)),
  )
}

const exemptionUnits = computed(() =>
  (props.stage?.units || []).filter((unit) =>
    Boolean(unit.allow_exemption || (unit.exemption_quals || []).some((qualification) => qualificationId(qualification))),
  ),
)

const selectedCount = computed(() =>
  exemptionUnits.value.filter((unit) => unit.unit_id && selectedUnitIds.value[unit.unit_id]).length,
)

function unitIsEligible(unit: StageExemptionUnit) {
  return (unit.exemption_quals || []).some(qualificationIsEligible)
}

function unitLabel(unit: StageExemptionUnit) {
  return firstString(unit.unit_name, unit.name, unit.unit_id, t.value.common.unknown)
}

function qualificationLabel(qualification: ExemptionQualification) {
  if (typeof qualification === "string") return qualification
  return firstString(
    qualification.name,
    qualification.name_hint,
    qualification.qual_id,
    qualification.cred_def_ulid,
    t.value.common.unknown,
  )
}

function close() {
  if (props.submitting) return
  emit("update:open", false)
}

function toggleUnit(unit: StageExemptionUnit, checked: boolean) {
  if (!unit.unit_id || !unitIsEligible(unit)) return
  selectedUnitIds.value = {
    ...selectedUnitIds.value,
    [unit.unit_id]: checked,
  }
}

function submit() {
  const selected = exemptionUnits.value
    .filter((unit) => unit.unit_id && unitIsEligible(unit) && selectedUnitIds.value[unit.unit_id])
    .map((unit) => String(unit.unit_id))
  emit("submit", selected)
}

async function loadEligibility() {
  selectedUnitIds.value = {}
  eligibleQualificationIds.value = new Set()
  eligibilityLoadFailed.value = false

  const qualificationIds = Array.from(new Set(
    exemptionUnits.value
      .flatMap((unit) => unit.exemption_quals || [])
      .map(qualificationId)
      .filter(Boolean),
  ))
  if (qualificationIds.length === 0) return

  eligibilityLoading.value = true
  try {
    const response = await apiClient(
      `/api/credentials/qualifications?qual_ulids=${encodeURIComponent(qualificationIds.join(","))}`,
      { suppressErrorToast: true },
    )
    eligibleQualificationIds.value = new Set(
      (Array.isArray(response?.qualifications) ? response.qualifications : [])
        .filter((qualification: any) =>
          Boolean(
            qualification?.eligible
            || String(qualification?.credential_status || "").trim().toUpperCase() === "CREDENTIAL_STATUS_ACTIVE",
          ),
        )
        .map((qualification: any) => firstString(qualification?.qual_id, qualification?.cred_def_ulid))
        .filter(Boolean),
    )
  } catch (error) {
    console.error("Failed to load stage exemption eligibility", error)
    eligibilityLoadFailed.value = true
  } finally {
    eligibilityLoading.value = false
  }
}

watch(
  () => [props.open, props.stage?.stage_id, props.stage?.stage_cc_ulid],
  ([open]) => {
    if (open) void loadEligibility()
  },
  { immediate: true },
)
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4 backdrop-blur-sm">
    <div class="flex max-h-[92vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl shadow-slate-950/20">
      <div class="flex items-start justify-between gap-4 border-b border-slate-100 px-6 py-5">
        <div class="flex min-w-0 items-start gap-3">
          <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-blue-50 text-primary">
            <GraduationCap class="h-5 w-5" />
          </div>
          <div class="min-w-0">
            <h2 class="text-xl font-bold text-slate-950">{{ t.learning.stageExemptionTitle }}</h2>
            <p class="mt-1 text-sm text-slate-500">
              {{ stage?.stage_name || stage?.name || t.learning.stageExemptionStageFallback }}
            </p>
          </div>
        </div>
        <button
          type="button"
          class="rounded-xl p-2 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700"
          :aria-label="t.paymentSession.close"
          :disabled="submitting"
          @click="close"
        >
          <X class="h-5 w-5" />
        </button>
      </div>

      <div class="overflow-y-auto px-6 py-5">
        <div class="rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm leading-6 text-blue-950">
          {{ t.learning.stageExemptionDesc }}
        </div>

        <div v-if="eligibilityLoading" class="mt-4 flex items-center justify-center gap-2 rounded-xl border border-slate-200 p-8 text-sm text-slate-500">
          <Loader2 class="h-4 w-4 animate-spin" />
          {{ t.learning.stageExemptionLoading }}
        </div>

        <div v-else-if="eligibilityLoadFailed" class="mt-4 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-900">
          {{ t.learning.stageExemptionLoadFailed }}
        </div>

        <div v-else-if="exemptionUnits.length === 0" class="mt-4 rounded-xl border border-dashed border-slate-200 p-8 text-center text-sm text-slate-500">
          {{ t.learning.stageExemptionEmpty }}
        </div>

        <div v-else class="mt-4 space-y-3">
          <label
            v-for="unit in exemptionUnits"
            :key="unit.unit_id || unitLabel(unit)"
            :class="[
              'flex gap-3 rounded-xl border p-4 transition-colors',
              unitIsEligible(unit)
                ? 'cursor-pointer border-slate-200 hover:border-primary/40 hover:bg-blue-50/40'
                : 'cursor-not-allowed border-slate-100 bg-slate-50 opacity-75',
              unit.unit_id && selectedUnitIds[unit.unit_id]
                ? 'border-emerald-300 bg-emerald-50/70 ring-1 ring-emerald-100'
                : '',
            ]"
          >
            <input
              class="sr-only"
              type="checkbox"
              :checked="Boolean(unit.unit_id && selectedUnitIds[unit.unit_id])"
              :disabled="!unitIsEligible(unit)"
              @change="toggleUnit(unit, ($event.target as HTMLInputElement).checked)"
            />
            <span
              :class="[
                'mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-md border',
                unit.unit_id && selectedUnitIds[unit.unit_id]
                  ? 'border-emerald-500 bg-emerald-500 text-white'
                  : 'border-slate-300 bg-white text-transparent',
              ]"
            >
              <Check class="h-3.5 w-3.5 stroke-[3]" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="font-semibold text-slate-950">{{ unitLabel(unit) }}</span>
                <span
                  :class="[
                    'badge text-xs',
                    unitIsEligible(unit)
                      ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
                      : 'border-amber-200 bg-amber-50 text-amber-700',
                  ]"
                >
                  {{ unitIsEligible(unit) ? t.learning.stageExemptionEligible : t.learning.stageExemptionUnavailable }}
                </span>
              </span>
              <span class="mt-2 flex flex-wrap gap-1.5">
                <span
                  v-for="qualification in unit.exemption_quals || []"
                  :key="qualificationId(qualification)"
                  class="rounded-full border border-slate-200 bg-white px-2 py-1 text-xs text-slate-500"
                >
                  {{ qualificationLabel(qualification) }}
                </span>
              </span>
            </span>
          </label>
        </div>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-slate-100 px-6 py-4">
        <span class="text-sm text-slate-500">
          {{ t.learning.stageExemptionSelected.replace("{{count}}", String(selectedCount)) }}
        </span>
        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" class="btn btn-outline rounded-lg" :disabled="submitting" @click="close">
            {{ t.common.cancel }}
          </button>
          <button
            type="button"
            class="btn btn-primary rounded-lg"
            :disabled="eligibilityLoading || submitting"
            @click="submit"
          >
            <Loader2 v-if="submitting" class="mr-2 h-4 w-4 animate-spin" />
            {{ submitting
              ? t.learning.stageExemptionSubmitting
              : selectedCount > 0
                ? t.learning.stageExemptionConfirm
                : t.learning.stageExemptionContinueWithout
            }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
