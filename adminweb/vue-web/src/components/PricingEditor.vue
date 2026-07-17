<script setup lang="ts">
import { Plus, Trash2 } from "lucide-vue-next"
import { computed, nextTick, ref, watch } from "vue"
import { useAdminLanguage } from "@/lib/language"

type SelectOption = {
  id: string
  label: string
  subtitle?: string
  durationMonths?: number
}

type StripeRef = {
  stripe_price_id: string
  stripe_product_id: string
}

type UnitPricing = {
  key: number
  unit_id: string
  access: StripeRef
  retake: StripeRef
  exemption: StripeRef
}

type UnlockPricing = StripeRef & {
  key: number
  target_id: string
}

type MembershipPricing = StripeRef & {
  key: number
  membership_id: string
  discount_coupon: string
  duration_months: number
}

type QualReviewPricing = StripeRef & {
  key: number
  qual_id: string
}

type PricingState = {
  package_coupon: string
  units: UnitPricing[]
  unlocks: UnlockPricing[]
  memberships: MembershipPricing[]
  qual_reviews: QualReviewPricing[]
}

const props = withDefaults(defineProps<{
  modelValue: string
  unitOptions?: SelectOption[]
  pipelineOptions?: SelectOption[]
  membershipOptions?: SelectOption[]
}>(), {
  unitOptions: () => [],
  pipelineOptions: () => [],
  membershipOptions: () => [],
})

const emit = defineEmits<{
  "update:modelValue": [value: string]
}>()

const { t } = useAdminLanguage()
const copy = computed(() => t.value.pricingEditor)

let nextRowKey = 1
let syncingFromProp = false

function rowKey() {
  const key = nextRowKey
  nextRowKey += 1
  return key
}

function emptyStripeRef(): StripeRef {
  return {
    stripe_price_id: "",
    stripe_product_id: "",
  }
}

function emptyState(): PricingState {
  return {
    package_coupon: "",
    units: [],
    unlocks: [],
    memberships: [],
    qual_reviews: [],
  }
}

const state = ref<PricingState>(emptyState())
const availableUnitOptions = computed(() => {
  const selected = new Set(state.value.units.map((unit) => unit.unit_id).filter(Boolean))
  return props.unitOptions.filter((option) => !selected.has(option.id))
})
const availableUnlockOptions = computed(() => {
  const selected = new Set(state.value.unlocks.map((unlock) => unlock.target_id).filter(Boolean))
  return props.pipelineOptions.filter((option) => !selected.has(option.id))
})

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : null
}

function stripeRef(value: unknown): StripeRef {
  const record = asRecord(value)
  return {
    stripe_price_id: String(record?.stripe_price_id || ""),
    stripe_product_id: String(record?.stripe_product_id || ""),
  }
}

function parsePricing(value: string): PricingState {
  let parsed: unknown = {}
  try {
    parsed = JSON.parse(value || "{}")
  } catch {
    return emptyState()
  }
  const record = asRecord(parsed) || {}
  const unlocks = asRecord(record.unlocks) || {}

  return {
    package_coupon: String(record.package_coupon || ""),
    units: Array.isArray(record.units)
      ? record.units.map((value) => {
          const unit = asRecord(value) || {}
          return {
            key: rowKey(),
            unit_id: String(unit.unit_id || ""),
            access: stripeRef(unit.access),
            retake: stripeRef(unit.retake),
            exemption: stripeRef(unit.exemption),
          }
        })
      : [],
    unlocks: Object.entries(unlocks).map(([targetId, value]) => ({
      key: rowKey(),
      target_id: targetId,
      ...stripeRef(value),
    })),
    memberships: Array.isArray(record.memberships)
      ? record.memberships.map((value) => {
          const membership = asRecord(value) || {}
          return {
            key: rowKey(),
            membership_id: String(membership.membership_id || ""),
            stripe_price_id: String(membership.stripe_price_id || ""),
            stripe_product_id: String(membership.stripe_product_id || ""),
            discount_coupon: String(membership.discount_coupon || ""),
            duration_months: Number(membership.duration_months) || 0,
          }
        })
      : [],
    qual_reviews: Array.isArray(record.qual_reviews)
      ? record.qual_reviews.map((value) => {
          const review = asRecord(value) || {}
          return {
            key: rowKey(),
            qual_id: String(review.qual_id || ""),
            stripe_price_id: String(review.stripe_price_id || ""),
            stripe_product_id: String(review.stripe_product_id || ""),
          }
        })
      : [],
  }
}

