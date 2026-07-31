import type { LocalizedText } from "@/lib/gfiSite"

const assetBase = "/gfi/ecosystem"

export const ecosystemAssets = {
  programme: `${assetBase}/programme.webp`,
  partnershipCta: `${assetBase}/partnership-cta.webp`,
  professional: `${assetBase}/professional.webp`,
}

export type Partner = { name: LocalizedText; logo: string; url: string; programmes?: string[] }

export const partnerGroups: Array<{ key: string; title: LocalizedText; description?: LocalizedText; partners: Partner[] }> = [
  {
    key: "education",
    title: { zh: "教育合作伙伴", en: "Education Partners" },
    description: {
      zh: "这些合作伙伴与GFI在金融科技和数字金融教育以及人才途径方面合作。",
      en: "These partners collaborate with GFI on fintech and digital finance education and talent pathways.",
    },
    partners: [
      { name: { zh: "新加坡社科大学", en: "Singapore University of Social Sciences" }, logo: `${assetBase}/suss.webp`, url: "https://www.suss.edu.sg/", programmes: ["金融学士（BFIN）", "金融硕士（MFIN）", "金融科技硕士（中文）（在读中）"] },
      { name: { zh: "新加坡管理大学", en: "Singapore Management University" }, logo: `${assetBase}/smu.png`, url: "https://www.smu.edu.sg/", programmes: ["应用金融科技理学硕士（MAF）", "MITB（金融技术与分析），计算与信息系统学院（WIP）"] },
      { name: { zh: "上海财经大学", en: "Shanghai University of Finance and Economics" }, logo: `${assetBase}/sufe.webp`, url: "https://english.sufe.edu.cn/", programmes: ["金融学学士", "双学位项目（金融科技基础班）", "金融科技理学硕士"] },
      { name: { zh: "新加坡国立大学", en: "National University of Singapore" }, logo: `${assetBase}/nus.png`, url: "https://nus.edu.sg/", programmes: ["工商管理学士（BBA）", "金融理学硕士（MFIN）"] },
      { name: { zh: "南洋理工大学", en: "Nanyang Technological University" }, logo: `${assetBase}/ntu.png`, url: "https://www.ntu.edu.sg/", programmes: ["工商管理学士（银行与金融），国家银行学院", "金融技术理学硕士，SPMS"] },
      { name: { zh: "默多克大学", en: "Murdoch University" }, logo: `${assetBase}/murdoch.webp`, url: "https://www.murdoch.edu.au/", programmes: ["银行与金融工商管理学士（双专业）"] },
      { name: { zh: "凯利商学院", en: "Kelley School of Business" }, logo: `${assetBase}/kelley.png`, url: "https://kelley.iu.edu/index.html", programmes: ["金融理学硕士（MSF）"] },
      { name: { zh: "博阿齐奇大学", en: "Bogazici University" }, logo: `${assetBase}/bogazici.webp`, url: "https://bogazici.edu.tr/en", programmes: ["管理学学士"] },
    ],
  },
  {
    key: "associations",
    title: { zh: "协会", en: "Associations" },
    partners: [
      { name: { zh: "WAWEB3", en: "WAWEB3" }, logo: `${assetBase}/waweb3.webp`, url: "https://waweb3.org/" },
      { name: { zh: "银行与金融服务工会", en: "Banking and Financial Services Union" }, logo: `${assetBase}/bfsu.webp`, url: "https://www.bfsu.org.sg/" },
      { name: { zh: "新加坡经济学会", en: "Economic Society of Singapore" }, logo: `${assetBase}/ess.webp`, url: "https://ess.org.sg/" },
      { name: { zh: "非洲金融科技网络", en: "Africa FinTech Network" }, logo: `${assetBase}/africa.webp`, url: "https://africafintechnetwork.com/" },
    ],
  },
  {
    key: "industry",
    title: { zh: "行业", en: "Industry" },
    partners: [
      { name: { zh: "新加坡ICP枢纽网络", en: "ICP Hub Singapore" }, logo: `${assetBase}/icp.webp`, url: "https://www.icphubsg.com/" },
      { name: { zh: "泰坦网络", en: "Titan Network" }, logo: `${assetBase}/titan.webp`, url: "https://www.titannet.io/" },
      { name: { zh: "验证VASP", en: "VerifyVASP" }, logo: `${assetBase}/verifyvasp.webp`, url: "https://www.verifyvasp.com/en/" },
      { name: { zh: "登顿罗迪克与戴维森律师事务所", en: "Dentons Rodyk" }, logo: `${assetBase}/dentons.png`, url: "https://dentons.rodyk.com/" },
      { name: { zh: "科博", en: "Cobo" }, logo: `${assetBase}/cobo.webp`, url: "https://www.cobo.com/" },
      { name: { zh: "亚太交流", en: "Asia Pacific Exchange" }, logo: `${assetBase}/apx.webp`, url: "https://www.asiapacificex.com/" },
    ],
  },
]

