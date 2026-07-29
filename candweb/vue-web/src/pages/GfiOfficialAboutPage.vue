<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute } from "vue-router"
import { ArrowLeft, ArrowRight, ArrowUpRight, Check, ChevronDown, Search } from "lucide-vue-next"
import GfiFooter from "@/components/GfiFooter.vue"
import GfiHeader from "@/components/GfiHeader.vue"
import { useTranslation } from "@/lib/language"
import { localize } from "@/lib/gfiSite"
import { contacts, fellowBenefits, fellows, jobLinks, jobs, peoplePages, tx } from "@/lib/gfiAboutPages"

const route = useRoute()
const { lang } = useTranslation()
const pageKey = computed(() => route.path.replace(/^\/gfi\/?/, "").replace(/\/$/, ""))
const peoplePage = computed(() => peoplePages[pageKey.value as keyof typeof peoplePages])
const l = (value: ReturnType<typeof tx>) => localize(value, lang.value)
const search = ref("")
const selectedTag = ref("")
const jobFilter = ref<"all" | "full" | "intern">("all")
const openFaq = ref<number | null>(0)
const openYouthFaq = ref<number | null>(0)
const expandedPeople = ref<string[]>([])
const fellowViewport = ref<HTMLElement | null>(null)
let revealObserver: IntersectionObserver | null = null

const fellowTags = computed(() => [...new Set(fellows.flatMap((item) => item.tags.map((tag) => l(tag))))])
const filteredFellows = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  return fellows.filter((item) => {
    const tags = item.tags.map((tag) => l(tag))
    const haystack = [l(item.name), l(item.role), l(item.org), ...tags].join(" ").toLocaleLowerCase()
    return (!selectedTag.value || tags.includes(selectedTag.value)) && (!query || haystack.includes(query))
  })
})
const filteredJobs = computed(() => jobs.filter((job) => {
  if (jobFilter.value === "all") return true
  const type = l(job[2])
  return jobFilter.value === "intern" ? /实习|intern/i.test(type) : !/实习|intern/i.test(type)
}))

const industryFaqs = [
  [tx("什么是GFI的行业研究员？", "What is a GFI Industry Fellow?"), tx("全球金融科技学院的行业研究员是一位资深从业者，因其在金融科技生态系统中的专业知识、领导力和持续贡献而获得认可。研究员通过指导、对话以及对教育、政策讨论和行业标准的贡献，在推进负责任的金融科技实践中发挥积极作用。", "A GFI Industry Fellow is a senior practitioner recognised for expertise, leadership and sustained contribution to the fintech ecosystem.")],
  [tx("研究员是如何选出的？", "How are Fellows selected?"), tx("行业研究员通过邀请方式任命，基于其在金融、技术、监管或相关领域的专业地位、领域专业知识和已证明的影响力。选拔考虑个人独立、建设性和可信地贡献于GFI使命和社区的能力。", "Industry Fellows are appointed by invitation based on professional standing, domain expertise and demonstrated impact.")],
  [tx("时间承诺是什么？", "What is the time commitment?"), tx("时间承诺是灵活的，因个人而异。贡献可能包括参与选定的讨论、为项目提供建议、指导专业人士或支持特定倡议。没有固定的最低要求，但期望研究员随着时间的推移进行有意义的参与。", "The time commitment is flexible and varies by individual.")],
  [tx("组织可以提名候选人吗？", "Can organisations nominate candidates?"), tx("可以。组织可以提名高级领导者或主题专家作为行业研究员的候选人。所有提名都会经过审查，以确保与GFI的标准、价值观和重点领域保持一致。", "Yes. Organisations may nominate senior leaders or subject-matter experts.")],
  [tx("研究员如何为GFI项目和倡议做出贡献？", "How do Fellows contribute?"), tx("行业研究员通过在认证、高管项目、研究、政策对话和社区倡议中分享专业知识来做出贡献。这可能包括就课程相关性提供建议、参与闭门讨论、指导专业人士、为出版物贡献见解或支持生态系统合作。", "Industry Fellows contribute across certifications, executive programmes, research, policy dialogue and community initiatives.")],
] as const

