<script setup lang="ts">
import { Loader2, ShieldCheck } from "lucide-vue-next"
import { onMounted, ref } from "vue"
import { apiClient } from "@/lib/apiClient"

const error = ref("")

function reload() {
  window.location.reload()
}

onMounted(async () => {
  try {
    const callback = encodeURIComponent(`${window.location.origin}/callback`)
    const data = await apiClient<{ url?: string }>(`/api/auth/login-url?callback=${callback}`)
    if (data?.url) {
      window.location.href = data.url
      return
    }
    error.value = "未获取到登录地址，请稍后重试。"
  } catch (err) {
    console.error(err)
    error.value = "登录初始化失败，请检查后台认证服务。"
  }
})
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-slate-950 px-6 text-white">
    <div class="w-full max-w-sm rounded-3xl border border-white/10 bg-white/5 p-8 text-center shadow-2xl backdrop-blur">
      <div class="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-sky-500/20 text-sky-300">
        <ShieldCheck v-if="error" class="h-8 w-8" />
        <Loader2 v-else class="h-8 w-8 animate-spin" />
      </div>
      <h1 class="text-2xl font-black">{{ error ? "登录遇到问题" : "正在跳转认证中心" }}</h1>
      <p class="mt-3 text-sm leading-6 text-slate-300">
        {{ error || "正在为你建立安全会话，请不要关闭页面。" }}
      </p>
      <button
        v-if="error"
        class="mt-6 rounded-xl bg-sky-500 px-5 py-2 text-sm font-bold text-white hover:bg-sky-400"
        type="button"
        @click="reload"
      >
        重试
      </button>
    </div>
  </div>
</template>
