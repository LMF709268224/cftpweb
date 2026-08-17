<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import {
  ArrowLeft,
  ArrowRight,
  ArrowUpRight,
  Linkedin,
  Play,
  X,
} from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import { useTranslation } from "@/lib/language"

const homeAsset = (name: string) => `/gfi/home/${name}`

const zhHeroSlides = [
  {
    image: homeAsset("hero-1.webp"),
    title: "访问GFI学习门户",
    description: "登录您的个性化中心以获取所有必要的资源和课程。访问学习指南、跟踪您的进度并注册考试。",
    cta: "访问门户",
    to: "https://member.globalfintechinstitute.org/home.php",
    external: true,
  },
  {
    image: homeAsset("hero-2.webp"),
    title: "开启您的金融科技之旅",
    description: "无论您是金融科技新手还是经验丰富的金融专业人士，我们的项目都为您打开通向行业认可知识和职业机会的大门。",
    cta: "了解更多",
    to: "/gfi/programmes/cfta",
    external: false,
  },
  {
    image: homeAsset("hero-3.webp"),
    title: "通过特许金融科技专业人士(CFtP®)提升技能",
    description: "专为职业中期专业人士设计，CFtP®为您提供实用技能、实际应用，并让您接触全球金融科技领袖网络。立即加速您的职业发展。",
    cta: "立即报名",
    to: "/gfi/programmes/cftp",
    external: false,
  },
]

const enHeroSlides = [
  {
    image: homeAsset("hero-1.webp"),
    title: "Access GFI Learning Portal",
    description: "Log in to your personalised hub for essential resources and courses. Access study guides, track your progress, and register for examinations.",
    cta: "Access Portal",
    to: "https://member.globalfintechinstitute.org/home.php",
    external: true,
  },
  {
    image: homeAsset("hero-2.webp"),
    title: "Start Your Fintech Journey with GFI",
    description: "Whether you're new to fintech or a seasoned financial professional, our programmes open doors to industry-recognised knowledge and career opportunities.",
    cta: "Learn More",
    to: "/gfi/programmes/cfta",
    external: false,
  },
  {
    image: homeAsset("hero-3.webp"),
    title: "Upskill with the Chartered Fintech Professional (CFtP®)",
    description: "Designed for mid-career professionals, the CFtP® equips you with practical skills, real-world applications, and access to a global network of fintech leaders.",
    cta: "Enrol Now",
    to: "/gfi/programmes/cftp",
    external: false,
  },
]

const partners = [
  { name: "币安", image: homeAsset("partner-binance.webp") },
  { name: "Hashkey Group", image: homeAsset("partner-hashkey.webp") },
  { name: "FTI", image: homeAsset("partner-fti.webp") },
  { name: "新加坡国立大学", image: homeAsset("partner-nus.svg") },
  { name: "南洋理工大学", image: homeAsset("partner-ntu.webp") },
  { name: "新加坡管理大学", image: homeAsset("partner-smu.webp") },
  { name: "新加坡新跃社科大学", image: homeAsset("partner-suss.webp") },
]

const zhStrengths = [
  {
    number: "01",
    title: "行业验证的认证",
    text: "我们的特许金融科技助理(CFtA)和特许金融科技专业人士(CFtP®)认证是与100多位行业领袖和学术合作伙伴共同构建的。这确保每个模块都反映真实世界的实践、新兴技术和全球监管趋势——让您保持领先地位。",
    image: homeAsset("strength-1.webp"),
  },
  {
    number: "02",
    title: "金融科技教育的可信认证",
    text: "作为领先的认证机构，GFI为金融科技教育设定了基准。我们不仅认证专业人士，还为符合行业标准的合作伙伴项目提供认可，为监管机构、雇主和机构创建可信的质量印章。",
    image: homeAsset("strength-2.webp"),
  },
  {
    number: "03",
    title: "专业发展的清晰路径",
    text: "从新手到资深专业人士，GFI提供结构化的学习之旅：CFtA构建基础知识，而CFtP®为职业中期领导者提供先进的实用技能。这条路径使专业人士能够在每个职业阶段自信和有信誉地成长。",
    image: homeAsset("strength-3.webp"),
  },
]

const enStrengths = [
  {
    number: "01",
    title: "Industry-Validated Certifications",
    text: "Our Chartered Fintech Associate (CFtA) and Chartered Fintech Professional (CFtP®) certifications are built with over 100 industry leaders and academic partners, ensuring every module reflects real-world practice, emerging technology, and global regulatory trends.",
    image: homeAsset("strength-1.webp"),
  },
  {
    number: "02",
    title: "Trusted Accreditation for Fintech Education",
    text: "As a leading accreditation body, GFI sets the benchmark for fintech education. We certify professionals and recognise partner programmes that meet industry standards, creating a trusted seal of quality for regulators, employers, and institutions.",
    image: homeAsset("strength-2.webp"),
  },
  {
    number: "03",
    title: "Clear Pathway for Professional Growth",
    text: "From newcomers to seasoned professionals, GFI offers a structured learning journey: CFtA builds foundational knowledge while CFtP® equips mid-career leaders with advanced, practical skills.",
    image: homeAsset("strength-3.webp"),
  },
]

const zhStories = [
  {
    name: "Jag Foo、CFtP、CPP、PSP、PCI",
    date: "October 1, 2025",
    path: "/gfi/gfi-stories/jag-foo-cftp-cpp-psp-pci",
    image: homeAsset("story-jag.webp"),
  },
  {
    name: "达妍叶，CFtP",
    date: "October 1, 2025",
    path: "/gfi/gfi-stories/tat-yeen-yap-cftp",
    image: homeAsset("story-tat.webp"),
  },
  {
    name: "丁亚伦，CFtP",
    date: "October 1, 2025",
    path: "/gfi/gfi-stories/aaron-ting-cftp",
    image: homeAsset("story-aaron.webp"),
  },
]

