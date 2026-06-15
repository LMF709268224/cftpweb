<script setup lang="ts">
import { ref } from "vue"
import { ArrowRight, LockKeyhole, Network, ShieldCheck } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"

const loading = ref(false)

async function login() {
  loading.value = true
  try {
    const callback = `${window.location.origin}/callback`
    const resp = await apiClient(`/api/auth/login-url?callback=${encodeURIComponent(callback)}`)
    if (resp?.url) window.location.href = resp.url
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="relative grid min-h-screen overflow-hidden bg-[var(--page-bg)] lg:grid-cols-[1.1fr_0.9fr]">
    <section class="relative flex items-center px-6 py-12 lg:px-16">
      <div class="absolute left-10 top-10 h-28 w-28 rounded-full bg-[var(--clay)]/20 blur-3xl" />
      <div class="absolute bottom-20 right-8 h-40 w-40 rounded-full bg-[var(--moss)]/20 blur-3xl" />
      <div class="relative max-w-3xl">
        <div class="mb-8 inline-flex items-center gap-2 rounded-full border border-white/60 bg-white/55 px-4 py-2 text-sm font-black text-[var(--ink)] shadow-sm">
          <ShieldCheck class="h-4 w-4" />
          CFtP Admin Console
        </div>
        <h1 class="text-5xl font-black leading-[1.04] tracking-tight text-[var(--ink)] md:text-7xl">
          把微服务数据<br />摊开看清楚
        </h1>
        <p class="mt-6 max-w-xl text-lg leading-8 text-slate-600">
          新版 Vue 管理后台聚焦资源包配置和白盒化查看，把排查问题需要的 list/detail 数据集中到一个入口。
        </p>
        <div class="mt-10 grid max-w-2xl gap-4 sm:grid-cols-2">
          <div class="card p-5">
            <Network class="mb-4 h-7 w-7 text-[var(--clay)]" />
            <h2 class="font-black">统一入口</h2>
            <p class="mt-2 text-sm leading-6 text-slate-500">课程、订单、消息、证书、PDF、日志和资源包集中查看。</p>
          </div>
          <div class="card p-5">
            <LockKeyhole class="mb-4 h-7 w-7 text-[var(--moss)]" />
            <h2 class="font-black">沿用 SSO</h2>
            <p class="mt-2 text-sm leading-6 text-slate-500">继续使用现有 Casdoor 登录、cookie 和 Bearer token 机制。</p>
          </div>
        </div>
      </div>
    </section>

    <section class="flex items-center justify-center p-6">
      <div class="card w-full max-w-md p-8">
        <p class="text-xs font-black uppercase tracking-[0.28em] text-slate-500">Secure Sign In</p>
        <h2 class="mt-3 text-3xl font-black text-[var(--ink)]">管理员登录</h2>
        <p class="mt-3 text-sm leading-6 text-slate-500">点击下方按钮跳转统一认证系统，完成后会自动回到管理后台。</p>
        <button class="btn btn-primary mt-8 h-12 w-full" :disabled="loading" @click="login">
          {{ loading ? "正在跳转..." : "使用 SSO 登录" }}
          <ArrowRight class="h-4 w-4" />
        </button>
      </div>
    </section>
  </main>
</template>
