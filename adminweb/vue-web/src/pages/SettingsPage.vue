<script setup lang="ts">
import { Loader2, Save, Shield, UserRound } from "lucide-vue-next"
import { onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { apiClient } from "@/lib/apiClient"
import { clearAuthSession, setAuthSession } from "@/lib/authStorage"
import type { JsonRecord } from "@/lib/display"

type ProfileForm = {
  name: string
  display_name: string
  email: string
  affiliation: string
  title: string
  real_name: string
  bio: string
  gender: string
  birthday: string
  education: string
}

const router = useRouter()
const activeTab = ref<"profile" | "account">("profile")
const profileLoading = ref(false)
const profileSaving = ref(false)
const passwordSaving = ref(false)

const profile = ref<ProfileForm>({
  name: "",
  display_name: "",
  email: "",
  affiliation: "",
  title: "",
  real_name: "",
  bio: "",
  gender: "",
  birthday: "",
  education: "",
})

const oldPassword = ref("")
const newPassword = ref("")
const confirmPassword = ref("")

function applyProfile(data: JsonRecord) {
  profile.value = {
    name: String(data.name || ""),
    display_name: String(data.display_name || ""),
    email: String(data.email || ""),
    affiliation: String(data.affiliation || ""),
    title: String(data.title || ""),
    real_name: String(data.real_name || ""),
    bio: String(data.bio || ""),
    gender: String(data.gender || ""),
    birthday: String(data.birthday || ""),
    education: String(data.education || ""),
  }
}

async function loadProfile() {
  profileLoading.value = true
  try {
    const data = await apiClient<JsonRecord>("/api/user/me")
    applyProfile(data || {})
  } catch (err) {
    console.error(err)
    toast.error("个人资料加载失败")
  } finally {
    profileLoading.value = false
  }
}

async function saveProfile() {
  profileSaving.value = true
  try {
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify({
        display_name: profile.value.display_name,
        email: profile.value.email,
        affiliation: profile.value.affiliation,
        title: profile.value.title,
        real_name: profile.value.real_name,
        bio: profile.value.bio,
        gender: profile.value.gender,
        birthday: profile.value.birthday,
        education: profile.value.education,
      }),
    })
    setAuthSession("", profile.value.display_name || profile.value.name || "Admin")
    window.dispatchEvent(new Event("storage"))
    toast.success("个人资料已保存")
  } catch (err) {
    console.error(err)
    toast.error("个人资料保存失败")
  } finally {
    profileSaving.value = false
  }
}

async function savePassword() {
  if (!oldPassword.value || !newPassword.value) {
    toast.error("请填写当前密码和新密码")
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    toast.error("两次输入的新密码不一致")
    return
  }

  passwordSaving.value = true
  try {
    await apiClient("/api/user/password", {
      method: "PUT",
      body: JSON.stringify({
        old_password: oldPassword.value,
        new_password: newPassword.value,
      }),
    })
    toast.success("密码已更新，请重新登录")
    clearAuthSession()
    setTimeout(() => router.push("/login"), 800)
  } catch (err) {
    console.error(err)
    toast.error("密码更新失败")
  } finally {
    passwordSaving.value = false
  }
}

