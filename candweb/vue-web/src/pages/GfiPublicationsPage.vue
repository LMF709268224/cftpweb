<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowUpRight, ChevronRight } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import GfiLineBackground from "@/components/GfiLineBackground.vue"
import publicationSource from "@/lib/gfiPublicationPosts.json"
import { useTranslation } from "@/lib/language"
import { formatBackendDateOnly } from "@/lib/utils"

type PublicationKind = "reports" | "insights" | "news"
type PublicationPost = {
  title: string
  slug: string
  date: string
  image: string
  description: string
  categories: string[]
}

const publicationData = publicationSource as Record<PublicationKind, Record<"zh" | "en", PublicationPost[]>>
const route = useRoute()
const { lang } = useTranslation()
const query = ref("")
const selectedCategory = ref("")
const selectedYear = ref("")
let revealObserver: IntersectionObserver | null = null

const pageKey = computed(() => route.path.replace(/\/$/, "").split("/").pop() || "reports")
const isJournals = computed(() => pageKey.value === "journals")
const kind = computed<PublicationKind>(() => {
  if (pageKey.value === "insights" || pageKey.value === "news") return pageKey.value
  return "reports"
})

const pageMeta = computed(() => {
  const meta = {
    reports: {
      title: { zh: "报告", en: "Reports" },
      description: {
        zh: "深入分析、政策洞察和行业观点，塑造金融科技的未来。探索GFI与监管机构、行业领袖和学术机构合作开发的研究成果。",
        en: "In-depth analysis, policy insights, and industry perspectives shaping the future of fintech. Explore GFI's research outputs developed in collaboration with regulators, industry leaders, and academic institutions.",
      },
    },
    insights: {
      title: { zh: "洞察", en: "Insights" },
      description: {
        zh: "关于监管发展、市场变化和行业优先事项的及时观点。GFI的政策简报和行业报告将复杂的变化转化为实用的洞察，为金融、金融科技和公共政策领域的决策者提供参考。",
        en: "Timely perspectives on regulatory developments, market shifts, and industry priorities. GFI's policy briefs and industry reports translate complex changes into practical insights for decision-makers across finance, fintech, and public policy.",
      },
    },
    news: {
      title: { zh: "新闻与更新", en: "News & Updates" },
      description: { zh: "", en: "" },
    },
  } as const
  return meta[kind.value]
})

const posts = computed(() => publicationData[kind.value][lang.value])
const categories = computed(() => {
  const counts = new Map<string, number>()
  posts.value.forEach((post) => post.categories.filter((category) => !category.startsWith("#")).forEach((category) => {
    counts.set(category, (counts.get(category) || 0) + 1)
  }))
  const order = kind.value === "news" ? ["Announcements", "Events", "Publications"] : ["Research Briefs", "Whitepapers"]
  return [...counts.entries()].map(([name, count]) => ({ name, count })).sort((left, right) => order.indexOf(left.name) - order.indexOf(right.name))
})
const years = computed(() => {
  const counts = new Map<string, number>()
  posts.value.forEach((post) => {
    const year = new Date(post.date).getUTCFullYear()
    const label = `${year}-${String(year + 1).slice(-2)}`
    counts.set(label, (counts.get(label) || 0) + 1)
  })
  return [...counts.entries()].map(([name, count]) => ({ name, count }))
})
const filteredPosts = computed(() => {
  const needle = query.value.trim().toLocaleLowerCase()
  return posts.value.filter((post) => {
    const matchesQuery = !needle || `${post.title} ${post.description}`.toLocaleLowerCase().includes(needle)
    const matchesCategory = !selectedCategory.value || post.categories.includes(selectedCategory.value)
    const year = new Date(post.date).getUTCFullYear()
    const range = `${year}-${String(year + 1).slice(-2)}`
    return matchesQuery && matchesCategory && (!selectedYear.value || selectedYear.value === range)
  })
})