const youthFeatures = [
  [tx("实践经验", "Real-world Experience"), tx("通过在GFI会议、小组讨论和闭门圆桌会议上做志愿者，获得实际接触机会。", "Gain practical exposure by volunteering at GFI conferences, panels and closed-door roundtables."), "real-world-experience.cxQG5lJx_TS6nf.webp"],
  [tx("金融科技学习机会", "Fintech Learning Opportunities"), tx("通过适合学生的会议和活动，探索支付、人工智能、区块链、监管和风险等主题。", "Explore payments, AI, blockchain, regulation and risk through student-friendly sessions and events."), "fintech-learning-opportunities.Bl_g268g_ZeD3mQ.webp"],
  [tx("职业与认证途径", "Career & Certification Pathways"), tx("通过特许金融科技助理（CFtA）计划，发现进入金融科技角色的途径。", "Discover pathways into fintech roles through the Chartered Fintech Associate (CFtA) programme."), "career-certification-pathways.C-H--jds_J5Bky.webp"],
  [tx("社区与校园参与", "Community & Campus Engagement"), tx("通过共享的倡议和项目，与各大学、金融科技社团和青年组织的同行建立联系。", "Connect with peers across universities, fintech societies and youth organisations through shared initiatives."), "community-campus-engagement.CKXT0_f0_bmAh3.webp"],
] as const
const youthEngagement = [
  [tx("校园外展", "Campus Outreach"), tx("通过学生主导的会议、大学路演和跨校园的协作活动，探索金融科技。", "Explore fintech through student-led sessions, university roadshows and cross-campus collaboration."), "campus-outreach.CasbyaLu_Z1hfUj7.webp"],
  [tx("志愿服务机会", "Volunteering Opportunities"), tx("支持GFI的活动、小组讨论和项目，获得实际接触机会，并直接参与行业对话。", "Support GFI events, panels and programmes while gaining practical exposure."), "volunteering-opportunities.DHVAShHq_Z5AeHj.webp"],
  [tx("行业参与", "Industry Engagement"), tx("通过GFI的全球网络和倡议，与金融科技专业人士、组织和生态系统合作伙伴建立联系。", "Connect with fintech professionals and ecosystem partners through GFI's global network."), "industry-engagement.rJw7XPD8_PlQp1.webp"],
] as const
const youthFaqs = [
  [tx("什么是GFI青年部？", "What is the GFI Youth Wing?"), tx("青年部是一个为希望获得金融科技行业实践经验的学生和早期职业人士而设的社区。成员支持GFI活动，参加学习会议，并与整个生态系统的专业人士建立联系。", "The Youth Wing is a community for students and early-career individuals seeking practical fintech experience.")],
  [tx("谁可以加入？", "Who can join?"), tx("任何对金融科技感兴趣的人都欢迎加入——大学生、理工学院学生和早期职业人士。不需要先前的金融科技知识。", "University students, polytechnic students and early-career professionals are welcome.")],
  [tx("加入是免费的吗？", "Is it free to join?"), tx("是的。加入青年部是免费的。可选途径如CFtA认证有单独的费用。", "Yes. Joining the Youth Wing is free.")],
  [tx("作为青年部成员，我将做什么？", "What will I do as a member?"), tx("您可以根据自己的兴趣选择活动：\n• 在GFI小组讨论和会议上做志愿者\n• 参加学习会议\n• 帮助校园外展\n• 支持内容、研究或活动后勤\n• 加入GFI主导的项目", "You can choose activities based on your interests: volunteer at GFI panels and conferences, attend learning sessions, help with campus outreach, support content, research or event logistics, and join GFI-led projects.")],
  [tx("我需要投入多少时间？", "How much time is required?"), tx("大多数角色都很灵活——通常每月3到6小时。您可以报名参加适合您时间表的机会。", "Most roles are flexible, typically three to six hours per month.")],
  [tx("我能参加GFI活动吗？", "Can I attend GFI events?"), tx("是的。青年部成员经常支持GFI活动，并获得与演讲者、讨论和行业见解的近距离接触。您也可以在LinkedIn上关注GFI以获取更新和机会。", "Yes. Youth Wing members often support GFI events and gain close access to speakers, discussions and industry insights. You can also follow GFI on LinkedIn for updates and opportunities.")],
  [tx("我需要金融科技背景才能加入吗？", "Do I need a fintech background?"), tx("不需要。欢迎所有院系和学科的学生。兴趣和好奇心就足够了。", "No. Students from all faculties and disciplines are welcome.")],
  [tx("什么是CFtA认证？", "What is the CFtA certification?"), tx("特许金融科技助理（CFtA）认证建立数字金融的基础知识，涵盖支付、人工智能、区块链、风险和监管。建议（但不是强制）青年部成员参加。了解更多：https://bit.ly/cfta-apply", "The Chartered Fintech Associate (CFtA) certification builds foundational knowledge in digital finance, covering payments, AI, blockchain, risk and regulation. It is recommended, but not mandatory, for Youth Wing members.")],
  [tx("有领导机会吗？", "Are there leadership opportunities?"), tx("是的。随着青年部的扩展，选定的志愿者可以担任活动协调、外展、沟通或项目领导等角色——非常适合获得作品集和简历经验。", "Yes. As the Youth Wing expands, selected volunteers may take on roles in event coordination, outreach, communications or project leadership, ideal for building portfolio and resume experience.")],
] as const

const togglePerson = (name: string) => expandedPeople.value = expandedPeople.value.includes(name) ? expandedPeople.value.filter((item) => item !== name) : [...expandedPeople.value, name]
const scrollFellows = (direction: number) => fellowViewport.value?.scrollBy({ left: direction * fellowViewport.value.clientWidth * .86, behavior: "smooth" })
const setupReveal = () => {
  revealObserver?.disconnect()
  revealObserver = new IntersectionObserver((entries) => entries.forEach((entry) => {
    if (!entry.isIntersecting) return
    entry.target.classList.add("is-revealed")
    revealObserver?.unobserve(entry.target)
  }), { threshold: .08, rootMargin: "0px 0px -25px" })
  document.querySelectorAll(".official-about [data-reveal]").forEach((element) => revealObserver?.observe(element))
}
watch(pageKey, async () => {
  search.value = ""; selectedTag.value = ""; jobFilter.value = "all"; openFaq.value = 0; openYouthFaq.value = 0; expandedPeople.value = []
  await nextTick(); setupReveal()
})
onMounted(async () => { await nextTick(); setupReveal() })
onBeforeUnmount(() => revealObserver?.disconnect())
</script>