const enStories = [
  { ...zhStories[0], name: "Jag Foo, CFtP, CPP, PSP, PCI", date: "October 1, 2025" },
  { ...zhStories[1], name: "Tat Yeen Yap, CFtP", date: "October 1, 2025" },
  { ...zhStories[2], name: "Aaron Ting, CFtP", date: "October 1, 2025" },
]

const zhTestimonials = [
  {
    quote: "GFI通过结构化学习、全球参与和一流培训促进金融科技领域的协作、道德和知识——赋能专业人士推动行业转型和面向未来的金融科技领导力。",
    name: "Nizam Ismail",
    role: "CEO & Founder",
    companyName: "Ethikom Consultancy",
    avatar: homeAsset("avatar-nizam.webp"),
    company: homeAsset("company-ethikom.webp"),
  },
  {
    quote: "CFtP®提供了关于AI、区块链和金融科技基础设施的结构化见解，将学术严谨性与全球最佳实践相结合——为我提供了跨境贸易和监管协调的可操作工具。",
    name: "Gary Loh",
    role: "Founder",
    companyName: "DiMuto",
    avatar: homeAsset("avatar-gary.webp"),
    company: homeAsset("company-dimuto.webp"),
  },
  {
    quote: "CFtP®使我能够通过桥接金融科技和公共政策为数字基础设施做出贡献——帮助政府通过道德创新和有意义的生态系统协作建立坚实的基础。",
    name: "Aaron Ting",
    role: "Co-Founder",
    companyName: "ICP Hub Singapore",
    avatar: homeAsset("avatar-aaron.webp"),
    company: homeAsset("company-icp.webp"),
  },
  {
    quote: "CFtP®提供结构化、行业相关的学习，具有强大的道德基础。社区促进致力于现实世界变革的金融科技领袖之间的有影响力的协作和知识共享。",
    name: "Tat Yeen Yap",
    role: "Head of Supply Chain Solutions",
    companyName: "Maybank",
    avatar: homeAsset("avatar-tat.webp"),
    company: homeAsset("company-maybank.webp"),
  },
  {
    quote: "通过CFtP®，GFI使专业人士能够保持相关性并应对新兴技术对金融系统和现实世界应用的影响。该计划通过深入、实用的知识支持终身金融科技学习。",
    name: "Dr. Ernie Teo",
    role: "Program Director",
    companyName: "NTU Singapore",
    avatar: homeAsset("avatar-ernie.webp"),
    company: homeAsset("company-ntu.webp"),
  },
  {
    quote: "众所周知，CFtP®在治理、道德和合规方面提供必要的培训。这为金融科技专业人士在Web3和传统金融融合的不断发展的环境中建立了坚实的基础。",
    name: "Jag Foo",
    role: "Chairman",
    companyName: "Blockchain Security & Compliance Committee",
    avatar: homeAsset("avatar-jag.webp"),
    company: "",
  },
]

const enTestimonials = [
  { ...zhTestimonials[0], quote: "GFI fosters collaboration, ethics, and knowledge across fintech through structured learning, global engagement, and best-in-class training—empowering professionals to drive industry transformation and future-ready fintech leadership." },
  { ...zhTestimonials[1], quote: "The CFtP® offered structured insights into AI, blockchain, and fintech infrastructure, combining academic rigour with global best practices and equipping me with actionable tools for cross-border trade and regulatory alignment." },
  { ...zhTestimonials[2], quote: "The CFtP® empowered me to contribute to digital infrastructure by bridging fintech and public policy, helping governments build solid foundations through ethical innovation and meaningful ecosystem collaboration." },
  { ...zhTestimonials[3], quote: "CFtP® offers structured, industry-relevant learning with strong ethical grounding. The community fosters impactful collaboration and knowledge-sharing among fintech leaders committed to real-world change." },
  { ...zhTestimonials[4], quote: "Through the CFtP®, GFI empowers professionals to stay relevant and respond to the impact of emerging technologies on financial systems and real-world applications." },
  { ...zhTestimonials[5], quote: "The CFtP® provides essential training in governance, ethics, and compliance, building a strong foundation for fintech professionals as Web3 and traditional finance converge." },
]

const zhNews = [
  {
    date: "July 13, 2026",
    path: "/gfi/news/global-fintech-institute-and-solusfutura-sign-mou-to-bring-chartered-fintech-professional-certification-to-hong-kong-and-the-greater-bay-area",
    title: "Global Fintech Institute and SolusFutura Sign MOU to Bring Chartered Fintech Professional Certification to Hong Kong and the Greater Bay Area",
    description: "SINGAPORE, 13 July 2026. Fintech capital and technology move across borders in an instant, but professional standards travel only as fast as the institutions that carry them. This week, the Global Fintech Institute takes a step toward carrying them further.",
    image: homeAsset("news-1.webp"),
  },
  {
    date: "July 10, 2026",
    path: "/gfi/news/cfta-is-now-live-enhanced-curriculum-for-a-fast-moving-fintech-industry",
    title: "CFtA is Now Live: Enhanced Curriculum for a Fast-Moving Fintech Industry",
    description: "10 July 2026，Singapore — The Global Fintech Institute is pleased to announce that the Chartered Fintech Associate (CFtA) is now live with an enhanced curriculum, broader partner ecosystem, and a continued commitment to credible fintech education.",
    image: homeAsset("news-2.webp"),
  },
  {
    date: "July 3, 2026",
    path: "/gfi/news/introducing-the-digital-assets-security-and-compliance-subcommittee",
    title: "Introducing the Digital Assets Security and Compliance Subcommittee",
    description: "The Global Fintech Institute is convening the Digital Assets Security and Compliance Subcommittee to advance institutional thinking on cybersecurity, compliance, and operational resilience within digital asset ecosystems.",
    image: homeAsset("news-3.webp"),
  },
]

