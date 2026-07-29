<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { ArrowUpRight, BadgeCheck, BriefcaseBusiness, Check, ChevronDown, Link2, Mail, MapPin, Search } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import { useTranslation } from "@/lib/language"
import { localize } from "@/lib/gfiSite"
import { aboutImages, contacts, fellowBenefits, fellows, jobLinks, jobs, peoplePages, tx } from "@/lib/gfiAboutPages"

const route = useRoute()
const { lang } = useTranslation()
const search = ref("")
const selectedTag = ref("")
const jobFilter = ref<"all" | "full" | "intern">("all")
const openFaq = ref<number | null>(0)
const openYouthFaq = ref<number | null>(0)
const expandedPeople = ref<string[]>([])
const statsSection = ref<HTMLElement | null>(null)
const statValues = ref([0, 0, 0])
const activeLine = ref(7)
let statsObserver: IntersectionObserver | null = null
let statsFrame = 0
let lineTimer = 0
let revealObserver: IntersectionObserver | null = null

const pageKey = computed(() => route.path.replace(/^\/gfi\/?/, "").replace(/\/$/, ""))
const peoplePage = computed(() => peoplePages[pageKey.value as keyof typeof peoplePages])
const l = (value: ReturnType<typeof tx>) => localize(value, lang.value)

const fellowTags = computed(() => {
  const values = fellows.flatMap((fellow) => fellow.tags.map((tag) => l(tag)))
  return [...new Set(values)]
})

const filteredFellows = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return fellows.filter((fellow) => {
    const tags = fellow.tags.map((tag) => l(tag))
    const matchesTag = !selectedTag.value || tags.includes(selectedTag.value)
    const haystack = [l(fellow.name), l(fellow.role), l(fellow.org), ...tags].join(" ").toLocaleLowerCase()
    return matchesTag && (!query || haystack.includes(query))
  })
})

const filteredJobs = computed(() => jobs.filter((job) => {
  if (jobFilter.value === "all") return true
  const type = l(job[2])
  return jobFilter.value === "intern" ? /实习|intern/i.test(type) : !/实习|intern/i.test(type)
}))

const togglePerson = (name: string) => {
  expandedPeople.value = expandedPeople.value.includes(name)
    ? expandedPeople.value.filter((item) => item !== name)
    : [...expandedPeople.value, name]
}


const faqs = [
  [tx("什么是GFI的行业研究员？", "What is a GFI Industry Fellow?"), tx("全球金融科技学院的行业研究员是一位资深从业者，因其在金融科技生态系统中的专业知识、领导力和持续贡献而获得认可。研究员通过指导、对话以及对教育、政策讨论和行业标准的贡献，在推进负责任的金融科技实践中发挥积极作用。", "A GFI Industry Fellow is a senior practitioner recognised for expertise, leadership and sustained contribution to the fintech ecosystem. Fellows advance responsible practice through mentoring, dialogue and contributions to education, policy and industry standards.")],
  [tx("研究员是如何选出的？", "How are Fellows selected?"), tx("行业研究员通过邀请方式任命，基于其在金融、技术、监管或相关领域的专业地位、领域专业知识和已证明的影响力。选拔考虑个人独立、建设性和可信地贡献于GFI使命和社区的能力。", "Industry Fellows are appointed by invitation based on professional standing, domain expertise, demonstrated impact and the ability to contribute independently and constructively to GFI's mission.")],
  [tx("时间承诺是什么？", "What is the time commitment?"), tx("时间承诺是灵活的，因个人而异，考虑到研究员的资深地位。贡献可能包括参与选定的讨论、为项目提供建议、指导专业人士或支持特定倡议。没有固定的最低要求，但期望研究员随着时间的推移进行有意义的参与。", "The commitment is flexible and may include selected discussions, programme advice, mentoring professionals or supporting specific initiatives.")],
  [tx("组织可以提名候选人吗？", "Can organisations nominate candidates?"), tx("可以。组织可以提名高级领导者或主题专家作为行业研究员的候选人。所有提名都会经过审查，以确保与GFI的标准、价值观和重点领域保持一致。", "Yes. Organisations may nominate senior leaders or subject-matter experts, with every nomination reviewed against GFI's standards, values and focus areas.")],
  [tx("研究员如何为GFI项目和倡议做出贡献？", "How do Fellows contribute to GFI programmes and initiatives?"), tx("行业研究员通过在认证、高管项目、研究、政策对话和社区倡议中分享专业知识来做出贡献。这可能包括就课程相关性提供建议、参与闭门讨论、指导专业人士、为出版物贡献见解或支持生态系统合作。", "Industry Fellows contribute by sharing expertise across certifications, executive programmes, research, policy dialogue and community initiatives.")],
] as const

const youthFeatures = [
  [tx("实践经验", "Real-world Experience"), tx("通过在GFI会议、小组讨论和闭门圆桌会议上做志愿者，获得实际接触机会。", "Gain practical exposure by volunteering at GFI conferences, panels and closed-door roundtables."), "real-world-experience.cxQG5lJx_TS6nf.webp"],
  [tx("金融科技学习机会", "Fintech Learning Opportunities"), tx("通过适合学生的会议和活动，探索支付、人工智能、区块链、监管和风险等主题。", "Explore payments, AI, blockchain, regulation and risk through student-friendly sessions and events."), "fintech-learning-opportunities.Bl_g268g_ZeD3mQ.webp"],
  [tx("职业与认证途径", "Career & Certification Pathways"), tx("通过特许金融科技助理（CFtA）计划，发现进入金融科技角色的途径。", "Discover pathways into fintech roles through the Chartered Fintech Associate (CFtA) programme."), "career-certification-pathways.C-H--jds_J5Bky.webp"],
  [tx("社区与校园参与", "Community & Campus Engagement"), tx("通过共享的倡议和项目，与各大学、金融科技社团和青年组织的同行建立联系。", "Connect with peers across universities, fintech societies and youth organisations through shared initiatives."), "community-campus-engagement.CKXT0_f0_bmAh3.webp"],
] as const

const formatStat = (index: number) => `${statValues.value[index]}${index === 0 ? "K+" : "+"}`

