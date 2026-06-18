<script setup lang="ts">
import { computed, ref } from "vue"
import { Check, Crown, Download, HelpCircle, Percent, Shield, Star, Video, Zap } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { useTranslation } from "@/lib/language"

const { t } = useTranslation()
const activeTab = ref("benefits")

const tabs = computed(() => [
  { id: "intro", label: t.value.membership.tabs.intro },
  { id: "benefits", label: t.value.membership.tabs.benefits },
  { id: "levels", label: t.value.membership.tabs.levels },
  { id: "settings", label: t.value.membership.tabs.settings },
  { id: "orders", label: t.value.membership.tabs.orders },
])

const benefits = computed(() => [
  { icon: Zap, title: t.value.membership.benefitsList.b1Title, description: t.value.membership.benefitsList.b1Desc },
  { icon: Video, title: t.value.membership.benefitsList.b2Title, description: t.value.membership.benefitsList.b2Desc },
  { icon: Download, title: t.value.membership.benefitsList.b3Title, description: t.value.membership.benefitsList.b3Desc },
  { icon: Shield, title: t.value.membership.benefitsList.b4Title, description: t.value.membership.benefitsList.b4Desc },
  { icon: Percent, title: t.value.membership.benefitsList.b5Title, description: t.value.membership.benefitsList.b5Desc },
  { icon: HelpCircle, title: t.value.membership.benefitsList.b6Title, description: t.value.membership.benefitsList.b6Desc },
])

const membershipLevels = computed(() => [
  { id: "basic", name: t.value.membership.levelsTitle.basic, englishName: t.value.membership.levelsEnglishName.basic, price: t.value.membership.priceFree, features: t.value.membership.basicBenefits },
  { id: "certified", name: t.value.membership.levelsTitle.certified, englishName: t.value.membership.levelsEnglishName.certified, price: t.value.membership.priceYearly1999, features: t.value.membership.certifiedBenefits, highlight: true },
  { id: "premium", name: t.value.membership.levelsTitle.premium, englishName: t.value.membership.levelsEnglishName.premium, price: t.value.membership.priceYearly4999, features: t.value.membership.premiumBenefits },
])
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <Crown class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.membership.title }}</span>
      </header>

      <main class="px-5 py-8 md:px-8 lg:px-10">
        <div class="mb-6">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.membership.title }}</h1>
          <p class="mt-2 text-muted-foreground">{{ t.membership.subtitle }}</p>
        </div>

    <div class="mb-4 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="flex items-center gap-3">
        <div class="flex h-11 w-11 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Crown class="h-5 w-5" />
        </div>
        <div>
          <h2 class="font-semibold text-card-foreground">{{ t.membership.currentMember }}</h2>
          <p class="text-sm text-muted-foreground">{{ t.membership.devNotice }}</p>
        </div>
      </div>
    </div>

    <div class="mb-4 rounded-md bg-white px-8 pt-6">
      <div class="flex flex-wrap gap-10 border-b border-[#edf0f2]">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          :class="['relative cursor-pointer whitespace-nowrap px-1 pb-7 text-base font-medium transition-colors duration-200', activeTab === tab.id ? 'text-primary' : 'text-[#111827] hover:text-primary']"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
          <span v-if="activeTab === tab.id" class="absolute bottom-[-1px] left-0 h-0.5 w-full rounded-full bg-primary" />
        </button>
      </div>
    </div>

    <div v-if="activeTab === 'benefits'" class="rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <h2 class="mb-4 text-lg font-semibold text-card-foreground">{{ t.membership.currentBenefits }}</h2>
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div v-for="benefit in benefits" :key="benefit.title" class="group flex gap-4 rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:shadow-md hover:shadow-primary/10">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105">
            <component :is="benefit.icon" class="h-5 w-5" />
          </div>
          <div>
            <h3 class="mb-1 font-medium text-card-foreground">{{ benefit.title }}</h3>
            <p class="text-sm text-muted-foreground">{{ benefit.description }}</p>
          </div>
        </div>
      </div>
    </div>

    <div v-if="activeTab === 'levels'" class="grid gap-4 md:grid-cols-3">
      <div v-for="level in membershipLevels" :key="level.id" :class="['relative overflow-hidden rounded-[16px] p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)] transition-all hover:-translate-y-0.5 hover:shadow-md', level.highlight ? 'bg-primary/5 shadow-primary/10' : 'bg-white']">
        <div :class="['absolute left-0 top-0 h-full w-1', level.highlight ? 'bg-primary' : level.id === 'premium' ? 'bg-amber-500' : 'bg-slate-300']" />
        <div class="mb-4 text-center">
          <div :class="['mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-xl', level.id === 'basic' ? 'bg-slate-100 text-slate-600' : level.id === 'certified' ? 'bg-primary/10 text-primary' : 'bg-amber-100 text-amber-600']">
            <Star v-if="level.id === 'basic'" class="h-7 w-7" />
            <Crown v-else class="h-7 w-7" />
          </div>
          <h3 class="text-lg font-semibold text-card-foreground">{{ level.name }}</h3>
          <p class="text-sm text-muted-foreground">{{ level.englishName }}</p>
        </div>
        <div class="mb-4 text-center"><span class="text-2xl font-bold text-card-foreground">{{ level.price }}</span></div>
        <ul class="space-y-3">
          <li v-for="feature in level.features" :key="feature" class="flex items-center gap-2 text-sm">
            <Check :class="['h-4 w-4 shrink-0', level.highlight ? 'text-primary' : 'text-emerald-500']" />
            <span class="text-card-foreground">{{ feature }}</span>
          </li>
        </ul>
      </div>
    </div>

    <div v-if="activeTab === 'intro'" class="rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <h2 class="mb-4 text-lg font-semibold text-card-foreground">{{ t.membership.introTitle }}</h2>
      <p class="leading-relaxed text-muted-foreground">{{ t.membership.introDesc }}</p>
    </div>

    <div v-if="activeTab === 'settings' || activeTab === 'orders'" class="rounded-[16px] bg-white p-4 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <div class="flex flex-col items-center justify-center py-12 text-center">
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-xl bg-primary/10">
          <Crown class="h-8 w-8 text-primary" />
        </div>
        <h3 class="mb-2 text-lg font-semibold text-foreground">{{ activeTab === 'settings' ? t.membership.tabs.settings : t.membership.tabs.orders }}</h3>
        <p class="text-muted-foreground">{{ t.membership.devNotice }}</p>
      </div>
    </div>
      </main>
    </div>
  </AppShell>
</template>