const enNews = [
  { ...zhNews[0], date: "July 13, 2026", title: "Global Fintech Institute and SolusFutura Sign MOU to Bring Chartered Fintech Professional Certification to Hong Kong and the Greater Bay Area" },
  { ...zhNews[1], date: "July 10, 2026", title: "CFtA is Now Live: Enhanced Curriculum for a Fast-Moving Fintech Industry" },
  { ...zhNews[2], date: "July 3, 2026", title: "Introducing the Digital Assets Security and Compliance Subcommittee" },
]

const { lang } = useTranslation()
const heroSlides = computed(() => (lang.value === "zh" ? zhHeroSlides : enHeroSlides))
const strengths = computed(() => (lang.value === "zh" ? zhStrengths : enStrengths))
const stories = computed(() => (lang.value === "zh" ? zhStories : enStories))
const testimonials = computed(() => (lang.value === "zh" ? zhTestimonials : enTestimonials))
const news = computed(() => (lang.value === "zh" ? zhNews : enNews))
const copy = computed(() => lang.value === "zh" ? {
  heroNote: "在全球金融科技学院，我们通过让专业人士直接向塑造金融科技格局的资深从业者学习来帮助他们推进职业生涯。通过我们的认证、研究和全球社区，您将获得在快速发展的行业中保持领先所需的技能和关系。",
  partnerIntro: "我们很自豪能与这些杰出的合作伙伴一起工作。",
  aboutTitle: "通过全球金融科技学院推动您的职业生涯",
  aboutLearn: "了解更多关于我们",
  stats: ["加入全球专业人士、导师和行业领袖的社区——准备为您开启下一个职业里程碑的大门。", "从数十年的综合专业知识中受益，构建的课程旨在帮助您在快速变化的世界中自信地领导。", "提升您的就业能力。GFI认证的专业人士被公认为面向未来的领导者，为高管机会和职业发展做好准备。"],
  certificationEyebrow: "行业认可的认证",
  programmes: "我们的项目",
  programmeTitle: "加密货币监管与合规基础",
  programmeText: "由全球金融科技学院(GFI)与币安合作提供的严格16小时短期课程，旨在为专业人士提供数字资产监管、区块链基础设施和新兴合规框架的深入理解。",
  learnMore: "了解更多",
  whyEyebrow: "为什么选择GFI",
  whyTitle: "从基础到领导力：我们塑造金融科技职业生涯",
  storiesEyebrow: "鼓舞人心的校友历程",
  storiesTitle: "GFI 成功故事",
  allStories: "查看所有故事",
  storyTag: "story",
  readStory: "阅读故事",
  testimonials: "客户评价",
  alumniSay: "我们的校友对GFI的评价",
  createAccount: "创建账户",
  reviewFrom: "Review From",
  news: "新闻",
  latestNews: "获取GFI的最新新闻和更新",
  allNews: "所有新闻",
  readMore: "Read More",
  newsTag: "news",
} : {
  heroNote: "At the Global Fintech Institute, professionals learn directly from established practitioners shaping the fintech landscape, gaining the skills and connections to stay ahead.",
  partnerIntro: "We are proud to work alongside these distinguished partners.",
  aboutTitle: "Power Your Career with Global Fintech Institute",
  aboutLearn: "Learn More About Us",
  stats: ["Step into a global community of professionals, mentors, and industry leaders, ready to open doors to your next career milestone.", "Benefit from decades of combined expertise, built into a curriculum designed for confident leadership in a fast-changing world.", "Boost your employability. GFI-certified professionals are recognised as future-ready leaders."],
  certificationEyebrow: "Industry-Recognised Certifications",
  programmes: "Our Programmes",
  programmeTitle: "Foundation in Crypto Regulation and Compliance",
  programmeText: "A rigorous 16-hour short course co-delivered by GFI and Binance, designed to build a deep understanding of digital asset regulation, blockchain infrastructure, and emerging compliance frameworks.",
  learnMore: "Learn More",
  whyEyebrow: "Why Choose GFI",
  whyTitle: "From Foundations to Leadership: We Shape Fintech Careers",
  storiesEyebrow: "Inspiring Alumni Journeys",
  storiesTitle: "Our Charterholders",
  allStories: "View All Stories",
  storyTag: "story",
  readStory: "Read Story",
  testimonials: "Testimonials",
  alumniSay: "What Our Alumni Say About GFI",
  createAccount: "Create Account",
  reviewFrom: "Review from LinkedIn",
  news: "News",
  latestNews: "Access the Latest News and Updates from GFI",
  allNews: "All News",
  newsTag: "news",
  readMore: "Read More",
})

const currentSlide = ref(0)
const videoOpen = ref(false)
const statsElement = ref<HTMLElement | null>(null)
const statValues = ref([0, 0, 0])
let heroTimer: ReturnType<typeof setInterval> | undefined
let revealObserver: IntersectionObserver | undefined
let statsObserver: IntersectionObserver | undefined

const activeHero = computed(() => heroSlides.value[currentSlide.value])
const marqueePartners = computed(() => [...partners, ...partners])
const marqueeTestimonials = computed(() => [...testimonials.value, ...testimonials.value])

function setSlide(index: number) {
  currentSlide.value = (index + heroSlides.value.length) % heroSlides.value.length
}

function selectSlide(index: number) {
  if (heroTimer) clearInterval(heroTimer)
  heroTimer = undefined
  setSlide(index)
}

function animateStats() {
  const targets = [500, 20, 10]
  const start = performance.now()
  const duration = 1500
  const tick = (now: number) => {
    const progress = Math.min((now - start) / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    statValues.value = targets.map((target) => Math.round(target * eased))
    if (progress < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") videoOpen.value = false
}

onMounted(() => {
  heroTimer = setInterval(() => setSlide(currentSlide.value + 1), 4000)
  window.addEventListener("keydown", handleKeydown)
  revealObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (!entry.isIntersecting) return
      entry.target.classList.add("is-revealed")
      revealObserver?.unobserve(entry.target)
    })
  }, { threshold: 0.08, rootMargin: "0px 0px -40px" })
  document.querySelectorAll(".gfi-page [data-reveal]").forEach((element) => revealObserver?.observe(element))
  statsObserver = new IntersectionObserver(([entry]) => {
    if (!entry?.isIntersecting) return
    animateStats()
    statsObserver?.disconnect()
  }, { threshold: 0.3 })
  if (statsElement.value) statsObserver.observe(statsElement.value)
})

