<script setup lang="ts">
import { Loader2, RefreshCw, Save, Shield, UserRound } from "lucide-vue-next"
import { computed, onMounted, ref } from "vue"
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

type SettingsSection = "profile" | "password"

const router = useRouter()
const activeSection = ref<SettingsSection>("profile")
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

const sections = computed(() => [
  {
    key: "profile" as const,
    title: "个人资料",
    description: "维护 Casdoor 同步过来的管理员资料。",
    count: profile.value.display_name || profile.value.name ? 1 : 0,
  },
  {
    key: "password" as const,
    title: "修改密码",
    description: "通过 Casdoor 修改当前管理员密码。",
    count: 1,
  },
])

function normalizeGender(value: unknown) {
  const text = String(value || "").trim().toLowerCase()
  if (text === "male" || text === "m" || text === "男") return "男"
  if (text === "female" || text === "f" || text === "女") return "女"
  return ""
}

function normalizeBirthday(value: unknown) {
  const text = String(value || "").trim()
  if (!text) return ""
  return text.slice(0, 10)
}

function applyProfile(data: JsonRecord) {
  profile.value = {
    name: String(data.name || ""),
    display_name: String(data.display_name || ""),
    email: String(data.email || ""),
    affiliation: String(data.affiliation || ""),
    title: String(data.title || ""),
    real_name: String(data.real_name || ""),
    bio: String(data.bio || ""),
    gender: normalizeGender(data.gender),
    birthday: normalizeBirthday(data.birthday),
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
        display_name: profile.value.display_name.trim(),
        affiliation: profile.value.affiliation.trim(),
        title: profile.value.title.trim(),
        real_name: profile.value.real_name.trim(),
        bio: profile.value.bio.trim(),
        gender: profile.value.gender,
        birthday: profile.value.birthday,
        education: profile.value.education.trim(),
      }),
    })
    setAuthSession("", profile.value.display_name.trim() || profile.value.name || "Admin")
    window.dispatchEvent(new Event("storage"))
    toast.success("个人资料已保存")
    await loadProfile()
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
  if (newPassword.value.length < 8) {
    toast.error("新密码至少 8 个字符")
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
  if (params.get("tab") === "password" || params.get("tab") === "account") activeSection.value = "password"
  void loadProfile()
})
</script>

