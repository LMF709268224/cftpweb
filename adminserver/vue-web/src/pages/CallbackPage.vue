<script setup lang="ts">
import { onMounted, ref } from "vue"
import { LoaderCircle } from "lucide-vue-next"
import { apiClient } from "@/lib/apiClient"

const message = ref("正在完成登录...")

onMounted(async () => {
  const url = new URL(window.location.href)
  const code = url.searchParams.get("code") || ""
  const state = url.searchParams.get("state") || ""
  if (!code || !state) {
    message.value = "缺少登录回调参数"
    return
  }

  try {
    const resp = await apiClient("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ code, state }),
    })
    if (resp?.token) localStorage.setItem("access_token", resp.token)
    if (resp?.user?.name) localStorage.setItem("admin_name", resp.user.name)
    window.location.href = "/"
  } catch (error) {
    message.value = error instanceof Error ? error.message : "登录失败"
  }
})
</script>

<template>
  <main class="flex min-h-screen items-center justify-center p-6">
    <div class="card max-w-md p-8 text-center">
      <LoaderCircle class="mx-auto h-10 w-10 animate-spin text-[var(--clay)]" />
      <h1 class="mt-5 text-2xl font-black">登录处理中</h1>
      <p class="mt-3 text-sm text-slate-500">{{ message }}</p>
    </div>
  </main>
</template>
