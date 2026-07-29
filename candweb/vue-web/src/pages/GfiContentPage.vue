<script setup lang="ts">
import { computed } from "vue"
import { useRoute } from "vue-router"
import { ArrowUpRight, Check, Home } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import { useTranslation } from "@/lib/language"
import { getGfiPage, localize } from "@/lib/gfiSite"

const route = useRoute()
const { lang } = useTranslation()
const pagePath = computed(() => route.path.replace(/^\/gfi\/?/, "").replace(/\/$/, ""))
const definition = computed(() => getGfiPage(pagePath.value))

const page = computed(() => {
  const source = definition.value
  if (!source) return null
  return {
    title: localize(source.title, lang.value),
    eyebrow: localize(source.eyebrow, lang.value),
    description: localize(source.description, lang.value),
    image: source.image,
    sections: source.sections.map((section) => ({
      title: localize(section.title, lang.value),
      text: localize(section.text, lang.value),
      bullets: section.bullets?.map((bullet) => localize(bullet, lang.value)) || [],
    })),
  }
})
</script>

<template>
  <div class="gfi-content-page">
    <GfiHeader />

    <main v-if="page">
      <section class="content-hero" :style="{ backgroundImage: `url(${page.image})` }">
        <div class="content-hero-overlay" />
        <div class="content-container content-hero-inner">
          <nav class="breadcrumbs" :aria-label="lang === 'zh' ? '面包屑' : 'Breadcrumb'">
            <RouterLink to="/gfi"><Home />{{ lang === "zh" ? "首页" : "Home" }}</RouterLink>
            <span>/</span>
            <span>{{ page.eyebrow }}</span>
          </nav>
          <span class="content-eyebrow">{{ page.eyebrow }}</span>
          <h1>{{ page.title }}</h1>
          <p>{{ page.description }}</p>
        </div>
      </section>

      <section class="content-intro">
        <div class="content-container intro-grid">
          <div>
            <span>{{ page.eyebrow }}</span>
            <h2>{{ page.title }}</h2>
          </div>
          <p>{{ page.description }}</p>
        </div>
      </section>

      <section class="content-sections">
        <div class="content-container section-stack">
          <article v-for="(section, index) in page.sections" :key="section.title" class="content-section-card" :class="{ reverse: index % 2 === 1 }">
            <div class="section-number">{{ String(index + 1).padStart(2, "0") }}</div>
            <div class="section-copy">
              <h2>{{ section.title }}</h2>
              <p>{{ section.text }}</p>
              <ul v-if="section.bullets.length">
                <li v-for="bullet in section.bullets" :key="bullet"><Check />{{ bullet }}</li>
              </ul>
            </div>
            <div class="section-visual">
              <span>{{ page.eyebrow }}</span>
              <strong>{{ section.title }}</strong>
            </div>
          </article>
        </div>
      </section>

      <section class="content-cta">
        <div class="content-container cta-inner">
          <div>
            <span>{{ lang === "zh" ? "迈出下一步" : "Take the Next Step" }}</span>
            <h2>{{ lang === "zh" ? "加入全球金融科技专业社群" : "Join the Global Fintech Professional Community" }}</h2>
          </div>
          <a href="https://portal.globalfintechinstitute.org" target="_blank" rel="noopener noreferrer">
            {{ lang === "zh" ? "访问学习门户" : "Access Learning Portal" }} <ArrowUpRight />
          </a>
        </div>
      </section>
    </main>

    <main v-else class="missing-page">
      <h1>{{ lang === "zh" ? "页面不存在" : "Page Not Found" }}</h1>
      <RouterLink to="/gfi">{{ lang === "zh" ? "返回GFI首页" : "Back to GFI Home" }}</RouterLink>
    </main>

    <GfiFooter />
  </div>
</template>

