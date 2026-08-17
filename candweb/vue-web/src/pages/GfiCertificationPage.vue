<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowUpRight, BarChart3, Link2 } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import { useTranslation } from "@/lib/language"
import { localize } from "@/lib/gfiSite"
import { certificationAssets, certificationPrograms, pathwayStages, type CertificationContent } from "@/lib/gfiCertificationPages"

const route = useRoute()
const { lang } = useTranslation()
const pageKey = computed(() => route.path.replace(/^\/gfi\/?/, "").replace(/\/$/, ""))
const isOverview = computed(() => pageKey.value === "certifications")
const isPathway = computed(() => pageKey.value === "certifications/pathway")
const programKey = computed(() => pageKey.value.endsWith("cftp") ? "cftp" : "cfta")
const program = computed(() => pageKey.value.startsWith("programmes/") ? certificationPrograms[programKey.value] : null)
const activeTabKey = ref("overview")
const activeTab = computed(() => program.value?.tabs.find((item) => item.key === activeTabKey.value) ?? program.value?.tabs[0])
const stats = ref([0, 0, 0])
const statTargets = [1, 50, 10]
let revealObserver: IntersectionObserver | null = null
let statsObserver: IntersectionObserver | null = null
let animationFrame = 0

const l = (value: { zh: string; en: string }) => localize(value, lang.value)
const visibleParagraphs = (section: CertificationContent) => section.paragraphs?.filter((item) => l(item)) ?? []
const visibleBullets = (section: CertificationContent) => section.bullets?.filter((item) => l(item)) ?? []
const visibleGroupBullets = (items: Array<{ zh: string; en: string }>) => items.filter((item) => l(item))

