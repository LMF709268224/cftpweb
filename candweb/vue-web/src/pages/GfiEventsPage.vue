<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowUpRight, ChevronRight } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import GfiLineBackground from "@/components/GfiLineBackground.vue"
import eventSource from "@/lib/gfiEventPosts.json"
import { useTranslation } from "@/lib/language"

type EventKind = "all" | "webinars" | "conferences"

interface EventPost {
  title: string
  slug: string
  date: string
  image: string
  description: string
  label: string
  categories: Array<{ name: string; slug: string }>
  status: string
}

const eventData = eventSource as Record<EventKind, Record<"zh" | "en", EventPost[]>>
const route = useRoute()
const { lang } = useTranslation()
const query = ref("")
const selectedCategory = ref("")
const selectedPeriod = ref("")
const selectedYear = ref("")
let revealObserver: IntersectionObserver | null = null

const kind = computed<EventKind>(() => {
  if (route.path.includes("webinar-recordings")) return "webinars"
  if (route.path.includes("conferences")) return "conferences"
  return "all"
})

const pageMeta = computed(() => {
  if (kind.value === "webinars") {
    return {
      title: { zh: "网络研讨会录像", en: "Webinar Recordings" },
      description: "",
      searchTitle: { zh: "搜索网络研讨会", en: "Search Webinars" },
      searchPlaceholder: { zh: "搜索网络研讨会...", en: "Search webinars..." },
    }
  }
  if (kind.value === "conferences") {
    return {
      title: { zh: "Conferences & Roundtables", en: "Conferences & Roundtables" },
      description: "High-trust convenings that bring together regulators, industry leaders, and academia to exchange insights, shape standards, and advance responsible fintech innovation.",
      searchTitle: { zh: "搜索活动", en: "Search Events" },
      searchPlaceholder: { zh: "搜索活动...", en: "Search events..." },
    }
  }
  return {
    title: { zh: "所有活动", en: "All Events" },
    description: "",
    searchTitle: { zh: "搜索活动", en: "Search Events" },
    searchPlaceholder: { zh: "搜索活动...", en: "Search events..." },
  }
})

const emptyMessage = computed(() => {
  if (kind.value === "webinars") return "No Webinars Found"
  if (kind.value === "conferences") return "No News Found"
  return "No Events Found"
})

const posts = computed(() => eventData[kind.value][lang.value])
const categories = computed(() => {
  const values = new Map<string, string>()
  posts.value.forEach((post) => post.categories.forEach((category) => values.set(category.slug, category.name)))
  return Array.from(values, ([slug, name]) => ({ slug, name }))
})
const years = computed(() => Array.from(new Set(posts.value.map((post) => new Date(post.date).getUTCFullYear()))).sort((a, b) => b - a))

function isUpcoming(post: EventPost) {
  return post.status !== "past-event" && new Date(post.date).getTime() > Date.now()
}

const filteredPosts = computed(() => {
  const search = query.value.trim().toLocaleLowerCase()
  return posts.value.filter((post) => {
    if (search && !`${post.title} ${post.description}`.toLocaleLowerCase().includes(search)) return false
    if (selectedCategory.value && !post.categories.some((category) => category.slug === selectedCategory.value)) return false
    if (selectedYear.value && new Date(post.date).getUTCFullYear().toString() !== selectedYear.value) return false
    if (selectedPeriod.value === "upcoming" && !isUpcoming(post)) return false
    if (selectedPeriod.value === "past" && isUpcoming(post)) return false
    return true
  })
})

function categoryCount(slug: string) {
  return posts.value.filter((post) => !slug || post.categories.some((category) => category.slug === slug)).length
}

function yearCount(year?: number) {
  return posts.value.filter((post) => !year || new Date(post.date).getUTCFullYear() === year).length
}

function periodCount(period: string) {
  if (!period) return posts.value.length
  return posts.value.filter((post) => period === "upcoming" ? isUpcoming(post) : !isUpcoming(post)).length
}

function formatDate(date: string, short = false) {
  return new Intl.DateTimeFormat(short ? "en-GB" : "en-US", short
    ? { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }
    : { month: "long", day: "numeric", year: "numeric", timeZone: "UTC" }).format(new Date(date))
}

function yearLabel(year: number) {
  return `${year}-${String((year + 1) % 100).padStart(2, "0")}`
}

function toggleCategory(slug: string) {
  selectedCategory.value = selectedCategory.value === slug ? "" : slug
}