<style scoped>
.gfi-content-page { min-height: 100vh; width: 100%; overflow: hidden; background: #fff; color: #4a566b; font-family: Inter, "PingFang SC", "Microsoft YaHei", Arial, sans-serif; letter-spacing: 0; }
.gfi-content-page * { box-sizing: border-box; }
.gfi-content-page a { text-decoration: none; }
.content-container { width: min(1288px, calc(100% - 64px)); margin: 0 auto; }
.content-hero { position: relative; min-height: 550px; background-position: center; background-size: cover; color: #fff; }
.content-hero-overlay { position: absolute; inset: 0; background: rgba(8,29,69,.74); }
.content-hero-inner { position: relative; display: flex; min-height: 550px; flex-direction: column; justify-content: center; padding-top: 35px; }
.breadcrumbs { position: absolute; top: 38px; left: 0; display: flex; align-items: center; gap: 10px; font-size: 13px; color: rgba(255,255,255,.72); }
.breadcrumbs a { display: inline-flex; align-items: center; gap: 7px; color: inherit; }
.breadcrumbs svg { width: 14px; height: 14px; }
.content-eyebrow { display: inline-flex; align-self: flex-start; margin-bottom: 23px; padding: 4px 17px; border: 1px solid rgba(255,255,255,.3); border-radius: 18px; background: rgba(255,255,255,.08); font-size: 14px; }
.content-hero h1 { max-width: 850px; margin: 0; color: #fff; font-size: 55px; line-height: 1.18; font-weight: 500; overflow-wrap: anywhere; }
.content-hero p { max-width: 710px; margin: 24px 0 0; padding-left: 20px; border-left: 1px solid rgba(255,255,255,.58); color: rgba(255,255,255,.9); font-size: 17px; line-height: 1.75; }
.content-intro { padding: 105px 0 80px; }
.intro-grid { display: grid; grid-template-columns: 1.05fr .95fr; align-items: end; gap: 95px; }
.intro-grid span { display: inline-flex; margin-bottom: 17px; padding: 4px 18px; border: 1px solid #cfdbf1; border-radius: 18px; background: #f3f6ff; color: #2864dc; font-size: 14px; }
.intro-grid h2 { margin: 0; color: #111b36; font-size: 43px; line-height: 1.25; font-weight: 500; overflow-wrap: anywhere; }
.intro-grid > p { margin: 0 0 6px; font-size: 17px; line-height: 1.82; }
.content-sections { padding: 20px 0 110px; }
.section-stack { display: grid; gap: 22px; }
.content-section-card { display: grid; grid-template-columns: 115px minmax(0, 1.45fr) minmax(260px, .8fr); min-height: 335px; border: 1px solid #cfdbf1; background: #fff; }
.content-section-card.reverse { grid-template-columns: 115px minmax(260px, .8fr) minmax(0, 1.45fr); }
.content-section-card.reverse .section-copy { order: 3; }
.content-section-card.reverse .section-visual { order: 2; }
.section-number { display: flex; align-items: flex-start; justify-content: center; padding-top: 53px; border-right: 1px solid #e0e6f0; color: #2864ff; font-size: 29px; font-weight: 600; }
.section-copy { display: flex; flex-direction: column; justify-content: center; padding: 48px 60px; }
.section-copy h2 { margin: 0 0 16px; color: #111b36; font-size: 29px; line-height: 1.35; font-weight: 500; }
.section-copy p { margin: 0; font-size: 16px; line-height: 1.8; }
.section-copy ul { display: grid; gap: 10px; margin: 22px 0 0; padding: 0; list-style: none; }
.section-copy li { display: flex; align-items: flex-start; gap: 10px; }
.section-copy li svg { width: 18px; height: 18px; flex: 0 0 18px; margin-top: 3px; color: #2864ff; }
.section-visual { display: flex; min-height: 250px; flex-direction: column; justify-content: flex-end; padding: 36px; background: #edf3ff url("https://globalfintechinstitute.org/assets/bg.CZWEzqel_Z5lu0T.svg") center/cover; color: #143878; }
.section-visual span { margin-bottom: 9px; font-size: 13px; text-transform: uppercase; }
.section-visual strong { color: #1d4bad; font-size: 25px; line-height: 1.35; }
.content-cta { padding: 85px 0; background: #101f47; color: #fff; }
.cta-inner { display: flex; align-items: center; justify-content: space-between; gap: 60px; }
.cta-inner span { display: block; margin-bottom: 12px; color: #bdd1ff; font-size: 14px; }
.cta-inner h2 { max-width: 730px; margin: 0; color: #fff; font-size: 36px; line-height: 1.3; font-weight: 500; }
.cta-inner a { display: inline-flex; min-height: 56px; flex: 0 0 auto; align-items: center; gap: 11px; padding: 0 24px; border-radius: 28px; background: #fff; color: #1e56d4; font-weight: 600; }
.cta-inner svg { width: 18px; height: 18px; }
.missing-page { display: grid; min-height: 60vh; place-content: center; text-align: center; }
.missing-page h1 { color: #111b36; }
.missing-page a { color: #2864ff; }

@media (max-width: 900px) {
  .content-container { width: calc(100% - 40px); }
  .content-hero, .content-hero-inner { min-height: 500px; }
  .content-hero h1 { font-size: 43px; }
  .intro-grid { grid-template-columns: 1fr; gap: 30px; }
  .content-section-card, .content-section-card.reverse { grid-template-columns: 80px 1fr; }
  .section-copy, .content-section-card.reverse .section-copy { order: 2; }
  .section-visual, .content-section-card.reverse .section-visual { grid-column: 1 / -1; order: 3; }
  .cta-inner { align-items: flex-start; flex-direction: column; }
}

@media (max-width: 600px) {
  .content-container { width: calc(100% - 32px); }
  .content-hero, .content-hero-inner { min-height: 460px; }
  .breadcrumbs { top: 27px; }
  .content-hero h1 { font-size: 34px; }
  .content-hero p { font-size: 15px; }
  .content-intro { padding: 72px 0 55px; }
  .intro-grid h2 { font-size: 33px; }
  .content-sections { padding-bottom: 75px; }
  .content-section-card, .content-section-card.reverse { grid-template-columns: 1fr; }
  .section-number { justify-content: flex-start; padding: 28px 24px 0; border-right: 0; }
  .section-copy, .content-section-card.reverse .section-copy { order: 2; padding: 25px 24px 38px; }
  .section-visual, .content-section-card.reverse .section-visual { grid-column: auto; min-height: 210px; order: 3; padding: 25px; }
  .content-cta { padding: 65px 0; }
  .cta-inner h2 { font-size: 29px; }
}
</style>