onBeforeUnmount(() => {
  if (heroTimer) clearInterval(heroTimer)
  revealObserver?.disconnect()
  statsObserver?.disconnect()
  window.removeEventListener("keydown", handleKeydown)
})
</script>

<template>
  <div class="gfi-page">
    <GfiHeader />

    <main id="top">
      <section class="hero">
        <div :key="`media-${currentSlide}`" class="hero-media"><img :src="activeHero.image" alt="Banner Image"></div>
        <div class="hero-shade" />
        <div class="container hero-layout">
          <div class="hero-copy" :key="currentSlide">
            <h1>{{ activeHero.title }}</h1>
            <p>{{ activeHero.description }}</p>
            <a v-if="activeHero.external" class="outline-action" :href="activeHero.to" target="_blank" rel="noopener noreferrer">
              <span>{{ activeHero.cta }}</span>
              <span class="action-icon"><ArrowUpRight /></span>
            </a>
            <RouterLink v-else class="outline-action" :to="activeHero.to">
              <span>{{ activeHero.cta }}</span>
              <span class="action-icon"><ArrowUpRight /></span>
            </RouterLink>
          </div>

          <div class="hero-note">
            <button :aria-label="lang === 'zh' ? '播放介绍视频' : 'Play introduction video'" :title="lang === 'zh' ? '播放介绍视频' : 'Play introduction video'" :aria-expanded="videoOpen" @click="videoOpen = true"><Play /></button>
            <p>{{ copy.heroNote }}</p>
          </div>

          <div class="hero-controls">
            <button aria-label="上一张" title="上一张" @click="selectSlide(currentSlide - 1)"><ArrowLeft /></button>
            <span />
            <button aria-label="下一张" title="下一张" @click="selectSlide(currentSlide + 1)"><ArrowRight /></button>
          </div>
        </div>
      </section>

      <section class="partner-strip" aria-label="合作伙伴" data-reveal>
        <div class="container partner-layout">
          <p>{{ copy.partnerIntro }}</p>
          <div class="partner-window">
            <div class="partner-track">
              <RouterLink v-for="(partner, index) in marqueePartners" :key="`${partner.name}-${index}`" to="/gfi/partnerships" :aria-hidden="index >= partners.length" :tabindex="index >= partners.length ? -1 : undefined">
                <img :src="partner.image" :alt="partner.name" />
              </RouterLink>
            </div>
          </div>
        </div>
      </section>

      <section id="about" class="about-section">
        <div class="container">
          <div class="section-heading about-heading" data-reveal>
            <h2>{{ copy.aboutTitle }}</h2>
            <RouterLink class="text-action" to="/gfi/about">{{ copy.aboutLearn }} <ArrowUpRight /></RouterLink>
          </div>

          <div ref="statsElement" class="stats-grid" data-reveal>
            <article>
              <strong>{{ statValues[0] }}<span>+</span></strong>
              <p>{{ copy.stats[0] }}</p>
            </article>
            <article>
              <strong>{{ statValues[1] }}<span>+</span></strong>
              <p>{{ copy.stats[1] }}</p>
            </article>
            <article>
              <strong>{{ statValues[2] }}<span>X</span></strong>
              <p>{{ copy.stats[2] }}</p>
            </article>
          </div>
        </div>
      </section>

      <section id="programs" class="programs-section">
        <div class="programs-band">
          <div class="container section-heading light-heading" data-reveal>
            <div>
              <span class="eyebrow dark-eyebrow">{{ copy.certificationEyebrow }}</span>
              <h2>{{ copy.programmes }}</h2>
            </div>
          </div>
        </div>
        <div class="container program-card-wrap">
          <article class="program-card" data-reveal>
            <div class="program-image">
              <img :src="homeAsset('programme.webp')" alt="加密货币监管与合规基础课程" loading="lazy" decoding="async" />
            </div>
            <div class="program-copy">
              <h3>{{ copy.programmeTitle }}</h3>
              <p>{{ copy.programmeText }}</p>
              <RouterLink to="/gfi/programmes/executive-program">Read More <ArrowUpRight /></RouterLink>
            </div>
          </article>
          <div class="program-controls" aria-hidden="true">
            <button title="上一个项目" disabled><ArrowLeft /></button>
            <button title="下一个项目" disabled><ArrowRight /></button>
          </div>
        </div>
      </section>

      <section id="why" class="why-section">
        <div class="container">
          <div class="section-heading why-heading" data-reveal>
            <div>
              <span class="eyebrow">{{ copy.whyEyebrow }}</span>
              <h2>{{ copy.whyTitle }}</h2>
            </div>
          </div>

          <div class="strength-stack">
            <article v-for="(strength, index) in strengths" :key="strength.number" class="strength-card" :class="{ reversed: index % 2 === 1 }" data-reveal>
              <div class="strength-copy">
                <strong>{{ strength.number }}</strong>
                <h3>{{ strength.title }}</h3>
                <p>{{ strength.text }}</p>
              </div>
              <div class="strength-art">
                <img :src="strength.image" :alt="strength.title" loading="lazy" decoding="async" />
              </div>
            </article>
          </div>
        </div>
      </section>

      <section id="stories" class="stories-section">
        <div class="container">
          <div class="section-heading split-heading" data-reveal>
            <div>
              <span class="eyebrow">{{ copy.storiesEyebrow }}</span>
              <h2>{{ copy.storiesTitle }}</h2>
            </div>
            <RouterLink class="filled-action" to="/gfi/gfi-stories">{{ copy.allStories }} <ArrowUpRight /></RouterLink>
          </div>
          <div class="story-grid">
            <article v-for="story in stories" :key="story.name" class="story-card" data-reveal>
              <img :src="story.image" :alt="story.name" loading="lazy" decoding="async" />
              <div class="story-overlay">
                <h3>{{ story.name }}</h3>
                <p>{{ story.date }} <span /> #{{ copy.storyTag }}</p>
                <RouterLink :to="story.path">{{ copy.readStory }} <ArrowUpRight /></RouterLink>
              </div>
            </article>
          </div>
        </div>
      </section>

      <section id="testimonials" class="testimonial-section">
        <div class="container section-heading split-heading testimonial-heading" data-reveal>
          <div>
            <span class="eyebrow">{{ copy.testimonials }}</span>
            <h2>{{ copy.alumniSay }}</h2>
          </div>
          <button class="account-button" disabled>{{ copy.createAccount }} <span><ArrowUpRight /></span></button>
        </div>
        <div class="testimonial-window">
          <div class="testimonial-track">
            <article v-for="(item, index) in marqueeTestimonials" :key="`${item.name}-${index}`" class="testimonial-card" :aria-hidden="index >= testimonials.length">
              <div class="testimonial-company">
                <div><img v-if="item.company" :src="item.company" alt="" loading="lazy" decoding="async"><strong>{{ item.companyName }}</strong></div>
                <span>{{ copy.reviewFrom }} <Linkedin /></span>
              </div>
              <blockquote>{{ item.quote }}</blockquote>
              <footer>
                <img :src="item.avatar" :alt="item.name" loading="lazy" decoding="async" />
                <div><strong>{{ item.name }}</strong><span>{{ item.role }}</span></div>
                <p aria-label="5 stars">★ ★ ★ ★ ★</p>
              </footer>
            </article>
          </div>
        </div>
      </section>

      <section id="news" class="news-section">
        <div class="container">
          <div class="section-heading split-heading" data-reveal>
            <div>
              <span class="eyebrow">{{ copy.news }}</span>
              <h2>{{ copy.latestNews }}</h2>
            </div>
            <RouterLink class="filled-action" to="/gfi/news">{{ copy.allNews }} <ArrowUpRight /></RouterLink>
          </div>
          <div class="news-grid">
            <article v-for="item in news" :key="item.title" class="news-card" data-reveal>
              <div class="news-image"><img :src="item.image" :alt="item.title" loading="lazy" decoding="async" /></div>
              <div class="news-copy">
                <p>{{ item.date }} <span /> #{{ copy.newsTag }}</p>
                <h3>{{ item.title }}</h3>
                <div class="news-description">{{ item.description }}</div>
                <RouterLink :to="item.path">{{ copy.readMore }} <ArrowUpRight /></RouterLink>
              </div>
            </article>
          </div>
        </div>
      </section>
    </main>

    <div v-if="videoOpen" class="video-modal" role="dialog" aria-modal="true" :aria-label="lang === 'zh' ? 'GFI介绍视频' : 'GFI introduction video'" @click.self="videoOpen = false">
      <div class="video-dialog">
        <button class="video-close" :aria-label="lang === 'zh' ? '关闭视频' : 'Close video'" :title="lang === 'zh' ? '关闭视频' : 'Close video'" @click="videoOpen = false"><X /></button>
        <video controls autoplay playsinline>
          <source src="/gfi/home/gfi-showcase.mp4" type="video/mp4" />
        </video>
      </div>
    </div>

    <GfiFooter />
  </div>