const animateStats = () => {
  cancelAnimationFrame(animationFrame)
  const start = performance.now()
  const duration = 1200
  const tick = (now: number) => {
    const progress = Math.min((now - start) / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    stats.value = statTargets.map((target) => Math.round(target * eased))
    if (progress < 1) animationFrame = requestAnimationFrame(tick)
  }
  animationFrame = requestAnimationFrame(tick)
}

const setupObservers = () => {
  revealObserver?.disconnect()
  statsObserver?.disconnect()
  revealObserver = new IntersectionObserver((entries) => entries.forEach((entry) => {
    if (!entry.isIntersecting) return
    entry.target.classList.add("is-revealed")
    revealObserver?.unobserve(entry.target)
  }), { threshold: .08, rootMargin: "0px 0px -25px" })
  document.querySelectorAll(".certification-page [data-reveal]").forEach((element) => revealObserver?.observe(element))

  const statsElement = document.querySelector(".certification-page .cert-stats")
  if (statsElement) {
    statsObserver = new IntersectionObserver(([entry]) => {
      if (!entry?.isIntersecting) return
      animateStats()
      statsObserver?.disconnect()
    }, { threshold: .25 })
    statsObserver.observe(statsElement)
  }
}

watch(pageKey, async () => {
  activeTabKey.value = "overview"
  stats.value = [0, 0, 0]
  await nextTick()
  setupObservers()
})

watch(activeTabKey, async () => {
  await nextTick()
  setupObservers()
})

onMounted(async () => {
  await nextTick()
  setupObservers()
})

onBeforeUnmount(() => {
  revealObserver?.disconnect()
  statsObserver?.disconnect()
  cancelAnimationFrame(animationFrame)
})
</script>

<template>
  <div class="certification-page">
    <GfiHeader theme="light" />

    <main v-if="isOverview">
      <section class="cert-hero" :style="{ backgroundImage: `url(${certificationAssets.hero})` }">
        <div class="cert-container" data-reveal>
          <h1>{{ lang === "zh" ? "金融科技专业认证" : "Professional Fintech Certifications" }}</h1>
          <p>{{ lang === "zh" ? "行业认可的认证，旨在在您职业生涯的每个阶段建立金融科技能力、可信度和领导力。" : "Industry-recognised certifications designed to build fintech capability, credibility, and leadership at every stage of your career." }}</p>
          <RouterLink to="/gfi/certifications/pathway"><span>{{ lang === "zh" ? "探索您的认证路径" : "Explore Your Certification Path" }}</span><i><ArrowUpRight /></i></RouterLink>
        </div>
      </section>

      <section class="cert-impact">
        <div class="cert-container impact-grid">
          <div data-reveal>
            <span class="cert-pill">{{ lang === "zh" ? "认证影响力" : "Certifications Impact" }}</span>
            <h2>{{ lang === "zh" ? "树立金融科技全球标准" : "Setting the Global Standard for Fintech" }}</h2>
            <p>{{ lang === "zh" ? "全球金融科技学院（GFI）为在金融、科技、监管与创新交汇处前行的专业人士，提供结构化的认证路径。" : "The Global Fintech Institute (GFI) offers a structured certification pathway for professionals navigating the intersection of finance, technology, regulation, and innovation." }}</p>
            <article><BarChart3 /><div><h3>{{ lang === "zh" ? "行业、学术与政策共创" : "Co-created with industry, academia, and policy" }}</h3><p>{{ lang === "zh" ? "我们的认证由业界从业者、学术机构与政策相关方共同打造，确保内容的前瞻性、严谨性与实际应用价值。" : "Our certifications are developed in collaboration with industry practitioners, academic institutions, and policy stakeholders to ensure relevance, rigour, and real-world applicability." }}</p></div></article>
            <article><Link2 /><div><h3>{{ lang === "zh" ? "覆盖职业发展的可信标杆" : "Trusted benchmark across your career" }}</h3><p>{{ lang === "zh" ? "无论您是开启金融科技之旅，还是迈向领导力阶段，GFI认证都是值得信赖的专业卓越基准。" : "Whether you are starting your fintech journey or advancing into leadership, GFI certifications provide a trusted benchmark for professional excellence." }}</p></div></article>
          </div>
          <img data-reveal :src="certificationAssets.globalStandard" :alt="lang === 'zh' ? '树立金融科技全球标准' : 'Setting the global standard for fintech'">
        </div>
      </section>

      <section class="flagship-section">
        <div class="cert-container">
          <h2 data-reveal>{{ lang === "zh" ? "我们的旗舰认证" : "Our Flagship Certifications" }}</h2>
          <div class="flagship-grid">
            <article data-reveal>
              <h3>{{ lang === "zh" ? "特许金融科技助理 (CFtA)" : "Chartered Fintech Associate (CFtA)" }}</h3>
              <strong>{{ lang === "zh" ? "基础认证" : "Foundational certification" }}</strong>
              <p>{{ lang === "zh" ? "CFtA专为寻求金融科技基础知识扎实基础的个人而设计。它介绍了数字金融、新兴技术、监管和负责任的创新等关键概念。" : "The CFtA is designed for individuals seeking a strong grounding in fintech fundamentals. It introduces key concepts across digital finance, emerging technologies, regulation, and responsible innovation." }}</p>
              <h4>Who it's for</h4>
              <ul><li v-for="item in (lang === 'zh' ? ['学生和早期职业专业人士','进入金融科技领域的职业转换者','寻求基础金融科技素养的专业人士'] : ['Students and early-career professionals','Career switchers entering fintech','Professionals seeking foundational fintech literacy'])" :key="item">{{ item }}</li></ul>
              <h4>What it delivers</h4>
              <ul><li v-for="item in (lang === 'zh' ? ['对金融科技生态系统的广泛理解','接触关键技术和市场结构','进入专业金融科技教育的认可入口'] : ['Broad understanding of the fintech ecosystem','Exposure to key technologies and market structures','A recognised entry point into professional fintech education'])" :key="item">{{ item }}</li></ul>
              <RouterLink to="/gfi/programmes/cfta"><span>{{ lang === "zh" ? "了解更多关于CFtA" : "Learn more about CFtA" }}</span><i><ArrowUpRight /></i></RouterLink>
            </article>
            <article data-reveal>
              <h3>{{ lang === "zh" ? "特许金融科技专业人士 (CFtP®)" : "Chartered Fintech Professional (CFtP®)" }}</h3>
              <strong>{{ lang === "zh" ? "高级专业称号" : "Advanced professional designation" }}</strong>
              <p>{{ lang === "zh" ? "CFtP®是GFI的旗舰称号，面向希望展示金融科技领域高级能力、道德基础和领导力准备的专业人士。专为具有现有经验、正在担任或准备担任金融、金融科技、技术、风险和政策领域高级职位的专业人士而设计。" : "The CFtP® is GFI's flagship designation for professionals who wish to demonstrate advanced capability, ethical grounding, and leadership readiness in fintech. It is designed for professionals with existing experience who are operating at, or preparing for, senior roles across finance, fintech, technology, risk, and policy." }}</p>
              <h4>Who it's for</h4>
              <ul><li v-for="item in (lang === 'zh' ? ['中高级专业人士','在受监管或复杂的金融科技环境中工作的从业者','寻求认可专业地位的领导者'] : ['Mid- to senior-level professionals','Practitioners working in regulated or complex fintech environments','Leaders seeking recognised professional standing'])" :key="item">{{ item }}</li></ul>
              <h4>What it delivers</h4>
              <ul><li v-for="item in (lang === 'zh' ? ['深入的技术和监管能力','全球专业认可','访问高级网络、理事会和职业机会'] : ['Deep technical and regulatory competence','Global professional recognition','Access to senior networks, councils, and career opportunities'])" :key="item">{{ item }}</li></ul>
              <RouterLink to="/gfi/programmes/cftp"><span>{{ lang === "zh" ? "了解更多关于CFtP®" : "Learn more about CFtP®" }}</span><i><ArrowUpRight /></i></RouterLink>
            </article>
          </div>
        </div>
      </section>

      <section class="cert-why">
        <div class="cert-container why-grid">
          <div data-reveal>
            <h2>{{ lang === "zh" ? "为什么选择GFI认证" : "Why Choose GFI Certifications" }}</h2>
            <p>{{ lang === "zh" ? "GFI认证旨在反映金融科技在当今的实际运作方式——跨越边界、技术和监管环境。它们专为需要在复杂的现实环境中做出明智决策的专业人士而设计，而不仅仅是通过考试。" : "GFI certifications are designed to reflect how fintech actually operates today — across borders, technologies, and regulatory environments. They are built for professionals who need to make informed decisions in complex, real-world settings, not just pass an exam." }}</p>
            <article><BarChart3 /><div><h3>{{ lang === "zh" ? "行业验证，全球相关" : "Industry-Validated, Globally Relevant" }}</h3><p>{{ lang === "zh" ? "与从业者和学术合作伙伴共同开发，GFI认证提供严格、实用的知识，适用于各种市场和监管环境，而不仅仅是单一司法管辖区。" : "Developed with practitioners and academic partners, GFI certifications deliver rigorous, real-world knowledge that applies across markets and regulatory environments, not just a single jurisdiction." }}</p></div></article>
            <article><Link2 /><div><h3>{{ lang === "zh" ? "为负责任的长期职业而建" : "Built for Responsible, Long-Term Careers" }}</h3><p>{{ lang === "zh" ? "道德和治理嵌入核心，支持从基础到特许持有者的清晰进展，确保专业人士建立随着金融科技发展而持续的能力和信誉。" : "Ethics and governance are embedded at the core, supported by a clear progression from foundation to charterholder, ensuring professionals build capability and credibility that endure as fintech evolves." }}</p></div></article>
          </div>
          <div class="cert-stats" data-reveal>
            <article><strong>{{ stats[0] }}K+</strong><h3>{{ lang === "zh" ? "专业人士" : "Professionals" }}</h3><p>{{ lang === "zh" ? "在我们不断增长的全球网络中" : "In our growing global network" }}</p></article>
            <article><strong>{{ stats[1] }}+</strong><h3>{{ lang === "zh" ? "合作伙伴" : "Partners" }}</h3><p>{{ lang === "zh" ? "与行业领袖和学术机构合作" : "Collaborating with industry leaders and academic institutions" }}</p></article>
            <article><strong>{{ stats[2] }}+</strong><h3>{{ lang === "zh" ? "国家" : "Countries" }}</h3><p>{{ lang === "zh" ? "在我们认证社区中的代表" : "Represented across our certified community" }}</p></article>
          </div>
        </div>
      </section>

      <section class="cert-cta">
        <div class="cert-container" data-reveal>
          <div><small>{{ lang === "zh" ? "认证" : "Certifications" }}</small><h2>{{ lang === "zh" ? "探索您的认证路径" : "Explore Your Certification Path" }}</h2><p>{{ lang === "zh" ? "了解GFI认证如何适合您的背景和职业目标。" : "Learn how GFI certifications fit your background and career goals." }}</p><RouterLink to="/gfi/certifications/pathway">{{ lang === "zh" ? "查看认证路径" : "View Certification Pathway" }} <ArrowUpRight /></RouterLink></div>
          <img :src="certificationAssets.professional" :alt="lang === 'zh' ? '探索您的认证路径' : 'Explore Your Certification Path'">
        </div>
      </section>
    </main>

    <main v-else-if="isPathway" class="pathway-main" :style="{ backgroundImage: `url(${certificationAssets.pathwayLines})` }">
      <section class="pathway-intro">
        <div class="cert-container" data-reveal>
          <h1><span>{{ lang === "zh" ? "您在金融科技领域的" : "Your Pathway to" }}</span> <strong>{{ lang === "zh" ? "专业认可之路" : "Professional Recognition in FinTech" }}</strong></h1>
          <p>{{ lang === "zh" ? "GFI的认证框架设计为渐进式路径，允许专业人士根据其背景在合适的级别进入，并随着其能力和经验的增长而进步。" : "GFI's certification framework is designed as a progressive pathway, allowing professionals to enter at the right level based on their background, and advance as their capability and experience grow." }}</p>
        </div>
      </section>
      <section class="pathway-steps">
        <div class="cert-container">
          <header data-reveal><span class="cert-pill">{{ lang === "zh" ? "认证路径" : "Certification Pathway" }}</span><h2>{{ lang === "zh" ? "三步解锁您的路径" : "Unlock Your Pathway in 3-Steps" }}</h2></header>
          <article v-for="stage in pathwayStages" :key="stage.number" class="pathway-card" data-reveal>
            <div><b>{{ stage.number }}</b><h3>{{ l(stage.title) }}</h3><strong>{{ l(stage.level) }}</strong><p v-for="paragraph in stage.paragraphs" :key="l(paragraph)">{{ l(paragraph) }}</p><RouterLink v-if="stage.link" :to="stage.link">{{ l(stage.linkLabel!) }} <ArrowUpRight /></RouterLink></div>
            <div class="pathway-art" :style="{ backgroundImage: `url(${certificationAssets.pathwayPattern})` }"><img :src="stage.image" :alt="l(stage.title)"></div>
          </article>
          <aside class="direct-entry" data-reveal><h2>{{ lang === "zh" ? "直接入学与豁免" : "Direct Entry & Exemptions" }}</h2><p>{{ lang === "zh" ? "GFI认可先前的学习。具有以下条件的候选人可能有资格直接进入CFtP®：" : "GFI recognizes prior learning. Direct entry into CFtP® may be available for candidates with:" }}</p><ul><li v-for="item in (lang === 'zh' ? ['来自认可机构的相关大学学位','专业认证，如CFA、CAIA或同等资格','GFI认可的认证项目'] : ['Relevant university degrees from recognised institutions','Professional certifications such as CFA, CAIA, or equivalent','Accredited programmes recognised by GFI'])" :key="item">{{ item }}</li></ul><p>{{ lang === "zh" ? "符合条件的候选人可能获得选定基础或一级组件的豁免，需经审查。" : "Eligible candidates may receive exemptions from selected foundation or Level 1 components, subject to review." }}</p><p>{{ lang === "zh" ? "这确保路径既严格又灵活，尊重先前的学习，同时保持标准。" : "This ensures the pathway remains rigorous yet flexible, respecting prior learning while maintaining standards." }}</p></aside>
        </div>
      </section>
      <section class="cert-cta pathway-cta"><div class="cert-container" data-reveal><div><small>{{ lang === "zh" ? "获得指导" : "Get Guidance" }}</small><h2>{{ lang === "zh" ? "不确定您适合哪个？" : "Not sure where you fit?" }}</h2><p>{{ lang === "zh" ? "告诉我们您的背景，我们将指导您找到合适的认证路径。" : "Tell us about your background and we will guide you to the right certification pathway." }}</p><RouterLink to="/gfi/contact">{{ lang === "zh" ? "联系我们" : "Contact Us" }} <ArrowUpRight /></RouterLink></div><img :src="certificationAssets.professional" :alt="lang === 'zh' ? '不确定您适合哪个？' : 'Not sure where you fit?'"></div></section>
    </main>

    <main v-else-if="program" class="programme-main">
      <section class="programme-banner"><img :src="program.banner" :alt="l(program.title)"></section>
      <section class="programme-meta"><div class="cert-container" data-reveal><span>{{ l(program.eyebrow) }}</span><h1>{{ l(program.title) }}</h1><dl><div><dt>{{ lang === "zh" ? "地点" : "Location" }}</dt><dd>{{ lang === "zh" ? "全球 - 在线" : "Global - Online" }}</dd></div><div><dt>Program Type</dt><dd>{{ l(program.type) }}</dd></div><div><dt>Application Deadline</dt><dd>{{ lang === "zh" ? "持续进行中" : "Ongoing" }}</dd></div></dl></div></section>
      <section class="programme-details"><div class="cert-container programme-grid">
        <div class="programme-panel" data-reveal>
          <nav class="programme-tabs" :aria-label="lang === 'zh' ? '项目内容' : 'Programme content'"><button v-for="tab in program.tabs" :key="tab.key" :class="{ active: activeTabKey === tab.key }" @click="activeTabKey = tab.key">{{ l(tab.label) }}</button></nav>
          <div class="programme-content">
            <section v-for="section in activeTab?.content" :key="l(section.heading)">
              <h2>{{ l(section.heading) }}</h2>
              <p v-for="paragraph in visibleParagraphs(section)" :key="l(paragraph)">{{ l(paragraph) }}</p>
              <ul v-if="visibleBullets(section).length"><li v-for="bullet in visibleBullets(section)" :key="l(bullet)"><i></i><span>{{ l(bullet) }}</span></li></ul>
              <div v-for="group in section.groups" :key="l(group.heading)" class="programme-group"><h3>{{ l(group.heading) }}</h3><ul v-if="visibleGroupBullets(group.bullets).length"><li v-for="bullet in visibleGroupBullets(group.bullets)" :key="l(bullet)"><i></i><span>{{ l(bullet) }}</span></li></ul></div>
            </section>
          </div>
        </div>
        <aside class="purchase-card" data-reveal><strong><small>USD</small> {{ program.price }}</strong><a :href="program.registerUrl" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "立即注册" : "Register Now" }}</a></aside>
      </div></section>
    </main>

    <GfiFooter />
  </div>