<template>
  <section class="mx-auto flex min-h-screen w-full max-w-[1480px] flex-col gap-6 px-8 py-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-4xl font-black tracking-tight">账户设置</h1>
        <p class="mt-2 text-slate-600">维护管理员个人资料和登录密码；字段来自 Casdoor 用户资料接口。</p>
      </div>
      <button class="inline-flex items-center gap-2 rounded-xl border bg-white px-4 py-3 text-sm font-bold shadow-sm" type="button" @click="loadProfile">
        <RefreshCw class="h-4 w-4" :class="profileLoading ? 'animate-spin' : ''" />
        重新加载
      </button>
    </header>

    <div class="grid gap-6 xl:grid-cols-[0.75fr_1.25fr]">
      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="border-b border-slate-100 px-5 py-4">
          <h2 class="text-xl font-black">设置项</h2>
          <p class="mt-1 text-sm text-slate-500">左侧选择设置项，右侧进行查看或保存。</p>
        </div>
        <button
          v-for="section in sections"
          :key="section.key"
          class="grid w-full grid-cols-[1fr_auto] gap-4 border-b border-slate-100 px-5 py-4 text-left last:border-b-0 hover:bg-sky-50"
          :class="activeSection === section.key ? 'bg-sky-50' : ''"
          type="button"
          @click="activeSection = section.key"
        >
          <div>
            <div class="font-black">{{ section.title }}</div>
            <div class="mt-1 text-sm text-slate-500">{{ section.description }}</div>
          </div>
          <span class="h-fit rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600">{{ section.count }}</span>
        </button>
      </section>

      <section class="overflow-hidden rounded-3xl border border-slate-200 bg-white shadow-sm">
        <div class="flex items-center justify-between border-b border-slate-100 px-5 py-4">
          <div>
            <h2 class="text-xl font-black">{{ activeSection === "profile" ? "个人资料" : "修改密码" }}</h2>
            <p class="mt-1 text-sm text-slate-500">
              {{ activeSection === "profile" ? "登录 ID 和邮箱只读；其他字段通过 /api/user/profile 保存。" : "密码通过 /api/user/password 修改，成功后会退出登录。" }}
            </p>
          </div>
          <UserRound v-if="activeSection === 'profile'" class="h-5 w-5 text-blue-600" />
          <Shield v-else class="h-5 w-5 text-blue-600" />
        </div>

        <form v-if="activeSection === 'profile'" class="space-y-6 p-5" @submit.prevent="saveProfile">
          <div v-if="profileLoading" class="rounded-2xl bg-slate-50 p-8 text-center text-slate-500">
            <Loader2 class="mx-auto mb-2 h-6 w-6 animate-spin" />
            正在加载个人资料...
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <label class="block">
              <span class="text-sm font-bold">登录 ID</span>
              <input v-model="profile.name" class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" disabled />
            </label>
            <label class="block">
              <span class="text-sm font-bold">邮箱</span>
              <input v-model="profile.email" class="mt-2 w-full rounded-xl border border-slate-200 bg-slate-100 px-4 py-3 text-slate-500" disabled />
            </label>
            <label class="block">
              <span class="text-sm font-bold">显示名称</span>
              <input v-model.trim="profile.display_name" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="80" placeholder="用于后台显示的名称" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">真实姓名</span>
              <input v-model.trim="profile.real_name" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="80" placeholder="真实姓名" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">性别</span>
              <select v-model="profile.gender" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3">
                <option value="">请选择性别</option>
                <option value="男">男</option>
                <option value="女">女</option>
              </select>
            </label>
            <label class="block">
              <span class="text-sm font-bold">生日</span>
              <input v-model="profile.birthday" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" type="date" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">机构</span>
              <input v-model.trim="profile.affiliation" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="160" placeholder="所属机构" />
            </label>
            <label class="block">
              <span class="text-sm font-bold">职位</span>
              <input v-model.trim="profile.title" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="120" placeholder="职位/头衔" />
            </label>
            <label class="block md:col-span-2">
              <span class="text-sm font-bold">教育背景</span>
              <input v-model.trim="profile.education" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="200" placeholder="教育背景" />
            </label>
            <label class="block md:col-span-2">
              <span class="text-sm font-bold">简介</span>
              <textarea v-model.trim="profile.bio" class="mt-2 min-h-28 w-full rounded-xl border border-slate-200 p-4" maxlength="1000" placeholder="管理员简介" />
              <span class="mt-1 block text-right text-xs text-slate-400">{{ profile.bio.length }}/1000</span>
            </label>
          </div>

          <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="profileSaving" type="submit">
            <Loader2 v-if="profileSaving" class="h-4 w-4 animate-spin" />
            <Save v-else class="h-4 w-4" />
            {{ profileSaving ? "保存中..." : "保存资料" }}
          </button>
        </form>

        <form v-else class="max-w-2xl space-y-4 p-5" @submit.prevent="savePassword">
          <label class="block">
            <span class="text-sm font-bold">当前密码</span>
            <input v-model="oldPassword" autocomplete="current-password" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="128" required type="password" />
          </label>
          <label class="block">
            <span class="text-sm font-bold">新密码</span>
            <input v-model="newPassword" autocomplete="new-password" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="128" minlength="8" required type="password" />
            <span class="mt-1 block text-xs text-slate-400">至少 8 个字符。</span>
          </label>
          <label class="block">
            <span class="text-sm font-bold">确认新密码</span>
            <input v-model="confirmPassword" autocomplete="new-password" class="mt-2 w-full rounded-xl border border-slate-200 px-4 py-3" maxlength="128" minlength="8" required type="password" />
          </label>

          <button class="inline-flex items-center gap-2 rounded-xl bg-[#0b4ea2] px-5 py-3 font-bold text-white disabled:opacity-50" :disabled="passwordSaving" type="submit">
            <Loader2 v-if="passwordSaving" class="h-4 w-4 animate-spin" />
            <Shield v-else class="h-4 w-4" />
            {{ passwordSaving ? "更新中..." : "更新密码" }}
          </button>
        </form>
      </section>
    </div>
  </section>
</template>
