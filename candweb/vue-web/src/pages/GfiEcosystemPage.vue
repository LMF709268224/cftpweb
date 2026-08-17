<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, useRoute } from "vue-router"
import { ArrowLeft, ArrowRight, ArrowUpRight } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import GfiLineBackground from "@/components/GfiLineBackground.vue"
import { useTranslation } from "@/lib/language"
import { localize } from "@/lib/gfiSite"
import {
  corporateFaqs,
  corporateRows,
  corporateTiers,
  ecosystemAssets,
  membershipRows,
  membershipTiers,
  partnerGroups,
} from "@/lib/gfiEcosystemPages"

const route = useRoute()
const { lang } = useTranslation()
const page = computed(() => route.path.replace(/^\/gfi\/?/, "").replace(/\/$/, ""))
const isProgrammes = computed(() => page.value === "programmes")
const isPartnerships = computed(() => page.value === "partnerships")
const isMembership = computed(() => page.value === "membership/benefits")
const isCorporate = computed(() => page.value === "membership/corporate")
const l = (value: { zh: string; en: string }) => localize(value, lang.value)

const tierIndex = ref(0)
const compactCarousel = ref(false)
const openFaq = ref(0)
const stats = ref([0, 0, 0])
const visibleCorporateTiers = computed(() => corporateTiers
  .map((_, offset) => corporateTiers[(tierIndex.value + offset) % corporateTiers.length])
  .slice(0, 3))
const corporateSlideCount = computed(() => compactCarousel.value ? corporateTiers.length : 2)
let revealObserver: IntersectionObserver | null = null
let statObserver: IntersectionObserver | null = null
let tierTimer = 0
let animationFrame = 0

function moveTier(direction: number) {
  tierIndex.value = (tierIndex.value + direction + corporateSlideCount.value) % corporateSlideCount.value
}

function syncCarouselViewport() {
  compactCarousel.value = window.innerWidth < 980
  if (tierIndex.value >= corporateSlideCount.value) tierIndex.value = 0
}