function serializedStripeRef(value: StripeRef) {
  const stripePriceId = value.stripe_price_id.trim()
  const stripeProductId = value.stripe_product_id.trim()
  if (!stripePriceId && !stripeProductId) return null
  return {
    stripe_product_id: stripeProductId,
    stripe_price_id: stripePriceId,
  }
}

function serializePricing(value: PricingState) {
  const out: Record<string, unknown> = {}
  const packageCoupon = value.package_coupon.trim()
  if (packageCoupon) out.package_coupon = packageCoupon

  if (value.units.length) {
    out.units = value.units.map((unit) => {
      const serialized: Record<string, unknown> = { unit_id: unit.unit_id.trim() }
      const access = serializedStripeRef(unit.access)
      const retake = serializedStripeRef(unit.retake)
      const exemption = serializedStripeRef(unit.exemption)
      if (access) serialized.access = access
      if (retake) serialized.retake = retake
      if (exemption) serialized.exemption = exemption
      return serialized
    })
  }

  const unlocks: Record<string, unknown> = {}
  for (const unlock of value.unlocks) {
    const targetId = unlock.target_id.trim()
    const price = serializedStripeRef(unlock)
    if (targetId || price) {
      unlocks[targetId] = price || {
        stripe_product_id: "",
        stripe_price_id: "",
      }
    }
  }
  if (Object.keys(unlocks).length) out.unlocks = unlocks

  if (value.memberships.length) {
    out.memberships = value.memberships.map((membership) => ({
      membership_id: membership.membership_id.trim(),
      stripe_product_id: membership.stripe_product_id.trim(),
      stripe_price_id: membership.stripe_price_id.trim(),
      discount_coupon: membership.discount_coupon.trim(),
      duration_months: Number(membership.duration_months) || 0,
    }))
  }

  if (value.qual_reviews.length) {
    out.qual_reviews = value.qual_reviews.map((review) => ({
      qual_id: review.qual_id.trim(),
      stripe_product_id: review.stripe_product_id.trim(),
      stripe_price_id: review.stripe_price_id.trim(),
    }))
  }

  return JSON.stringify(out, null, 2)
}

watch(
  () => props.modelValue,
  async (value) => {
    syncingFromProp = true
    state.value = parsePricing(value)
    await nextTick()
    syncingFromProp = false
  },
  { immediate: true },
)

watch(
  state,
  (value) => {
    if (syncingFromProp) return
    const serialized = serializePricing(value)
    if (serialized !== props.modelValue) emit("update:modelValue", serialized)
  },
  { deep: true },
)

function addUnit() {
  const option = availableUnitOptions.value[0]
  state.value.units.push({
    key: rowKey(),
    unit_id: option?.id || "",
    access: emptyStripeRef(),
    retake: emptyStripeRef(),
    exemption: emptyStripeRef(),
  })
}

function addUnlock() {
  const option = availableUnlockOptions.value[0]
  if (!option) return
  state.value.unlocks.push({
    key: rowKey(),
    target_id: option.id,
    ...emptyStripeRef(),
  })
}

function addMembership() {
  if (state.value.memberships.length >= 1) return
  const option = props.membershipOptions[0]
  state.value.memberships.push({
    key: rowKey(),
    membership_id: option?.id || "",
    stripe_price_id: "",
    stripe_product_id: "",
    discount_coupon: "",
    duration_months: Number(option?.durationMonths) || 0,
  })
}

function addQualReview() {
  state.value.qual_reviews.push({
    key: rowKey(),
    qual_id: "",
    ...emptyStripeRef(),
  })
}

function updateMembershipDuration(membership: MembershipPricing) {
  const option = props.membershipOptions.find((item) => item.id === membership.membership_id)
  if (option?.durationMonths) membership.duration_months = option.durationMonths
}

function unitOptionDisabled(optionId: string, currentKey: number) {
  return state.value.units.some((unit) => unit.key !== currentKey && unit.unit_id === optionId)
}

function unlockOptionDisabled(optionId: string, currentKey: number) {
  return state.value.unlocks.some((unlock) => unlock.key !== currentKey && unlock.target_id === optionId)
}
</script>

