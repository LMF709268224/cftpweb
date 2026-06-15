<script setup lang="ts">
import AdminShell from "@/components/AdminShell.vue"
</script>

<template>
  <AdminShell>
    <section class="card p-6">
      <p class="text-xs font-black uppercase tracking-[0.24em] text-slate-500">Settings</p>
      <h1 class="mt-3 text-3xl font-black">账号设置</h1>
      <p class="mt-3 max-w-2xl text-sm leading-6 text-slate-500">
        这一版先保留账号设置入口，后续可继续把管理员资料、密码修改和权限角色管理迁移到 Vue。
      </p>
    </section>
  </AdminShell>
</template>
