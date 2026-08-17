<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { toast } from "vue-sonner"
import { ChevronDown, Languages, Menu, X } from "lucide-vue-next"
import { isAuthenticated } from "@/lib/authStorage"
import { startGfiLogin } from "@/lib/gfiLogin"
import { useTranslation } from "@/lib/language"
import { gfiNavGroups, localize } from "@/lib/gfiSite"

const props = withDefaults(defineProps<{ theme?: "dark" | "light"; authTarget?: string; authNewTab?: boolean }>(), {
  theme: "dark",
  authTarget: "/marketplace",
  authNewTab: true,
})

const route = useRoute()
const { t, lang, changeLanguage } = useTranslation()
const openMenu = ref<number | null>(null)
const languageOpen = ref(false)
const mobileOpen = ref(false)
const authenticated = ref(isAuthenticated())
const authStarting = ref(false)

const logo = "/gfi/gfi-logo-blue.svg"
const currentLanguage = computed(() => (lang.value === "zh" ? "CN" : "EN"))
const usesDirectThirdPartyLogin = computed(() => props.authTarget === "/login")

async function beginAuthentication() {
  if (authStarting.value) return
  authStarting.value = true
  try {
    await startGfiLogin()
  } catch (error) {
    console.error("Unable to start GFI login:", error)
    toast.error(t.value.loginPage.errorTitle)
    authStarting.value = false
  }
}

function isGroupActive(index: number) {
  return gfiNavGroups[index].items.some((item) => route.path === item.to || route.path.startsWith(`${item.to}/`))
}

function toggleMenu(index: number) {
  openMenu.value = openMenu.value === index ? null : index
  languageOpen.value = false
}

function selectLanguage(next: "zh" | "en") {
  changeLanguage(next)
  languageOpen.value = false
  mobileOpen.value = false
}

function syncAuthentication() {
  authenticated.value = isAuthenticated()
}

onMounted(() => {
  window.addEventListener("storage", syncAuthentication)
  window.addEventListener("focus", syncAuthentication)
})

onBeforeUnmount(() => {
  window.removeEventListener("storage", syncAuthentication)
  window.removeEventListener("focus", syncAuthentication)
})

watch(
  () => route.fullPath,
  () => {
    syncAuthentication()
    openMenu.value = null
    languageOpen.value = false
    mobileOpen.value = false
  },
)
</script>

<template>
  <header class="gfi-header" :class="`gfi-header--${theme}`">
    <div class="gfi-header-inner">
      <RouterLink to="/" class="gfi-brand" aria-label="Global Fintech Institute">
        <img :src="logo" alt="Global Fintech Institute" />
      </RouterLink>

      <nav class="gfi-desktop-nav" :aria-label="lang === 'zh' ? '主导航' : 'Main navigation'">
        <div
          v-for="(group, index) in gfiNavGroups"
          :key="group.label.en"
          class="gfi-nav-group"
          @mouseenter="openMenu = index"
          @mouseleave="openMenu = null"
        >
          <button
            :class="{ active: openMenu === index || isGroupActive(index) }"
            :aria-expanded="openMenu === index"
            @click="toggleMenu(index)"
          >
            {{ localize(group.label, lang) }}
            <ChevronDown :class="{ rotated: openMenu === index }" />
          </button>
          <div v-show="openMenu === index" class="gfi-dropdown">
            <RouterLink v-for="item in group.items" :key="item.to" :to="item.to">
              {{ localize(item.label, lang) }}
            </RouterLink>
          </div>
        </div>
      </nav>

      <div class="gfi-header-actions">
        <div class="gfi-language" @mouseenter="languageOpen = true" @mouseleave="languageOpen = false">
          <button :aria-expanded="languageOpen" @click="languageOpen = !languageOpen">
            <Languages /> {{ currentLanguage }} <ChevronDown :class="{ rotated: languageOpen }" />
          </button>
          <div v-show="languageOpen" class="gfi-language-menu">
            <button v-if="lang === 'zh'" @click="selectLanguage('en')">English</button>
            <button v-else @click="selectLanguage('zh')">中文</button>
          </div>
        </div>
        <span />
        <RouterLink v-if="authenticated" to="/dashboard" class="gfi-console-link">
          {{ lang === "zh" ? "控制台" : "Dashboard" }}
        </RouterLink>
        <template v-else-if="usesDirectThirdPartyLogin">
          <button type="button" class="gfi-auth-link" :disabled="authStarting" :aria-busy="authStarting" @click="beginAuthentication">
            {{ lang === "zh" ? "登录" : "Log In" }}
          </button>
          <span />
          <button type="button" class="gfi-auth-link" :disabled="authStarting" :aria-busy="authStarting" @click="beginAuthentication">
            {{ lang === "zh" ? "注册" : "Register" }}
          </button>
        </template>
        <template v-else>
          <RouterLink :to="props.authTarget" :target="props.authNewTab ? '_blank' : undefined" :rel="props.authNewTab ? 'noopener noreferrer' : undefined">
            {{ lang === "zh" ? "登录" : "Log In" }}
          </RouterLink>
          <span />
          <RouterLink :to="props.authTarget" :target="props.authNewTab ? '_blank' : undefined" :rel="props.authNewTab ? 'noopener noreferrer' : undefined">
            {{ lang === "zh" ? "注册" : "Register" }}
          </RouterLink>
        </template>
      </div>

      <button class="gfi-menu-toggle" :aria-expanded="mobileOpen" :aria-label="mobileOpen ? 'Close menu' : 'Open menu'" @click="mobileOpen = !mobileOpen">
        <X v-if="mobileOpen" />
        <Menu v-else />
      </button>
    </div>

    <nav v-if="mobileOpen" class="gfi-mobile-nav">
      <div v-for="(group, index) in gfiNavGroups" :key="group.label.en">
        <button :aria-expanded="openMenu === index" @click="toggleMenu(index)">
          {{ localize(group.label, lang) }}
          <ChevronDown :class="{ rotated: openMenu === index }" />
        </button>
        <div v-show="openMenu === index" class="gfi-mobile-submenu">
          <RouterLink v-for="item in group.items" :key="item.to" :to="item.to">{{ localize(item.label, lang) }}</RouterLink>
        </div>
      </div>
      <div class="gfi-mobile-language">
        <button :class="{ selected: lang === 'zh' }" @click="selectLanguage('zh')">中文</button>
        <button :class="{ selected: lang === 'en' }" @click="selectLanguage('en')">English</button>
      </div>
      <button
        v-if="!authenticated && usesDirectThirdPartyLogin"
        type="button"
        class="gfi-mobile-login"
        :disabled="authStarting"
        :aria-busy="authStarting"
        @click="beginAuthentication"
      >
        {{ lang === "zh" ? "登录 / 注册" : "Log In / Register" }}
      </button>
      <RouterLink
        v-else
        :to="authenticated ? '/dashboard' : props.authTarget"
        class="gfi-mobile-login"
        :target="!authenticated && props.authNewTab ? '_blank' : undefined"
        :rel="!authenticated && props.authNewTab ? 'noopener noreferrer' : undefined"
      >
        {{ authenticated ? (lang === "zh" ? "控制台" : "Dashboard") : (lang === "zh" ? "登录 / 注册" : "Log In / Register") }}
      </RouterLink>
    </nav>
  </header>