const startStats = () => {
  const targets = [10, 50, 10]
  const startedAt = performance.now()
  const tick = (now: number) => {
    const progress = Math.min((now - startedAt) / 1800, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    statValues.value = targets.map((target) => Math.round(target * eased))
    if (progress < 1) statsFrame = requestAnimationFrame(tick)
  }
  statsFrame = requestAnimationFrame(tick)
}

const observeStats = () => {
  statsObserver?.disconnect()
  if (!statsSection.value) return
  statsObserver = new IntersectionObserver(([entry]) => {
    if (!entry?.isIntersecting) return
    startStats()
    statsObserver?.disconnect()
  }, { threshold: 0.3 })
  statsObserver.observe(statsSection.value)
}

const observeReveals = () => {
  revealObserver?.disconnect()
  revealObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (!entry.isIntersecting) return
      entry.target.classList.add("is-revealed")
      revealObserver?.unobserve(entry.target)
    })
  }, { threshold: 0.08, rootMargin: "0px 0px -30px" })
  document.querySelectorAll(".about-pages [data-reveal]").forEach((element) => revealObserver?.observe(element))
}

watch(pageKey, async () => {
  search.value = ""
  selectedTag.value = ""
  jobFilter.value = "all"
  openFaq.value = 0
  openYouthFaq.value = 0
  expandedPeople.value = []
  statValues.value = [0, 0, 0]
  await nextTick()
  observeStats()
  observeReveals()
})

onMounted(async () => {
  await nextTick()
  observeStats()
  observeReveals()
  lineTimer = window.setInterval(() => {
    let next = activeLine.value
    while (next === activeLine.value) next = 3 + Math.floor(Math.random() * 10)
    activeLine.value = next
  }, 5000)
})

onBeforeUnmount(() => {
  statsObserver?.disconnect()
  revealObserver?.disconnect()
  cancelAnimationFrame(statsFrame)
  window.clearInterval(lineTimer)
})
</script>