const journals = [
  {
    title: "World Scientific Annual Review of Fintech",
    url: "https://www.worldscientific.com/toc/wsarft/03",
    image: "/gfi/publications/journal-annual-review.webp",
    paragraphs: [
      "World Scientific Annual Review of Fintech is a leading publication dedicated to high-quality survey and review papers across major areas of fintech and sustainable development. Covering topics such as AI, blockchain, financial inclusion, green finance, regulation, governance, and other emerging trends, the journal brings together insights that matter to both academia and industry.",
      "Published annually around a contemporary theme in sustainable fintech, the journal serves a global audience of researchers, regulators, central bankers, policy makers, and practitioners. With an esteemed international editorial team, it aims to bridge academic research and real-world application through rigorous article selection, meaningful interdisciplinary perspectives, and a timely review process.",
    ],
  },
  {
    title: "The Journal of FinTech",
    url: "https://www.worldscientific.com/worldscinet/JFT",
    image: "/gfi/publications/journal-fintech.webp",
    paragraphs: [
      "The Journal of FinTech explores how technology is reshaping the world of finance. From banking, insurance, and investments to credit analysis and digital assets, the journal provides a broad platform for research and discussion on the trends, technologies, and challenges defining this fast-evolving field.",
      "The journal welcomes work across key areas such as artificial intelligence, blockchain, crypto, data, energy, quantum, regulation, security, and technology. It also covers interdisciplinary topics including automated trading, reg-tech, large language models, differential privacy, tokenomics, cybersecurity, data infrastructure, and other innovations at the intersection of finance and technology.",
      "By bringing together research across these themes, The Journal of FinTech supports deeper understanding of how emerging technologies are influencing financial systems, institutions, and markets.",
    ],
  },
]

function visibleCategory(post: PublicationPost) {
  const category = post.categories.find((item) => !item.startsWith("#"))
  if (category) return `#${category}`
  return kind.value === "insights" ? "#Industry Report" : kind.value === "reports" ? "#Report" : "#news"
}

function selectCategory(value: string) {
  selectedCategory.value = selectedCategory.value === value ? "" : value
}

function selectYear(value: string) {
  selectedYear.value = selectedYear.value === value ? "" : value
}

async function initialiseReveal() {
  await nextTick()
  revealObserver?.disconnect()
  revealObserver = new IntersectionObserver((entries) => entries.forEach((entry) => {
    if (!entry.isIntersecting) return
    entry.target.classList.add("is-revealed")
    revealObserver?.unobserve(entry.target)
  }), { threshold: 0.08, rootMargin: "0px 0px -30px" })
  document.querySelectorAll(".publication-page [data-reveal]").forEach((element) => revealObserver?.observe(element))
}

watch([pageKey, lang], () => {
  query.value = ""
  selectedCategory.value = ""
  selectedYear.value = ""
  initialiseReveal()
})
watch(filteredPosts, initialiseReveal)
onMounted(initialiseReveal)
onBeforeUnmount(() => revealObserver?.disconnect())
</script>