export const membershipTiers = [
  {
    title: { zh: "附属会员", en: "Affiliate Member" }, price: 49,
    description: { zh: "附属会员是进入GFI生态系统的入口。专为探索金融科技、建立基础知识或寻求全球行业洞察和网络接触的个人而设计。", en: "Affiliate Membership is the entry point into the GFI ecosystem. It is designed for individuals exploring fintech, building foundational knowledge, or seeking exposure to global industry insights and networks." },
    benefits: { zh: ["访问网络研讨会录像和回放", "优先邀请参加GFI活动", "访问GFI报告和白皮书", "GFI通讯", "GFI学习门户短期课程享10%折扣"], en: ["Access to webinar recordings and replays", "Priority invitations to GFI events", "Access to GFI reports and whitepapers", "GFI newsletter", "10% discount on short courses on the GFI Learning Portal"] },
    best: { zh: "学生、早期职业专业人士和探索金融科技基础知识和全球行业接触的职业转换者。", en: "Students, early-career professionals, and career switchers exploring fintech fundamentals and global industry exposure." },
  },
  {
    title: { zh: "准会员", en: "Associate Member" }, price: 129,
    description: { zh: "准会员认可正在积极建立金融科技能力并朝着正式认证方向发展的专业人士。它支持更深入的学习、实践接触以及与GFI社区更紧密的互动。", en: "Associate Membership recognises professionals who are actively building fintech capability and progressing towards formal certification. It supports deeper learning, practical exposure, and closer engagement with the GFI community." },
    benefits: { zh: ["附属会员全部福利，以及", "使用CFtA称号的权利", "访问GFI精选资源和洞察", "通过GFI合作网络获得职业机会", "GFI学习门户短期课程享15%折扣"], en: ["All Affiliate Member benefits, plus", "Right to use the CFtA designation", "Access to curated GFI resources and insights", "Career opportunities through GFI's partner network", "15% discount on short courses on the GFI Learning Portal"] },
    best: { zh: "积极建立结构化金融科技专业知识并朝着认证方向发展的CFtP候选人和CFtA持有者。", en: "CFtP candidates and CFtA holders actively building structured fintech expertise and progressing towards certification." },
  },
  {
    title: { zh: "特许会员", en: "Charterholder Member" }, price: 169,
    description: { zh: "特许会员代表GFI最高级别的专业认可。专为CFtP®特许持有者保留，它标志着掌握、可信度和对道德和负责任的金融科技领导的承诺。", en: "Charterholder Membership represents GFI's highest level of professional recognition. Reserved for CFtP® charterholders, it signals mastery, credibility, and a commitment to ethical and responsible fintech leadership." },
    benefits: { zh: ["准会员全部福利，以及", "使用CFtP称号的权利", "受邀加入FLEX国际中心（人才与公司投资者对接平台）", "GFI学习门户短期课程享20%折扣"], en: ["All Associate Member benefits, plus", "Right to use the CFtP designation", "Invitation to join the FLEX International Hub", "20% discount on short courses on the GFI Learning Portal"] },
    best: { zh: "寻求认可的专业地位、领导机会和持续行业参与的CFtP®特许持有者。", en: "CFtP® charterholders seeking recognised professional standing, leadership opportunities, and continued industry engagement." },
  },
]