<template>
  <div class="official-about">
    <GfiHeader theme="light" />

    <main v-if="peoplePage">
      <section class="page-title patterned"><div data-reveal><h1>{{ l(peoplePage.title) }}</h1><nav><RouterLink to="/gfi">Home</RouterLink><i>/</i><RouterLink to="/gfi">Cn</RouterLink><i>/</i><RouterLink to="/gfi/about">About</RouterLink><i>/</i><b>{{ peoplePage.title.en }}</b></nav></div></section>
      <section class="overlap-intro container" data-reveal><img src="https://globalfintechinstitute.org/assets/gfi-about-1.BPJKq5hL_1NRqCB.webp" :alt="l(peoplePage.introTitle)"><div><h2>{{ l(peoplePage.introTitle) }}</h2><p>{{ l(peoplePage.intro) }}</p></div></section>
      <section class="people-section"><div class="container"><header data-reveal><span>{{ l(peoplePage.eyebrow) }}</span><h2>{{ l(peoplePage.membersTitle) }}</h2></header><article v-for="person in peoplePage.members" :key="person.name" class="person-row" data-reveal><img :src="person.image" :alt="person.name"><div><h3>{{ person.name }}</h3><strong>{{ person.role.en }}</strong><p :class="{ clamped: !expandedPeople.includes(person.name) }">{{ person.bio.en }}</p><button @click="togglePerson(person.name)">{{ expandedPeople.includes(person.name) ? "Read less" : "Read more" }} <ChevronDown :class="{ rotated: expandedPeople.includes(person.name) }" /></button></div></article></div></section>
      <section class="round-cta" data-reveal><div class="container"><div><h2>{{ lang === "zh" ? "从治理到集体领导" : "From Governance to Collective Leadership" }}</h2><p>{{ lang === "zh" ? "除了董事会，GFI的工作还通过其委员会和理事会推进——这些由实践者领导的团体贡献专业知识，指导标准，并在教育、政策和行业参与方面塑造项目。" : "Beyond the Board, GFI's work advances through committees and councils led by practitioners." }}</p><RouterLink to="/gfi/subcommittees">{{ lang === "zh" ? "探索小组委员会" : "Explore SubCommittees" }} <ArrowUpRight /></RouterLink></div><span></span></div></section>
    </main>

    <main v-else-if="pageKey === 'subcommittees'">
      <section class="page-title patterned"><div data-reveal><h1>{{ lang === "zh" ? "子委员会" : "SubCommittees" }}</h1><nav><RouterLink to="/gfi">Home</RouterLink><i>/</i><RouterLink to="/gfi">Cn</RouterLink><i>/</i><b>Subcommittees</b></nav></div></section>
      <section class="overlap-intro container" data-reveal><img src="https://globalfintechinstitute.org/assets/gfi-about-1.BPJKq5hL_1NRqCB.webp" alt="GFI"><div><h2>{{ lang === "zh" ? "支持GFI使命的专业专长" : "Specialist Expertise Supporting GFI's Mission" }}</h2><p>{{ lang === "zh" ? "GFI子委员会是在GFI委员会下设立的特定领域工作组，旨在解决金融科技生态系统中新兴、复杂和高影响力的领域。它们汇聚高级从业者和主题专家，以开发见解、指导项目发展并支持知情的行业对话。在GFI的治理框架内运作，子委员会将战略方向转化为与不断发展的技术和监管现实相一致的专注、应用性工作。" : "GFI SubCommittees are domain-specific working groups established to address emerging, complex and high-impact areas across the fintech ecosystem." }}</p></div></section>
      <section class="committee-section"><div class="container"><header data-reveal><span>{{ lang === "zh" ? "子委员会" : "SubCommittees" }}</span><h2>{{ lang === "zh" ? "GFI子委员会" : "GFI SubCommittees" }}</h2></header><div class="committee-grid"><article data-reveal><b>01</b><h3>{{ lang === "zh" ? "新兴技术融合（人工智能、区块链与量子计算）" : "Emerging Technology Convergence (AI, Blockchain & Quantum Computing)" }}</h3><p>{{ lang === "zh" ? "本子委员会探讨人工智能、区块链和量子计算等先进技术如何融合以重塑金融服务。重点关注治理、风险、安全与监管考量，支持金融科技生态系统内的知情采用与负责任创新。" : "This SubCommittee explores how advanced technologies converge to reshape financial services." }}</p><a href="https://globalfintechinstitute.org/cn/subcommittees/convergence-emerging-technologies/" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "了解更多" : "Learn more" }} <ArrowUpRight /></a></article><article data-reveal><b>02</b><h3>{{ lang === "zh" ? "数字资产安全与合规子委员会" : "Digital Assets Security & Compliance SubCommittee" }}</h3><p>{{ lang === "zh" ? "本子委员会致力于加强数字资产生态系统的安全性、合规性与运营韧性。它涵盖托管、基础设施风险、监管预期与治理等关键议题，为在复杂且高度监管环境中运营的机构提供支持。" : "This SubCommittee strengthens security, compliance and operational resilience across digital asset ecosystems." }}</p><a href="https://globalfintechinstitute.org/cn/subcommittees/digital-assets-security-compliance/" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "了解更多" : "Learn more" }} <ArrowUpRight /></a></article></div></div></section>
      <section class="round-cta" data-reveal><div class="container"><div><h2>{{ lang === "zh" ? "有兴趣参与？" : "Interested in Contributing?" }}</h2><p>{{ lang === "zh" ? "若您希望探讨贡献专业能力的机会，欢迎与我们联系或进一步了解我们的治理机构。" : "Contact us to explore opportunities to contribute your expertise." }}</p><RouterLink to="/gfi/contact">{{ lang === "zh" ? "联系我们" : "Contact Us" }} <ArrowUpRight /></RouterLink></div><span></span></div></section>
    </main>

    <main v-else-if="pageKey === 'industry-fellow'">
      <section class="fellow-hero" style="background-image:url('https://globalfintechinstitute.org/assets/gfi-banner-3.X5tYPN3w_ZLlOGO.webp')"><div class="container" data-reveal><h1>{{ lang === "zh" ? "行业研究员" : "Industry Fellows" }}</h1><p>{{ lang === "zh" ? "一个邀请制的专业网络，由高级领导者组成，通过教育、对话和生态系统合作，贡献专业知识以推进负责任的金融科技实践。" : "An invitation-only professional network of senior leaders contributing expertise through education, dialogue and ecosystem collaboration." }}</p><a href="https://airtable.com/appCg8CSsvuJBv582/pagDK5m706Bo4Qect/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "表达兴趣" : "Express Interest" }} <ArrowUpRight /></a></div></section>
      <section class="fellow-program"><div class="container program-layout"><div data-reveal><span class="pill">{{ lang === "zh" ? "行业研究员计划" : "Industry Fellow Programme" }}</span><h2>{{ lang === "zh" ? "领导力平台，而不仅仅是认可" : "A Platform for Leadership, Not Just Recognition" }}</h2><p>{{ lang === "zh" ? "行业研究员计划汇集了金融科技各个垂直领域的资深专业人士，通过指导、政策和标准对话，以及与包括监管机构、投资者和行业领导者在内的主要利益相关者的合作，加强生态系统。" : "The programme brings together senior professionals across fintech verticals to strengthen the ecosystem." }}</p><article v-for="(benefit,index) in fellowBenefits" :key="l(benefit[0])"><b>0{{ index + 1 }}</b><div><h3>{{ l(benefit[0]) }}</h3><p>{{ l(benefit[1]) }}</p></div></article></div><img data-reveal src="https://globalfintechinstitute.org/assets/image.Aw2I5NvI_1JP4zj.webp" alt="Industry Fellow Programme"></div></section>
      <section class="fellows-section"><div class="container"><header data-reveal><h2>{{ lang === "zh" ? "认识GFI行业研究员" : "Meet GFI Industry Fellows" }}</h2><p>{{ lang === "zh" ? "按领域搜索或筛选，了解我们的资深从业者网络。" : "Search or filter by domain to discover our network of senior practitioners." }}</p></header><p class="search-label">{{ lang === "zh" ? "搜索行业研究员" : "Search Industry Fellows" }}</p><label class="fellow-search"><Search /><input v-model="search" :placeholder="lang === 'zh' ? '按姓名、职位、机构或领域搜索…' : 'Search by name, role, organisation or domain…'"></label><p class="filter-label">{{ lang === "zh" ? "按领域专长筛选：" : "Filter by expertise:" }}</p><div class="filter-chips"><button :class="{ active: !selectedTag }" @click="selectedTag=''">{{ lang === "zh" ? "全部领域" : "All Domains" }}</button><button v-for="tag in fellowTags" :key="tag" :class="{ active:selectedTag===tag }" @click="selectedTag=tag">{{ tag }}</button></div><button v-if="search || selectedTag" class="clear-filters" @click="search=''; selectedTag=''">{{ lang === "zh" ? "清除筛选" : "Clear Filters" }}</button><strong class="result-count">{{ filteredFellows.length }} {{ lang === "zh" ? "位行业研究员" : "Industry Fellows" }}</strong><div v-if="filteredFellows.length" ref="fellowViewport" class="fellow-viewport"><article v-for="item in filteredFellows" :key="l(item.name)"><span v-if="l(item.name).includes('CFtP')">CFtP®</span><h3>{{ l(item.name) }}</h3><strong>{{ l(item.role) }}</strong><p>{{ l(item.org) }}</p><div><em v-for="tag in item.tags" :key="l(tag)">{{ l(tag) }}</em></div><a :href="item.url" target="_blank" rel="noopener noreferrer">View profile <ArrowUpRight /></a></article></div><div v-else class="empty-fellows"><h3>{{ lang === "zh" ? "未找到行业研究员" : "No Industry Fellows Found" }}</h3><p>{{ lang === "zh" ? "请尝试修改搜索词或选择「全部领域」。" : "Try adjusting your search or select All Domains." }}</p><button @click="search=''; selectedTag=''">{{ lang === "zh" ? "清除筛选" : "Clear Filters" }}</button></div><div v-if="filteredFellows.length" class="carousel-controls"><button @click="scrollFellows(-1)" aria-label="Previous"><ArrowLeft /></button><i></i><button @click="scrollFellows(1)" aria-label="Next"><ArrowRight /></button></div></div></section>
      <section class="faq-section"><div class="container"><h2 data-reveal>FAQs</h2><article v-for="(faq,index) in industryFaqs" :key="l(faq[0])" data-reveal><button @click="openFaq=openFaq===index?null:index"><span>{{ l(faq[0]) }}</span><ChevronDown :class="{ rotated:openFaq===index }" /></button><p v-if="openFaq===index">{{ l(faq[1]) }}</p></article></div></section>
      <section class="round-cta" data-reveal><div class="container"><div><small>{{ lang === "zh" ? "行业研究员计划" : "Industry Fellow Programme" }}</small><h2>{{ lang === "zh" ? "帮助塑造负责任的金融科技实践" : "Help Shape Responsible Fintech Practice" }}</h2><p>{{ lang === "zh" ? "如果您是一位具有影响力记录并致力于推进专业标准的资深从业者，我们欢迎您表达兴趣。" : "If you are a senior practitioner committed to advancing professional standards, we welcome your interest." }}</p><a href="https://airtable.com/appCg8CSsvuJBv582/pagDK5m706Bo4Qect/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "表达兴趣" : "Express Interest" }} <ArrowUpRight /></a></div><span></span></div></section>
    </main>

    <main v-else-if="pageKey === 'youth-wing'">
      <section class="page-title patterned"><div data-reveal><h1>{{ lang === "zh" ? "GFI 青年部" : "GFI Youth Wing" }}</h1><nav><RouterLink to="/gfi">Home</RouterLink><i>/</i><RouterLink to="/gfi">Cn</RouterLink><i>/</i><b>Youth Wing</b></nav></div></section>
      <section class="overlap-intro container" data-reveal><img src="https://globalfintechinstitute.org/assets/youth-wing-hero.tPwGWJkI_ZkzQJw.webp" alt="GFI Youth Wing"><div><h2>{{ lang === "zh" ? "一个为塑造金融科技未来的学生和年轻专业人士而设的社区。" : "A Community for Students and Young Professionals Shaping Fintech's Future." }}</h2><p>{{ lang === "zh" ? "GFI青年部是全球金融科技研究所的学生和早期职业社区。我们汇聚了对数字金融充满好奇、渴望获得实践经验、并准备参与塑造行业的真实对话的年轻人。" : "The GFI Youth Wing is our student and early-career community." }}</p><a href="https://airtable.com/appY1MeInT0J7XPkm/pagyCg86VFuc5foSn/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "立即注册" : "Register Now" }} <ArrowUpRight /></a></div></section>
      <section class="youth-features"><div class="container"><header data-reveal><h2>{{ lang === "zh" ? "塑造未来的金融科技人才" : "Shaping Future Fintech Talent" }}</h2><p>{{ lang === "zh" ? "为青年提供知识、经验和网络，使他们能够有意义地参与全球金融科技生态系统。我们通过以下方式实现：" : "Giving young people the knowledge, experience and networks to participate meaningfully in the global fintech ecosystem." }}</p></header><div class="youth-grid"><article v-for="feature in youthFeatures" :key="l(feature[0])" data-reveal><img :src="`https://globalfintechinstitute.org/assets/${feature[2]}`" :alt="l(feature[0])"><div><h3>{{ l(feature[0]) }}</h3><p>{{ l(feature[1]) }}</p></div></article></div></div></section>
      <section class="youth-benefits"><div class="container"><div data-reveal><h2>{{ lang === "zh" ? "我们要建立和推动的目标" : "What We Aim to Build and Advance" }}</h2><p>{{ lang === "zh" ? "一个由学生主导的空间，用于学习、贡献和探索金融科技领域的真实机会——从行业活动到基础培训，再到跨校园的社区建设。" : "A student-led space to learn, contribute and explore real opportunities in fintech." }}</p><ul><li v-for="item in (lang==='zh'?['在行业小组讨论、会议和圆桌会议上做志愿者。','支持活动运营、研究和数字内容。','通过CFtA计划获得基础金融科技学习。','与各大学和金融科技社区的同行建立联系。','加入学生主导的项目、学习圈和青年部倡议。','探索支付、人工智能、区块链、政策和风险等领域的真实角色。']:['Volunteer at industry panels, conferences and roundtables.','Support event operations, research and digital content.','Build foundational fintech knowledge through CFtA.','Connect with peers across universities and fintech communities.','Join student-led projects and Youth Wing initiatives.','Explore real roles across fintech domains.'])" :key="item"><Check />{{ item }}</li></ul><a href="https://airtable.com/appY1MeInT0J7XPkm/pagyCg86VFuc5foSn/form" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "加入GFI青年部" : "Join the GFI Youth Wing" }} <ArrowUpRight /></a></div><img data-reveal src="https://globalfintechinstitute.org/assets/gfi-benefits.CSK4AmvV_7iTVc.webp" alt="GFI Benefits"></div></section>
      <section class="engagement-section"><div class="container"><header data-reveal><span>{{ lang === "zh" ? "参与其中" : "Get Involved" }}</span><h2>{{ lang === "zh" ? "关注青年部举办的活动与参与机会" : "Look Out for Youth Wing Events and Opportunities" }}</h2></header><div><article v-for="item in youthEngagement" :key="l(item[0])" data-reveal><img :src="`https://globalfintechinstitute.org/assets/${item[2]}`" :alt="l(item[0])"><h3>{{ l(item[0]) }}</h3><p>{{ l(item[1]) }}</p></article></div></div></section>
      <section class="faq-section youth-faq"><div class="container"><header class="faq-heading" data-reveal><h2>{{ lang === "zh" ? "仍然不确定？我们为您提供支持" : "Still Unsure? We Have You Covered" }}</h2><a href="https://globalfintechinstitute.org/faq/" target="_blank" rel="noopener noreferrer">{{ lang === "zh" ? "查看所有常见问题" : "View All FAQs" }} <ArrowUpRight /></a></header><article v-for="(faq,index) in youthFaqs" :key="l(faq[0])" data-reveal><button @click="openYouthFaq=openYouthFaq===index?null:index"><span>{{ l(faq[0]) }}</span><ChevronDown :class="{ rotated:openYouthFaq===index }" /></button><p v-if="openYouthFaq===index">{{ l(faq[1]) }}</p></article></div></section>
      <section class="cfta-section" data-reveal><div class="container"><div><h2>{{ lang === "zh" ? "认证途径：CFtA" : "Certification Pathway: CFtA" }}</h2><p>{{ lang === "zh" ? "特许金融科技助理（CFtA）认证是青年部成员的推荐基础，提供数字支付、金融中的人工智能和数据、区块链和基础设施、监管和合规以及风险和金融素养方面的基础知识。优秀成员也可能被考虑获得未来的奖学金机会。" : "The CFtA certification is the recommended foundation for Youth Wing members." }}</p><RouterLink to="/gfi/programmes/cfta">{{ lang === "zh" ? "了解更多" : "Learn More" }} <ArrowUpRight /></RouterLink></div><strong>10+<small>{{ lang === "zh" ? "大学与机构" : "Universities & Institutions" }}</small></strong><img src="https://globalfintechinstitute.org/assets/youth-wing-hero.tPwGWJkI_ZkzQJw.webp" alt="CFtA"></div></section>
    </main>

    <main v-else-if="pageKey === 'career'">
      <section class="page-title patterned career-title"><div data-reveal><h1>{{ lang === "zh" ? "立即加入GFI,在金融科技领域建立您的职业生涯" : "Join GFI and Build Your Career in Fintech" }}</h1><nav><RouterLink to="/gfi">Home</RouterLink><i>/</i><RouterLink to="/gfi">Cn</RouterLink><i>/</i><b>Career</b></nav></div></section>
      <section class="overlap-intro container" data-reveal><img src="https://globalfintechinstitute.org/assets/intro.CZOZAx-m_ZzLPPa.webp" alt="Careers at GFI"><div><h2>{{ lang === "zh" ? "立即加入GFI,在金融科技领域建立您的职业生涯" : "Join GFI and Build Your Career in Fintech" }}</h2><p>{{ lang === "zh" ? "加入我们,共同塑造金融科技的未来。在GFI,我们为专业人士、学生和志愿者提供参与研究、教育和社区建设的机会。无论是通过全职职位、实习还是志愿者岗位,您都将成为推动金融科技创新、标准和影响力的全球网络的一部分。" : "Join us in shaping the future of fintech. GFI offers professionals, students and volunteers opportunities across research, education and community building." }}</p><a href="#openings">{{ lang === "zh" ? "查看所有职位" : "View All Positions" }} <ArrowUpRight /></a></div></section>
      <section id="openings" class="jobs-section"><div class="container"><h2 data-reveal>{{ lang === "zh" ? "当前职位" : "Current Openings" }}</h2><div class="job-tabs"><button :class="{active:jobFilter==='all'}" @click="jobFilter='all'">All</button><button :class="{active:jobFilter==='full'}" @click="jobFilter='full'">Full-time</button><button :class="{active:jobFilter==='intern'}" @click="jobFilter='intern'">Internship</button></div><div v-if="filteredJobs.length" class="job-list"><article v-for="job in filteredJobs" :key="l(job[1])" data-reveal><span>{{ l(job[0]) }}</span><h3>{{ l(job[1]) }}</h3><p><strong>{{ l(job[2]) }}</strong><i></i>{{ l(job[3]) }}</p><a :href="`https://globalfintechinstitute.org/cn/career/${jobLinks[jobs.indexOf(job)]}/`" target="_blank" rel="noopener noreferrer">View More Details <ArrowUpRight /></a></article></div><p v-else class="no-jobs">No openings found for this filter.</p></div></section>
    </main>

    <main v-else-if="pageKey === 'contact'">
      <section class="contact-section patterned"><div class="container"><header data-reveal><h1>{{ lang === "zh" ? "与全球金融科技学院取得联系" : "Get in Touch with the Global Fintech Institute" }}</h1><p>{{ lang === "zh" ? "无论您是在探索我们的认证、合作伙伴关系还是即将推出的计划，我们的团队随时为您提供帮助。请通过以下相关联系方式与我们联系，以便我们能够高效地回复。我们的团队通常在2-3个工作日内回复。" : "Whether you are exploring our certifications, partnerships or upcoming programmes, our team is here to help. We typically reply within 2-3 business days." }}</p></header><div class="contact-grid"><article v-for="item in contacts" :key="l(item[0])" data-reveal><h2>{{ l(item[0]) }}</h2><p>{{ l(item[1]) }}</p><div><span>Email</span><i></i><a :href="`mailto:${item[2]}`">{{ item[2] }}</a></div><a class="contact-link" :href="`mailto:${item[2]}`">Contact Us <ArrowUpRight /></a></article></div></div></section>
    </main>

    <GfiFooter />
  </div>