<template>
  <div class="publication-page">
    <GfiHeader theme="light" />
    <main>
      <section class="publication-hero" :class="{ 'listing-hero': !isJournals && kind !== 'news', 'news-hero': kind === 'news' }">
        <GfiLineBackground />
        <div class="publication-container hero-copy" data-reveal>
          <template v-if="isJournals">
            <h1>Journals</h1>
            <p>Featured journals under Publications.</p>
          </template>
          <template v-else>
            <h1>{{ pageMeta.title[lang] }}</h1>
            <p v-if="pageMeta.description[lang]">{{ pageMeta.description[lang] }}</p>
          </template>
          <nav aria-label="Breadcrumb">
            <RouterLink to="/gfi">Home</RouterLink><i>/</i>
            <span v-if="lang === 'zh'">Cn</span><i v-if="lang === 'zh'">/</i>
            <span>Publications</span><i>/</i>
            <span>{{ isJournals ? "Journals" : kind === "news" ? "News" : pageMeta.title.en }}</span>
          </nav>
        </div>
      </section>

      <section v-if="isJournals" class="journal-list publication-container">
        <article v-for="journal in journals" :key="journal.title" data-reveal>
          <a class="journal-cover" :href="journal.url" target="_blank" rel="noopener noreferrer"><img :src="journal.image" :alt="journal.title" loading="lazy" decoding="async"></a>
          <div>
            <h2>{{ journal.title }}</h2>
            <a class="journal-url" :href="journal.url" target="_blank" rel="noopener noreferrer">{{ journal.url }}</a>
            <p v-for="paragraph in journal.paragraphs" :key="paragraph">{{ paragraph }}</p>
          </div>
        </article>
      </section>

      <section v-else class="publication-content publication-container">
        <div class="post-grid">
          <article v-for="post in filteredPosts" :key="post.slug" class="post-card" data-reveal>
            <div class="post-image">
              <a v-if="kind !== 'news'" href="/marketplace" target="_blank" rel="noopener noreferrer" :aria-label="post.title"><img :src="post.image" :alt="post.title" loading="lazy" decoding="async"></a>
              <RouterLink v-else :to="`/gfi${post.slug}`" :aria-label="post.title"><img :src="post.image" :alt="post.title" loading="lazy" decoding="async"></RouterLink>
            </div>
            <div class="post-copy">
              <div class="post-meta"><span>{{ formatBackendDateOnly(post.date) }}</span><i></i><b>{{ visibleCategory(post) }}</b></div>
              <h2>{{ post.title }}</h2>
              <p>{{ post.description }}</p>
              <a v-if="kind !== 'news'" href="/marketplace" target="_blank" rel="noopener noreferrer">Access Full Report <ArrowUpRight /></a>
              <RouterLink v-else :to="`/gfi${post.slug}`">Read More <ArrowUpRight /></RouterLink>
            </div>
          </article>
          <div v-if="!filteredPosts.length" class="no-results" role="status">
            <img src="/gfi/events/404.svg" alt="" aria-hidden="true">
            <p>No News Found</p>
          </div>
        </div>

        <aside class="publication-sidebar" data-reveal>
          <section>
            <h2>{{ lang === "zh" ? "搜索新闻和更新" : "Search News & Updates" }}</h2>
            <label><input v-model="query" :placeholder="lang === 'zh' ? '搜索新闻和更新...' : 'Search news and updates...'" type="search"><button type="button" :aria-label="lang === 'zh' ? '搜索' : 'Search'"><ArrowUpRight /></button></label>
          </section>
          <section>
            <h2>{{ lang === "zh" ? "分类" : "Categories" }}</h2>
            <div v-if="categories.length" class="filter-list">
              <button :class="{ active: !selectedCategory }" @click="selectedCategory = ''"><ChevronRight /><span>All Categories</span><b>{{ posts.length }}</b></button>
              <button v-for="category in categories" :key="category.name" :class="{ active: selectedCategory === category.name }" @click="selectCategory(category.name)"><ChevronRight /><span>{{ category.name }}</span><b>{{ category.count }}</b></button>
            </div>
            <p v-else class="empty-filter">No categories to filter by</p>
          </section>
          <section>
            <h2>Filter by Year</h2>
            <div class="filter-list">
              <button :class="{ active: !selectedYear }" @click="selectedYear = ''"><ChevronRight /><span>All Years</span><b>{{ posts.length }}</b></button>
              <button v-for="year in years" :key="year.name" :class="{ active: selectedYear === year.name }" @click="selectYear(year.name)"><ChevronRight /><span>{{ year.name }}</span><b>{{ year.count }}</b></button>
            </div>
          </section>
          <section v-if="kind !== 'news'" class="newsletter-card">
            <img src="/gfi/publications/newsletter.jpg.webp" alt="GFI publications">
            <div>
              <h2>{{ kind === "reports" ? (lang === "zh" ? "随时了解GFI报告动态" : "Stay Informed with GFI Reports") : (lang === "zh" ? "随时了解政策与行业发展动态" : "Stay Informed with Policy and Industry Developments") }}</h2>
              <p>{{ kind === "reports" ? (lang === "zh" ? "当新的报告和洞察发布时，接收更新通知。加入GFI会员，即可获得完整报告和会员专属洞察。" : "Receive updates when new reports and insights are published. Get full access to reports and member-exclusive insights when you join GFI Membership.") : (lang === "zh" ? "当新的政策简报和行业报告发布时，接收更新通知。加入GFI会员，即可获得深入分析和会员专属洞察。" : "Receive updates when new policy briefs and industry reports are published. Join GFI Membership for in-depth analysis and member-exclusive insights.") }}</p>
              <a href="https://www.linkedin.com/company/globalfintechinstitute/" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "订阅GFI通讯" : "Subscribe to GFI Newsletter" }} <ArrowUpRight /></a>
            </div>
          </section>
        </aside>
      </section>
    </main>
    <GfiFooter />
  </div>