<template>
  <div class="about-pages">
    <GfiHeader theme="light" />

    <main v-if="pageKey === 'about'">
      <section class="story-intro">
        <div class="story-line-field" aria-hidden="true">
          <svg viewBox="0 0 1920 819" preserveAspectRatio="xMidYMid slice">
            <defs>
              <linearGradient id="about-line-glow" x1="0" y1="0" x2="1" y2="1">
                <stop offset="0" stop-color="#ffffff" />
                <stop offset=".47" stop-color="#ffffff" />
                <stop offset=".5" stop-color="#5d8cff" />
                <stop offset=".53" stop-color="#ffffff" />
                <stop offset="1" stop-color="#ffffff" />
              </linearGradient>
            </defs>
            <g class="story-line-paths">
              <path v-for="index in 16" :key="index" :class="{ active: activeLine === index }" :d="`M ${-390 + index * 105} -190 C ${-180 + index * 95} 120, ${-40 + index * 80} 520, ${180 + index * 105} 900`" />
            </g>
          </svg>
        </div>
        <div class="site-container story-copy">
          <h1 class="story-reveal">
            {{ lang === "zh" ? "推进" : "Advancing" }}
            <strong>
              <svg class="story-doodle" viewBox="0 0 69 33" aria-hidden="true">
                <path d="M54.03 29.58c3.52-1.88 7.15-3.53 10.9-4.94M49.14 23.71c4.32-4.33 9.03-8.22 14.05-11.69M35.73 13.13c.96-3.67 2.2-7.25 3.7-10.72M12.79 25.62c-4.17-1.1-8.37-.72-11.54 1.01 2.29-4.55 6.83-7.54 11.89-7.84-1.16-3.67-.81-7.72.96-11.14 2.54 3.22 4.02 7.12 4.25 11.08 2.87-1.33 6.18-1.48 9.16-.42-2.56 1.69-4.7 4-6.19 6.68" />
              </svg>
              {{ lang === "zh" ? "全球金融科技标准" : "Global Fintech Standards" }}
            </strong>
          </h1>
          <p class="story-reveal story-delay-1">{{ lang === "zh" ? "我们是一个非营利智库和专业机构，致力于塑造金融科技的未来。总部位于新加坡的全球金融科技学院（GFI）连接监管机构、企业和学术界，制定全球标准，提供世界级教育，并在整个金融生态系统中促进负责任的创新。" : "We are a non-profit think tank and professional body shaping the future of fintech. Headquartered in Singapore, GFI connects regulators, corporates and academia to set global standards, deliver world-class education and foster responsible innovation across the financial ecosystem." }}</p>
        </div>
        <div
          class="photo-carousel story-reveal story-delay-2"
          role="region"
          :aria-label="lang === 'zh' ? 'GFI活动图片轮播' : 'GFI community photo carousel'"
        >
          <div class="photo-ribbon">
            <div v-for="setIndex in 2" :key="setIndex" class="photo-set" :aria-hidden="setIndex === 2">
              <figure v-for="(image, index) in aboutImages" :key="`${setIndex}-${image}`" class="photo-slide">
                <img :src="image" :alt="setIndex === 1 ? `GFI ${index + 1}` : ''" :loading="setIndex === 1 && index < 4 ? 'eager' : 'lazy'" draggable="false" />
              </figure>
            </div>
          </div>
        </div>
      </section>

      <section id="achievements" class="impact-section">
        <div class="site-container impact-layout">
          <div class="impact-copy">
            <span class="eyebrow">{{ lang === "zh" ? "成就" : "Achievements" }}</span>
            <h2>{{ lang === "zh" ? "推动金融科技教育和标准的影响" : "Driving Impact in Fintech Education and Standards" }}</h2>
            <p class="impact-lead">{{ lang === "zh" ? "自成立以来，GFI通过创建全球认可的认证、建立协作网络以及领导影响行业和政策的对话，推进了金融科技知识和实践。" : "Since its founding, GFI has advanced fintech knowledge and practice by creating globally recognised certifications, building collaborative networks and leading dialogue that influences industry and policy." }}</p>
            <div class="impact-features">
              <article><span class="feature-icon"><BadgeCheck /></span><div><h3>{{ lang === "zh" ? "行业认可的认证" : "Industry-recognised Certifications" }}</h3><p>{{ lang === "zh" ? "我们的特许金融科技助理（CFtA）和特许金融科技专业人士（CFtP®）认证为专业人士提供在快速发展的数字经济中茁壮成长的技能。" : "Our Chartered Fintech Associate (CFtA) and Chartered Fintech Professional (CFtP®) certifications equip professionals with the skills to thrive in a fast-moving digital economy." }}</p></div></article>
              <article><span class="feature-icon"><Link2 /></span><div><h3>{{ lang === "zh" ? "全球协作" : "Global Collaboration" }}</h3><p>{{ lang === "zh" ? "通过与监管机构、大学和企业的合作，我们建立最佳实践和认证标准，塑造金融的未来。" : "Working with regulators, universities and corporates, we establish best practices and certification standards that shape the future of finance." }}</p></div></article>
            </div>
          </div>
          <div ref="statsSection" class="impact-stats">
            <div class="stat-row"><div class="stat-number"><strong>{{ formatStat(0) }}</strong><span>{{ lang === "zh" ? "专业人士" : "Professionals" }}</span></div><p>{{ lang === "zh" ? "在我们不断增长的全球网络中" : "In our growing global network" }}</p></div>
            <div class="stat-row stat-reverse"><p>{{ lang === "zh" ? "与行业领袖和学术机构合作" : "Collaborating with industry leaders and academic institutions" }}</p><div class="stat-number"><strong>{{ formatStat(1) }}</strong><span>{{ lang === "zh" ? "合作伙伴" : "Partners" }}</span></div></div>
            <div class="stat-row"><div class="stat-number"><strong>{{ formatStat(2) }}</strong><span>{{ lang === "zh" ? "国家" : "Countries" }}</span></div><p>{{ lang === "zh" ? "在我们认证社区中的代表" : "Represented in our certification community" }}</p></div>
          </div>
        </div>
      </section>
    </main>

    <main v-else-if="peoplePage">
      <section class="title-hero patterned">
        <div data-reveal><h1>{{ l(peoplePage.title) }}</h1><nav class="breadcrumb"><RouterLink to="/gfi">Home</RouterLink><span>/</span><RouterLink to="/gfi">Cn</RouterLink><span>/</span><RouterLink to="/gfi/about">About</RouterLink><span>/</span><b>{{ peoplePage.title.en }}</b></nav></div>
      </section>
      <section class="overlap-intro site-container" data-reveal>
        <img src="https://globalfintechinstitute.org/assets/gfi-about-1.BPJKq5hL_1NRqCB.webp" :alt="l(peoplePage.introTitle)" />
        <div class="overlap-panel"><h2>{{ l(peoplePage.introTitle) }}</h2><p>{{ l(peoplePage.intro) }}</p></div>
      </section>
      <section class="people-section white-section">
        <div class="site-container">
          <div class="people-heading" data-reveal><span class="eyebrow">{{ l(peoplePage.eyebrow) }}</span><h2>{{ l(peoplePage.membersTitle) }}</h2></div>
          <div class="people-list">
            <article v-for="person in peoplePage.members" :key="person.name" class="person-row" data-reveal>
              <img :src="person.image" :alt="person.name" />
              <div><h3>{{ person.name }}</h3><strong>{{ person.role.en }}</strong><p :class="{ clamped: !expandedPeople.includes(person.name) }">{{ person.bio.en }}</p><button type="button" @click="togglePerson(person.name)">{{ expandedPeople.includes(person.name) ? "Read less" : "Read more" }} <ChevronDown :class="{ rotated: expandedPeople.includes(person.name) }" /></button></div>
            </article>
          </div>
        </div>
      </section>
      <section class="governance-cta" data-reveal><div class="site-container"><div><h2>{{ lang === "zh" ? "从治理到集体领导" : "From Governance to Collective Leadership" }}</h2><p>{{ lang === "zh" ? "除了董事会，GFI的工作还通过其委员会和理事会推进——这些由实践者领导的团体贡献专业知识，指导标准，并在教育、政策和行业参与方面塑造项目。" : "Beyond the Board, GFI's work advances through committees and councils led by practitioners who contribute expertise, guide standards and shape programmes across education, policy and industry engagement." }}</p><RouterLink to="/gfi/subcommittees">{{ lang === "zh" ? "探索小组委员会" : "Explore SubCommittees" }} <ArrowUpRight /></RouterLink></div><span class="cta-circle" aria-hidden="true"></span></div></section>
    </main>

    <main v-else-if="pageKey === 'subcommittees'">
      <section class="title-hero patterned"><div data-reveal><h1>{{ lang === "zh" ? "子委员会" : "SubCommittees" }}</h1><nav class="breadcrumb"><RouterLink to="/gfi">Home</RouterLink><span>/</span><RouterLink to="/gfi">Cn</RouterLink><span>/</span><b>Subcommittees</b></nav></div></section>
      <section class="overlap-intro site-container" data-reveal><img src="https://globalfintechinstitute.org/assets/gfi-about-1.BPJKq5hL_1NRqCB.webp" alt="GFI"><div class="overlap-panel"><h2>{{ lang === "zh" ? "支持GFI使命的专业专长" : "Specialist Expertise Supporting GFI's Mission" }}</h2><p>{{ lang === "zh" ? "GFI子委员会是在GFI委员会下设立的特定领域工作组，旨在解决金融科技生态系统中新兴、复杂和高影响力的领域。它们汇聚高级从业者和主题专家，以开发见解、指导项目发展并支持知情的行业对话。在GFI的治理框架内运作，子委员会将战略方向转化为与不断发展的技术和监管现实相一致的专注、应用性工作。" : "GFI SubCommittees are domain-specific working groups established to address emerging, complex and high-impact areas across the fintech ecosystem. They bring together senior practitioners and subject-matter experts to develop insights, guide programme development and support informed industry dialogue." }}</p></div></section>
      <section class="committee-section"><div class="site-container"><div data-reveal><span class="eyebrow">{{ lang === "zh" ? "子委员会" : "SubCommittees" }}</span><h2>{{ lang === "zh" ? "GFI子委员会" : "GFI SubCommittees" }}</h2></div><div class="committee-grid"><article data-reveal><span>01</span><h3>{{ lang === "zh" ? "新兴技术融合（人工智能、区块链与量子计算）" : "Emerging Technology Convergence (AI, Blockchain & Quantum Computing)" }}</h3><p>{{ lang === "zh" ? "本子委员会探讨人工智能、区块链和量子计算等先进技术如何融合以重塑金融服务。重点关注治理、风险、安全与监管考量，支持金融科技生态系统内的知情采用与负责任创新。" : "This SubCommittee explores how AI, blockchain and quantum computing converge to reshape financial services, with a focus on governance, risk, security and regulation." }}</p><a href="https://globalfintechinstitute.org/cn/subcommittees/emerging-technology-convergence/" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "了解更多" : "Learn more" }} <ArrowUpRight /></a></article><article data-reveal><span>02</span><h3>{{ lang === "zh" ? "数字资产安全与合规子委员会" : "Digital Assets Security & Compliance SubCommittee" }}</h3><p>{{ lang === "zh" ? "本子委员会致力于加强数字资产生态系统的安全性、合规性与运营韧性。它涵盖托管、基础设施风险、监管预期与治理等关键议题，为在复杂且高度监管环境中运营的机构提供支持。" : "This SubCommittee strengthens security, compliance and operational resilience across digital asset ecosystems, covering custody, infrastructure risk, regulatory expectations and governance." }}</p><a href="https://globalfintechinstitute.org/cn/subcommittees/digital-asset-security-compliance/" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "了解更多" : "Learn more" }} <ArrowUpRight /></a></article></div></div></section>
      <section class="simple-cta" data-reveal><div class="site-container"><div><h2>{{ lang === "zh" ? "有兴趣参与？" : "Interested in Contributing?" }}</h2><p>{{ lang === "zh" ? "若您希望探讨贡献专业能力的机会，欢迎与我们联系或进一步了解我们的治理机构。" : "Contact us to explore opportunities to contribute your expertise or learn more about our governance bodies." }}</p><RouterLink to="/gfi/contact">{{ lang === "zh" ? "联系我们" : "Contact Us" }} <ArrowUpRight /></RouterLink></div><span class="cta-circle" aria-hidden="true"></span></div></section>
    </main>

    <main v-else-if="pageKey === 'industry-fellow'">
      <section class="fellow-hero" style="background-image:url('https://globalfintechinstitute.org/assets/gfi-banner-3.X5tYPN3w_ZLlOGO.webp')"><div class="site-container"><h1>{{ lang === "zh" ? "行业研究员" : "Industry Fellows" }}</h1><p>{{ lang === "zh" ? "一个邀请制的专业网络，由高级领导者组成，通过教育、对话和生态系统合作，贡献专业知识以推进负责任的金融科技实践。" : "An invitation-only professional network of senior leaders contributing expertise through education, dialogue and ecosystem collaboration." }}</p><a href="https://airtable.com/appCg8CSsvuJBv582/pagDK5m706Bo4Qect/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "表达兴趣" : "Express Interest" }} <ArrowUpRight /></a></div></section>
      <section class="fellow-program"><div class="site-container"><span class="eyebrow">{{ lang === "zh" ? "行业研究员计划" : "Industry Fellow Programme" }}</span><h2>{{ lang === "zh" ? "领导力平台，而不仅仅是认可" : "A Platform for Leadership, Not Just Recognition" }}</h2><p class="lead">{{ lang === "zh" ? "行业研究员计划汇集了金融科技各个垂直领域的资深专业人士，通过指导、政策和标准对话，以及与监管机构、投资者和行业领导者的合作，加强生态系统。" : "The programme brings together senior professionals across fintech verticals to strengthen the ecosystem through mentoring, policy and standards dialogue, and collaboration with regulators, investors and industry leaders." }}</p><div class="benefit-grid"><article v-for="(benefit,index) in fellowBenefits" :key="l(benefit[0])"><span>0{{ index + 1 }}</span><h3>{{ l(benefit[0]) }}</h3><p>{{ l(benefit[1]) }}</p></article></div></div></section>
      <section class="fellows-section"><div class="site-container"><h2>{{ lang === "zh" ? "认识GFI行业研究员" : "Meet GFI Industry Fellows" }}</h2><p>{{ lang === "zh" ? "按领域搜索或筛选，了解我们的资深从业者网络。" : "Search or filter by domain to discover our network of senior practitioners." }}</p><div class="fellow-tools"><label><Search /><input v-model="search" :placeholder="lang === 'zh' ? '搜索行业研究员' : 'Search Industry Fellows'"></label><select v-model="selectedTag"><option value="">{{ lang === "zh" ? "全部领域" : "All Domains" }}</option><option v-for="tag in fellowTags" :key="tag" :value="tag">{{ tag }}</option></select></div><strong class="result-count">{{ filteredFellows.length }} {{ lang === "zh" ? "位行业研究员" : "Industry Fellows" }}</strong><div class="fellow-grid"><article v-for="fellow in filteredFellows" :key="l(fellow.name)"><span class="credential" v-if="l(fellow.name).includes('CFtP')">CFtP®</span><h3>{{ l(fellow.name) }}</h3><strong>{{ l(fellow.role) }}</strong><p>{{ l(fellow.org) }}</p><div class="tag-row"><span v-for="tag in fellow.tags" :key="l(tag)">{{ l(tag) }}</span></div><a :href="fellow.url" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "查看资料" : "View profile" }} <ArrowUpRight /></a></article></div></div></section>
      <section class="faq-section"><div class="site-container"><h2>FAQs</h2><article v-for="(faq,index) in faqs" :key="l(faq[0])"><button @click="openFaq = openFaq === index ? null : index"><span>{{ l(faq[0]) }}</span><ChevronDown :class="{ rotated: openFaq === index }" /></button><p v-if="openFaq === index">{{ l(faq[1]) }}</p></article></div></section>
    </main>

    <main v-else-if="pageKey === 'youth-wing'">
      <section class="title-hero patterned"><h1>{{ lang === "zh" ? "GFI 青年部" : "GFI Youth Wing" }}</h1></section>
      <section class="youth-intro site-container"><img src="https://globalfintechinstitute.org/assets/youth-wing-hero.tPwGWJkI_ZkzQJw.webp" alt="GFI Youth Wing"><div><h2>{{ lang === "zh" ? "一个为塑造金融科技未来的学生和年轻专业人士而设的社区。" : "A Community for Students and Young Professionals Shaping Fintech's Future." }}</h2><p>{{ lang === "zh" ? "GFI青年部是全球金融科技研究所的学生和早期职业社区。我们汇聚了对数字金融充满好奇、渴望获得实践经验、并准备参与塑造行业的真实对话的年轻人。" : "The GFI Youth Wing is our student and early-career community, bringing together young people curious about digital finance, eager for practical experience and ready to join real industry conversations." }}</p><a href="https://airtable.com/appY1MeInT0J7XPkm/pagyCg86VFuc5foSn/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "立即注册" : "Register Now" }} <ArrowUpRight /></a></div></section>
      <section class="youth-features"><div class="site-container"><h2>{{ lang === "zh" ? "塑造未来的金融科技人才" : "Shaping Future Fintech Talent" }}</h2><p class="lead">{{ lang === "zh" ? "为青年提供知识、经验和网络，使他们能够有意义地参与全球金融科技生态系统。" : "Giving young people the knowledge, experience and networks to participate meaningfully in the global fintech ecosystem." }}</p><div class="youth-grid"><article v-for="feature in youthFeatures" :key="l(feature[0])"><img :src="`https://globalfintechinstitute.org/assets/${feature[2]}`" :alt="l(feature[0])"><div><h3>{{ l(feature[0]) }}</h3><p>{{ l(feature[1]) }}</p></div></article></div></div></section>
      <section class="youth-benefits"><div class="site-container"><div><h2>{{ lang === "zh" ? "我们要建立和推动的目标" : "What We Aim to Build and Advance" }}</h2><p>{{ lang === "zh" ? "一个由学生主导的空间，用于学习、贡献和探索金融科技领域的真实机会。" : "A student-led space to learn, contribute and explore real opportunities in fintech." }}</p><ul><li v-for="item in (lang === 'zh' ? ['在行业小组讨论、会议和圆桌会议上做志愿者。','支持活动运营、研究和数字内容。','通过CFtA计划获得基础金融科技学习。','与各大学和金融科技社区的同行建立联系。','加入学生主导的项目、学习圈和青年部倡议。'] : ['Volunteer at industry panels, conferences and roundtables.','Support event operations, research and digital content.','Build foundational fintech knowledge through CFtA.','Connect with peers across universities and fintech communities.','Join student-led projects, learning circles and Youth Wing initiatives.'])" :key="item"><Check />{{ item }}</li></ul></div><img src="https://globalfintechinstitute.org/assets/gfi-benefits.CSK4AmvV_7iTVc.webp" alt="GFI Youth Wing benefits"></div></section>
    </main>

    <main v-else-if="pageKey === 'career'">
      <section class="career-hero patterned"><div class="site-container"><h1>{{ lang === "zh" ? "立即加入GFI，在金融科技领域建立您的职业生涯" : "Join GFI and Build Your Career in Fintech" }}</h1></div></section>
      <section class="career-intro site-container"><div><h2>{{ lang === "zh" ? "立即加入GFI，在金融科技领域建立您的职业生涯" : "Join GFI and Build Your Career in Fintech" }}</h2><p>{{ lang === "zh" ? "加入我们，共同塑造金融科技的未来。在GFI，我们为专业人士、学生和志愿者提供参与研究、教育和社区建设的机会。无论是通过全职职位、实习还是志愿者岗位，您都将成为推动金融科技创新、标准和影响力的全球网络的一部分。" : "Join us in shaping the future of fintech. GFI offers professionals, students and volunteers opportunities across research, education and community building. Through full-time roles, internships or volunteering, you will become part of a global network advancing fintech innovation, standards and impact." }}</p><a href="#openings">{{ lang === "zh" ? "查看所有职位" : "View All Positions" }} <ArrowUpRight /></a></div><img src="https://globalfintechinstitute.org/assets/intro.CZOZAx-m_ZzLPPa.webp" alt="Careers at GFI"></section>
      <section id="openings" class="jobs-section"><div class="site-container"><h2>{{ lang === "zh" ? "当前职位" : "Current Openings" }}</h2><div class="job-tabs"><button :class="{active:jobFilter==='all'}" @click="jobFilter='all'">{{ lang === "zh" ? "全部" : "All" }}</button><button :class="{active:jobFilter==='full'}" @click="jobFilter='full'">{{ lang === "zh" ? "全职" : "Full-time" }}</button><button :class="{active:jobFilter==='intern'}" @click="jobFilter='intern'">{{ lang === "zh" ? "实习" : "Internship" }}</button></div><div class="job-list"><article v-for="job in filteredJobs" :key="l(job[1])"><BriefcaseBusiness /><div><span>{{ l(job[0]) }}</span><h3>{{ l(job[1]) }}</h3><p><strong>{{ l(job[2]) }}</strong><MapPin />{{ l(job[3]) }}</p></div><a :href="`https://globalfintechinstitute.org/${lang === 'zh' ? 'cn' : 'career'}/${lang === 'zh' ? 'career/' : ''}${jobLinks[jobs.indexOf(job)]}/`" target="_blank" rel="noopener noreferrer" :aria-label="l(job[1])"><ArrowUpRight /></a></article></div></div></section>
    </main>

    <main v-else-if="pageKey === 'contact'">
      <section class="contact-section patterned"><div class="site-container"><h1>{{ lang === "zh" ? "与全球金融科技学院取得联系" : "Get in Touch with the Global Fintech Institute" }}</h1><p>{{ lang === "zh" ? "无论您是在探索我们的认证、合作伙伴关系还是即将推出的计划，我们的团队随时为您提供帮助。请通过以下相关联系方式与我们联系，以便我们能够高效地回复。我们的团队通常在2-3个工作日内回复。" : "Whether you are exploring our certifications, partnerships or upcoming programmes, our team is here to help. Use the relevant contact below so we can respond efficiently. We typically reply within 2-3 business days." }}</p><div class="contact-grid"><article v-for="contact in contacts" :key="l(contact[0])"><Mail /><h2>{{ l(contact[0]) }}</h2><p>{{ l(contact[1]) }}</p><span>Email</span><a :href="`mailto:${contact[2]}`">{{ contact[2] }}</a><a class="contact-link" :href="`mailto:${contact[2]}`">{{ lang === "zh" ? "联系我们" : "Contact Us" }} <ArrowUpRight /></a></article></div></div></section>
    </main>

    <GfiFooter />
  </div>