function startStats() {
  const started = performance.now()
  const targets = [500, 10, 30]
  const animate = (now: number) => {
    const progress = Math.min((now - started) / 1100, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    stats.value = targets.map((target) => Math.round(target * eased))
    if (progress < 1) animationFrame = requestAnimationFrame(animate)
  }
  animationFrame = requestAnimationFrame(animate)
}

async function initialiseAnimations() {
  await nextTick()
  revealObserver?.disconnect()
  statObserver?.disconnect()
  revealObserver = new IntersectionObserver((entries) => entries.forEach((entry) => {
    if (!entry.isIntersecting) return
    entry.target.classList.add("is-revealed")
    revealObserver?.unobserve(entry.target)
  }), { threshold: .08, rootMargin: "0px 0px -30px" })
  document.querySelectorAll(".ecosystem-page [data-reveal]").forEach((element) => revealObserver?.observe(element))
  const statElement = document.querySelector(".ecosystem-page .flex-stats")
  if (statElement) {
    statObserver = new IntersectionObserver(([entry]) => {
      if (!entry?.isIntersecting) return
      startStats()
      statObserver?.disconnect()
    }, { threshold: .25 })
    statObserver.observe(statElement)
  }
}

onMounted(() => {
  syncCarouselViewport()
  window.addEventListener("resize", syncCarouselViewport)
  initialiseAnimations()
  tierTimer = window.setInterval(() => {
    if (isCorporate.value) moveTier(1)
  }, 5500)
})
watch([page, lang], initialiseAnimations)
onBeforeUnmount(() => {
  revealObserver?.disconnect()
  statObserver?.disconnect()
  cancelAnimationFrame(animationFrame)
  window.clearInterval(tierTimer)
  window.removeEventListener("resize", syncCarouselViewport)
})
</script>

<template>
  <div class="ecosystem-page">
    <GfiHeader theme="light" />

    <main v-if="isProgrammes" class="programme-page">
      <GfiLineBackground />
      <section class="programme-intro section-container" data-reveal>
        <h1>{{ lang === "zh" ? "金融科技领导力高管教育" : "Executive Education for FinTech Leadership" }}</h1>
        <p>{{ lang === "zh" ? "GFI的高管项目专为高级专业人士、监管机构和决策者设计，旨在深入理解具有战略、监管和系统性影响的金融科技发展。" : "GFI's executive programmes are designed for senior professionals, regulators, and decision-makers seeking a deeper understanding of fintech developments with strategic, regulatory, and systemic implications." }}</p>
        <p>{{ lang === "zh" ? "这些项目超越了技术概述。它们专注于治理、风险、政策和实际执行，使领导者能够应对复杂性，做出明智的决策，并与整个金融生态系统的利益相关者进行可信的互动。" : "These programmes go beyond technical overviews. They focus on governance, risk, policy, and real-world execution, enabling leaders to navigate complexity, make informed decisions, and engage credibly with stakeholders across the financial ecosystem." }}</p>
      </section>
      <section class="programme-list section-container">
        <header data-reveal><span class="eyebrow">Current Programmes</span><h2>Our Fintech Programmes</h2></header>
        <article class="programme-card" data-reveal>
          <img :src="ecosystemAssets.programme" alt="Foundation in Crypto Regulation and Compliance">
          <div>
            <h3>Foundation in Crypto Regulation and Compliance</h3>
            <i aria-hidden="true"></i>
            <p>A rigorous 16-hour short course co-delivered by the Global Fintech Institute (GFI), in partnership with Binance—designed to equip professionals with a deep understanding of digital asset regulation, blockchain infrastructure, and emerging compliance frameworks.</p>
            <RouterLink to="/gfi/programmes/executive-program">Read More <ArrowUpRight /></RouterLink>
          </div>
        </article>
      </section>
    </main>

    <main v-else-if="isPartnerships" class="partnership-page">
      <section class="partnership-hero">
        <GfiLineBackground />
        <div class="section-container" data-reveal>
          <h1>{{ lang === "zh" ? "与GFI合作" : "Become a Partner" }}</h1>
          <p>{{ lang === "zh" ? "GFI与全球30多个合作伙伴网络合作，通过学术合作、应用研究和行业驱动的教育项目，在商业、政府和社会中创造有意义的影响。" : "Reach targeted audiences, connect with senior decision-makers, form strategic partnerships, and showcase your brand or product on a global stage. Partner with Global Fintech Institute to co-create programmes, advance fintech standards, and build talent pipelines across Asia and beyond." }}</p>
          <div class="hero-actions"><a href="https://airtable.com/appeDc6m5JlxCGaHn/pagLzTRtb9YDubupY/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "提交合作意向" : "Submit Partnership Interest" }} <ArrowUpRight /></a><a href="#our-partners" class="secondary">{{ lang === "zh" ? "查看我们的合作伙伴" : "View Our Partners" }}</a></div>
        </div>
      </section>
      <section id="our-partners" class="partner-directory section-container">
        <header class="section-heading" data-reveal><h2>{{ lang === "zh" ? "我们的合作伙伴" : "Our Partners" }}</h2><p>{{ lang === "zh" ? "全球金融科技学院与教育、行业和更广泛生态系统的组织合作。以下是与GFI在课程、研究、人才、政策对话或战略合作方面有过接触的机构选择。" : "The Global Fintech Institute collaborates with organisations across education, industry, and the wider ecosystem. Below is a selection of institutions that have engaged with GFI across programmes, research, talent, policy dialogue, or strategic collaboration." }}</p></header>
        <section class="patron-block" data-reveal><h3>{{ lang === "zh" ? "白金企业赞助商" : "Platinum Corporate Patron" }}</h3><p>{{ lang === "zh" ? "2026年白金企业赞助商" : "2026 Platinum Corporate Patron" }}</p><a class="patron-card patron-card-rich" href="https://www.binance.com/en" target="_blank" rel="noopener noreferrer"><div class="logo-frame"><img src="/gfi/ecosystem/binance.svg" alt="Binance"></div><div class="partner-copy"><span>&lt;链接&gt;</span><u>https://www.binance.com/en</u><span>&lt;链接&gt;</span><span>&lt;内容&gt;</span><strong>币安</strong><span>&lt;内容&gt;</span></div></a></section>
        <section class="patron-block" data-reveal><h3>Silver Corporate Patron</h3><p>2026 Silver Corporate Patron</p><a class="patron-card" href="https://uqpay.com" target="_blank" rel="noopener noreferrer"><div class="logo-frame"><img src="/gfi/ecosystem/uqpay.webp" alt="UQPay"></div><div class="partner-copy"><strong>UQPay</strong></div></a></section>

        <template v-for="group in partnerGroups" :key="group.key">
          <section v-if="group.key === 'education'" class="partner-group">
            <header data-reveal><h3>{{ l(group.title) }}</h3><p>{{ group.description ? l(group.description) : "" }}</p></header>
            <div class="partner-grid education-grid">
              <a v-for="partner in group.partners" :key="partner.url" :href="partner.url" target="_blank" rel="noopener noreferrer" data-reveal>
                <div class="logo-frame"><img :src="partner.logo" :alt="l(partner.name)"></div><div class="partner-copy rich-partner-copy"><span>&lt;链接&gt;</span><u>{{ partner.url }}</u><span>&lt;链接&gt;</span><span>&lt;内容&gt;</span><ul><li v-for="item in partner.programmes" :key="item"><strong>{{ item }}</strong></li></ul><span>&lt;内容&gt;</span></div>
              </a>
            </div>
          </section>
        </template>
        <section class="partner-group community-group">
          <header data-reveal><h3>{{ lang === "zh" ? "社区与生态系统合作伙伴" : "Community & Ecosystem Partners" }}</h3><p>{{ lang === "zh" ? "与GFI在生态系统建设、合规和治理以及跨境对话方面合作的行业协会、网络、公司和机构。" : "Industry associations, networks, companies, and institutions working with GFI on ecosystem building, compliance and governance, and cross-border dialogue." }}</p></header>
          <template v-for="group in partnerGroups.filter((item) => item.key !== 'education')" :key="group.key"><h4 class="subgroup-title" data-reveal>{{ l(group.title) }}</h4><div class="partner-grid compact-grid"><a v-for="partner in group.partners" :key="partner.url" :href="partner.url" target="_blank" rel="noopener noreferrer" data-reveal><div class="logo-frame"><img :src="partner.logo" :alt="l(partner.name)"></div><div class="partner-copy rich-partner-copy"><span>&lt;链接&gt;</span><u>{{ partner.url }}</u><span>&lt;链接&gt;</span><span>&lt;内容&gt;</span><strong>{{ l(partner.name) }}</strong><span>&lt;内容&gt;</span></div></a></div></template>
        </section>
        <section class="media-block" data-reveal><h3>{{ lang === "zh" ? "媒体合作伙伴" : "Media Partners" }}</h3><p>{{ lang === "zh" ? "与GFI在思想领导力、活动报道和金融科技与数字金融公共教育方面合作的媒体和内容平台。媒体合作伙伴帮助将GFI圆桌会议、项目和研究的见解传播给更广泛的全球受众。" : "Media and content platforms working with GFI on thought leadership, event coverage, and public education across fintech and digital finance." }}</p><strong>{{ lang === "zh" ? "暂无合作伙伴" : "No partners yet" }}</strong></section>
      </section>
      <section class="image-cta"><div class="section-container" data-reveal><div><span class="eyebrow">{{ lang === "zh" ? "合作机会" : "Partnership Opportunity" }}</span><h2>{{ lang === "zh" ? "准备探索与GFI的合作？" : "Ready to Explore a Partnership with GFI?" }}</h2><p>{{ lang === "zh" ? "与我们分享一些详细信息，我们将在与当前重点领域有强烈契合的地方跟进。" : "Share a few details with us and we will follow up where there is a strong fit with current priorities." }}</p><a href="https://airtable.com/appeDc6m5JlxCGaHn/pagLzTRtb9YDubupY/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "提交合作意向" : "Submit Partnership Interest" }} <ArrowUpRight /></a></div><img :src="ecosystemAssets.partnershipCta" alt="GFI partnership"></div></section>
    </main>

    <main v-else-if="isMembership" class="membership-page">
      <section class="membership-hero"><GfiLineBackground /><div class="section-container" data-reveal><h1>{{ lang === "zh" ? "GFI会员计划" : "GFI Membership Programme" }}</h1><p>{{ lang === "zh" ? "在全球金融科技学院，会员不仅仅是订阅——这是对专业卓越的承诺，是金融科技领域的终身职业旅程。我们的分层会员结构支持从探索行业的学生到塑造未来的高级专业人士的每个阶段。" : "At the Global Fintech Institute, membership is more than a subscription—it is a commitment to professional excellence and a lifelong career journey in fintech. Our tiered membership structure supports every stage, from students exploring the industry to senior professionals shaping its future." }}</p></div></section>
      <section class="tier-section section-container"><div class="membership-grid"><article v-for="tier in membershipTiers" :key="tier.price" data-reveal><h2>{{ l(tier.title) }}</h2><div class="price"><strong>{{ tier.price }}</strong><b>USD</b><i aria-hidden="true"></i><span>Year</span></div><p>{{ l(tier.description) }}</p><h3>Key Benefits</h3><ul><li v-for="benefit in tier.benefits[lang]" :key="benefit"><b>•</b>{{ benefit }}</li></ul><div class="best"><h3>Best for:</h3><p>{{ l(tier.best) }}</p></div><a href="/marketplace" target="_blank" rel="noopener noreferrer"><span>Join Now</span><i><ArrowUpRight /></i></a></article></div></section>
      <section class="comparison-section"><div class="section-container"><header class="section-heading" data-reveal><h2>{{ lang === "zh" ? "会员对比一览" : "Membership at a Glance" }}</h2><p>{{ lang === "zh" ? "比较每个会员等级的福利和功能，找到最适合您职业发展的选择。" : "Compare the benefits and features of each membership level to find the right fit for your professional development." }}</p></header><div class="table-scroll" data-reveal><table><thead><tr><th>{{ lang === "zh" ? "适用对象" : "Designed for" }}</th><th>{{ lang === "zh" ? "向所有人开放" : "Open to all" }}<small>49 USD / Year</small></th><th>{{ lang === "zh" ? "CFtP候选人及CFtA持有者" : "CFtP candidates & CFtA holders" }}<small>129 USD / Year</small></th><th>{{ lang === "zh" ? "仅限CFtP特许持有者" : "CFtP charterholders only" }}<small>169 USD / Year</small></th></tr></thead><tbody><tr v-for="row in membershipRows" :key="row[0]"><th>{{ row[0] }}</th><td v-for="cell in row.slice(1)" :key="cell">{{ cell }}</td></tr></tbody></table></div></div></section>
      <section class="flex-section"><div class="section-container flex-layout"><div data-reveal><span class="eyebrow">{{ lang === "zh" ? "您通往经过验证的金融科技机会的门户" : "Your Gateway to Verified FinTech Opportunities" }}</span><h2>{{ lang === "zh" ? "FLEX职业门户" : "FLEX Career Portal" }}</h2><p>{{ lang === "zh" ? "FLEX职业门户是GFI的全球人才平台，旨在将经过验证的金融科技专业人士与雇主、机构和生态系统合作伙伴联系起来。" : "The FLEX Career Portal is GFI's global talent platform, connecting verified fintech professionals with employers, institutions, and ecosystem partners." }}</p><p>{{ lang === "zh" ? "仅限CFtP®特许会员使用，FLEX超越了职位列表——基于证书、专业知识和专业地位进行发现，而不仅仅是简历。" : "Available exclusively to CFtP® Charterholder Members, FLEX goes beyond job listings—enabling discovery based on credentials, expertise, and professional standing, not just resumes." }}</p><a href="https://match.flex.sg/auth/login" target="_blank" rel="noopener noreferrer"><span>{{ lang === "zh" ? "探索FLEX" : "Explore FLEX" }}</span><i><ArrowUpRight /></i></a></div><div class="flex-stats" data-reveal><article><strong>{{ String(stats[0]).padStart(3, "0") }}+</strong><h3>{{ lang === "zh" ? "专业人士" : "Professionals" }}</h3><p>{{ lang === "zh" ? "跨角色和市场的经过验证的金融科技人才全球社区" : "A global community of verified fintech talent across roles and markets" }}</p></article><article><strong>{{ String(stats[1]).padStart(2, "0") }}+</strong><h3>{{ lang === "zh" ? "国家" : "Countries" }}</h3><p>{{ lang === "zh" ? "跨越金融中心、创新中心和新兴市场的机会" : "Opportunities across financial centres, innovation hubs, and emerging markets" }}</p></article><article><strong>{{ String(stats[2]).padStart(2, "0") }}+</strong><h3>{{ lang === "zh" ? "合作伙伴组织" : "Partner Organisations" }}</h3><p>{{ lang === "zh" ? "与经过验证的人才互动的雇主、机构和生态系统合作伙伴" : "Employers, institutions, and ecosystem partners engaging verified talent" }}</p></article></div></div></section>
      <section class="image-cta member-cta"><div class="section-container" data-reveal><div><span class="eyebrow">{{ lang === "zh" ? "保持联系" : "Stay Connected" }}</span><h2>{{ lang === "zh" ? "成为GFI社区的一员" : "Be Part of the GFI Community" }}</h2><p>{{ lang === "zh" ? "及时了解即将推出的项目、网络研讨会、研究和会员更新。无论您是在探索金融科技还是已经是生态系统的一部分，我们都会让您了解重要信息。" : "Stay informed about upcoming programmes, webinars, research, and membership updates. Whether you are exploring fintech or already part of the ecosystem, we keep you connected to what matters." }}</p><a href="https://www.linkedin.com/company/globalfintechinstitute/" target="_blank" rel="noopener noreferrer"><span>{{ lang === "zh" ? "关注更新" : "Follow for Updates" }}</span><i><ArrowUpRight /></i></a></div><div class="member-art"><i aria-hidden="true"></i><img :src="ecosystemAssets.professional" alt="GFI community"></div></div></section>
    </main>

    <main v-else-if="isCorporate" class="corporate-page">
      <section class="corporate-hero"><GfiLineBackground /><div class="section-container" data-reveal><h1>Corporate Patron Programme</h1><p>GFI's Corporate Patron Programme is designed for organisations that want to do more than sponsor events. It enables institutions to <strong>build internal fintech capability</strong>, <strong>engage regulators and industry leaders</strong>, and <strong>contribute meaningfully to the global fintech ecosystem</strong> through education, talent, and standards.</p><p>Our patron tiers provide a structured progression — from participation and visibility, to influence and leadership — aligned with your organisation's scale and strategic intent.</p></div></section>
      <section class="corporate-tiers section-container"><button class="edge-arrow edge-arrow-left" aria-label="Previous tier" @click="moveTier(-1)"><ArrowLeft /></button><button class="edge-arrow edge-arrow-right" aria-label="Next tier" @click="moveTier(1)"><ArrowRight /></button><div class="corporate-grid"><article v-for="tier in visibleCorporateTiers" :key="tier.name" data-reveal><h2>{{ tier.name }}</h2><div class="price"><strong>{{ tier.price }}</strong><b>USD</b><i aria-hidden="true"></i><span>Year</span></div><p>{{ tier.description }}</p><h3>Key Benefits</h3><ul><li v-for="benefit in tier.benefits" :key="benefit"><b>•</b>{{ benefit }}</li></ul><div class="best"><h3>Best for:</h3><p>{{ tier.best }}</p></div><a href="https://airtable.com/appd7xPwALyxFOYFx/pagDMRvuwDoH5JbpY/form" target="_blank" rel="noopener noreferrer"><span>Contact Us</span><i><ArrowUpRight /></i></a></article></div><div class="carousel-dots"><button v-for="index in corporateSlideCount" :key="index" :class="{ active: tierIndex === index - 1 }" :aria-label="`Show slide ${index}`" @click="tierIndex = index - 1"></button></div></section>
      <section class="segments-section"><div class="section-container"><header class="section-heading" data-reveal><h2>Patron Segments</h2><p>Compare the benefits and features of each patron tier to find the perfect fit for your organisation.</p></header><div class="table-scroll" data-reveal><table><thead><tr><th>Category</th><th>Benefit</th><th>Bronze<small>USD 5,000</small></th><th>Silver<small>USD 10,000</small></th><th>Gold<small>USD 20,000</small></th><th>Platinum<small>USD 50,000</small></th></tr></thead><tbody><tr v-for="row in corporateRows" :key="row.join('-')"><th>{{ row[0] }}</th><td v-for="cell in row.slice(1)" :key="cell">{{ cell }}</td></tr></tbody></table></div></div></section>
      <section class="faq-section section-container"><header class="section-heading" data-reveal><h2>Frequently Asked Questions</h2></header><div class="faq-list" data-reveal><article v-for="(faq, index) in corporateFaqs" :key="faq[0]" :class="{ open: openFaq === index }"><button :aria-expanded="openFaq === index" @click="openFaq = openFaq === index ? -1 : index"><span>{{ faq[0] }}</span><ArrowUpRight /></button><div v-show="openFaq === index"><p>{{ faq[1] }}</p></div></article></div></section>
      <section class="image-cta corporate-cta"><div class="section-container" data-reveal><div><span class="eyebrow">业务目标</span><h2>Interested in Becoming a Corporate Patron?</h2><p>Partner with GFI to align your organisation with global fintech standards, talent development, and ecosystem leadership.</p><p>📩 Partnership enquiries: <a class="email-link" href="mailto:partnerships@globalfintechinstitute.org">partnerships@globalfintechinstitute.org</a></p><a class="cta-button" href="https://airtable.com/appd7xPwALyxFOYFx/pagDMRvuwDoH5JbpY/form" target="_blank" rel="noopener noreferrer"><span>Contact Us</span><i><ArrowUpRight /></i></a></div><div class="member-art"><i aria-hidden="true"></i><img :src="ecosystemAssets.professional" alt="Corporate Patron Programme"></div></div></section>
    </main>

    <GfiFooter />
  </div>
</template>

<style scoped>
.ecosystem-page { --blue:#225ef5; --navy:#0e214d; --ink:#17233f; color:var(--ink); background:#fff; font-family:Arial,"Microsoft YaHei",sans-serif; }
.ecosystem-page * { box-sizing:border-box; }
.ecosystem-page a { text-decoration:none; }
.section-container { width:min(1288px,calc(100% - 64px)); margin:0 auto; }
h1,h2,h3,h4,p { margin-top:0; overflow-wrap:anywhere; letter-spacing:0; }
[data-reveal] { opacity:1; transform:none; }
[data-reveal].is-revealed { animation:gfi-reveal-in .75s ease both; }
@keyframes gfi-reveal-in { from { opacity:0; transform:translateY(25px); } to { opacity:1; transform:translateY(0); } }
.partner-grid [data-reveal],.membership-grid [data-reveal],.corporate-grid [data-reveal] { opacity:1 !important; transform:none !important; }
.patterned,.pattern-page { background-color:#f8faff; background-position:center top; background-size:cover; }
.eyebrow { display:inline-block; margin-bottom:14px; padding:5px 18px; border:1px solid #b9ccff; border-radius:20px; color:var(--blue); font-size:14px; }
.section-heading { max-width:780px; margin:0 auto 55px; text-align:center; }
.section-heading h2 { margin-bottom:18px; font-size:40px; font-weight:500; }
.section-heading p { color:#546079; font-size:16px; line-height:1.8; }

.programme-intro { max-width:960px; padding:135px 0 170px; text-align:center; }
.programme-intro h1 { margin-bottom:34px; color:#101e42; font-size:50px; font-weight:500; }
.programme-intro p { max-width:850px; margin:0 auto 20px; color:#526077; font-size:17px; line-height:1.85; }
.programme-list { padding:0 0 150px; }
.programme-list header { margin-bottom:45px; }
.programme-list h2 { font-size:40px; font-weight:500; }
.programme-card { width:418px; overflow:hidden; border-radius:6px; background:#fff; box-shadow:0 12px 35px rgba(24,45,88,.13); }
.programme-card > img { display:block; width:100%; height:240px; object-fit:cover; }
.programme-card > div { padding:30px; }
.programme-card h3 { font-size:23px; font-weight:500; line-height:1.35; }
.programme-card p { color:#5d677b; font-size:14px; line-height:1.75; }
.programme-card a,.hero-actions a,.image-cta a,.flex-section a { display:inline-flex; align-items:center; gap:10px; color:var(--blue); font-weight:600; }
.programme-card svg,.image-cta svg,.hero-actions svg,.flex-section svg { width:17px; }

.partnership-hero,.membership-hero,.corporate-hero { padding:125px 0 150px; text-align:center; }
.partnership-hero .section-container,.membership-hero .section-container,.corporate-hero .section-container { max-width:920px; }
.partnership-hero h1,.membership-hero h1,.corporate-hero h1 { margin-bottom:30px; font-size:52px; font-weight:500; }
.partnership-hero p,.membership-hero p,.corporate-hero p { color:#4e5c74; font-size:17px; line-height:1.85; }
.hero-actions { display:flex; justify-content:center; gap:14px; margin-top:38px; }
.hero-actions a { padding:14px 24px; border:1px solid var(--blue); border-radius:3px; background:var(--blue); color:#fff; }
.hero-actions a.secondary { background:#fff; color:var(--blue); }
.partner-directory { padding:140px 0 115px; }
.patron-block { margin-bottom:95px; text-align:center; }
.patron-block h3,.partner-group > header h3,.media-block h3 { margin-bottom:12px; font-size:30px; font-weight:500; }
.patron-block > p,.partner-group > header p,.media-block > p { color:#647087; line-height:1.7; }
.patron-card { display:flex; width:300px; min-height:300px; margin:30px auto 0; padding:30px; flex-direction:column; align-items:center; justify-content:center; gap:20px; border:1px solid #edf0f5; border-radius:8px; color:var(--ink); box-shadow:0 10px 30px rgba(22,42,83,.1); transition:transform .3s ease,box-shadow .3s ease; }
.patron-card:hover,.partner-grid > a:hover { transform:translateY(-8px); box-shadow:0 18px 36px rgba(22,42,83,.15); }
.patron-card img { width:180px; height:185px; object-fit:contain; }
.partner-group { margin:115px 0; }
.partner-group > header { max-width:820px; margin:0 auto 42px; text-align:center; }
.partner-grid { display:flex; flex-wrap:wrap; justify-content:center; gap:24px; }
.partner-grid > a { display:flex; width:280px; min-height:390px; flex-direction:column; overflow:hidden; border:1px solid #edf0f4; border-radius:8px; background:#fff; color:var(--ink); box-shadow:0 8px 25px rgba(22,42,83,.09); transition:transform .3s ease,box-shadow .3s ease; }
.logo-frame { display:flex; height:200px; padding:28px; align-items:center; justify-content:center; border-bottom:1px solid #eef1f5; background:linear-gradient(145deg,#fafbfc,#f4f6f9); }
.logo-frame img { max-width:100%; max-height:150px; object-fit:contain; }
.partner-copy { padding:22px; }
.partner-copy h4 { margin-bottom:13px; font-size:17px; }
.partner-copy ul { margin:0; padding-left:18px; color:#58657b; font-size:14px; line-height:1.7; }
.subgroup-title { margin:45px 0 25px; text-align:center; font-size:23px; font-weight:500; }
.compact-grid > a { min-height:280px; }
.compact-grid .partner-copy { text-align:center; }
.media-block { max-width:850px; margin:120px auto 0; text-align:center; }
.media-block strong { display:block; margin-top:25px; color:#8a93a5; font-weight:400; }
.image-cta { background:#f1f5fb; }
.image-cta .section-container { display:grid; min-height:450px; grid-template-columns:1fr 480px; align-items:center; }
.image-cta .section-container > div { padding:75px 80px 75px 0; }
.image-cta h2 { font-size:39px; font-weight:500; }
.image-cta p { color:#56637a; line-height:1.75; }
.image-cta img { width:100%; height:100%; max-height:450px; object-fit:cover; object-position:center top; }

.tier-section { padding:0 0 130px; }
.membership-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:24px; transform:translateY(-55px); }
.membership-grid > article,.corporate-grid > article { display:flex; min-height:680px; padding:38px 32px; flex-direction:column; border:1px solid #e8edf5; border-radius:6px; background:#fff; box-shadow:0 13px 35px rgba(23,42,80,.09); }
.membership-grid h2 { font-size:28px; font-weight:500; }
.price { display:flex; align-items:center; gap:12px; margin:4px 0 22px; color:var(--blue); }
.price strong { font-size:50px; font-weight:500; }
.price span,.price small { color:#758097; font-size:13px; line-height:1.35; }
.membership-grid article > p,.corporate-grid article > p { min-height:115px; color:#5d687d; line-height:1.65; }
.membership-grid h3,.corporate-grid h3 { margin:20px 0 14px; font-size:16px; }
.membership-grid ul,.corporate-grid ul { margin:0; padding:0; list-style:none; }
.membership-grid li,.corporate-grid li { display:flex; gap:10px; margin-bottom:10px; color:#46536b; font-size:14px; line-height:1.45; }
.membership-grid li svg,.corporate-grid li svg { width:16px; height:16px; flex:0 0 16px; color:var(--blue); }
.best { margin-top:auto; padding-top:20px; border-top:1px solid #edf0f5; }
.best p { color:#5d687d; font-size:14px; line-height:1.6; }
.membership-grid article > a,.corporate-grid article > a { display:flex; margin-top:20px; padding:13px 20px; justify-content:center; align-items:center; gap:9px; border:1px solid var(--blue); border-radius:3px; background:var(--blue); color:#fff; font-weight:600; }
.membership-grid a svg,.corporate-grid a svg { width:17px; }
.comparison-section,.segments-section { padding:110px 0; background:#f5f7fb; }
.table-scroll { overflow-x:auto; border-radius:4px; box-shadow:0 8px 24px rgba(22,42,83,.07); }
table { width:100%; min-width:950px; border-collapse:collapse; background:#fff; }
th,td { padding:20px 18px; border:1px solid #e5e9f0; text-align:left; font-size:14px; line-height:1.45; }
thead th { background:#132953; color:#fff; font-weight:500; }
thead small { display:block; margin-top:7px; color:#b9c6df; }
tbody th { color:#182541; font-weight:500; }
tbody td { color:#5b667a; }
.flex-section { padding:125px 0; }
.flex-layout { display:grid; grid-template-columns:1fr 1fr; gap:110px; align-items:center; }
.flex-layout > div:first-child h2 { font-size:42px; font-weight:500; }
.flex-layout > div:first-child p { color:#58657b; line-height:1.75; }
.flex-stats { display:grid; grid-template-columns:1fr 1fr; }
.flex-stats article { min-height:175px; padding:28px; border-right:1px solid #dde3ed; border-bottom:1px solid #dde3ed; }
.flex-stats article:nth-child(2) { border-right:0; }
.flex-stats article:nth-child(3) { grid-column:2; border-right:0; border-bottom:0; }
.flex-stats strong { color:var(--blue); font-size:38px; }
.flex-stats h3 { margin:9px 0; font-size:16px; font-weight:500; }
.flex-stats p { color:#6d778a; font-size:13px; line-height:1.55; }
.member-cta .section-container { min-height:490px; }

.corporate-tiers { padding:120px 0; }
.tier-head { display:flex; margin-bottom:45px; justify-content:space-between; align-items:flex-end; }
.tier-head h2 { margin:0; font-size:40px; font-weight:500; }
.tier-head button { width:48px; height:48px; margin-left:8px; border:1px solid #dbe2ee; border-radius:50%; background:#fff; color:var(--navy); cursor:pointer; }
.tier-head svg { width:19px; }
.corporate-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:24px; overflow:hidden; }
.corporate-grid > article { min-height:760px; }
.corporate-grid > article > span { color:var(--blue); font-size:25px; font-weight:600; }
.corporate-grid .price strong { font-size:43px; }
.corporate-grid article > p { min-height:125px; }
.carousel-dots { display:flex; justify-content:center; gap:8px; margin-top:32px; }
.carousel-dots button { width:9px; height:9px; padding:0; border:0; border-radius:50%; background:#c7cfde; cursor:pointer; }
.carousel-dots button.active { width:28px; border-radius:8px; background:var(--blue); }
.segments-section table { min-width:1200px; }
.faq-section { padding:120px 0; }
.faq-list { max-width:900px; margin:0 auto; border-top:1px solid #dde3ec; }
.faq-list article { border-bottom:1px solid #dde3ec; }
.faq-list button { display:flex; width:100%; padding:24px 0; justify-content:space-between; align-items:center; border:0; background:transparent; color:var(--ink); font-size:18px; text-align:left; cursor:pointer; }
.faq-list button svg { width:19px; transition:transform .25s ease; }
.faq-list article.open button svg { transform:rotate(180deg); }
.faq-list article > div p { padding:0 40px 25px 0; color:#617087; line-height:1.75; }

/* Official GFI page geometry */
.programme-page,.partnership-hero,.membership-hero,.corporate-hero { position:relative; overflow:hidden; background:#fff; }
.programme-intro,.programme-list,.partnership-hero > .section-container,.membership-hero > .section-container,.corporate-hero > .section-container { position:relative; z-index:1; }
.programme-intro { max-width:1040px; padding:112px 0 92px; }
.programme-intro h1 { margin-bottom:34px; font-size:48px; }
.programme-intro p { max-width:850px; margin-bottom:20px; font-size:17px; line-height:1.72; }
.programme-list { padding:0 0 120px; }
.programme-list header { margin-bottom:68px; }
.programme-list h2 { font-size:38px; font-weight:600; }
.programme-card { width:416px; border:1px solid #e2e6ee; border-radius:7px; box-shadow:none; }
.programme-card > img { height:288px; }
.programme-card > div { padding:34px 32px 38px; }
.programme-card h3 { display:-webkit-box; min-height:70px; margin-bottom:9px; overflow:hidden; font-size:25px; font-weight:600; line-height:1.35; -webkit-box-orient:vertical; -webkit-line-clamp:2; }
.programme-card > div > i { display:block; width:40px; height:2px; margin:0 0 20px; background:#42b7e9; }
.programme-card p { display:-webkit-box; min-height:74px; margin-bottom:28px; overflow:hidden; font-size:15px; line-height:1.65; -webkit-box-orient:vertical; -webkit-line-clamp:3; }
.programme-card a { width:max-content; padding-bottom:8px; border-bottom:1px solid #2464f5; font-weight:500; }

.partnership-hero { min-height:555px; padding:105px 0 145px; }
.partnership-hero h1 { margin-bottom:34px; font-size:43px; }
.partnership-hero p { max-width:760px; margin:0 auto; font-size:16px; line-height:1.7; }
.hero-actions { margin-top:32px; }
.hero-actions a { padding:12px 22px; border-radius:24px; font-size:14px; }
.partner-directory { padding:130px 0 120px; }
.partner-directory > .section-heading { margin-bottom:100px; }
.section-heading h2 { font-size:35px; font-weight:600; }
.patron-block { margin-bottom:105px; }
.patron-block h3,.partner-group > header h3,.media-block h3 { font-size:29px; font-weight:600; }
.patron-block > p { font-size:14px; }
.patron-card { width:320px; min-height:0; padding:0; gap:0; align-items:stretch; justify-content:flex-start; border-radius:8px; box-shadow:0 8px 18px rgba(24,42,72,.09); }
.patron-card .logo-frame { height:220px; }
.patron-card .partner-copy { min-height:78px; text-align:left; }
.patron-card-rich .partner-copy { min-height:180px; }
.patron-card:hover { transform:none; box-shadow:0 8px 18px rgba(24,42,72,.09); }
.partner-group { margin:115px 0 135px; }
.partner-group > header { margin-bottom:45px; }
.partner-group > header h3 { margin-bottom:16px; }
.partner-grid { max-width:1120px; margin:0 auto; gap:60px 72px; align-items:flex-start; }
.partner-grid > a { width:280px; min-height:0; overflow:visible; border:0; border-radius:8px; box-shadow:none; }
.partner-grid > a:hover { transform:none; box-shadow:none; }
.partner-grid .logo-frame { height:200px; border:1px solid #edf0f5; border-radius:8px; box-shadow:0 9px 17px rgba(24,42,72,.1); }
.partner-grid .partner-copy { padding:20px 0 0; }
.rich-partner-copy { display:flex; flex-direction:column; align-items:flex-start; gap:8px; color:#566176; font-size:13px; line-height:1.5; }
.rich-partner-copy u { width:100%; overflow-wrap:anywhere; color:#17233f; text-decoration:underline; }
.rich-partner-copy ul { margin:4px 0; padding-left:27px; color:#17233f; line-height:1.85; }
.rich-partner-copy > strong { color:#17233f; }
.compact-grid > a { min-height:0; }
.subgroup-title { margin:55px 0 34px; }
.media-block { margin-top:135px; }

.membership-hero { min-height:590px; padding:85px 0 280px; }
.membership-hero h1 { margin-bottom:28px; font-size:51px; font-weight:600; }
.membership-hero p { max-width:920px; margin:0 auto; font-size:16px; line-height:1.65; }
.tier-section { padding:0 0 82px; }
.membership-grid { align-items:start; gap:42px; transform:translateY(-245px); margin-bottom:-245px; }
.membership-grid > article,.corporate-grid > article { min-height:0; padding:34px 32px 38px; border:0; border-radius:0; box-shadow:0 10px 28px rgba(26,43,74,.06); }
.membership-grid h2,.corporate-grid h2 { margin-bottom:14px; font-size:22px; font-weight:500; }
.membership-grid .price,.corporate-grid .price { gap:7px; margin:0 0 24px; }
.membership-grid .price strong,.corporate-grid .price strong { font-size:43px; font-weight:600; }
.price b { align-self:flex-end; padding-bottom:7px; color:var(--blue); font-size:14px; }
.price > i { width:1px; height:29px; margin:0 12px; transform:skew(-20deg); background:#bcd0ff; }
.price > span { align-self:flex-end; padding-bottom:7px; color:var(--blue); font-size:14px; }
.membership-grid article > p,.corporate-grid article > p { min-height:0; margin-bottom:22px; color:#4f5b70; font-size:14px; line-height:1.55; }
.membership-grid h3,.corporate-grid h3 { margin:0; padding-bottom:13px; border-bottom:1px dotted #b9c8ea; color:#2364f5; font-size:14px; font-weight:500; }
.membership-grid ul,.corporate-grid ul { padding:17px 0 14px; border-bottom:1px dotted #b9c8ea; }
.membership-grid li,.corporate-grid li { gap:12px; margin-bottom:13px; font-size:13px; }
.membership-grid li > b,.corporate-grid li > b { color:#2767ff; }
.best { margin:22px 0 0; padding:14px 16px; border:0; border-radius:4px; background:#f7f9fd; }
.best h3 { padding:0; border:0; font-size:12px; }
.best p { margin:7px 0 0; font-size:12px; line-height:1.5; }
.membership-grid article > a,.corporate-grid article > a,.flex-section a,.member-cta a.cta-button { width:max-content; margin-top:22px; padding:0; gap:0; border:0; background:transparent; color:#2364f5; }
.membership-grid article > a > span,.corporate-grid article > a > span,.flex-section a > span,.member-cta a.cta-button > span { display:flex; min-width:105px; height:42px; padding:0 20px; align-items:center; justify-content:center; border:1px solid #2364f5; border-radius:24px; }
.membership-grid article > a > i,.corporate-grid article > a > i,.flex-section a > i,.member-cta a.cta-button > i { display:flex; width:44px; height:44px; margin-left:4px; align-items:center; justify-content:center; border-radius:50%; background:#2364f5; color:#fff; }
.comparison-section { padding:78px 0 92px; }
.comparison-section .section-heading { margin-bottom:44px; }
.comparison-section table { box-shadow:0 8px 22px rgba(27,43,72,.12); }
.comparison-section th,.comparison-section td { padding:15px 20px; border:0; }
.comparison-section tbody tr:nth-child(even) { background:#f7f8fa; }
.comparison-section tbody tr:nth-child(odd) { background:#fff; }
.flex-section { min-height:690px; padding:125px 0; }
.flex-layout { grid-template-columns:1.08fr .92fr; gap:120px; }
.flex-layout > div:first-child h2 { font-size:36px; font-weight:500; }
.flex-layout > div:first-child p { max-width:630px; font-size:14px; line-height:1.55; }
.flex-stats article { min-height:170px; padding:28px 20px; }
.flex-stats strong { font-size:44px; font-weight:600; }
.flex-stats h3 { margin:7px 0; font-size:15px; }
.flex-stats p { font-size:12px; }
.member-cta,.corporate-cta { overflow:hidden; background:#f4f7fb; }
.member-cta .section-container,.corporate-cta .section-container { min-height:570px; grid-template-columns:1fr 550px; }
.member-cta .section-container > div:first-child,.corporate-cta .section-container > div:first-child { padding:95px 50px 95px 0; }
.member-art { position:relative; align-self:stretch; }
.member-art { padding:0 !important; }
.member-art > i { position:absolute; right:-75px; bottom:-210px; width:620px; height:620px; border:55px solid #326dff; border-radius:50%; }
.member-art img { position:absolute; right:-35px; bottom:0; z-index:1; width:520px; height:auto; max-height:none; object-fit:contain; }

.corporate-hero { min-height:445px; padding:70px 0 100px; }
.corporate-hero h1 { margin-bottom:34px; font-size:50px; font-weight:600; }
.corporate-hero p { max-width:900px; margin:0 auto 26px; font-size:16px; line-height:1.55; }
.corporate-tiers { position:relative; padding:0 0 100px; }
.corporate-grid { align-items:start; gap:28px; overflow:hidden; }
.corporate-grid > article { min-height:690px; padding:26px 24px 30px; box-shadow:0 7px 20px rgba(26,43,74,.04); }
.corporate-grid > article > h2 { font-size:20px; }
.corporate-grid .price strong { font-size:39px; }
.corporate-grid article > p { min-height:76px; font-size:12px; }
.corporate-grid li { font-size:12px; line-height:1.45; }
.edge-arrow { position:absolute; top:315px; z-index:2; display:flex; width:45px; height:45px; align-items:center; justify-content:center; border:0; border-radius:50%; background:#fff; box-shadow:0 5px 18px rgba(24,42,72,.12); color:#627087; cursor:pointer; }
.edge-arrow svg { width:17px; }
.edge-arrow-left { left:-28px; }
.edge-arrow-right { right:-28px; }
.carousel-dots { margin-top:55px; }
.carousel-dots button { width:10px; height:10px; }
.carousel-dots button.active { width:10px; border-radius:50%; }
.segments-section { padding:85px 0 100px; }
.segments-section .section-heading { margin-bottom:50px; }
.segments-section table { box-shadow:0 8px 22px rgba(27,43,72,.12); }
.segments-section th,.segments-section td { padding:15px 17px; border:0; font-size:12px; }
.segments-section tbody tr:nth-child(even) { background:#f7f8fa; }
.faq-section { min-height:760px; padding:115px 0 135px; }
.faq-section .section-heading { margin-bottom:65px; }
.faq-list { max-width:1020px; }
.faq-list button { padding:25px 0; color:#6b7588; font-size:15px; }
.faq-list button svg { width:15px; transform:none !important; }
.faq-list article > div p { padding:0 35px 30px 18px; white-space:pre-line; color:#677287; font-size:13px; line-height:1.65; }
.corporate-cta .email-link { display:inline; color:#2364f5; font-weight:400; }

@media (max-width:980px) {
  .section-container { width:min(100% - 36px,720px); }
  .programme-intro { padding:90px 0 110px; }
  .programme-intro h1,.partnership-hero h1,.membership-hero h1,.corporate-hero h1 { font-size:38px; }
  .membership-grid,.corporate-grid { grid-template-columns:1fr; }
  .membership-grid { transform:none; padding-top:65px; }
  .corporate-grid > article:not(:first-child) { display:none; }
  .image-cta .section-container { grid-template-columns:1fr 330px; }
  .image-cta .section-container > div { padding-right:35px; }
  .flex-layout { grid-template-columns:1fr; gap:60px; }
  .membership-grid { margin-bottom:0; transform:none; }
  .membership-hero { min-height:auto; padding-bottom:90px; }
  .corporate-hero { min-height:auto; }
  .corporate-tiers { padding-top:70px; }
  .edge-arrow { top:390px; }
  .edge-arrow-left { left:-12px; }
  .edge-arrow-right { right:-12px; }
  .member-cta .section-container,.corporate-cta .section-container { grid-template-columns:1fr 360px; }
}
@media (max-width:640px) {
  .section-container { width:calc(100% - 32px); }
  .programme-intro,.partnership-hero,.membership-hero,.corporate-hero { padding:75px 0 95px; }
  .programme-intro h1,.partnership-hero h1,.membership-hero h1,.corporate-hero h1 { font-size:32px; }
  .programme-intro p,.partnership-hero p,.membership-hero p,.corporate-hero p { font-size:15px; }
  .programme-card { width:100%; }
  .programme-list { padding-bottom:90px; }
  .hero-actions { flex-direction:column; }
  .hero-actions a { justify-content:center; }
  .partner-directory { padding:85px 0; }
  .partner-grid > a { width:100%; }
  .image-cta .section-container { display:block; }
  .image-cta .section-container > div { padding:65px 0 40px; }
  .image-cta h2,.section-heading h2,.tier-head h2,.flex-layout > div:first-child h2 { font-size:30px; }
  .image-cta img { height:330px; }
  .tier-head { align-items:center; }
  .tier-head { display:block; }
  .tier-head > div:last-child { margin-top:22px; }
  .tier-head .eyebrow { display:none; }
  .membership-grid > article,.corporate-grid > article { padding:30px 24px; }
  .comparison-section,.segments-section,.flex-section,.faq-section,.corporate-tiers { padding:80px 0; }
  .flex-stats article { padding:18px; }
  .programme-intro { padding:75px 0 70px; }
  .programme-list header { margin-bottom:45px; }
  .partnership-hero { min-height:auto; padding:75px 0 95px; }
  .partner-grid { gap:50px; }
  .membership-grid { padding-top:0; }
  .membership-hero { padding:70px 0 85px; }
  .membership-hero h1 { font-size:34px; }
  .corporate-hero { padding:70px 0 90px; }
  .corporate-hero h1 { font-size:34px; }
  .corporate-hero p { font-size:14px; }
  .edge-arrow { top:430px; }
  .edge-arrow-left { left:-6px; }
  .edge-arrow-right { right:-6px; }
  .table-scroll {
    --sticky-column-width:clamp(116px,36vw,142px);
    position:relative;
    overflow-x:auto;
    overscroll-behavior-inline:contain;
    scrollbar-color:#9eacc8 #e9edf5;
    scrollbar-width:thin;
    box-shadow:inset -18px 0 18px -18px rgba(19,41,83,.42),0 8px 24px rgba(22,42,83,.07);
    -webkit-overflow-scrolling:touch;
  }
  .table-scroll::-webkit-scrollbar { height:6px; }
  .table-scroll::-webkit-scrollbar-track { background:#e9edf5; }
  .table-scroll::-webkit-scrollbar-thumb { border-radius:999px; background:#9eacc8; }
  .comparison-section table { min-width:760px; table-layout:fixed; }
  .segments-section table { min-width:1040px; table-layout:fixed; }
  .comparison-section th,.comparison-section td { padding:14px 12px; font-size:14px; }
  .segments-section th,.segments-section td { padding:14px 12px; font-size:14px; }
  .table-scroll thead th { vertical-align:top; }
  .table-scroll thead small { font-size:12px; }
  .table-scroll thead th:first-child,
  .table-scroll tbody th:first-child {
    position:sticky;
    left:0;
    width:var(--sticky-column-width);
    min-width:var(--sticky-column-width);
    max-width:var(--sticky-column-width);
    box-shadow:8px 0 14px -12px rgba(19,41,83,.72);
  }
  .table-scroll thead th:first-child { z-index:3; background:#132953; }
  .table-scroll tbody th:first-child { z-index:2; background:#fff; }
  .comparison-section tbody tr:nth-child(even) th:first-child,
  .segments-section tbody tr:nth-child(even) th:first-child { background:#f7f8fa; }
  .member-cta .section-container,.corporate-cta .section-container { display:block; }
  .member-cta .section-container > div:first-child,.corporate-cta .section-container > div:first-child { padding:70px 0 30px; }
  .member-art { min-height:390px; }
  .member-art > i { right:-145px; bottom:-240px; width:520px; height:520px; }
  .member-art img { right:-40px; width:390px; }
}
@media (prefers-reduced-motion:reduce) { [data-reveal],[data-reveal].is-revealed { opacity:1; transform:none; animation:none !important; transition:none; } }
</style>