</template>

<style scoped>
.publication-page { --primary:#245fff; --navy:#101f45; min-height:100vh; overflow:hidden; background:#fff; color:var(--navy); font-family:Arial,"Microsoft YaHei",sans-serif; letter-spacing:0; }
.publication-page * { box-sizing:border-box; }
.publication-page a { text-decoration:none; }
.publication-container { width:min(1248px,calc(100% - 64px)); margin:0 auto; }
[data-reveal] { opacity:1; transform:none; }
[data-reveal].is-revealed { animation:publication-reveal .75s ease both; }
@keyframes publication-reveal { from { opacity:0; transform:translateY(24px); } to { opacity:1; transform:translateY(0); } }
.publication-hero { position:relative; min-height:410px; overflow:hidden; background:linear-gradient(to bottom,#f7f9fb,#fff); }
.publication-hero.listing-hero { min-height:428px; }
.publication-hero.news-hero { min-height:354px; }
.hero-copy { position:relative; z-index:1; padding:96px 0 88px; text-align:center; }
.hero-copy h1 { margin:0 0 26px; font-size:49px; font-weight:500; line-height:1.18; }
.hero-copy > p { max-width:860px; margin:0 auto 24px; color:#30333a; font-size:17px; line-height:1.65; }
.hero-copy nav { display:inline-flex; min-height:42px; padding:0 26px; align-items:center; justify-content:center; gap:20px; border:1px solid #dfe2e8; border-radius:999px; background:#fff; color:#777; font-size:13px; }
.hero-copy nav a { color:var(--primary); }
.hero-copy nav i { color:#d9dde4; font-style:normal; }
.publication-content { display:grid; padding:0 0 130px; grid-template-columns:minmax(0,2fr) 394px; gap:44px; align-items:start; }
.post-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:24px; }
.post-card { overflow:hidden; border-radius:7px; background:#fff; box-shadow:0 9px 26px rgba(19,42,85,.08); }
.post-image { height:256px; overflow:hidden; background:#eef2f7; }
.post-image a { display:block; width:100%; height:100%; }
.post-image img { display:block; width:100%; height:100%; object-fit:cover; transition:transform .5s ease; }
.post-card:hover .post-image img { transform:scale(1.05); }
.post-copy { padding:25px 24px 28px; }
.post-meta { display:flex; padding-bottom:14px; align-items:center; gap:10px; border-bottom:1px solid #e0e3e9; color:#343943; font-size:13px; }
.post-meta span { color:var(--primary); }
.post-meta i { width:3px; height:3px; border-radius:50%; background:var(--primary); }
.post-meta b { font-weight:400; }
.post-copy h2 { display:-webkit-box; min-height:65px; margin:14px 0 15px; overflow:hidden; color:#0d1c43; font-size:22px; font-weight:600; line-height:1.45; -webkit-box-orient:vertical; -webkit-line-clamp:2; }
.post-copy p { display:-webkit-box; min-height:66px; margin:0 0 24px; overflow:hidden; color:#4f5664; font-size:14px; line-height:1.58; white-space:pre-line; -webkit-box-orient:vertical; -webkit-line-clamp:3; }
.post-copy > a { display:inline-flex; padding-bottom:7px; align-items:center; gap:7px; border-bottom:1px solid var(--primary); color:var(--primary); font-size:14px; }
.post-copy > a svg { width:14px; height:14px; }
.no-results { display:flex; min-height:560px; padding:68px 24px 48px; grid-column:1/-1; flex-direction:column; align-items:center; justify-content:flex-start; text-align:center; }
.no-results img { display:block; width:min(100%,620px); height:auto; }
.no-results p { margin:26px 0 0; color:#071539; font-size:46px; font-weight:700; line-height:1.15; }
.publication-sidebar { padding:0 20px 20px; background:#fff; }
.publication-sidebar > section { margin-bottom:30px; }
.publication-sidebar h2 { margin:0; padding:20px 0; border-bottom:1px solid #e7e9ed; font-size:21px; font-weight:500; }
.publication-sidebar label { display:flex; height:55px; margin-top:15px; border:1px solid #dfe3e9; }
.publication-sidebar input { min-width:0; flex:1; padding:0 20px; border:0; outline:0; color:#343a46; font:inherit; }
.publication-sidebar label button { width:51px; border:0; background:#f2f5f9; color:#536078; cursor:pointer; }
.publication-sidebar label svg { width:22px; }
.filter-list button { display:flex; width:100%; min-height:52px; padding:7px 6px; align-items:center; gap:15px; border:0; background:#fff; color:#3c4149; text-align:left; cursor:pointer; transition:color .25s ease,background .25s ease; }
.filter-list button:hover,.filter-list button.active { color:var(--primary); }
.filter-list svg { width:15px; }
.filter-list span { flex:1; }
.filter-list b { display:flex; width:38px; height:38px; align-items:center; justify-content:center; border-radius:50%; background:#f7f9fc; font-size:13px; font-weight:400; }
.empty-filter { margin:20px 0 0; color:#6d727d; font-size:13px; }
.newsletter-card { position:relative; min-height:295px; overflow:hidden; color:#fff; }
.newsletter-card > img { position:absolute; inset:0; width:100%; height:100%; object-fit:cover; }
.newsletter-card::after { position:absolute; inset:42% 0 0; content:""; background:linear-gradient(to bottom,rgba(44,101,245,.64),#2b63f0); }
.newsletter-card > div { position:relative; z-index:1; padding:164px 20px 24px; }
.newsletter-card h2 { margin:0 0 12px; padding:0; border:0; color:#fff; font-size:21px; font-weight:600; line-height:1.4; }
.newsletter-card p { margin:0 0 16px; font-size:13px; line-height:1.55; }
.newsletter-card a { display:inline-flex; padding-bottom:5px; align-items:center; gap:7px; border-bottom:1px solid #fff; color:#fff; font-size:13px; }
.newsletter-card svg { width:13px; }
.journal-list { position:relative; z-index:2; width:min(1216px,calc(100% - 64px)); margin-top:-77px; padding-bottom:130px; }
.journal-list article { display:grid; margin-bottom:32px; padding:24px; grid-template-columns:240px minmax(0,1fr); gap:24px; border:1px solid #dce2eb; border-radius:24px; background:#fff; box-shadow:0 2px 4px rgba(15,31,69,.04); }
.journal-cover { overflow:hidden; border:1px solid #dce2eb; border-radius:16px; background:#f8fafc; }
.journal-cover img { display:block; width:100%; height:100%; min-height:340px; object-fit:cover; }
.journal-list h2 { margin:0 0 18px; font-size:31px; line-height:1.25; }
.journal-url { display:block; margin-bottom:18px; color:#2563eb; text-decoration:underline !important; overflow-wrap:anywhere; }
.journal-list p { margin:0 0 18px; color:#26344e; font-size:16px; line-height:2; }
@media (max-width:980px) {
  .publication-content { grid-template-columns:1fr; }
  .publication-sidebar { padding:0; }
  .newsletter-card { max-width:420px; }
}
@media (max-width:640px) {
  .publication-container { width:calc(100% - 32px); }
  .publication-hero { min-height:360px; }
  .hero-copy { padding:70px 0; }
  .hero-copy h1 { font-size:36px; }
  .hero-copy > p { font-size:15px; }
  .hero-copy nav { max-width:100%; padding:0 16px; gap:10px; flex-wrap:wrap; }
  .post-grid { grid-template-columns:1fr; }
  .post-image { height:230px; }
  .publication-content { padding-bottom:85px; }
  .no-results { min-height:430px; padding:48px 0 32px; }
  .no-results p { margin-top:18px; font-size:32px; }
  .journal-list { margin-top:-40px; }
  .journal-list article { grid-template-columns:1fr; padding:18px; border-radius:16px; }
  .journal-cover { max-width:240px; }
  .journal-list h2 { font-size:25px; }
  .journal-list p { font-size:14px; line-height:1.75; }
}
@media (prefers-reduced-motion:reduce) {
  [data-reveal],[data-reveal].is-revealed { opacity:1; transform:none; animation:none !important; }
  .post-image img { transition:none; }
}
</style>