</template>

<style scoped>
.about-pages { min-height: 100vh; overflow: hidden; background: #fff; color: #4a5568; font-family: Inter, "PingFang SC", "Microsoft YaHei", Arial, sans-serif; letter-spacing: 0; }
.about-pages * { box-sizing: border-box; }
.about-pages a { text-decoration: none; }
.site-container { width: min(1288px, calc(100% - 64px)); margin: 0 auto; }
.patterned { background: #fbfcfe url("https://globalfintechinstitute.org/assets/bg.CZWEzqel_Z5lu0T.svg") center/cover; background-blend-mode: screen; }
.eyebrow { display: inline-flex; margin-bottom: 17px; padding: 5px 17px; border: 1px solid #c9d8fa; border-radius: 18px; background: #f3f7ff; color: #2864ff; font-size: 13px; }
h1, h2, h3, p { overflow-wrap: anywhere; }
h1, h2, h3 { color: #151f37; font-weight: 500; }
.story-intro { position: relative; isolation: isolate; padding: 78px 0 0; overflow: hidden; background: linear-gradient(to top, #fff 0%, rgba(244,247,251,.86) 100%); }
.story-line-field { position: absolute; inset: 0 0 auto; z-index: -1; height: 819px; overflow: hidden; pointer-events: none; }
.story-line-field svg { width: 100%; height: 100%; }
.story-line-paths path { fill: none; stroke: #fff; stroke-width: 1.2; vector-effect: non-scaling-stroke; }
.story-line-paths path.active { stroke: url(#about-line-glow); stroke-width: 1.7; animation: line-glow 5s linear infinite; }
@keyframes line-glow {
  from { stroke-dasharray: 0 1900; stroke-dashoffset: 0; }
  12% { stroke-dasharray: 110 1790; }
  to { stroke-dasharray: 110 1790; stroke-dashoffset: -1900; }
}
.story-copy { position: relative; display: grid; grid-template-columns: 1.1fr .9fr; align-items: center; gap: 130px; min-height: 218px; }
.story-copy h1 { margin: 0; font-size: 37px; line-height: 1.35; }
.story-copy h1 strong { position: relative; color: #2864ff; font-weight: 600; white-space: nowrap; }
.story-doodle { position: absolute; top: -35px; left: 50%; width: 69px; height: 33px; transform: translateX(-50%); fill: none; stroke: currentColor; stroke-width: 1.4; stroke-linecap: round; stroke-linejoin: round; }
.story-copy p { margin: 0; color: #2f3a4e; font-size: 16px; line-height: 1.75; }
.story-reveal { animation: story-fade-up .75s both; }
.story-delay-1 { animation-delay: .2s; }
.story-delay-2 { animation-delay: .3s; }
@keyframes story-fade-up { from { opacity: 0; transform: translateY(22px); } to { opacity: 1; transform: translateY(0); } }
.photo-carousel { position: relative; width: 100%; margin-top: 0; overflow: hidden; }
.photo-ribbon { display: flex; width: max-content; align-items: center; animation: photo-marquee 50s linear infinite; will-change: transform; }
.photo-set { display: flex; align-items: center; }
.photo-slide { height: 28rem; flex: 0 0 25rem; margin: 0; padding: 0 1rem; overflow: hidden; }
.photo-set:first-child .photo-slide:nth-child(even), .photo-set:nth-child(2) .photo-slide:nth-child(odd) { height: 24rem; }
.photo-slide img { display: block; width: 100%; height: 100%; object-fit: cover; pointer-events: none; }
@keyframes photo-marquee {
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-50%, 0, 0); }
}
.impact-section { padding: 108px 0 110px; background: #f5f7fb; }
.impact-layout { display: grid; grid-template-columns: minmax(0, 1.2fr) minmax(390px, .82fr); gap: 135px; align-items: center; }
.impact-copy h2 { max-width: 650px; margin: 0; font-size: 36px; line-height: 1.3; }
.impact-lead { max-width: 650px; margin: 24px 0 0; font-size: 15px; line-height: 1.75; }
.impact-features { max-width: 650px; margin-top: 58px; }
.impact-features article { display: grid; grid-template-columns: 56px 1fr; gap: 24px; align-items: center; padding: 0 0 30px; }
.impact-features article + article { padding-top: 26px; border-top: 1px solid #dde3ec; }
.feature-icon { display: grid; width: 56px; height: 56px; place-items: center; border-radius: 8px; background: #eaf0ff; color: #2864ff; }
.feature-icon svg { width: 29px; height: 29px; stroke-width: 1.45; }
.impact-features h3 { margin: 0 0 4px; font-size: 21px; font-weight: 500; }
.impact-features p { margin: 0; font-size: 14px; line-height: 1.55; }
.impact-stats { border-left: 1px solid #d8dfe9; }
.stat-row { display: grid; grid-template-columns: 1fr 1fr; min-height: 143px; }
.stat-row + .stat-row { border-top: 1px solid #d8dfe9; }
.stat-row > * { display: flex; min-width: 0; justify-content: center; flex-direction: column; margin: 0; padding: 28px 16px; }
.stat-row > :nth-child(2) { border-left: 1px solid #d8dfe9; }
.stat-number { align-items: flex-end; text-align: right; }
.stat-number strong { color: #2864ff; font-size: 42px; font-weight: 600; line-height: 1; }
.stat-number span { margin-top: 14px; color: #32405a; font-size: 15px; }
.stat-row p { color: #505968; font-size: 14px; line-height: 1.55; }
.stat-reverse > p { align-items: flex-end; text-align: right; }
.stat-reverse .stat-number { align-items: flex-start; text-align: left; }
.section-heading { margin-bottom: 55px; }
.split-heading { display: grid; grid-template-columns: 1fr .85fr; gap: 90px; align-items: end; }
.section-heading h2, .people-section > .site-container > h2, .committee-section h2, .fellows-section h2, .jobs-section h2 { margin: 0; font-size: 43px; line-height: 1.25; }
.section-heading p, .lead { margin: 0; font-size: 16px; line-height: 1.8; }
.benefit-grid article > span, .committee-grid article > span { color: #2864ff; font-size: 14px; }
.benefit-grid h3, .committee-grid h3 { margin: 20px 0 14px; font-size: 25px; line-height: 1.35; }
.benefit-grid p, .committee-grid p { margin: 0; line-height: 1.8; }
.title-hero { display: grid; min-height: 390px; place-items: center; }
.title-hero h1 { margin: 0; font-size: 52px; }
.intro-section { display: grid; grid-template-columns: 1fr 1fr; gap: 90px; align-items: center; padding-top: 105px; padding-bottom: 105px; }
.intro-section > img { width: 100%; height: 470px; object-fit: cover; }
.intro-section h2, .youth-intro h2, .career-intro h2, .fellow-program h2 { margin: 0 0 25px; font-size: 39px; line-height: 1.28; }
.intro-section p, .youth-intro p, .career-intro p { margin: 0; font-size: 16px; line-height: 1.8; }
.people-section, .committee-section, .fellows-section, .jobs-section { padding: 105px 0; background: #f7f9fc; }
.people-grid { display: grid; gap: 28px; margin-top: 50px; }
.person-card { display: grid; grid-template-columns: 350px 1fr; min-height: 390px; overflow: hidden; border: 1px solid #dce3ee; background: #fff; }
.person-card:nth-child(even) { grid-template-columns: 1fr 350px; }
.person-card:nth-child(even) img { order: 2; }
.person-card img { width: 100%; height: 100%; object-fit: cover; object-position: top center; }
.person-card div { align-self: center; padding: 50px 65px; }
.person-card h3 { margin: 0 0 9px; font-size: 28px; }
.person-card strong { color: #2864ff; font-size: 14px; }
.person-card p { margin: 23px 0 0; line-height: 1.82; }
.governance-cta { padding: 82px 0; background: #101f47; color: #fff; }
.governance-cta .site-container { display: flex; justify-content: space-between; gap: 70px; align-items: center; }
.governance-cta h2 { margin: 8px 0 14px; color: #fff; font-size: 37px; }
.governance-cta p { max-width: 800px; margin: 0; color: rgba(255,255,255,.72); line-height: 1.75; }
.governance-cta a, .simple-cta a, .fellow-hero a, .youth-intro a, .career-intro a { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 8px; padding: 16px 23px; border-radius: 28px; background: #2864ff; color: #fff; }
.governance-cta svg, .simple-cta svg, .fellow-hero svg, .youth-intro svg, .career-intro svg { width: 18px; }
.committee-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin-top: 45px; }
.committee-grid article { min-height: 340px; padding: 45px; border: 1px solid #d7e0ef; background: #fff; }
.simple-cta { padding: 90px 0; text-align: center; }
.simple-cta h2 { margin: 0 0 15px; font-size: 38px; }
.simple-cta p { max-width: 680px; margin: 0 auto 28px; line-height: 1.75; }
.fellow-hero { position: relative; min-height: 610px; background-position: center; background-size: cover; color: #fff; }
.fellow-hero::before { content: ""; position: absolute; inset: 0; background: rgba(9,29,70,.72); }
.fellow-hero .site-container { position: relative; display: flex; min-height: 610px; flex-direction: column; justify-content: center; align-items: flex-start; }
.fellow-hero h1 { margin: 0; color: #fff; font-size: 58px; }
.fellow-hero p { max-width: 700px; margin: 24px 0 30px; font-size: 17px; line-height: 1.75; }
.fellow-program { padding: 105px 0; }
.fellow-program h2 { max-width: 780px; }
.fellow-program .lead { max-width: 850px; }
.benefit-grid { display: grid; grid-template-columns: repeat(3,1fr); margin-top: 55px; border-top: 1px solid #d7dfec; }
.benefit-grid article { padding: 42px 38px 0 0; }
.benefit-grid article + article { padding-left: 38px; border-left: 1px solid #d7dfec; }
.fellows-section > .site-container > p { margin: 15px 0 32px; }
.fellow-tools { display: flex; gap: 14px; margin-bottom: 20px; }
.fellow-tools label { display: flex; flex: 1; max-width: 520px; align-items: center; gap: 10px; padding: 0 16px; border: 1px solid #cfd8e8; background: #fff; }
.fellow-tools svg { width: 19px; }
.fellow-tools input, .fellow-tools select { min-height: 48px; border: 0; outline: 0; background: transparent; }
.fellow-tools input { width: 100%; }
.fellow-tools select, .fellow-tools + select { min-width: 230px; padding: 0 14px; border: 1px solid #cfd8e8; background: #fff; }
.result-count { display: block; margin-bottom: 22px; color: #1a2741; }
.fellow-grid { display: grid; grid-template-columns: repeat(3,1fr); gap: 22px; }
.fellow-grid article { display: flex; min-height: 300px; flex-direction: column; padding: 28px; border: 1px solid #d8e0ed; background: #fff; }
.fellow-grid h3 { margin: 13px 0 10px; font-size: 22px; }
.fellow-grid article > strong { color: #263651; }
.fellow-grid article > p { margin: 8px 0 18px; }
.credential, .tag-row span { align-self: flex-start; padding: 5px 10px; border: 1px solid #c7d8ff; border-radius: 15px; background: #f4f7ff; color: #2864ff; font-size: 12px; }
.tag-row { display: flex; flex-wrap: wrap; gap: 7px; }
.fellow-grid article > a { display: inline-flex; align-items: center; gap: 6px; margin-top: auto; padding-top: 20px; border-top: 1px solid #e2e7ef; color: #2864ff; }
.fellow-grid article > a svg { width: 16px; }
.faq-section { padding: 105px 0; }
.faq-section .site-container { max-width: 930px; }
.faq-section h2 { margin: 0 0 35px; font-size: 42px; }
.faq-section article { border-top: 1px solid #d7dfeb; }
.faq-section article:last-child { border-bottom: 1px solid #d7dfeb; }
.faq-section button { display: flex; width: 100%; align-items: center; justify-content: space-between; padding: 24px 0; border: 0; background: transparent; color: #17223b; font-size: 18px; text-align: left; }
.faq-section button svg { width: 20px; transition: transform .2s; }
.faq-section article p { margin: -5px 0 25px; line-height: 1.75; }
.rotated { transform: rotate(180deg); }
.youth-intro, .career-intro { display: grid; grid-template-columns: 1fr 1fr; gap: 80px; align-items: center; padding-top: 105px; padding-bottom: 105px; }
.youth-intro img, .career-intro img { width: 100%; min-height: 480px; object-fit: cover; }
.youth-intro a, .career-intro a { margin-top: 28px; }
.youth-features { padding: 105px 0; background: #f7f9fc; }
.youth-features h2, .youth-benefits h2 { margin: 0 0 15px; font-size: 42px; }
.youth-features .lead { max-width: 720px; }
.youth-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 22px; margin-top: 50px; }
.youth-grid article { display: grid; grid-template-columns: 180px 1fr; min-height: 220px; background: #fff; border: 1px solid #dae2ee; }
.youth-grid img { width: 100%; height: 100%; object-fit: cover; }
.youth-grid article div { align-self: center; padding: 28px; }
.youth-grid h3 { margin: 0 0 12px; font-size: 22px; }
.youth-grid p { margin: 0; line-height: 1.7; }
.youth-benefits { padding: 105px 0; background: #101f47; color: #fff; }
.youth-benefits .site-container { display: grid; grid-template-columns: 1fr 1fr; gap: 75px; align-items: center; }
.youth-benefits h2 { color: #fff; }
.youth-benefits p { color: rgba(255,255,255,.72); }
.youth-benefits img { width: 100%; height: 500px; object-fit: cover; }
.youth-benefits ul { display: grid; gap: 13px; padding: 20px 0 0; list-style: none; }
.youth-benefits li { display: flex; gap: 10px; }
.youth-benefits li svg { width: 18px; flex: 0 0 18px; color: #7ea3ff; }
.career-hero { min-height: 420px; }
.career-hero .site-container { display: flex; min-height: 420px; align-items: center; }
.career-hero h1 { max-width: 1050px; margin: 0; font-size: 48px; line-height: 1.25; }
.jobs-section { scroll-margin-top: 20px; }
.job-tabs { display: flex; gap: 5px; margin: 35px 0 28px; }
.job-tabs button { padding: 10px 21px; border: 1px solid #cfd8e7; background: #fff; color: #33415a; }
.job-tabs button.active { border-color: #2864ff; background: #2864ff; color: #fff; }
.job-list { display: grid; gap: 12px; }
.job-list article { display: grid; grid-template-columns: 48px 1fr 48px; gap: 18px; align-items: center; padding: 25px 28px; border: 1px solid #d9e1ed; background: #fff; }
.job-list article > svg { padding: 10px; width: 44px; height: 44px; background: #edf3ff; color: #2864ff; }
.job-list span { color: #2864ff; font-size: 13px; }
.job-list h3 { margin: 6px 0 10px; font-size: 21px; }
.job-list p { display: flex; align-items: center; gap: 13px; margin: 0; font-size: 14px; }
.job-list p svg { width: 16px; }
.job-list article > a { display: grid; width: 44px; height: 44px; place-items: center; border: 1px solid #cad5e7; border-radius: 50%; background: #fff; color: #1b4ec2; }
.job-list article > a svg { width: 18px; }
.contact-section { padding: 105px 0 115px; }
.contact-section h1 { max-width: 780px; margin: 0 0 20px; font-size: 45px; }
.contact-section > .site-container > p { max-width: 780px; margin: 0; line-height: 1.75; }
.contact-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin-top: 65px; }
.contact-grid article { min-height: 300px; padding: 38px; border: 1px solid #d8e1ee; background: rgba(255,255,255,.92); }
.contact-grid article > svg { width: 30px; color: #2864ff; }
.contact-grid h2 { margin: 22px 0 12px; font-size: 25px; }
.contact-grid p { min-height: 50px; margin: 0 0 24px; line-height: 1.65; }
.contact-grid article > span { margin-right: 18px; color: #8993a4; font-size: 13px; }
.contact-grid article > a:not(.contact-link) { color: #2b3b58; }
.contact-link { display: flex; width: max-content; align-items: center; gap: 6px; margin-top: 22px; padding-bottom: 6px; border-bottom: 1px solid #2864ff; color: #2864ff; }
.contact-link svg { width: 16px; }

@media (max-width: 900px) {
  .site-container { width: calc(100% - 40px); }
  .story-copy, .split-heading, .intro-section, .youth-intro, .career-intro, .youth-benefits .site-container { grid-template-columns: 1fr; gap: 38px; }
  .story-intro { padding-top: 45px; }
  .story-copy { padding-bottom: 48px; }
  .photo-slide { width: 18.75rem; flex-basis: 18.75rem; }
  .impact-layout { grid-template-columns: 1fr; gap: 65px; }
  .impact-copy, .impact-copy h2, .impact-lead, .impact-features { max-width: none; }
  .impact-stats { width: min(100%, 560px); margin: 0 auto; }
  .person-card, .person-card:nth-child(even) { grid-template-columns: 260px 1fr; }
  .person-card:nth-child(even) img { order: initial; }
  .person-card div { padding: 36px; }
  .fellow-grid, .benefit-grid { grid-template-columns: 1fr 1fr; }
  .benefit-grid article + article { border-left: 0; }
  .benefit-grid article:nth-child(2) { padding-left: 25px; border-left: 1px solid #d7dfec; }
  .youth-benefits img { height: 390px; }
}

@media (max-width: 650px) {
  .site-container { width: calc(100% - 32px); }
  .story-copy h1, .section-heading h2, .people-section > .site-container > h2, .committee-section h2, .fellows-section h2, .jobs-section h2, .youth-features h2, .youth-benefits h2 { font-size: 32px; }
  .story-copy h1 strong { white-space: normal; }
  .story-doodle { top: -30px; left: 38%; width: 56px; }
  .photo-slide { height: 24rem; flex-basis: 18.75rem; }
  .photo-set:first-child .photo-slide:nth-child(even), .photo-set:nth-child(2) .photo-slide:nth-child(odd) { height: 20rem; }
  .impact-section, .people-section, .committee-section, .fellows-section, .jobs-section, .fellow-program, .faq-section, .youth-features, .youth-benefits, .contact-section { padding: 72px 0; }
  .impact-copy h2 { font-size: 31px; }
  .impact-features article { grid-template-columns: 48px 1fr; gap: 16px; }
  .feature-icon { width: 48px; height: 48px; }
  .impact-stats { border-left: 0; }
  .stat-row { min-height: 128px; }
  .stat-number strong { font-size: 35px; }
  .stat-row > * { padding: 22px 10px; }
  .committee-grid, .fellow-grid, .benefit-grid, .youth-grid, .contact-grid { grid-template-columns: 1fr; }
  .title-hero { min-height: 280px; }
  .title-hero h1, .fellow-hero h1 { font-size: 39px; }
  .intro-section { padding-top: 65px; padding-bottom: 65px; }
  .intro-section > img, .youth-intro img, .career-intro img { height: 330px; min-height: 0; }
  .intro-section h2, .youth-intro h2, .career-intro h2, .fellow-program h2 { font-size: 31px; }
  .person-card, .person-card:nth-child(even) { grid-template-columns: 1fr; }
  .person-card img { height: 360px; }
  .governance-cta .site-container { align-items: flex-start; flex-direction: column; }
  .fellow-hero, .fellow-hero .site-container { min-height: 530px; }
  .benefit-grid { border-top: 0; }
  .benefit-grid article, .benefit-grid article:nth-child(2), .benefit-grid article + article { padding: 30px 0; border-top: 1px solid #d7dfec; border-left: 0; }
  .fellow-tools { flex-direction: column; }
  .fellow-tools select { min-height: 48px; }
  .youth-grid article { grid-template-columns: 1fr; }
  .youth-grid img { height: 220px; }
  .career-hero, .career-hero .site-container { min-height: 350px; }
  .career-hero h1, .contact-section h1 { font-size: 35px; }
  .job-list article { grid-template-columns: 42px 1fr; padding: 20px; }
  .job-list article > a { grid-column: 2; }
  .job-list p { align-items: flex-start; flex-wrap: wrap; }
  .contact-grid article > a:not(.contact-link) { font-size: 13px; }
}
</style>
