<script setup lang="ts">
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
</script>

<template>
  <section class="mt-12 min-w-0" aria-labelledby="membership-comparison-title" data-testid="membership-comparison">
    <div class="mb-8 text-center md:mb-12">
      <h2 id="membership-comparison-title" class="text-3xl font-bold tracking-tight text-[#292929] md:text-4xl">
        {{ t.membership.comparison.title }}
      </h2>
      <p class="mx-auto mt-4 max-w-3xl text-sm leading-6 text-[#606060] md:text-base">
        {{ t.membership.comparison.subtitle }}
      </p>
    </div>

    <div
      class="max-w-full overflow-x-auto rounded-xl shadow-[0_4px_12px_rgba(0,0,0,0.12)] focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-primary"
      role="region"
      tabindex="0"
      aria-labelledby="membership-comparison-title"
    >
      <table class="w-full min-w-[600px] table-fixed border-collapse text-sm leading-6" aria-labelledby="membership-comparison-title">
        <colgroup>
          <col class="w-2/5" />
          <col class="w-[30%]" />
          <col class="w-[30%]" />
        </colgroup>
        <thead class="bg-[#2254f4] text-white">
          <tr>
            <th scope="col" class="px-6 py-3 text-left font-bold">{{ t.membership.comparison.audience }}</th>
            <th scope="col" class="px-6 py-3 text-center font-bold">
              {{ t.membership.comparison.affiliateAudience }}
              <span class="block text-xs">{{ t.membership.comparison.affiliatePrice }}</span>
            </th>
            <th scope="col" class="px-6 py-3 text-center font-bold">
              {{ t.membership.comparison.charterholderAudience }}
              <span class="block text-xs">{{ t.membership.comparison.charterholderPrice }}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in t.membership.comparison.rows" :key="row.label" class="bg-white even:bg-[#f8f9fa]">
            <th scope="row" class="px-6 py-2.5 text-left font-normal text-slate-900">{{ row.label }}</th>
            <td class="px-6 py-2.5 text-center text-slate-600">{{ row.affiliate }}</td>
            <td class="px-6 py-2.5 text-center text-slate-600">{{ row.charterholder }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