</template>

<style scoped>
.certification-page { min-height:100vh; color:#151d31; background:#fff; font-family:Arial,"Microsoft YaHei",sans-serif; }
.certification-page * { box-sizing:border-box; }
.certification-page a { text-decoration:none; }
.cert-container { width:min(1288px,calc(100% - 64px)); margin:0 auto; }
h1,h2,h3,h4,p { overflow-wrap:anywhere; letter-spacing:0; }
[data-reveal] { opacity:0; transform:translateY(24px); transition:opacity .72s ease,transform .72s ease; }
[data-reveal].is-revealed { opacity:1; transform:translateY(0); }
.cert-pill { display:inline-block; padding:4px 21px; border:1px solid #b8ceff; border-radius:18px; color:#2364f5; font-size:14px; }

.cert-hero { position:relative; min-height:732px; background-position:center; background-size:cover; color:#fff; }
.cert-hero::before { content:""; position:absolute; inset:0; background:linear-gradient(90deg,rgba(9,27,67,.98) 0%,rgba(10,30,72,.82) 42%,rgba(9,25,61,.1) 100%); }
.cert-hero .cert-container { position:relative; display:flex; min-height:732px; align-items:flex-start; flex-direction:column; justify-content:center; }
.cert-hero h1 { margin:0 0 25px; color:#fff; font-size:52px; font-weight:500; }
.cert-hero p { max-width:760px; margin:0 0 38px; padding-left:20px; border-left:1px solid #fff; font-size:16px; line-height:1.65; }
.cert-hero a,.flagship-grid a { display:flex; align-items:center; color:inherit; }
.cert-hero a > span,.flagship-grid a > span { padding:16px 30px; border:1px solid currentColor; border-radius:28px; }
.cert-hero a i,.flagship-grid a i { display:grid; width:56px; height:56px; place-items:center; margin-left:-1px; border-radius:50%; background:#fff; color:#10234c; }
.cert-hero a svg,.flagship-grid a svg { width:22px; }

.cert-impact,.cert-why { padding:110px 0; background:#f6f8fc; }
.impact-grid,.why-grid { display:grid; grid-template-columns:1.08fr .92fr; gap:110px; align-items:center; }
.impact-grid h2,.why-grid h2 { margin:14px 0 28px; color:#303237; font-size:40px; font-weight:500; }
.impact-grid > div > p,.why-grid > div > p { max-width:680px; line-height:1.7; }
.impact-grid article,.why-grid article { display:grid; grid-template-columns:56px 1fr; gap:22px; align-items:start; padding:27px 0; border-bottom:1px solid #e0e4eb; }
.impact-grid article svg,.why-grid article svg { width:56px; height:56px; padding:15px; border-radius:8px; background:#eaf0ff; color:#2764f4; }
.impact-grid article h3,.why-grid article h3 { margin:0 0 6px; color:#15213b; font-size:21px; font-weight:500; }
.impact-grid article p,.why-grid article p { margin:0; font-size:14px; line-height:1.55; }
.impact-grid > img { width:100%; max-height:470px; object-fit:contain; }

.flagship-section { padding:105px 0 115px; }
.flagship-section > .cert-container > h2 { margin:0 0 68px; text-align:center; font-size:49px; }
.flagship-grid { display:grid; grid-template-columns:1fr 1fr; gap:48px; }
.flagship-grid > article { display:flex; min-height:700px; flex-direction:column; padding:42px 34px; border:2px dashed #dce1e8; }
.flagship-grid h3 { margin:0 0 12px; font-size:28px; }
.flagship-grid > article > strong { color:#777; font-weight:400; }
.flagship-grid p { min-height:94px; line-height:1.7; }
.flagship-grid h4 { margin:18px 0 8px; padding-bottom:9px; border-bottom:1px dashed #ccd3df; color:#2764f4; font-size:16px; }
.flagship-grid ul { display:grid; gap:12px; margin:0; padding:6px 0 20px 15px; color:#60646d; }
.flagship-grid li::marker,.direct-entry li::marker { color:#2764f4; }
.flagship-grid a { width:max-content; margin-top:auto; color:#2764f4; }
.flagship-grid a i { background:#2764f4; color:#fff; }

.why-grid { grid-template-columns:1.15fr .85fr; }
.cert-stats { display:grid; grid-template-columns:1fr 1fr; }
.cert-stats article { display:flex; min-height:155px; grid-template-columns:none; gap:0; align-items:flex-end; justify-content:center; flex-direction:column; padding:24px; border-right:1px solid #dde2e9; border-bottom:1px solid #dde2e9; text-align:right; }
.cert-stats article:nth-child(2) { align-items:flex-start; border-right:0; text-align:left; }
.cert-stats article:nth-child(3) { grid-column:1 / -1; align-items:center; border-right:0; border-bottom:0; text-align:center; }
.cert-stats strong { color:#2764f4; font-size:43px; }
.cert-stats h3 { margin:4px 0; font-size:16px; }
.cert-stats p { margin:0; color:#60656f; font-size:13px; }

.cert-cta { position:relative; overflow:hidden; background:#10234d; color:#fff; }
.cert-cta .cert-container { display:grid; min-height:480px; grid-template-columns:1fr 520px; align-items:center; }
.cert-cta small { color:#7fa1ff; font-size:14px; }
.cert-cta h2 { margin:15px 0 18px; color:#fff; font-size:42px; }
.cert-cta p { margin:0 0 28px; color:#d4daea; }
.cert-cta a { display:flex; width:max-content; align-items:center; gap:8px; padding-bottom:7px; border-bottom:1px solid #fff; color:#fff; }
.cert-cta a svg { width:16px; }
.cert-cta img { align-self:end; width:100%; max-height:445px; object-fit:contain; object-position:bottom; }

.pathway-main { background-color:#f8fafc; background-repeat:repeat; background-size:450px; }
.pathway-intro { padding:96px 0 115px; }
.pathway-intro .cert-container { display:grid; grid-template-columns:1.1fr .9fr; gap:120px; align-items:center; }
.pathway-intro h1 { margin:0; color:#333; font-size:37px; font-weight:400; line-height:1.35; }
.pathway-intro h1 strong { color:#2764f4; }
.pathway-intro p { margin:0; line-height:1.65; }
.pathway-steps { padding-bottom:110px; }
.pathway-steps header { margin-bottom:70px; text-align:center; }
.pathway-steps header h2 { margin:16px 0 0; color:#303237; font-size:38px; font-weight:500; }
.pathway-card { display:grid; min-height:475px; grid-template-columns:1.25fr .85fr; margin-bottom:58px; border:1px solid #abc4ff; background:#fff; }
.pathway-card:nth-of-type(even) > div:first-child { order:2; }
.pathway-card:nth-of-type(even) .pathway-art { order:1; }
.pathway-card > div:first-child { padding:72px 64px; }
.pathway-card b { display:block; margin-bottom:25px; color:#2764f4; font-size:48px; }
.pathway-card h3 { margin:0 0 12px; font-size:27px; }
.pathway-card > div > strong { display:block; margin-bottom:26px; color:#777; font-weight:400; }
.pathway-card p { line-height:1.72; }
.pathway-card a { display:flex; width:max-content; align-items:center; gap:7px; margin-top:26px; padding-bottom:7px; border-bottom:1px solid #2764f4; color:#2764f4; }
.pathway-card a svg { width:16px; }
.pathway-art { display:grid; min-height:100%; place-items:center; overflow:hidden; background-size:cover; }
.pathway-art img { width:70%; max-height:360px; object-fit:contain; }
.direct-entry { padding:38px 40px; border-left:4px solid #3978ff; border-radius:8px; background:linear-gradient(135deg,#eef5ff,#fff); }
.direct-entry h2 { margin:0 0 20px; font-size:24px; }
.direct-entry p { line-height:1.7; }
.direct-entry ul { display:grid; gap:14px; padding-left:20px; }

.programme-main { background:#fff; }
.programme-banner { margin-top:96px; }
.programme-banner img { display:block; width:100%; height:335px; object-fit:cover; }
.programme-meta { padding:110px 0 48px; background:#f6f8fc; }
.programme-meta span { font-size:15px; }
.programme-meta h1 { margin:15px 0 28px; color:#071638; font-size:39px; font-weight:500; }
.programme-meta dl { display:flex; gap:48px; margin:0; }
.programme-meta dl > div { min-width:80px; }
.programme-meta dt { margin-bottom:16px; color:#8a8d93; letter-spacing:2px; }
.programme-meta dd { margin:0; font-size:14px; }
.programme-details { padding:64px 0 95px; }
.programme-grid { display:grid; grid-template-columns:minmax(0,716px) minmax(320px,404px); gap:168px; align-items:start; }
.programme-panel { border:1px solid #e5e7eb; }
.programme-tabs { display:flex; overflow-x:auto; }
.programme-tabs button { min-width:140px; min-height:69px; flex:1 0 140px; padding:12px; border:0; border-right:1px solid #e5e7eb; border-bottom:1px solid #e5e7eb; background:#fff; color:#252a32; font-size:15px; cursor:pointer; }
.programme-tabs button.active { border-bottom:2px solid #2764f4; background:#f0f3fb; color:#2764f4; font-weight:600; }
.programme-content { min-height:510px; padding:60px 24px 70px; }
.programme-content section + section { margin-top:38px; }
.programme-content h2 { margin:0 0 20px; color:#151c2c; font-size:25px; }
.programme-content p { line-height:1.8; }
.programme-content ul { display:grid; gap:13px; margin:0; padding:0; list-style:none; }
.programme-content li { display:flex; gap:17px; line-height:1.55; }
.programme-content li i { width:5px; height:5px; flex:0 0 5px; margin-top:9px; border-radius:50%; background:#2764f4; }
.programme-group { margin-top:32px; }
.programme-group h3 { margin:0 0 16px; font-size:19px; }
.purchase-card { position:sticky; top:115px; margin-top:-257px; padding:28px 24px 25px; background:#fff; box-shadow:0 10px 35px rgba(29,49,85,.08); }
.purchase-card strong { display:block; margin-bottom:26px; color:#050505; font-size:37px; text-align:center; }
.purchase-card strong small { color:#394050; font-size:16px; }
.purchase-card a { display:block; padding:14px; border:1px solid #2764f4; border-radius:6px; background:#2764f4; color:#fff; font-weight:600; text-align:center; }
@media (max-width:1000px) {
  .impact-grid,.why-grid,.pathway-intro .cert-container { grid-template-columns:1fr; gap:55px; }
  .flagship-grid { grid-template-columns:1fr; }
  .programme-grid { grid-template-columns:minmax(0,1fr) 320px; gap:40px; }
  .cert-cta .cert-container { grid-template-columns:1fr 390px; }
}

@media (max-width:700px) {
  .certification-page :deep(.gfi-header) { height:calc(91px + var(--app-safe-area-top)); }
  .certification-page :deep(.gfi-header-inner) { width:calc(100% - 90px); }
  .certification-page :deep(.gfi-brand) { width:122px; flex-basis:122px; }
  .certification-page :deep(.gfi-menu-toggle) { top:calc(24px + var(--app-safe-area-top)); right:calc(45px + var(--app-safe-area-right)); }
  .certification-page :deep(.gfi-mobile-nav) { top:calc(91px + var(--app-safe-area-top)); max-height:calc(var(--app-viewport-height) - 91px - var(--app-safe-area-top)); }
  .cert-container { width:calc(100% - 32px); }
  .cert-hero,.cert-hero .cert-container { min-height:385px; }
  .cert-hero { background-position:50% center; }
  .cert-hero .cert-container { justify-content:flex-start; padding-top:124px; }
  .cert-hero::before { background:linear-gradient(90deg,rgba(9,27,67,.97),rgba(9,27,67,.68)); }
  .cert-hero h1 { font-size:32px; line-height:1.2; white-space:normal; }
  .cert-hero p { width:100%; max-width:100%; overflow-wrap:anywhere; font-size:14px; }
  .cert-hero a > span,.flagship-grid a > span { padding:13px 19px; }
  .cert-hero a i,.flagship-grid a i { width:48px; height:48px; }
  .cert-impact { padding:110px 0; }
  .cert-why,.flagship-section { padding:72px 0; }
  .cert-pill { padding:2px 21px; }
  .impact-grid h2,.why-grid h2,.flagship-section > .cert-container > h2 { font-size:31px; }
  .impact-grid article,.why-grid article { grid-template-columns:48px 1fr; gap:15px; }
  .impact-grid article svg,.why-grid article svg { width:48px; height:48px; padding:13px; }
  .flagship-section > .cert-container > h2 { margin-bottom:40px; }
  .flagship-grid > article { min-height:0; padding:30px 23px; }
  .flagship-grid h3 { font-size:24px; }
  .flagship-grid p { min-height:0; }
  .cert-stats { grid-template-columns:1fr; }
  .cert-stats article,.cert-stats article:nth-child(2),.cert-stats article:nth-child(3) { min-height:130px; grid-column:auto; align-items:flex-start; border-right:0; border-bottom:1px solid #dde2e9; text-align:left; }
  .cert-cta .cert-container { min-height:560px; grid-template-columns:1fr; padding-top:65px; }
  .cert-cta h2 { font-size:32px; }
  .cert-cta img { max-height:300px; }
  .pathway-intro { padding:67px 0 134px; }
  .pathway-intro h1 { font-size:31px; }
  .pathway-intro p { display:none; }
  .pathway-steps header h2 { margin-top:10px; font-size:31px; }
  .pathway-card { grid-template-columns:1fr; }
  .pathway-card:nth-of-type(even) > div:first-child { order:1; }
  .pathway-card:nth-of-type(even) .pathway-art { order:2; }
  .pathway-card > div:first-child { padding:38px 24px; }
  .pathway-card b { font-size:40px; }
  .pathway-card h3 { font-size:24px; }
  .pathway-art { min-height:300px; }
  .direct-entry { padding:30px 24px; }
  .programme-banner { margin-top:46px; }
  .programme-banner img { height:116px; }
  .programme-meta { padding:64px 0 37px; }
  .programme-meta h1 { margin:5px 0 18px; font-size:31px; }
  .programme-meta dl { display:grid; grid-template-columns:minmax(0,1fr); gap:18px; }
  .programme-meta dl > div { min-width:0; }
  .programme-meta dt { margin-bottom:6px; white-space:normal; }
  .programme-meta dd { overflow-wrap:anywhere; }
  .programme-details { padding:15px 0 70px; }
  .programme-grid { display:flex; flex-direction:column; gap:28px; }
  .programme-panel,.purchase-card { width:100%; }
  .purchase-card { position:static; margin:0; }
  .programme-tabs { flex-direction:column; overflow:visible; }
  .programme-tabs button { width:100%; min-width:0; min-height:51px; flex:0 0 51px; padding:12px 15px; border-right:0; text-align:left; }
  .programme-tabs button.active { border-bottom:1px solid #e5e7eb; border-left:3px solid #2764f4; }
  .programme-content { padding:40px 22px 55px; }
  .programme-content h2 { font-size:23px; }
}
</style>