onMounted(() => {
  const params = new URLSearchParams(window.location.search)
  if (params.get("tab") === "account") activeTab.value = "account"
  void loadProfile()
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1280px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">账户设置</h1>
        <p class="mt-2 text-slate-600">维护管理员个人资料和登录密码。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadProfile">
        <Loader2 v-if="profileLoading" class="h-4 w-4 animate-spin" />
        <UserRound v-else class="h-4 w-4" />
        重新加载
      </button>
    </header>

    <div class="flex gap-2 rounded-2xl border border-slate-200 bg-white p-2 shadow-sm">
      <button
        class="inline-flex items-center gap-2 rounded-xl px-5 py-3 text-sm font-black transition"
        :class="activeTab === 'profile' ? 'bg-[#0b7bdc] text-white shadow-lg shadow-sky-200' : 'text-slate-600 hover:bg-slate-50'"
        type="button"
        @click="activeTab = 'profile'"
      >
        <UserRound class="h-4 w-4" />
        个人资料
      </button>
      <button
        class="inline-flex items-center gap-2 rounded-xl px-5 py-3 text-sm font-black transition"
        :class="activeTab === 'account' ? 'bg-[#0b7bdc] text-white shadow-lg shadow-sky-200' : 'text-slate-600 hover:bg-slate-50'"
        type="button"
        @click="activeTab = 'account'"
      >
        <Shield class="h-4 w-4" />
        修改密码
      </button>
    </div>

    <form v-if="activeTab === 'profile'" class="rounded-3xl border border-slate-200 bg-white p-6 shadow-sm" @submit.prevent="saveProfile">
      <div class="mb-6">
        <h2 class="text-2xl font-black">个人资料</h2>
        <p class="mt-1 text-sm text-slate-500">登录 ID 和邮箱由认证系统同步，其他字段可以在这里维护。</p>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <label class="block">
          <span class="text-sm font-bold">登录 ID</span>
          <input v-model="profile.name" class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-slate-500" disabled />
        </label>
        <label class="block">
          <span class="text-sm font-bold">邮箱</span>
          <input v-model="profile.email" class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-slate-500" disabled />
        </label>
        <label class="block">
          <span class="text-sm font-bold">显示名称</span>
          <input v-model="profile.display_name" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" placeholder="用于后台显示的名称" />
        </label>
        <label class="block">
          <span class="text-sm font-bold">真实姓名</span>
          <input v-model="profile.real_name" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
        </label>
        <label class="block">
          <span class="text-sm font-bold">性别</span>
          <input v-model="profile.gender" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
        </label>
        <label class="block">
          <span class="text-sm font-bold">生日</span>
          <input v-model="profile.birthday" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" type="date" />
        </label>
        <label class="block">
          <span class="text-sm font-bold">机构</span>
          <input v-model="profile.affiliation" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
        </label>
        <label class="block">
          <span class="text-sm font-bold">职位</span>
          <input v-model="profile.title" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
        </label>
        <label class="block md:col-span-2">
          <span class="text-sm font-bold">教育背景</span>
          <input v-model="profile.education" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" />
        </label>
        <label class="block md:col-span-2">
          <span class="text-sm font-bold">简介</span>
          <textarea v-model="profile.bio" class="mt-2 min-h-28 w-full rounded-xl border border-slate-200 p-4" />
        </label>
      </div>

      <button class="mt-6 inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="profileSaving" type="submit">
        <Loader2 v-if="profileSaving" class="h-4 w-4 animate-spin" />
        <Save v-else class="h-4 w-4" />
        {{ profileSaving ? "保存中..." : "保存资料" }}
      </button>
    </form>

    <form v-else class="max-w-2xl rounded-3xl border border-slate-200 bg-white p-6 shadow-sm" @submit.prevent="savePassword">
      <div class="mb-6">
        <h2 class="text-2xl font-black">修改密码</h2>
        <p class="mt-1 text-sm text-slate-500">密码更新成功后会退出当前登录，需要重新登录后台。</p>
      </div>

      <label class="block">
        <span class="text-sm font-bold">当前密码</span>
        <input v-model="oldPassword" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" required type="password" />
      </label>
      <label class="mt-4 block">
        <span class="text-sm font-bold">新密码</span>
        <input v-model="newPassword" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" required type="password" />
      </label>
      <label class="mt-4 block">
        <span class="text-sm font-bold">确认新密码</span>
        <input v-model="confirmPassword" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" required type="password" />
      </label>

      <button class="mt-6 inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="passwordSaving" type="submit">
        <Loader2 v-if="passwordSaving" class="h-4 w-4 animate-spin" />
        <Shield v-else class="h-4 w-4" />
        {{ passwordSaving ? "更新中..." : "更新密码" }}
      </button>
    </form>
  </section>
</template>