export const membershipRows = [
  ["职业阶段", "探索阶段", "发展路径", "公认领导力"], ["年费（美元）", "$49", "$129", "$169"],
  ["访问网络研讨会和洞察", "精选/基础", "高级", "高级"], ["访问研究和白皮书", "核心内容", "扩展访问", "完全访问"],
  ["课程和考试折扣", "标准", "增强", "增强"], ["GFI活动邀请", "标准", "优先", "优先"],
  ["实习机会", "—", "可用", "完全访问"], ["职业机会", "—", "—", "完全访问"],
  ["FLEX职业门户网站列表", "—", "—", "包含"], ["使用CFtP称号", "—", "—", "包含"],
  ["闭门领导论坛", "—", "—", "包含"], ["课程和内容更新", "—", "—", "包含"],
  ["专业地位", "社区成员", "认证候选人", "特许专业人士"],
]

export const corporateTiers = [
  { name: "Platinum Patron", price: "50,000", description: "Platinum is GFI's highest patron tier, offering first-tier access to flagship initiatives, co-branded events, and the strongest visibility across GFI platforms.", benefits: ["15 CFtP full-pathway scholarships per year", "Preferential, highest-visibility access on the FLEX International Hub", "Nomination of up to 5 senior leaders for consideration as GFI Industry Fellows", "First-tier, reserved access to key councils, investor forums, and regulatory roundtables", "First call on GFI flagship and strategic initiatives", "Inclusion of 2 branded webinars and 1 co-branded flagship event", "Priority consideration for major features and case studies in the Annual Journal of Fintech", "Prominent logo placement, including homepage visibility where appropriate", "Recognition across key GFI-led and flagship events", "Senior-level strategic coordination and ongoing partnership management", "Optional add-ons: additional flagship events, webinars, or CFtP seats"], best: "Global institutions and market leaders seeking long-term influence, leadership positioning, and deep integration into the fintech ecosystem." },
  { name: "Gold Patron", price: "20,000", description: "Gold Patrons receive enhanced access across programmes, content, and talent, including branded webinars and reserved participation in key dialogues.", benefits: ["6 CFtP full-pathway scholarships per year", "Enhanced visibility and access on the FLEX International Hub", "Nomination of 2 senior leaders for consideration as GFI Industry Fellows", "Reserved invitations to selected closed-door councils, roundtables, and policy dialogues", "Priority access across a wider slate of GFI programmes and initiatives", "Inclusion of 1 branded webinar co-designed with GFI", "Eligibility to contribute full case studies or thought pieces to the Annual Journal of Fintech", "Prominent logo placement under \"Corporate Patrons – Gold\"", "Stronger recognition across GFI-led and co-branded events", "Onboarding call plus ongoing coordination support", "Optional add-ons: additional webinars, CFtP seats, or flagship events"], best: "Organisations actively shaping fintech strategy, policy dialogue, and ecosystem initiatives." },
  { name: "Silver Patron", price: "10,000", description: "Silver Patrons gain stronger access to GFI programmes, priority invitations, and the ability to nominate senior leaders for consideration as Industry Fellows.", benefits: ["2 CFtP full-pathway scholarships per year", "Corporate listing and enhanced access on the FLEX International Hub", "Nomination of 1 senior leader for consideration as a GFI Industry Fellow", "Priority invitations to strategic GFI events, briefings, and selected closed-door sessions", "Eligibility to contribute practitioner notes and short pieces to the Annual Journal of Fintech", "Logo placement under \"Corporate Patrons – Silver\" on the GFI website", "Recognition at GFI-led and co-branded events", "Onboarding and coordination support as needed", "Optional add-ons: additional CFtP seats, webinars, or flagship events"], best: "Institutions seeking stronger visibility, leadership participation, and structured engagement with the fintech ecosystem." },
  { name: "Bronze Patron", price: "5,000", description: "Designed for organisations beginning their engagement with GFI, the Bronze tier offers foundational access to talent, insights, and selected programmes, while signalling support for professional fintech standards.", benefits: ["1 CFtP full-pathway scholarship per year", "Base-level access to the FLEX International Hub for job and internship postings", "Select invitations to GFI webinars, briefings, and relevant events", "Eligibility to contribute short practitioner notes to the Annual Journal of Fintech", "Logo placement under \"Corporate Patrons – Bronze\" on the GFI website", "Recognition at GFI events where relevant", "1 onboarding strategy call (30–45 minutes)", "Optional add-ons: additional CFtP seats, webinars, or flagship events"], best: "Organisations beginning their engagement with fintech capability-building and ecosystem participation." },
]