function toggleYear(year: number) {
  const value = year.toString()
  selectedYear.value = selectedYear.value === value ? "" : value
}

function selectPeriod(period: string) {
  selectedPeriod.value = selectedPeriod.value === period ? "" : period
}

async function initialiseReveal() {
  await nextTick()
  revealObserver?.disconnect()
  revealObserver = new IntersectionObserver((entries) => entries.forEach((entry) => {
    if (!entry.isIntersecting) return
    entry.target.classList.add("is-revealed")
    revealObserver?.unobserve(entry.target)
  }), { threshold: 0.08, rootMargin: "0px 0px -30px" })
  document.querySelectorAll(".gfi-events-page [data-reveal]").forEach((element) => revealObserver?.observe(element))
}

watch([kind, lang], () => {
  query.value = ""
  selectedCategory.value = ""
  selectedPeriod.value = ""
  selectedYear.value = ""
  initialiseReveal()
})
watch(filteredPosts, initialiseReveal)
onMounted(initialiseReveal)
onBeforeUnmount(() => revealObserver?.disconnect())
</script>

<template>
  <div class="gfi-events-page">
    <GfiHeader theme="light" />
    <main class="events-main">
      <GfiLineBackground />
      <section class="events-hero" :class="{ 'conference-hero': kind === 'conferences' }">
        <div class="events-container hero-copy" data-reveal>
          <h1>{{ pageMeta.title[lang] }}</h1>
          <p v-if="pageMeta.description">{{ pageMeta.description }}</p>
        </div>
      </section>

      <section class="events-content events-container">
        <div class="events-list-panel">
          <p v-if="kind === 'conferences' && (selectedCategory || selectedPeriod)" class="filter-results-count">
            Showing <b>{{ filteredPosts.length }}</b> {{ filteredPosts.length === 1 ? "conference" : "conferences" }}
          </p>
          <div class="event-grid">
            <article v-for="(post, index) in filteredPosts" :key="post.slug" class="event-card" :class="{ 'conference-card': kind === 'conferences' }" data-reveal>
              <div class="event-image">
                <RouterLink :to="`/gfi${post.slug}`" :aria-label="post.title"><img :src="post.image" :alt="post.title" :loading="index < 2 ? 'eager' : 'lazy'" decoding="async" :fetchpriority="index < 2 ? 'high' : 'auto'"></RouterLink>
                <span class="image-action"><ArrowUpRight /></span>
              </div>
              <div class="event-copy">
                <div v-if="kind !== 'conferences'" class="event-meta top-meta">
                  <span>{{ formatDate(post.date) }}</span><i></i><b>{{ post.label }}</b>
                </div>
                <h2><RouterLink :to="`/gfi${post.slug}`">{{ post.title }}</RouterLink></h2>
                <p>{{ post.description }}</p>
                <div v-if="kind === 'conferences'" class="conference-meta">
                  <span>{{ formatDate(post.date, true) }}</span><i></i><span>Past Event</span>
                </div>
                <RouterLink class="card-action" :to="`/gfi${post.slug}`">
                  {{ kind === "conferences" ? "View Highlights" : "Read More" }} <ArrowUpRight />
                </RouterLink>
              </div>
            </article>
            <div v-if="!filteredPosts.length" class="no-results">
              <img src="/gfi/events/404.svg" alt="404">
              <h1>{{ emptyMessage }}</h1>
            </div>
          </div>
        </div>

        <aside class="events-sidebar" data-reveal>
          <section>
            <h2>{{ pageMeta.searchTitle[lang] }}</h2>
            <label class="search-control">
              <input v-model="query" type="search" :placeholder="pageMeta.searchPlaceholder[lang]">
              <button type="button" :aria-label="pageMeta.searchTitle[lang]"><ArrowUpRight /></button>
            </label>
          </section>

          <section>
            <h2>{{ lang === "zh" ? "分类" : "Categories" }}</h2>
            <p v-if="!categories.length" class="empty-filter">No categories to filter by</p>
            <div v-else class="filter-list">
              <button type="button" :class="{ active: !selectedCategory }" @click="selectedCategory = ''">
                <ChevronRight /><span>All Categories</span><b>{{ categoryCount("") }}</b>
              </button>
              <button v-for="category in categories" :key="category.slug" type="button" :class="{ active: selectedCategory === category.slug }" @click="toggleCategory(category.slug)">
                <ChevronRight /><span>{{ category.name }}</span><b>{{ categoryCount(category.slug) }}</b>
              </button>
            </div>
          </section>

          <section v-if="kind === 'webinars'">
            <h2>Filter by Year</h2>
            <div class="filter-list">
              <button type="button" :class="{ active: !selectedYear }" @click="selectedYear = ''">
                <ChevronRight /><span>All Years</span><b>{{ yearCount() }}</b>
              </button>
              <button v-for="year in years" :key="year" type="button" :class="{ active: selectedYear === year.toString() }" @click="toggleYear(year)">
                <ChevronRight /><span>{{ yearLabel(year) }}</span><b>{{ yearCount(year) }}</b>
              </button>
            </div>
          </section>

          <section v-else>
            <h2>{{ lang === "zh" ? "按时间筛选" : "Filter by Time" }}</h2>
            <div class="filter-list">
              <button v-for="period in [
                { value: '', zh: '所有活动', en: 'All Events' },
                { value: 'upcoming', zh: '即将举行的活动', en: 'Upcoming Events' },
                { value: 'past', zh: '过去的活动', en: 'Past Events' },
              ]" :key="period.value" type="button" :class="{ active: selectedPeriod === period.value }" @click="selectPeriod(period.value)">
                <ChevronRight /><span>{{ period[lang] }}</span><b>{{ periodCount(period.value) }}</b>
              </button>
            </div>
          </section>

          <section v-if="kind === 'all' && lang === 'en'" class="connected-card">
            <div class="connected-image"><img src="/gfi/events/industry-engagement.webp" alt="Stay connected to GFI events"><img src="/gfi/events/industry-overlay.webp" alt=""></div>
            <div><h3>Stay Connected to GFI Events</h3><p>Stay informed on upcoming conferences and roundtables, including member sessions and closed-door dialogues.</p><a href="https://www.linkedin.com/company/globalfintechinstitute" target="_blank" rel="noopener noreferrer">Follow Us on LinkedIn <ArrowUpRight /></a></div>
          </section>
        </aside>
      </section>

      <section v-if="kind === 'all'" class="community-promo events-container" data-reveal>
        <div>
          <span>{{ lang === "zh" ? "GFI 社区专属" : "GFI Community Exclusive" }}</span>
          <h2>{{ lang === "zh" ? "立即锁定席位！2026年黑天鹅峰会在珀斯" : "Secure Your Spot Now! 2026 Black Swan Summit in Perth" }}</h2>
          <p v-if="lang === 'zh'">GFI会员可享受行业通行证50%折扣，可全面参与2026年黑天鹅峰会的所有舞台、圆桌会议和顶级社交活动。此会员价格有效期至2026年1月9日。在结账时使用代码 BSSAU26-IND-PTNR-EARLYBIRD 即可享受折扣。</p>
          <p v-else>GFI members enjoy 50% off the Industry Pass, with full access to all stages, roundtables, and premier networking sessions at Black Swan Summit 2026. This member rate is valid until 9 January 2026. Use the code BSSAU26-IND-PTNR-EARLYBIRD at checkout to redeem the discount.</p>
          <a href="https://australia.blackswansummit.com/passes" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "获取早鸟优惠" : "Get Early Access" }} <ArrowUpRight /></a>
        </div>
        <img src="/gfi/events/black-swan-summit-2026.webp" :alt="lang === 'zh' ? '2026年黑天鹅峰会珀斯站 - GFI会员早鸟优惠' : '2026 Black Swan Summit in Perth - Early Bird Access for GFI Members'" loading="lazy" decoding="async">
      </section>
    </main>
    <GfiFooter />
  </div>