</template>

<style scoped>
@font-face { font-family:"DM Sans Home"; src:url("/gfi/fonts/dm-sans.woff2") format("woff2"); font-style:normal; font-weight:100 900; font-display:swap; }
@font-face { font-family:"Syne Home"; src:url("/gfi/fonts/syne-700.woff2") format("woff2"); font-style:normal; font-weight:700; font-display:swap; }
.gfi-page {
  --gfi-primary: #2864ff;
  --gfi-navy: #101f47;
  --gfi-dark: #111b36;
  --gfi-text: #48546a;
  --gfi-border: #dde5f1;
  min-height: 100vh;
  width: 100%;
  overflow: hidden;
  background: #fff;
  color: var(--gfi-text);
  font-family:"DM Sans Home", "PingFang SC", "Microsoft YaHei", Arial, sans-serif;
  letter-spacing: 0;
}

.gfi-page *,
.gfi-page *::before,
.gfi-page *::after { box-sizing: border-box; }
.gfi-page a { color: inherit; text-decoration: none; }
.gfi-page button { border: 0; }
.container { width: min(1288px, calc(100% - 64px)); margin: 0 auto; }

.hero {
  position: relative;
  height: calc(100vh - 95px);
  min-height: 725px;
  overflow:hidden;
  color: #fff;
}
.hero-media { position:absolute; inset:0; animation:hero-fade .8s ease both; }
.hero-media img { display:block; width:100%; height:100%; object-fit:cover; object-position:center top; animation:kenburns 4.8s ease-out both; }
.hero-shade { position: absolute; inset: 0; background: rgba(8, 27, 66, .69); }
.hero-layout { position: relative; height: 100%; }
.hero-copy { position: absolute; top: 242px; left: 0; width: min(760px, 62%); animation: fade-up .8s ease both; }
.hero-copy h1 { margin: 0 0 22px; overflow-wrap: anywhere; word-break: break-word; font-size: 51px; line-height: 1.18; font-weight: 500; color: #fff; letter-spacing: 0; }
.hero-copy p { width: min(720px, 100%); margin: 0; padding-left: 20px; border-left: 1px solid rgba(255,255,255,.62); font-size: 17px; line-height: 1.85; color: rgba(255,255,255,.9); }
.outline-action { display: inline-flex; align-items: center; gap: 0; margin-top: 36px; }
.outline-action > span:first-child { display: inline-flex; height: 56px; align-items: center; padding: 0 32px; border: 1px solid rgba(255,255,255,.72); border-radius: 28px; font-weight: 600; }
.action-icon { display: inline-flex; width: 56px; height: 56px; margin-left: 8px; align-items: center; justify-content: center; border-radius: 50%; background: #b8ceff; color: #143366; transition: transform .2s ease; }
.outline-action:hover .action-icon { transform: translate(2px, -2px); }
.action-icon svg { width: 22px; height: 22px; }
.hero-note { position: absolute; right: 0; bottom: 0; width: 465px; min-height: 298px; padding: 48px; background: rgba(255,255,255,.08); backdrop-filter: blur(7px); }
.hero-note button { display: flex; width: 80px; height: 80px; align-items: center; justify-content: center; margin-bottom: 30px; border-radius: 50%; background: #10234f; color: #bdd3ff; }
.hero-note button svg { width: 30px; height: 30px; margin-left: 4px; }
.hero-note p { margin: 0; font-size: 16px; line-height: 1.55; color: rgba(255,255,255,.9); }
.hero-controls { position: absolute; left: 0; bottom: 36px; display: flex; align-items: center; }
.hero-controls button { display: inline-flex; width: 56px; height: 56px; align-items: center; justify-content: center; border-radius: 50%; background: #b8ceff; color: #2156c7; }
.hero-controls button:hover { background: #fff; }
.hero-controls svg { width: 29px; height: 29px; }
.hero-controls span { width: 46px; height: 2px; margin: 0 -10px; background: #4176ff; }

.partner-strip { padding: 64px 0; background: #fff; }
.partner-layout { display: grid; grid-template-columns: 320px 1fr; align-items: center; gap: 72px; }
.partner-layout > p { margin: 0; font-size: 15px; line-height: 1.7; color: #596477; }
.partner-window { overflow: hidden; mask-image: linear-gradient(to right, transparent, #000 10%, #000 90%, transparent); }
.partner-track { display: flex; width: max-content; align-items: center; animation: partner-scroll 20s linear infinite; }
.partner-track a { display: flex; width: 180px; height: 77px; align-items: center; justify-content: center; padding: 10px 28px; }
.partner-track img { max-width: 118px; max-height: 50px; object-fit: contain; }

.about-section, .why-section, .news-section { padding: 108px 0; }
.section-heading { display: flex; justify-content: space-between; gap: 80px; align-items: flex-end; }
.section-heading h2 { margin: 0; color: var(--gfi-dark); font-size: 40px; line-height: 1.28; font-weight: 400; letter-spacing: 0; }
.eyebrow { display: inline-flex; margin-bottom: 17px; padding: 4px 18px; border: 1px solid #cfdbf1; border-radius: 18px; background: #f3f6ff; color: #2864dc; font-size: 14px; }
.about-heading h2 { max-width:520px; }
.text-action { display:inline-flex; align-items:center; gap:10px; padding-bottom:8px; border-bottom:1px solid var(--gfi-primary); color:var(--gfi-primary)!important; font-size:16px; }
.text-action svg { width:15px; }
.stats-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin-top: 100px; }
.stats-grid article { min-height:190px; padding:32px; border:1px solid #e2e9f4; background:linear-gradient(180deg,#fff,#f6f8ff); }
.stats-grid strong { display:block; min-height:64px; margin-bottom:18px; color:var(--gfi-primary); font-family:"Syne Home",sans-serif; font-size:58px; line-height:1; font-weight:700; }
.stats-grid strong span { font-size:38px; }
.stats-grid p { margin:0; color:#343a45; font-size:15px; line-height:1.65; }

.programs-section { position:relative; min-height:910px; padding-bottom:112px; background:#fff; }
.programs-band { min-height:541px; padding:72px 0 0; background:var(--gfi-navy); }
.light-heading h2 { color: #fff; }
.dark-eyebrow { border-color: rgba(255,255,255,.12); background: rgba(255,255,255,.08); color: #fff; }
.program-card-wrap { position:relative; margin-top:-288px; }
.program-card { display:block; width:416px; min-height:544px; overflow:hidden; border:1px solid var(--gfi-border); border-radius:8px; background:#fff; }
.program-image { overflow: hidden; }
.program-image { height:288px; }
.program-image img { width:100%; height:100%; object-fit:cover; transition:transform .5s ease; }
.program-card:hover .program-image img { transform: scale(1.035); }
.program-copy { display:flex; min-height:256px; flex-direction:column; padding:32px; }
.program-copy h3 { margin:0 0 18px; color:var(--gfi-dark); font-size:26px; line-height:1.3; font-weight:400; }
.program-copy h3::after { content:""; display:block; width:40px; height:2px; margin-top:10px; background:#30a7ff; }
.program-copy p { display:-webkit-box; overflow:hidden; margin:0; font-size:15px; line-height:1.6; -webkit-line-clamp:2; -webkit-box-orient:vertical; }
.program-copy a, .news-copy a { display:inline-flex; width:max-content; align-items:center; gap:8px; margin-top:auto; padding-bottom:6px; border-bottom:1px solid #235fe6; color:#235fe6; font-weight:500; }
.program-copy a svg, .news-copy a svg, .story-overlay a svg, .filled-action svg { width: 17px; height: 17px; }
.program-controls { position:absolute; top:266px; left:-16px; right:-16px; display:flex; justify-content:space-between; pointer-events:none; }
.program-controls button { display:flex; width:40px; height:40px; align-items:center; justify-content:center; border:1px solid #d7e0ef; border-radius:50%; background:rgba(255,255,255,.86); color:#536174; box-shadow:0 5px 14px rgba(16,31,71,.08); }
.program-controls button:disabled { cursor: default; opacity: .55; }

.why-section { background:#fff url("/gfi/home/line-pattern.svg") center top/cover no-repeat; }
.why-heading { justify-content:center; text-align:center; }
.why-heading h2 { max-width:720px; }
.strength-stack { display: grid; gap: 22px; margin-top: 70px; }
.strength-card { display: grid; grid-template-columns: 7fr 5fr; min-height: 444px; border: 1px solid #cfdbf1; background: #fff; }
.strength-card.reversed { grid-template-columns: 5fr 7fr; }
.strength-card.reversed .strength-copy { order: 2; }
.strength-copy { display: flex; flex-direction: column; justify-content: center; padding: 58px 64px; }
.strength-copy strong { margin-bottom: 20px; color: var(--gfi-primary); font-size: 64px; line-height: 1; font-weight: 600; }
.strength-copy h3 { margin: 0 0 15px; color: var(--gfi-dark); font-size: 28px; font-weight: 500; }
.strength-copy p { margin: 0; line-height: 1.8; }
.strength-art { display:flex; align-items:center; justify-content:center; min-height:320px; padding:35px; background:#f4f7ff url("/gfi/home/strength-bg.svg") center/cover; }
.strength-art img { width: 90%; max-height: 360px; object-fit: contain; }

.stories-section { padding:102px 0 160px; background:#fff; }
.split-heading { align-items: center; }
.filled-action { display: inline-flex; height: 50px; align-items: center; padding-left: 27px; border: 1px solid var(--gfi-primary); border-radius: 25px; background: transparent; color: var(--gfi-primary) !important; font-weight: 500; }
.filled-action svg { box-sizing: border-box; width: 50px; height: 50px; margin: -1px -1px -1px 18px; padding: 15px; border-radius: 50%; background: var(--gfi-primary); color: #fff; }
.story-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin-top: 62px; }
.story-card { position:relative; height:384px; overflow:hidden; border-radius:12px; background:#12264f; }
.story-card > img { width: 100%; height: 100%; object-fit: cover; transition: transform .45s ease; }
.story-card::after { content: ""; position: absolute; inset: 0; background: linear-gradient(to top, rgba(11,31,73,.94), rgba(11,31,73,0) 68%); }
.story-overlay { position: absolute; z-index: 2; left: 0; right: 0; bottom: -45px; padding: 30px; color: #fff; transition: bottom .3s ease; }
.story-card:hover .story-overlay { bottom: 0; }
.story-card:hover > img { transform: scale(1.04); }
.story-overlay h3 { margin: 0 0 10px; color: #fff; font-size: 22px; font-weight: 500; }
.story-overlay p { display: flex; align-items: center; gap: 8px; margin: 0; font-size: 13px; color: rgba(255,255,255,.8); }
.story-overlay p span { width: 3px; height: 3px; border-radius: 50%; background: #fff; }
.story-overlay a { display: inline-flex; align-items: center; gap: 7px; margin-top: 25px; font-weight: 600; }

.testimonial-section { position:relative; padding:112px 0 104px; overflow:hidden; background:#f5f7fb; }
.testimonial-section::after { content:""; position:absolute; left:32%; bottom:-100px; width:330px; height:210px; border-radius:50%; background:rgba(228,238,144,.22); filter:blur(35px); }
.testimonial-heading h2 { color:var(--gfi-dark); }
.account-button { display:flex; min-height:50px; align-items:center; gap:0; padding:0 0 0 25px; border:1px solid #8fb1ff!important; border-radius:28px; background:transparent; color:#7da3ff; font-weight:500; opacity:1!important; }
.account-button span { display:flex; width:50px; height:50px; margin:-1px -1px -1px 18px; align-items:center; justify-content:center; border-radius:50%; background:#9ab8ff; color:#fff; }
.account-button svg { width:20px; }
.testimonial-window { position: relative; z-index: 2; width: 100%; margin-top: 62px; overflow: hidden; }
.testimonial-track { display:flex; width:max-content; animation:testimonial-scroll 48s linear infinite; }
.testimonial-track:hover { animation-play-state: paused; }
.testimonial-card { position:relative; display:flex; width:435px; min-height:310px; flex-direction:column; margin:0 16px; padding:24px; background-color:#fff; background-image:linear-gradient(#edf1f8 1px,transparent 1px),linear-gradient(90deg,#edf1f8 1px,transparent 1px); background-size:78px 78px; color:var(--gfi-text); }
.testimonial-company { position:relative; display:flex; min-height:34px; align-items:center; justify-content:space-between; gap:12px; margin-bottom:30px; }
.testimonial-company > div { display:flex; min-width:0; align-items:center; gap:10px; }
.testimonial-company > div img { width:24px; height:24px; object-fit:contain; }
.testimonial-company > div strong { overflow:hidden; color:#7a8290; font-size:14px; font-weight:400; text-overflow:ellipsis; white-space:nowrap; }
.testimonial-company > span { display:flex; flex-shrink:0; align-items:center; gap:7px; padding:6px 12px; border:1px solid #b9d0ff; border-radius:18px; color:#7da3ff; font-size:12px; }
.testimonial-company svg { width:16px; height:16px; color:#6b9df6; }
.testimonial-card blockquote { position:relative; display:-webkit-box; overflow:hidden; flex:1; margin:0; color:#5b616b; font-size:15px; line-height:1.75; -webkit-box-orient:vertical; -webkit-line-clamp:4; }
.testimonial-card footer { display:flex; align-items:center; gap:12px; margin-top:24px; }
.testimonial-card footer > img { width:48px; height:48px; border-radius:50%; object-fit:cover; }
.testimonial-card footer div { display: grid; gap: 4px; }
.testimonial-card footer strong { color:#555d6a; font-size:14px; }
.testimonial-card footer span { color:#878d97; font-size:12px; }
.testimonial-card footer p { margin-left:auto; padding:7px 12px; border-radius:18px; background:#f8fafc; color:#ffc31a; font-size:17px; white-space:nowrap; }

.news-section { padding-top:99px; background:#fff url("/gfi/home/line-pattern.svg") center top/cover no-repeat; }
.news-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin-top: 62px; }
.news-card { overflow:hidden; border:1px solid var(--gfi-border); border-radius:12px; background:#fff; box-shadow:0 8px 20px rgba(16,31,71,.1); }
.news-image { height: 256px; overflow: hidden; }
.news-image img { width: 100%; height: 100%; object-fit: cover; transition: transform .45s ease; }
.news-card:hover .news-image img { transform: scale(1.04); }
.news-copy { padding: 26px; }
.news-copy > p { display: flex; align-items: center; gap: 9px; margin: 0 0 18px; padding-bottom: 15px; border-bottom: 1px solid #e6ebf3; color: #2961d8; font-size: 13px; }
.news-copy > p span { width: 3px; height: 3px; border-radius: 50%; background: #2961d8; }
.news-copy h3 { display:-webkit-box; overflow:hidden; min-height:60px; margin:0 0 20px; color:var(--gfi-dark); font-family:"Syne Home","DM Sans Home",sans-serif; font-size:20px; line-height:1.5; font-weight:700; -webkit-box-orient:vertical; -webkit-line-clamp:2; }
.news-description { display:-webkit-box; overflow:hidden; min-height:48px; color:#545b67; font-size:14px; line-height:1.7; -webkit-box-orient:vertical; -webkit-line-clamp:2; }

.video-modal { position: fixed; z-index: 100; inset: 0; display: flex; align-items: center; justify-content: center; padding-top: max(24px,var(--app-safe-area-top)); padding-right: max(24px,var(--app-safe-area-right)); padding-bottom: max(24px,var(--app-safe-area-bottom)); padding-left: max(24px,var(--app-safe-area-left)); background: rgba(0,0,0,.84); }
.video-dialog { position: relative; width: min(960px, 100%); }
.video-dialog video { display: block; width: 100%; max-height: calc(var(--app-viewport-height) - var(--app-safe-area-top) - var(--app-safe-area-bottom) - 100px); background: #000; }
.video-close { position: absolute; z-index: 2; top: -50px; right: 0; display: flex; width: 40px; height: 40px; align-items: center; justify-content: center; border-radius: 50%; background: #fff; color: #17213a; }
.video-close svg { width: 22px; height: 22px; }

@keyframes fade-up { from { opacity: 0; transform: translateY(18px); } to { opacity: 1; transform: translateY(0); } }
@keyframes hero-fade { from { opacity:0; } to { opacity:1; } }
@keyframes kenburns { from { transform:scale(1); } to { transform:scale(1.035); } }
@keyframes partner-scroll { to { transform: translateX(-50%); } }
@keyframes testimonial-scroll { to { transform: translateX(-50%); } }

[data-reveal] { opacity:0; transform:translateY(24px); transition:opacity .75s ease,transform .75s ease; }
[data-reveal].is-revealed { opacity:1; transform:none; }

@media (prefers-reduced-motion: reduce) {
  .partner-track, .testimonial-track { animation-play-state: paused; }
  .hero-copy { animation: none; }
  .hero-media,.hero-media img { animation:none; }
  [data-reveal],[data-reveal].is-revealed { opacity:1; transform:none; transition:none; }
}

@media (max-width: 1100px) {
  .hero-note { width: 400px; }
}

@media (max-width: 900px) {
  .container { width: min(100% - 40px, 720px); }
  .hero { height: auto; min-height: 780px; }
  .hero-copy { top: 130px; width: 100%; }
  .hero-copy h1 { font-size: 42px; }
  .hero-note { right: -20px; bottom: 0; width: calc(100% + 40px); min-height: 230px; padding: 30px 20px; }
  .hero-note button { width: 58px; height: 58px; margin-bottom: 20px; }
  .hero-controls { bottom: 258px; }
  .partner-layout { grid-template-columns: 1fr; gap: 25px; }
  .section-heading { display: grid; gap: 30px; }
  .about-heading > p { width: auto; }
  .stats-grid, .story-grid, .news-grid { grid-template-columns: 1fr; }
  .stats-grid article { min-height: auto; }
  .program-card { grid-template-columns: 1fr; }
  .program-image { height: 340px; }
  .program-controls { display: none; }
  .strength-card, .strength-card.reversed { grid-template-columns: 1fr; }
  .strength-card.reversed .strength-copy { order: 0; }
}

@media (max-width: 600px) {
  .container { width: calc(100% - 32px); }
  .hero { min-height: 710px; background-position: 62% center; }
  .hero-copy { top: 94px; }
  .hero-copy h1 { font-size: 34px; line-height: 1.25; }
  .hero-copy p { font-size: 15px; line-height: 1.7; }
  .outline-action { margin-top: 28px; }
  .outline-action > span:first-child { height: 48px; padding: 0 24px; }
  .action-icon { width: 48px; height: 48px; }
  .hero-note { min-height: 215px; }
  .hero-note p { font-size: 14px; }
  .hero-controls { bottom: 238px; }
  .hero-controls button { width: 48px; height: 48px; }
  .partner-strip { padding: 36px 0; }
  .about-section, .why-section, .stories-section, .news-section { padding: 72px 0; }
  .section-heading h2 { font-size: 34px; }
  .stats-grid { margin-top: 42px; }
  .stats-grid strong { font-size: 52px; }
  .programs-section { min-height: auto; padding-bottom: 72px; }
  .programs-band { min-height: 430px; padding-top: 70px; }
  .program-card-wrap { margin-top: -245px; }
  .program-card { width: 100%; }
  .program-image { height: 245px; }
  .program-copy { padding: 32px 24px; }
  .program-copy h3 { font-size: 25px; }
  .strength-stack, .story-grid, .news-grid { margin-top: 42px; }
  .strength-copy { padding: 38px 25px; }
  .strength-copy strong { font-size: 52px; }
  .strength-copy h3 { font-size: 24px; }
  .strength-art { min-height: 255px; }
  .story-card { height: 360px; }
  .story-overlay { bottom: 0; padding: 24px; }
  .testimonial-section { padding: 75px 0 85px; }
  .account-button { justify-self: start; }
  .testimonial-window { margin-top: 42px; }
  .testimonial-card { width: calc(100vw - 48px); min-height: 440px; padding: 25px; }
  .news-image { height: 220px; }
  .news-copy h3 { min-height: auto; }
}
</style>