export const corporateRows = [
  ["CFtP & Talent", "CFtP Scholarships (full pathway)", "1", "2", "6", "15"],
  ["", "FLEX Talent Hub access", "Base job & intern postings", "Corporate listing & access", "Enhanced visibility", "Preferential, highest visibility"],
  ["Influence & Access", "Industry Fellow nominations", "—", "1", "2", "5"],
  ["", "Closed-door councils & roundtables", "Selected invitations", "Priority invitations", "Reserved invitations", "First-tier reserved access"],
  ["", "Access to GFI programmes", "Selected webinars & briefings", "Priority for strategic programmes", "Priority across wider slate", "First call on flagship initiatives"],
  ["Events", "Branded webinars included", "—", "—", "1", "2"], ["", "Co-branded flagship events", "—", "—", "—", "1"],
  ["Insights & Content", "Annual Journal of Fintech", "Short practitioner notes", "Practitioner notes / short pieces", "Case studies / thought pieces", "Priority major features"],
  ["Branding", "Website logo placement", "Patrons – Bronze", "Patrons – Silver", "Patrons – Gold", "Prominent + homepage"],
  ["Support", "Onboarding & coordination", "1 call (30–45 mins)", "As needed", "Ongoing coordination", "Senior-level strategic coordination"],
  ["Add-ons", "Optional extensions", "CFtP seats, webinars, events", "Same", "Same (expanded)", "Same (expanded)"],
]

export const corporateFaqs = [
  ["1. How do organisations use the programme to build internal capability?", "Corporate Patrons use the programme to develop a CFtP-trained talent bench, equipping teams with structured fintech, governance, and regulatory knowledge to engage credibly with regulators, partners, and clients.\n\nBronze: Sponsor CFtP scholarships for high-potential staff or students\n\nSilver: Scale structured upskilling across priority teams\n\nGold & Platinum: Build a sustained internal pipeline aligned with long-term strategy"],
  ["2. How does the programme support leadership positioning and networks?", "The programme enables organisations to place senior leaders within trusted industry and policy networks through fellowships, councils, and closed-door dialogues.\n\nSilver: Nominate leaders for consideration as GFI Industry Fellows\n\nGold: Participate in reserved roundtables and policy discussions\n\nPlatinum: Anchor leadership presence in flagship forums and councils"],
  ["3. How do Corporate Patrons access fintech talent?", "Patrons leverage the FLEX International Hub to connect with curated fintech professionals across regions and roles.\n\nBronze: Post internships and entry-level roles\n\nSilver: Gain corporate visibility to attract early-career talent\n\nGold & Platinum: Access senior and specialised talent pools"],
  ["4. How does the programme support thought leadership and visibility?", "The programme allows organisations to contribute credible, non-promotional insights that build trust and influence.\n\nBronze & Silver: Contribute practitioner notes and short insights\n\nGold: Host a co-branded webinar or publish case studies\n\nPlatinum: Lead flagship conversations and major industry features"],
  ["5. How does becoming a Corporate Patron signal long-term commitment?", "Participation demonstrates a clear commitment to responsible fintech standards, education, and ecosystem development.\n\nAll tiers: Public recognition as a GFI Corporate Patron\n\nGold & Platinum: Visible leadership in shaping best practices and standards"],
]
