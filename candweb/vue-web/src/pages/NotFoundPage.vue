<script setup lang="ts">
import { ArrowLeft, Home, PanelLeft, SearchX } from "lucide-vue-next"
import { RouterLink, useRouter } from "vue-router"
import AppShell from "@/components/AppShell.vue"
import { useTranslation } from "@/lib/language"

const router = useRouter()
const { t } = useTranslation()

function goBack() {
  if (window.history.length > 1) {
    router.back()
    return
  }
  router.push("/")
}
</script>

<template>
  <AppShell content-class="p-0">
    <div class="page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <PanelLeft class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.notFound.title }}</span>
      </header>

      <main class="flex min-h-[calc(100vh-4rem)] items-center justify-center px-5 py-10 md:px-8 lg:px-10">
        <section class="w-full max-w-lg text-center">
          <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10 text-primary">
            <SearchX class="h-8 w-8" />
          </div>
          <p class="mt-6 text-sm font-semibold uppercase tracking-[0.18em] text-primary">{{ t.notFound.code }}</p>
          <h1 class="mt-3 text-3xl font-bold text-foreground sm:text-4xl">{{ t.notFound.title }}</h1>
          <p class="mt-4 text-sm leading-6 text-muted-foreground">{{ t.notFound.desc }}</p>
          <div class="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
            <button class="btn btn-outline cursor-pointer px-5" @click="goBack">
              <ArrowLeft class="h-4 w-4" />
              {{ t.notFound.back }}
            </button>
            <RouterLink class="btn btn-primary cursor-pointer px-5" to="/">
              <Home class="h-4 w-4" />
              {{ t.notFound.home }}
            </RouterLink>
          </div>
        </section>
      </main>
    </div>
  </AppShell>
</template>