</template>

<style scoped>
.official-about { min-height:100vh; overflow:hidden; background:#fff; color:#4d5668; font-family:Inter,"PingFang SC","Microsoft YaHei",Arial,sans-serif; letter-spacing:0; }
.official-about * { box-sizing:border-box; }
.official-about a { text-decoration:none; }
.container { width:min(1288px,calc(100% - 64px)); margin:0 auto; }
h1,h2,h3,p { overflow-wrap:anywhere; }
h1,h2,h3 { color:#101c3a; font-weight:500; }
[data-reveal] { opacity:0; transform:translateY(24px); transition:opacity .7s ease,transform .7s ease; }
[data-reveal].is-revealed { opacity:1; transform:translateY(0); }
.patterned { position:relative; overflow:hidden; background-color:#f7f9fc; background-image:repeating-linear-gradient(74deg,transparent 0,transparent 92px,rgba(255,255,255,.94) 93px,rgba(255,255,255,.94) 95px,transparent 96px,transparent 188px); }
.patterned::after { content:""; position:absolute; top:-85px; left:58%; width:1px; height:260px; background:#4e7fff; transform:rotate(-8deg); animation:blue-line 5s linear infinite; }
@keyframes blue-line { 0%,15%{opacity:0;transform:translateY(-90px) rotate(-8deg)} 35%,65%{opacity:1} 100%{opacity:0;transform:translateY(280px) rotate(-8deg)} }
.page-title { display:grid; min-height:315px; place-items:center; text-align:center; }
.page-title > div { position:relative; z-index:1; }
.page-title h1 { margin:0 0 28px; font-size:49px; line-height:1.2; }
.page-title nav { display:flex; min-height:42px; align-items:center; gap:24px; padding:0 24px; border:1px solid #dce1e8; border-radius:24px; background:#fff; font-size:13px; }
.page-title nav a { color:#2864ff; }
.page-title nav i { color:#dde2eb; font-style:normal; }
.page-title nav b { color:#838a96; font-weight:400; }
.career-title h1 { font-size:43px; }
.overlap-intro { position:relative; padding-top:70px; padding-bottom:120px; }
.overlap-intro > img { display:block; width:100%; height:560px; object-fit:cover; }
.overlap-intro > div { position:absolute; bottom:72px; left:92px; width:min(880px,calc(100% - 180px)); min-height:205px; padding:52px 44px; background:#fff; box-shadow:0 16px 38px rgba(25,43,79,.05); }
.overlap-intro h2 { margin:0 0 22px; color:#2864ff; font-size:34px; line-height:1.25; }
.overlap-intro p { margin:0; color:#3e4655; font-size:14px; line-height:1.7; }
.overlap-intro a,.round-cta a,.fellow-hero a,.youth-benefits a,.cfta-section a,.committee-grid a { display:inline-flex; align-items:center; gap:8px; margin-top:20px; color:#2864ff; }
.overlap-intro a svg,.round-cta a svg,.fellow-hero a svg,.youth-benefits a svg,.cfta-section a svg,.committee-grid a svg { width:16px; }
.people-section { padding:58px 0 112px; background:#fff; }
.people-section header,.committee-section header,.engagement-section header { margin-bottom:45px; }
.people-section header span,.committee-section header span,.engagement-section header span,.pill { display:inline-flex; margin-bottom:14px; padding:5px 16px; border:1px solid #cbd9fa; border-radius:18px; background:#f4f7ff; color:#2864ff; font-size:13px; }
.people-section header h2,.committee-section header h2,.engagement-section header h2 { margin:0; font-size:38px; }
.person-row { display:grid; grid-template-columns:92px 1fr; gap:30px; padding:27px 0 35px; border-bottom:1px solid #dfe4ec; }
.person-row > img { width:92px; height:92px; border-radius:50%; object-fit:cover; object-position:top; }
.person-row h3 { margin:0 0 6px; font-size:21px; font-weight:600; }
.person-row strong { display:block; margin-bottom:12px; color:#626b7c; font-size:14px; font-weight:400; }
.person-row p { margin:0; font-size:14px; line-height:1.7; }
.person-row p.clamped { display:-webkit-box; overflow:hidden; -webkit-box-orient:vertical; -webkit-line-clamp:2; }
.person-row button { display:inline-flex; align-items:center; gap:4px; margin-top:14px; padding:0; border:0; background:transparent; color:#2864ff; cursor:pointer; }
.person-row button svg { width:15px; transition:transform .2s; }
.rotated { transform:rotate(180deg); }
.round-cta { position:relative; min-height:330px; overflow:hidden; background:#f6f8fc; }
.round-cta .container { position:relative; display:flex; min-height:330px; align-items:center; }
.round-cta .container > div { position:relative; z-index:1; max-width:720px; }
.round-cta h2 { margin:0 0 16px; font-size:36px; }
.round-cta p { margin:0; line-height:1.75; }
.round-cta .container > span { position:absolute; right:40px; bottom:-250px; width:480px; height:480px; border-radius:50%; background:#4c79ee; }
.committee-section { padding:105px 0; background:#fff; }
.committee-grid { display:grid; grid-template-columns:1fr 1fr; gap:24px; }
.committee-grid article { display:flex; min-height:370px; flex-direction:column; padding:44px; border:1px solid #dce2ec; background:#fff; }
.committee-grid article > b { color:#2864ff; font-weight:400; }
.committee-grid h3 { margin:20px 0 15px; font-size:25px; line-height:1.35; }
.committee-grid p { margin:0; line-height:1.75; }
.committee-grid a { margin-top:auto; padding-top:24px; }
.fellow-hero { position:relative; min-height:630px; background-position:center; background-size:cover; color:#fff; }
.fellow-hero::before { content:""; position:absolute; inset:0; background:rgba(8,28,69,.75); }
.fellow-hero .container { position:relative; display:flex; min-height:630px; flex-direction:column; justify-content:center; align-items:flex-start; }
.fellow-hero h1 { margin:0; color:#fff; font-size:57px; }
.fellow-hero p { max-width:710px; margin:25px 0 5px; padding-left:24px; border-left:1px solid #fff; font-size:16px; line-height:1.75; }
.fellow-hero a { padding:14px 22px; border:1px solid #fff; border-radius:28px; color:#fff; }
.fellow-program { padding:108px 0; }
.program-layout { display:grid; grid-template-columns:1.15fr .85fr; gap:95px; align-items:center; }
.program-layout > img { width:100%; max-height:700px; object-fit:cover; object-position:top; }
.program-layout h2 { margin:0 0 20px; font-size:40px; line-height:1.25; }
.program-layout > div > p { margin:0 0 35px; line-height:1.75; }
.program-layout article { display:grid; grid-template-columns:38px 1fr; gap:20px; padding:20px 0; border-top:1px solid #dde3ec; }
.program-layout article b { color:#2864ff; font-weight:400; }
.program-layout article h3 { margin:0 0 7px; font-size:20px; }
.program-layout article p { margin:0; line-height:1.65; }
.fellows-section { padding:105px 0; background:#f5f7fb; }
.fellows-section header h2 { margin:0 0 12px; font-size:40px; }
.fellows-section header p { margin:0 0 32px; }
.search-label { margin:0 0 10px; color:#26334d; font-size:14px; font-weight:600; }
.fellow-search { display:flex; width:410px; align-items:center; gap:10px; padding:0 15px; border:1px solid #d9e0ea; background:#fff; }
.fellow-search svg { width:17px; }
.fellow-search input { width:100%; min-height:46px; border:0; outline:0; }
.filter-label { margin:22px 0 12px; font-size:13px; }
.filter-chips { display:flex; flex-wrap:wrap; gap:8px; }
.filter-chips button { padding:8px 15px; border:1px solid #dce2eb; border-radius:20px; background:#fff; color:#4d5668; cursor:pointer; }
.filter-chips button.active { border-color:#2864ff; background:#2864ff; color:#fff; }
.clear-filters,.empty-fellows button { margin-top:14px; padding:0 0 5px; border:0; border-bottom:1px solid #2864ff; background:transparent; color:#2864ff; cursor:pointer; }
.result-count { display:block; margin:28px 0 22px; }
.fellow-viewport { display:grid; grid-auto-columns:calc((100% - 44px)/3); grid-auto-flow:column; gap:22px; overflow-x:auto; scroll-snap-type:x mandatory; scrollbar-width:none; }
.fellow-viewport::-webkit-scrollbar { display:none; }
.fellow-viewport article { display:flex; min-height:320px; flex-direction:column; padding:28px; scroll-snap-align:start; border:1px solid #dce2eb; background:#fff; }
.fellow-viewport article > span,.fellow-viewport em { width:max-content; padding:5px 10px; border:1px solid #cad9ff; border-radius:15px; color:#2864ff; font-size:12px; font-style:normal; }
.fellow-viewport h3 { margin:18px 0 8px; font-size:21px; }
.fellow-viewport article > p { margin:7px 0 18px; }
.fellow-viewport article > div { display:flex; flex-wrap:wrap; gap:6px; }
.fellow-viewport article > a { display:flex; align-items:center; gap:6px; margin-top:auto; padding-top:18px; border-top:1px solid #e1e6ed; color:#2864ff; }
.fellow-viewport article > a svg { width:15px; }
.carousel-controls { display:flex; align-items:center; justify-content:space-between; margin-top:32px; }
.carousel-controls button { display:grid; width:44px; height:44px; place-items:center; border:0; border-radius:50%; background:#fff; box-shadow:0 5px 18px rgba(29,45,74,.1); color:#1c315d; }
.carousel-controls button svg { width:17px; }
.carousel-controls i { width:34px; height:5px; border-radius:4px; background:#2864ff; box-shadow:13px 0 #d0d8e7,26px 0 #d0d8e7; }
.empty-fellows { padding:54px 24px; text-align:center; background:#fff; }
.empty-fellows h3 { margin:0 0 10px; font-size:24px; }
.empty-fellows p { margin:0; color:#667085; }
.faq-section { padding:105px 0; }
.faq-section .container { max-width:930px; }
.faq-section h2 { margin:0 0 34px; font-size:41px; }
.faq-heading { display:flex; align-items:flex-end; justify-content:space-between; gap:24px; margin-bottom:34px; }
.faq-heading h2 { margin:0; }
.faq-heading > a { display:flex; flex:0 0 auto; align-items:center; gap:7px; padding-bottom:7px; border-bottom:1px solid #2864ff; color:#2864ff; }
.faq-heading > a svg { width:15px; }
.faq-section article { border-top:1px solid #d9e0e9; }
.faq-section article:last-child { border-bottom:1px solid #d9e0e9; }
.faq-section article button { display:flex; width:100%; align-items:center; justify-content:space-between; padding:23px 0; border:0; background:transparent; color:#17223b; font-size:18px; text-align:left; cursor:pointer; }
.faq-section article button svg { width:19px; transition:transform .2s; }
.faq-section article p { margin:-5px 0 24px; line-height:1.75; white-space:pre-line; }
.youth-features,.engagement-section { padding:105px 0; background:#f5f7fb; }
.youth-features header h2 { margin:0 0 14px; font-size:41px; }
.youth-features header p { max-width:760px; margin:0; line-height:1.75; }
.youth-grid { display:grid; grid-template-columns:1fr 1fr; gap:24px; margin-top:48px; }
.youth-grid article { display:grid; grid-template-columns:210px 1fr; min-height:240px; border:1px solid #dae1eb; background:#fff; }
.youth-grid img { width:100%; height:100%; object-fit:cover; }
.youth-grid article > div { align-self:center; padding:28px; }
.youth-grid h3 { margin:0 0 12px; font-size:22px; }
.youth-grid p { margin:0; line-height:1.7; }
.youth-benefits { padding:105px 0; background:#101f47; color:#fff; }
.youth-benefits .container { display:grid; grid-template-columns:1fr 1fr; gap:75px; align-items:center; }
.youth-benefits h2 { margin:0 0 18px; color:#fff; font-size:40px; }
.youth-benefits p { line-height:1.75; color:rgba(255,255,255,.75); }
.youth-benefits ul { display:grid; gap:11px; padding:15px 0 0; list-style:none; }
.youth-benefits li { display:flex; gap:10px; }
.youth-benefits li svg { width:17px; flex:0 0 17px; color:#7ba0ff; }
.youth-benefits img { width:100%; height:540px; object-fit:cover; }
.engagement-section { background:#fff; }
.engagement-section .container > div { display:grid; grid-template-columns:repeat(3,1fr); gap:24px; }
.engagement-section article { border:1px solid #dce2eb; }
.engagement-section article img { width:100%; height:240px; object-fit:cover; }
.engagement-section article h3 { margin:24px 25px 10px; font-size:22px; }
.engagement-section article p { margin:0 25px 28px; line-height:1.7; }
.cfta-section { padding:95px 0; background:#f5f7fb; }
.cfta-section .container { display:grid; grid-template-columns:1.1fr .35fr .75fr; gap:45px; align-items:center; }
.cfta-section h2 { margin:0 0 16px; font-size:38px; }
.cfta-section p { line-height:1.75; }
.cfta-section strong { color:#2864ff; font-size:48px; text-align:center; }
.cfta-section strong small { display:block; margin-top:10px; color:#38445b; font-size:14px; }
.cfta-section img { width:100%; height:330px; object-fit:cover; }
.jobs-section { padding:105px 0 120px; background:#fff; }
.jobs-section h2 { margin:0; font-size:42px; }
.job-tabs { display:flex; width:max-content; gap:6px; margin:58px 0 64px; padding:7px; border:1px solid #d5deeb; border-radius:30px; background:#f4f7fb; }
.job-tabs button { min-width:58px; padding:11px 18px; border:0; border-radius:24px; background:transparent; color:#3b465c; font-weight:600; cursor:pointer; }
.job-tabs button.active { background:#1260b7; color:#fff; box-shadow:0 6px 14px rgba(18,96,183,.23); }
.job-list { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:24px; }
.job-list article { display:flex; min-height:255px; flex-direction:column; padding:38px 32px 28px; border:1px solid #dce2eb; border-radius:8px; background:#fff; }
.job-list span { color:#343b48; font-size:15px; }
.job-list h3 { margin:18px 0 28px; color:#0e1b38; font-size:21px; line-height:1.45; }
.job-list p { display:flex; flex-wrap:wrap; align-items:center; gap:14px; margin:0 0 24px; color:#737984; font-size:14px; }
.job-list p strong { color:#737984; font-weight:400; }
.job-list p i { width:1px; height:24px; background:#d8dee7; transform:rotate(25deg); }
.job-list article > a { display:flex; width:max-content; align-items:center; gap:7px; margin-top:auto; padding-bottom:7px; border-bottom:1px solid #2864ff; color:#2864ff; }
.job-list article > a svg { width:15px; }
.no-jobs { margin:0; color:#667085; }
.contact-section { padding:95px 0 112px; }
.contact-section header h1 { max-width:760px; margin:0 0 22px; font-size:43px; }
.contact-section header p { max-width:760px; margin:0; font-size:16px; line-height:1.75; }
.contact-grid { display:grid; grid-template-columns:repeat(3,1fr); gap:24px; margin-top:66px; }
.contact-grid article { min-height:265px; padding:38px 32px; border:1px solid #d8dee7; background:rgba(255,255,255,.9); }
.contact-grid h2 { margin:0 0 20px; font-size:22px; }
.contact-grid article > p { min-height:46px; margin:0 0 28px; color:#737a86; line-height:1.65; }
.contact-grid article > div { display:flex; align-items:center; gap:18px; color:#777e88; }
.contact-grid article > div i { width:1px; height:24px; background:#d9dfe8; transform:rotate(24deg); }
.contact-grid article > div a { color:#777e88; }
.contact-link { display:flex; width:max-content; align-items:center; gap:7px; margin-top:20px; padding-bottom:7px; border-bottom:1px solid #2864ff; color:#2864ff; }
.contact-link svg { width:15px; }

@media (max-width:900px) {
  .container { width:calc(100% - 40px); }
  .program-layout,.youth-benefits .container,.cfta-section .container { grid-template-columns:1fr; }
  .overlap-intro > div { left:50px; width:calc(100% - 100px); }
  .committee-grid,.youth-grid { grid-template-columns:1fr; }
  .contact-grid { grid-template-columns:1fr 1fr; }
  .job-list { grid-template-columns:1fr 1fr; }
  .fellow-viewport { grid-auto-columns:calc((100% - 22px)/2); }
  .engagement-section .container > div { grid-template-columns:1fr 1fr; }
}
@media (max-width:650px) {
  .container { width:calc(100% - 32px); }
  .page-title { min-height:260px; }
  .page-title h1,.career-title h1 { font-size:34px; }
  .page-title nav { gap:11px; padding:0 14px; font-size:11px; }
  .overlap-intro { padding-top:45px; padding-bottom:65px; }
  .overlap-intro > img { height:310px; }
  .overlap-intro > div { position:relative; bottom:auto; left:auto; width:calc(100% - 20px); min-height:0; margin:-45px auto 0; padding:27px 23px; }
  .overlap-intro h2 { font-size:26px; }
  .people-section,.committee-section,.fellow-program,.fellows-section,.faq-section,.youth-features,.youth-benefits,.engagement-section,.jobs-section,.contact-section,.cfta-section { padding:70px 0; }
  .people-section header h2,.committee-section header h2,.engagement-section header h2,.program-layout h2,.fellows-section header h2,.youth-features header h2,.youth-benefits h2,.cfta-section h2,.contact-section header h1 { font-size:31px; }
  .person-row { grid-template-columns:72px 1fr; gap:18px; }
  .person-row > img { width:72px; height:72px; }
  .round-cta .container > span { right:-180px; }
  .fellow-hero,.fellow-hero .container { min-height:520px; }
  .fellow-hero h1 { font-size:42px; }
  .program-layout > img { max-height:430px; }
  .fellow-search { width:100%; }
  .fellow-viewport { grid-auto-columns:88%; }
  .youth-grid article { grid-template-columns:1fr; }
  .youth-grid article img { height:230px; }
  .engagement-section .container > div,.contact-grid { grid-template-columns:1fr; }
  .youth-benefits img { height:360px; }
  .faq-heading { align-items:flex-start; flex-direction:column; }
  .job-tabs { max-width:100%; margin:38px 0 40px; }
  .job-tabs button { min-width:0; padding:10px 14px; }
  .job-list { grid-template-columns:1fr; }
  .job-list article { min-height:235px; padding:30px 24px 25px; }
  .contact-grid article > div { gap:11px; font-size:12px; }
}
</style>