</template>

<style scoped>
.gfi-header { position: relative; z-index: 80; height: 94px; background: #101f47; color: #fff; font-family: Inter, "PingFang SC", "Microsoft YaHei", Arial, sans-serif; }
.gfi-header--light { border-bottom: 1px solid #eef1f6; background: #fff; color: #0e1b3d; }
.gfi-header * { box-sizing: border-box; }
.gfi-header a { color: inherit; text-decoration: none; }
.gfi-header-inner { width: min(1288px, calc(100% - 64px)); height: 100%; margin: 0 auto; display: flex; align-items: center; }
.gfi-brand { display: flex; width: 124px; flex: 0 0 124px; }
.gfi-brand img { width: 100%; filter: brightness(0) invert(1); }
.gfi-header--light .gfi-brand img { filter: none; }
.gfi-desktop-nav { position: absolute; top: 0; left: calc(50% - 40px); display: flex; height: 94px; align-items: stretch; gap: 2px; transform: translateX(-50%); }
.gfi-nav-group { position: relative; display: flex; align-items: center; }
.gfi-nav-group > button, .gfi-language > button { display: inline-flex; height: 100%; align-items: center; gap: 8px; padding: 0 17px; border: 0; background: transparent; color: #fff; font-size: 16px; }
.gfi-header--light .gfi-nav-group > button, .gfi-header--light .gfi-language > button { color: #0e1b3d; }
.gfi-nav-group > button.active { background: rgba(255,255,255,.065); }
.gfi-nav-group svg, .gfi-language svg:last-child { width: 15px; height: 15px; transition: transform .2s ease; }
.rotated { transform: rotate(180deg); }
.gfi-dropdown { position: absolute; top: calc(100% + 10px); left: 50%; width: max-content; min-width: 192px; max-width: 260px; padding: 10px; transform: translateX(-50%); border-radius: 5px; background: #fff; box-shadow: 0 16px 45px rgba(9,21,51,.2); }
.gfi-dropdown::before { content: ""; position: absolute; top: -12px; left: 0; right: 0; height: 12px; }
.gfi-dropdown a { display: block; padding: 11px 14px; border-radius: 3px; color: #333d50; font-size: 15px; line-height: 1.45; white-space: normal; }
.gfi-dropdown a:hover, .gfi-dropdown a.router-link-active { background: #f1f5ff; color: #225edf; }
.gfi-header-actions { position: absolute; top: 0; right: 40px; display: flex; height: 94px; align-items: center; gap: 13px; font-size: 15px; white-space: nowrap; }
.gfi-header-actions > span { width: 1px; height: 18px; background: rgba(255,255,255,.4); }
.gfi-header--light .gfi-header-actions > span { background: #cbd2df; }
.gfi-auth-link { padding:0; border:0; background:transparent; color:inherit; cursor:pointer; font:inherit; }
.gfi-auth-link:disabled { cursor:wait; opacity:.65; }
.gfi-console-link { color:#9fc5ff; font-weight:600; transition:color .2s ease; }
.gfi-console-link:hover { color:#fff; text-decoration:underline; text-underline-offset:4px; }
.gfi-console-link:focus-visible { outline:2px solid #9fc5ff; outline-offset:5px; border-radius:2px; }
.gfi-header--light .gfi-console-link { color:#2058b5; }
.gfi-header--light .gfi-console-link:hover { color:#174796; }
.gfi-language { position: relative; height: 100%; }
.gfi-language > button { padding: 0; font-size: 15px; }
.gfi-language svg:first-child { width: 16px; height: 16px; }
.gfi-language-menu { position: absolute; top: calc(100% + 10px); right: -15px; width: 128px; padding: 10px; border-radius: 5px; background: #fff; box-shadow: 0 16px 45px rgba(9,21,51,.2); }
.gfi-language-menu::before { content: ""; position: absolute; top: -12px; left: 0; right: 0; height: 12px; }
.gfi-language-menu button { width: 100%; padding: 11px; border: 0; border-radius: 3px; background: #fff; color: #243047; font-size: 15px; }
.gfi-language-menu button:hover { background: #f1f5ff; color: #225edf; }
.gfi-menu-toggle { display: none; width: 42px; height: 42px; flex: 0 0 42px; align-items: center; justify-content: center; border: 0; background: transparent; color: #fff; }
.gfi-header--light .gfi-menu-toggle { color: #0e1b3d; }
.gfi-menu-toggle svg { width: 26px; height: 26px; }
.gfi-mobile-nav { position: absolute; top: calc(78px + var(--app-safe-area-top)); left: 0; right: 0; max-height: calc(var(--app-viewport-height) - 78px - var(--app-safe-area-top)); overflow-y: auto; padding-top: 10px; padding-right: calc(20px + var(--app-safe-area-right)); padding-bottom: calc(24px + var(--app-safe-area-bottom)); padding-left: calc(20px + var(--app-safe-area-left)); background: #101f47; border-top: 1px solid rgba(255,255,255,.1); }
.gfi-mobile-nav > div > button { display: flex; width: 100%; align-items: center; justify-content: space-between; padding: 14px 0; border: 0; border-bottom: 1px solid rgba(255,255,255,.09); background: transparent; color: #fff; }
.gfi-mobile-nav svg { width: 17px; height: 17px; }
.gfi-mobile-submenu { display: grid; padding: 4px 0 9px 16px; }
.gfi-mobile-submenu a { padding: 10px 0; color: rgba(255,255,255,.78); font-size: 14px; }
.gfi-mobile-language { display: flex !important; gap: 8px; padding: 18px 0 10px; }
.gfi-mobile-language button { width: auto !important; padding: 8px 14px !important; border: 1px solid rgba(255,255,255,.22) !important; border-radius: 18px; }
.gfi-mobile-language button.selected { background: #fff !important; color: #101f47 !important; }
.gfi-mobile-login { display:block; width:100%; margin-top:10px; padding:13px; border:0; border-radius:4px; background:#2864ff; color:#fff; cursor:pointer; text-align:center; font:inherit; font-weight:600; }
.gfi-mobile-login:disabled { cursor:wait; opacity:.75; }

@media (max-width: 1180px) {
  .gfi-nav-group > button { padding: 0 10px; font-size: 15px; }
  .gfi-desktop-nav { left: 50%; }
  .gfi-header-actions { right: 24px; }
}

@media (max-width: 960px) {
  .gfi-header { height: calc(78px + var(--app-safe-area-top)); }
  .gfi-header-inner { width: calc(100% - 32px); padding-top: var(--app-safe-area-top); padding-right: var(--app-safe-area-right); padding-left: var(--app-safe-area-left); }
  .gfi-brand { width: 109px; flex-basis: 109px; }
  .gfi-desktop-nav, .gfi-header-actions { display: none; }
  .gfi-menu-toggle { position: absolute; top: calc(18px + var(--app-safe-area-top)); right: calc(16px + var(--app-safe-area-right)); z-index: 90; display: inline-flex !important; margin-left: auto; background: transparent; visibility: visible !important; opacity: 1 !important; }
  .gfi-header--light .gfi-menu-toggle svg { display: block !important; color: #0e1b3d !important; stroke: #0e1b3d !important; }
}
</style>