</template>

<style scoped>
@font-face { font-family:"DM Sans Local"; src:url("/gfi/fonts/dm-sans.woff2") format("woff2"); font-style:normal; font-weight:400 700; font-display:swap; }
@font-face { font-family:"Syne Local"; src:url("/gfi/fonts/syne-700.woff2") format("woff2"); font-style:normal; font-weight:700; font-display:swap; }
.gfi-events-page { --primary:#2562ff; --navy:#101f45; min-height:100vh; overflow:hidden; background:#fff; color:var(--navy); font-family:"DM Sans Local","Microsoft YaHei",sans-serif; letter-spacing:0; }
.gfi-events-page * { box-sizing:border-box; }
.gfi-events-page a { text-decoration:none; }
.events-main { position:relative; overflow:hidden; background:#fff; }
.events-container { width:min(1286px,calc(100% - 64px)); margin:0 auto; }
[data-reveal] { opacity:1; transform:none; }
[data-reveal].is-revealed { animation:event-reveal .75s ease both; }
@keyframes event-reveal { from { opacity:0; transform:translateY(24px); } to { opacity:1; transform:translateY(0); } }
.events-hero { position:relative; z-index:1; min-height:271px; }
.events-hero.conference-hero { min-height:356px; }
.hero-copy { padding-top:100px; text-align:center; }
.hero-copy h1 { margin:0; font-family:"DM Sans Local","Microsoft YaHei",sans-serif; font-size:49px; font-weight:400; line-height:1.2; }
.conference-hero .hero-copy { padding-top:103px; }
.conference-hero .hero-copy h1 { font-family:"Syne Local","DM Sans Local",sans-serif; font-size:45px; font-weight:700; }
.hero-copy p { max-width:850px; margin:30px auto 0; color:#4c4f56; font-size:17px; line-height:1.6; }
.events-content { position:relative; z-index:1; display:grid; padding-bottom:80px; grid-template-columns:minmax(0,2fr) 411px; gap:24px; align-items:start; }
.events-list-panel,.events-sidebar { background:#fff; }
.events-list-panel { padding:20px; }
.event-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:24px; }
.event-card { overflow:hidden; border-radius:11px; background:#fff; box-shadow:0 8px 24px rgba(20,38,75,.09); }
.event-image { position:relative; height:256px; overflow:hidden; background:#eef2f7; }
.event-image > a { display:block; width:100%; height:100%; }
.event-image img { display:block; width:100%; height:100%; object-fit:cover; transition:transform .5s ease; }
.event-card:hover .event-image img { transform:scale(1.05); }
.image-action { position:absolute; top:50%; left:50%; z-index:2; display:flex; width:64px; height:64px; align-items:center; justify-content:center; border-radius:50%; opacity:0; pointer-events:none; transform:translate(-50%,-50%); background:var(--primary); color:#fff; transition:opacity .35s ease; }
.image-action svg { width:22px; }
.event-card:hover .image-action { opacity:1; }
.event-copy { padding:24px; }
.event-meta { display:flex; min-height:33px; margin-bottom:14px; padding-bottom:12px; align-items:center; gap:10px; border-bottom:1px solid #dfe3ea; font-size:14px; }
.event-meta span { color:var(--primary); }
.event-meta i,.conference-meta i { width:3px; height:3px; border-radius:50%; background:var(--primary); }
.event-meta b { color:#3c3f45; font-weight:400; }
.event-copy h2 { display:-webkit-box; overflow:hidden; min-height:58px; margin:0 0 16px; color:var(--navy); font-family:"Syne Local","Microsoft YaHei",sans-serif; font-size:22px; font-weight:700; line-height:1.3; -webkit-box-orient:vertical; -webkit-line-clamp:2; }
.event-copy h2 a { color:inherit; transition:color .25s ease; }
.event-copy h2 a:hover { color:var(--primary); }
.event-copy > p { display:-webkit-box; overflow:hidden; min-height:68px; margin:0 0 16px; color:#4d5057; font-size:14px; line-height:1.6; -webkit-box-orient:vertical; -webkit-line-clamp:3; }
.card-action { display:inline-flex; padding-bottom:5px; align-items:center; gap:8px; border-bottom:1px solid var(--primary); color:var(--primary); font-size:16px; }
.card-action svg { width:13px; transition:transform .25s ease; }
.card-action:hover svg { transform:translate(3px,-3px); }
.conference-card .event-copy h2 { min-height:58px; margin-bottom:14px; }
.conference-card .event-copy > p { margin-bottom:12px; }
.conference-meta { display:flex; margin-bottom:17px; align-items:center; gap:8px; color:#4d5057; font-size:13px; }
.events-sidebar { padding:15px 20px 28px; }
.events-sidebar section { margin-bottom:26px; }
.events-sidebar h2 { margin:0 0 16px; padding-bottom:16px; border-bottom:1px solid #e3e6eb; font-size:24px; font-weight:500; line-height:1.35; }
.search-control { display:grid; grid-template-columns:1fr 52px; border:1px solid #dce1e8; }
.search-control input { width:100%; min-width:0; height:52px; padding:0 20px; border:0; outline:0; background:#fff; color:#20293b; font-size:14px; }
.search-control button { display:flex; width:52px; height:52px; padding:0; align-items:center; justify-content:center; border:0; border-left:1px solid #edf0f4; background:#f6f8fa; color:#536276; }
.search-control svg { width:20px; }
.empty-filter { margin:0; padding:14px 0 4px; color:#555960; font-size:14px; }
.filter-list { display:flex; flex-direction:column; }
.filter-list button { position:relative; display:grid; width:100%; min-height:44px; padding:4px 8px 4px 0; grid-template-columns:22px 1fr 38px; align-items:center; gap:8px; border:0; background:transparent; color:#565a61; text-align:left; }
.filter-list button svg { width:14px; transition:transform .25s ease; }
.filter-list button:hover svg,.filter-list button.active svg { transform:translateX(4px); }
.filter-list button:hover,.filter-list button.active { color:var(--primary); }
.filter-list button b { display:flex; width:36px; height:36px; align-items:center; justify-content:center; border-radius:50%; background:#f7f9fc; color:#555960; font-size:14px; font-weight:400; }
.connected-card { overflow:hidden; color:#fff; }
.connected-image { position:relative; height:240px; }
.connected-image img { position:absolute; inset:0; display:block; width:100%; height:100%; object-fit:cover; }
.connected-card > div:last-child { padding:20px 16px 24px; background:var(--primary); }
.connected-card h3 { margin:0 0 16px; font-size:24px; font-weight:500; }
.connected-card p { margin:0 0 18px; font-size:15px; line-height:1.6; }
.connected-card a { display:inline-flex; align-items:center; gap:8px; color:#fff; border-bottom:1px solid #fff; }
.connected-card a svg { width:13px; }
.filter-results-count { margin:0 0 18px; color:#565a61; font-size:15px; }
.filter-results-count b { color:var(--primary); }
.no-results { display:flex; width:100%; padding:64px 0; grid-column:1/-1; flex-direction:column; align-items:center; gap:40px; color:var(--navy); text-align:center; }
.no-results img { display:block; width:100%; max-width:512px; height:auto; }
.no-results h1 { margin:0; font-family:"Syne Local","DM Sans Local",sans-serif; font-size:54px; font-weight:700; line-height:1.15; }
.community-promo { position:relative; z-index:1; display:grid; width:min(1168px,calc(100% - 64px)); margin:0 auto 80px; padding:48px; grid-template-columns:1fr 1fr; gap:48px; align-items:center; border-top:3px solid #1e3a8a; border-right:3px solid #1e3a8a; border-radius:24px; background:#fff; }
.community-promo > div > span { display:inline-block; margin-bottom:16px; padding:5px 16px; border-radius:999px; background:rgba(37,98,255,.1); color:var(--primary); font-size:14px; }
.community-promo h2 { margin:0 0 16px; font-size:31px; line-height:1.3; }
.community-promo p { margin:0 0 24px; color:#40444d; font-size:16px; line-height:1.7; }
.community-promo a { display:inline-flex; padding:11px 20px; align-items:center; gap:8px; border-radius:999px; background:var(--primary); color:#fff; font-size:14px; font-weight:600; box-shadow:0 6px 16px rgba(37,98,255,.24); }
.community-promo a svg { width:16px; }
.community-promo > img { display:block; width:100%; max-width:448px; justify-self:end; border-radius:16px; box-shadow:0 12px 28px rgba(15,31,69,.18); }
@media (max-width:1050px) {
  .events-content { grid-template-columns:1fr; }
  .events-sidebar { padding-top:20px; }
}
@media (max-width:680px) {
  .events-container { width:calc(100% - 32px); }
  .events-hero { min-height:290px; }
  .events-hero.conference-hero { min-height:360px; }
  .hero-copy,.conference-hero .hero-copy { padding-top:70px; }
  .hero-copy h1,.conference-hero .hero-copy h1 { font-size:35px; }
  .hero-copy p { margin-top:22px; font-size:15px; }
  .events-content { gap:18px; padding-bottom:55px; }
  .events-list-panel { padding:0; background:transparent; }
  .event-grid { grid-template-columns:1fr; }
  .event-image { height:230px; }
  .events-sidebar { padding:20px 16px; }
  .no-results { padding:32px 0; }
  .no-results h1 { font-size:36px; }
  .community-promo { width:calc(100% - 32px); padding:28px 20px; grid-template-columns:1fr; gap:28px; }
  .community-promo h2 { font-size:25px; }
  .community-promo > img { grid-row:1; }
}
@media (prefers-reduced-motion:reduce) {
  [data-reveal],[data-reveal].is-revealed { opacity:1; transform:none; animation:none !important; }
  .event-image img,.card-action svg { transition:none; }
}
</style>