<template>
  <div class="grid gap-5">
    <section class="rounded-2xl bg-primary-50 p-4">
      <h4 class="text-sm font-semibold text-primary-700">{{ copy.title }}</h4>
      <p class="mt-1 text-sm leading-6 text-slate-600">{{ copy.stripePairHint }}</p>
    </section>

    <section class="rounded-2xl border border-edge bg-white p-4">
      <label class="grid gap-1.5">
        <span class="ui-label">{{ copy.packageCoupon }}</span>
        <input v-model="state.package_coupon" class="ui-input" :placeholder="copy.packageCouponPlaceholder" />
      </label>
    </section>

    <section class="rounded-2xl border border-edge bg-white p-4">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-hairline pb-3">
        <div>
          <h4 class="text-base font-semibold text-ink">{{ copy.unitsTitle }}</h4>
          <p class="mt-1 text-xs text-slate-400">{{ copy.unitsHint }}</p>
        </div>
        <button type="button" class="ui-btn ui-btn-secondary ui-btn-sm" :disabled="unitOptions.length > 0 && !availableUnitOptions.length" @click="addUnit">
          <Plus class="h-4 w-4 text-slate-400" />
          {{ copy.addUnit }}
        </button>
      </div>

      <div v-if="!state.units.length" class="rounded-[10px] border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
        {{ copy.emptyUnits }}
      </div>

      <div v-else class="grid gap-4">
        <div v-for="(unit, index) in state.units" :key="unit.key" class="rounded-2xl bg-slate-50 p-4">
          <div class="mb-4 flex items-center justify-between gap-3">
            <label class="grid flex-1 gap-1.5">
              <span class="ui-label">{{ copy.unitId }}</span>
              <select v-if="unitOptions.length" v-model="unit.unit_id" class="ui-input">
                <option value="">{{ copy.selectUnit }}</option>
                <option v-for="option in unitOptions" :key="option.id" :value="option.id" :disabled="unitOptionDisabled(option.id, unit.key)">{{ option.label }}</option>
              </select>
              <input v-else v-model="unit.unit_id" class="ui-input tnum font-mono text-xs" :placeholder="copy.unitIdPlaceholder" />
            </label>
            <button type="button" class="mt-[26px] inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-red-50 hover:text-red-600" :aria-label="copy.delete" @click="state.units.splice(index, 1)">
              <Trash2 class="h-4 w-4" />
            </button>
          </div>

          <div class="grid gap-3 xl:grid-cols-3">
            <div v-for="priceType in (['access', 'retake', 'exemption'] as const)" :key="priceType" class="rounded-[10px] border border-edge bg-white p-3">
              <div class="mb-3 text-xs font-medium uppercase tracking-wider text-slate-400">{{ copy[priceType] }}</div>
              <label class="grid gap-1.5">
                <span class="text-xs font-medium text-slate-500">{{ copy.productId }}</span>
                <input v-model="unit[priceType].stripe_product_id" class="ui-input tnum font-mono text-xs" placeholder="prod_..." />
              </label>
              <label class="mt-3 grid gap-1.5">
                <span class="text-xs font-medium text-slate-500">{{ copy.priceId }}</span>
                <input v-model="unit[priceType].stripe_price_id" class="ui-input tnum font-mono text-xs" placeholder="price_..." />
              </label>
            </div>
          </div>
        </div>
      </div>
    </section>

    <section class="rounded-2xl border border-edge bg-white p-4">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-hairline pb-3">
        <div>
          <h4 class="text-base font-semibold text-ink">{{ copy.membershipsTitle }}</h4>
          <p class="mt-1 text-xs text-slate-400">{{ copy.membershipsHint }}</p>
        </div>
        <button type="button" class="ui-btn ui-btn-secondary ui-btn-sm" :disabled="state.memberships.length >= 1 || !membershipOptions.length" @click="addMembership">
          <Plus class="h-4 w-4 text-slate-400" />
          {{ copy.addMembership }}
        </button>
      </div>

      <div v-if="!state.memberships.length" class="rounded-[10px] border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
        {{ membershipOptions.length ? copy.emptyMemberships : copy.noLinkedMembership }}
      </div>

      <div v-for="(membership, index) in state.memberships" v-else :key="membership.key" class="rounded-2xl bg-slate-50 p-4">
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.membershipId }}</span>
            <select v-model="membership.membership_id" class="ui-input" @change="updateMembershipDuration(membership)">
              <option value="">{{ copy.selectMembership }}</option>
              <option v-for="option in membershipOptions" :key="option.id" :value="option.id">{{ option.label }}</option>
            </select>
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.durationMonths }}</span>
            <input v-model.number="membership.duration_months" type="number" min="1" class="ui-input tnum" />
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.productId }}</span>
            <input v-model="membership.stripe_product_id" class="ui-input tnum font-mono text-xs" placeholder="prod_..." />
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.priceId }}</span>
            <input v-model="membership.stripe_price_id" class="ui-input tnum font-mono text-xs" placeholder="price_..." />
          </label>
          <label class="grid gap-1.5 lg:col-span-2">
            <span class="ui-label">{{ copy.discountCoupon }}</span>
            <input v-model="membership.discount_coupon" class="ui-input tnum font-mono text-xs" :placeholder="copy.discountCouponPlaceholder" />
          </label>
        </div>
        <div class="mt-4 flex justify-end">
          <button type="button" class="inline-flex h-8 items-center gap-2 rounded-lg px-3 text-sm font-medium text-red-600 transition-colors hover:bg-red-50" @click="state.memberships.splice(index, 1)">
            <Trash2 class="h-4 w-4" />
            {{ copy.delete }}
          </button>
        </div>
      </div>
    </section>

    <section class="rounded-2xl border border-edge bg-white p-4">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-hairline pb-3">
        <div>
          <h4 class="text-base font-semibold text-ink">{{ copy.qualReviewsTitle }}</h4>
          <p class="mt-1 text-xs text-slate-400">{{ copy.qualReviewsHint }}</p>
        </div>
        <button type="button" class="ui-btn ui-btn-secondary ui-btn-sm" @click="addQualReview">
          <Plus class="h-4 w-4 text-slate-400" />
          {{ copy.addQualReview }}
        </button>
      </div>

      <div v-if="!state.qual_reviews.length" class="rounded-[10px] border border-dashed border-slate-200 p-4 text-center text-sm text-slate-500">
        {{ copy.emptyQualReviews }}
      </div>

      <div v-else class="grid gap-3">
        <div v-for="(review, index) in state.qual_reviews" :key="review.key" class="grid gap-3 rounded-2xl bg-slate-50 p-4 lg:grid-cols-[1fr_1fr_1fr_auto]">
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.qualId }}</span>
            <input v-model="review.qual_id" class="ui-input tnum font-mono text-xs" :placeholder="copy.qualIdPlaceholder" />
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.productId }}</span>
            <input v-model="review.stripe_product_id" class="ui-input tnum font-mono text-xs" placeholder="prod_..." />
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.priceId }}</span>
            <input v-model="review.stripe_price_id" class="ui-input tnum font-mono text-xs" placeholder="price_..." />
          </label>
          <button type="button" class="mt-[26px] inline-flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-red-50 hover:text-red-600" :aria-label="copy.delete" @click="state.qual_reviews.splice(index, 1)">
            <Trash2 class="h-4 w-4" />
          </button>
        </div>
      </div>
    </section>

    <section class="rounded-2xl bg-amber-50 p-4">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-amber-200 pb-3">
        <div>
          <h4 class="text-base font-semibold text-amber-700">{{ copy.unlocksTitle }}</h4>
          <p class="mt-1 text-xs leading-5 text-amber-700">{{ copy.unlocksHint }}</p>
        </div>
        <button type="button" class="ui-btn ui-btn-secondary ui-btn-sm" :disabled="!availableUnlockOptions.length" @click="addUnlock">
          <Plus class="h-4 w-4 text-slate-400" />
          {{ copy.addUnlock }}
        </button>
      </div>

      <div v-if="!state.unlocks.length" class="rounded-[10px] border border-dashed border-amber-200 bg-white/70 p-4 text-center text-sm text-amber-700">
        {{ pipelineOptions.length ? copy.emptyUnlocks : copy.noLinkedPipeline }}
      </div>

      <div v-else class="grid gap-3">
        <div v-for="(unlock, index) in state.unlocks" :key="unlock.key" class="grid gap-3 rounded-2xl bg-white p-4 lg:grid-cols-[1fr_1fr_1fr_auto]">
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.targetId }}</span>
            <select v-model="unlock.target_id" class="ui-input">
              <option value="">{{ copy.selectPipeline }}</option>
              <option v-for="option in pipelineOptions" :key="option.id" :value="option.id" :disabled="unlockOptionDisabled(option.id, unlock.key)">{{ option.label }}</option>
            </select>
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.productId }}</span>
            <input v-model="unlock.stripe_product_id" class="ui-input tnum font-mono text-xs" placeholder="prod_..." />
          </label>
          <label class="grid gap-1.5">
            <span class="ui-label">{{ copy.priceId }}</span>
            <input v-model="unlock.stripe_price_id" class="ui-input tnum font-mono text-xs" placeholder="price_..." />
          </label>
          <button type="button" class="mt-[26px] inline-flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 transition-colors hover:bg-red-50 hover:text-red-600" :aria-label="copy.delete" @click="state.unlocks.splice(index, 1)">
            <Trash2 class="h-4 w-4" />
          </button>
        </div>
      </div>
    </section>
  </div>
</template>
