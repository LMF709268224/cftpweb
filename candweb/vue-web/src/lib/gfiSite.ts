import type { Lang } from "@/lib/language"

export type LocalizedText = { zh: string; en: string }
export type GfiSection = {
  title: LocalizedText
  text: LocalizedText
  bullets?: LocalizedText[]
}

export type GfiPageDefinition = {
  path: string
  eyebrow: LocalizedText
  title: LocalizedText
  description: LocalizedText
  image: string
  sections: GfiSection[]
}

export type GfiNavGroup = {
  label: LocalizedText
  items: Array<{ label: LocalizedText; to: string }>
}

const t = (zh: string, en: string): LocalizedText => ({ zh, en })
const assetBase = "https://globalfintechinstitute.org/assets"

export function localize(value: LocalizedText, lang: Lang) {
  return value[lang]
}

export const gfiNavGroups: GfiNavGroup[] = [
  {
    label: t("关于", "About"),
    items: [
      { label: t("我们的故事", "Our Story"), to: "/gfi/about" },
      { label: t("董事会", "Board of Directors"), to: "/gfi/about/board-of-directors" },
      { label: t("认识团队", "Meet the Team"), to: "/gfi/about/team" },
      { label: t("子委员会", "SubCommittees"), to: "/gfi/subcommittees" },
      { label: t("行业研究员", "Industry Fellows"), to: "/gfi/industry-fellow" },
      { label: t("青年部", "Youth Wing"), to: "/gfi/youth-wing" },
      { label: t("职业与志愿者", "Career & Volunteer"), to: "/gfi/career" },
      { label: t("联系我们", "Contact Us"), to: "/gfi/contact" },
    ],
  },
  {
    label: t("认证", "Certifications"),
    items: [
      { label: t("认证概览", "Overview of Certifications"), to: "/gfi/certifications" },
      { label: t("认证路径", "Certification Pathway"), to: "/gfi/certifications/pathway" },
      { label: t("特许金融科技助理（CFtA）", "Chartered Fintech Associate (CFtA)"), to: "/gfi/programmes/cfta" },
      { label: t("特许金融科技专业人士（CFtP®）", "Chartered Fintech Professional (CFtP®)"), to: "/gfi/programmes/cftp" },
    ],
  },
  {
    label: t("课程", "Courses"),
    items: [{ label: t("高管项目", "Executive Programmes"), to: "/gfi/programmes" }],
  },
  {
    label: t("生态系统", "Ecosystems"),
    items: [
      { label: t("合作伙伴", "Partners"), to: "/gfi/partnerships" },
      { label: t("个人会员", "Individual Membership"), to: "/gfi/membership/benefits" },
      { label: t("企业赞助人", "Corporate Patrons"), to: "/gfi/membership/corporate" },
    ],
  },
  {
    label: t("出版物", "Publications"),
    items: [
      { label: t("报告", "Reports"), to: "/gfi/publications/reports" },
      { label: t("洞察", "Insights"), to: "/gfi/publications/insights" },
      { label: t("新闻中心", "Press Room"), to: "/gfi/publications/news" },
      { label: t("期刊", "Journals"), to: "/gfi/publications/journals" },
    ],
  },
  {
    label: t("活动", "Events"),
    items: [
      { label: t("所有活动", "All Events"), to: "/gfi/events/all-events" },
      { label: t("网络研讨会与录像", "Webinars & Recordings"), to: "/gfi/events/webinar-recordings" },
      { label: t("会议与圆桌会议", "Conferences & Roundtables"), to: "/gfi/events/conferences" },
    ],
  },
]

const aboutImage = `${assetBase}/gfi-banner-2.COpPUN29_Z1eroN0.webp`
const certificationImage = `${assetBase}/gfi-banner-3.X5tYPN3w_ZLlOGO.webp`
const ecosystemImage = `${assetBase}/gfi-banner-1.CXyxJgfV_1motdG.webp`
const publicationImage = "https://gfi-landingpage-dev.s3.ap-southeast-1.amazonaws.com/ghost-images/Digital%20Assets%20Security%20%26%20Compliance%20Subcommittee%20Cover%20Page-b3bfb1be-1783085371057.png"

const common = {
  standards: {
    title: t("推动全球金融科技标准", "Advancing Global Fintech Standards"),
    text: t("GFI连接监管机构、企业和学术界，制定全球标准、提供专业教育，并促进金融生态系统中的负责任创新。", "GFI connects regulators, corporations, and academia to set global standards, deliver professional education, and foster responsible innovation across the financial ecosystem."),
  },
  network: {
    title: t("连接全球专业网络", "A Connected Global Network"),
    text: t("通过认证、研究、活动与合作伙伴关系，我们让专业人士能够持续学习并参与行业发展。", "Through certifications, research, events, and partnerships, we enable professionals to keep learning and contribute to the industry's development."),
  },
}

export const gfiPages: GfiPageDefinition[] = [
  {
    path: "about",
    eyebrow: t("关于GFI", "About GFI"),
    title: t("推动全球金融科技标准", "Advancing Global Fintech Standards"),
    description: t("我们是一个非营利智库和专业机构，致力于塑造金融科技的未来。", "We are a non-profit think tank and professional body dedicated to shaping the future of fintech."),
    image: aboutImage,
    sections: [common.standards, { title: t("影响金融科技教育与标准", "Driving Impact in Fintech Education and Standards"), text: t("自成立以来，GFI通过创建全球认可的认证、建立协作网络和引领行业对话，持续推动金融科技知识与实践。", "Since our founding, GFI has advanced fintech knowledge and practice through globally recognised certifications, collaborative networks, and industry dialogue.") }, common.network],
  },
  {
    path: "about/board-of-directors",
    eyebrow: t("治理", "Governance"),
    title: t("董事会", "Board of Directors"),
    description: t("由金融、科技、政策和学术领域的资深领袖共同指导GFI的长期方向。", "Senior leaders across finance, technology, policy, and academia guide GFI's long-term direction."),
    image: aboutImage,
    sections: [{ title: t("负责任的治理", "Responsible Governance"), text: t("董事会确保GFI始终坚持公共利益、专业诚信和全球视野。", "The Board ensures that GFI remains focused on public value, professional integrity, and a global outlook.") }, common.standards],
  },
  {
    path: "about/team",
    eyebrow: t("我们的团队", "Our Team"),
    title: t("认识GFI团队", "Meet the GFI Team"),
    description: t("一支跨学科团队，负责认证、研究、合作伙伴关系与全球社群发展。", "A multidisciplinary team advancing certification, research, partnerships, and global community growth."),
    image: aboutImage,
    sections: [{ title: t("共同塑造金融科技未来", "Shaping Fintech Together"), text: t("我们的团队将行业实践、教育设计和政策研究结合起来，交付可信且相关的项目。", "Our team combines industry practice, education design, and policy research to deliver trusted, relevant programmes.") }, common.network],
  },
  {
    path: "subcommittees",
    eyebrow: t("治理", "Governance"),
    title: t("GFI子委员会", "GFI SubCommittees"),
    description: t("召集专家就数字资产安全、合规、治理和新兴技术开展务实工作。", "Convening experts to advance practical work in digital asset security, compliance, governance, and emerging technology."),
    image: certificationImage,
    sections: [{ title: t("专家驱动的行业协作", "Expert-Led Industry Collaboration"), text: t("各委员会围绕关键议题提出研究方向、行业建议和专业标准。", "Each committee develops research priorities, industry recommendations, and professional standards around critical issues.") }, common.standards],
  },
  {
    path: "industry-fellow",
    eyebrow: t("专业社群", "Professional Community"),
    title: t("行业研究员", "Industry Fellows"),
    description: t("汇聚有影响力的从业者，贡献专业知识并推动金融科技生态系统发展。", "Bringing together influential practitioners who contribute expertise and advance the fintech ecosystem."),
    image: ecosystemImage,
    sections: [{ title: t("贡献行业洞察", "Contribute Industry Insight"), text: t("研究员参与思想领导力、项目评审、专业指导和全球行业交流。", "Fellows contribute to thought leadership, programme review, professional mentoring, and global industry exchange.") }, common.network],
  },
  {
    path: "youth-wing",
    eyebrow: t("青年部", "Youth Wing"),
    title: t("培养下一代金融科技领袖", "Developing the Next Generation of Fintech Leaders"),
    description: t("帮助学生和青年专业人士建立知识、网络与领导力。", "Helping students and young professionals build knowledge, networks, and leadership capability."),
    image: ecosystemImage,
    sections: [{ title: t("从学习走向领导", "From Learning to Leadership"), text: t("青年部通过活动、导师计划和行业项目，为年轻人才提供参与金融科技的路径。", "The Youth Wing creates pathways into fintech through events, mentoring, and industry projects.") }, common.network],
  },
  {
    path: "career",
    eyebrow: t("职业与志愿者", "Career & Volunteer"),
    title: t("与GFI共同创造影响", "Create Impact with GFI"),
    description: t("加入我们的团队或志愿者网络，为全球金融科技专业发展贡献力量。", "Join our team or volunteer network and contribute to global fintech professional development."),
    image: ecosystemImage,
    sections: [{ title: t("职业机会", "Career Opportunities"), text: t("我们寻找认同专业标准、负责任创新和全球协作的人才。", "We welcome people who share our commitment to professional standards, responsible innovation, and global collaboration.") }, { title: t("志愿者机会", "Volunteer Opportunities"), text: t("以导师、评审、研究贡献者或活动支持者的身份参与GFI。", "Contribute as a mentor, reviewer, research contributor, or event supporter.") }],
  },
  {
    path: "contact",
    eyebrow: t("联系我们", "Contact Us"),
    title: t("让我们开始对话", "Let's Start a Conversation"),
    description: t("就认证、会员、合作伙伴关系、研究或媒体合作联系我们。", "Contact us about certifications, membership, partnerships, research, or media opportunities."),
    image: aboutImage,
    sections: [{ title: t("新加坡办公室", "Singapore Office"), text: t("80 Robinson Road, #08-01, 80RR Fintech Hub SG, Singapore", "80 Robinson Road, #08-01, 80RR Fintech Hub SG, Singapore") }, { title: t("一般咨询", "General Enquiries"), text: t("我们的团队会将您的咨询转交给最合适的负责人。", "Our team will direct your enquiry to the most relevant person.") }],
  },
  {
    path: "certifications",
    eyebrow: t("认证影响力", "Certifications Impact"),
    title: t("树立全球金融科技标准", "Setting the Global Standard for Fintech"),
    description: t("GFI为横跨金融、技术、监管和创新领域的专业人士提供结构化认证路径。", "GFI offers a structured certification pathway for professionals navigating finance, technology, regulation, and innovation."),
    image: certificationImage,
    sections: [{ title: t("特许金融科技助理（CFtA）", "Chartered Fintech Associate (CFtA)"), text: t("为学生、转型人才和职业早期专业人士建立金融科技基础。", "A foundational certification for students, career switchers, and early-career professionals."), bullets: [t("数字金融与新兴技术", "Digital finance and emerging technologies"), t("监管与负责任创新", "Regulation and responsible innovation")] }, { title: t("特许金融科技专业人士（CFtP®）", "Chartered Fintech Professional (CFtP®)"), text: t("面向中高级专业人士的旗舰认证，强调高级能力、道德基础和领导准备度。", "GFI's flagship designation for experienced professionals, demonstrating advanced capability, ethical grounding, and leadership readiness.") }, common.standards],
  },
  {
    path: "certifications/pathway",
    eyebrow: t("认证路径", "Certification Pathway"),
    title: t("规划您的金融科技职业旅程", "Plan Your Fintech Career Journey"),
    description: t("从基础知识到专业领导力，选择与您的经验和目标相匹配的认证。", "Move from foundational knowledge to professional leadership with a certification matched to your experience and goals."),
    image: certificationImage,
    sections: [{ title: t("第一步：CFtA", "Step 1: CFtA"), text: t("建立涵盖金融体系、数据、人工智能、区块链、网络安全和治理的核心知识。", "Build core knowledge across financial systems, data, AI, blockchain, cybersecurity, and governance.") }, { title: t("第二步：CFtP®", "Step 2: CFtP®"), text: t("发展应对复杂、受监管金融科技环境所需的高级实践能力。", "Develop advanced practical capability for complex, regulated fintech environments.") }, { title: t("持续发展", "Continuous Development"), text: t("通过会员、研究和全球社群持续更新能力与专业信誉。", "Maintain capability and professional standing through membership, research, and the global community.") }],
  },
  {
    path: "programmes",
    eyebrow: t("项目", "Programmes"),
    title: t("面向未来金融科技职业的学习项目", "Programmes for Future-Ready Fintech Careers"),
    description: t("探索GFI的专业认证与高管短期课程。", "Explore GFI professional certifications and executive short courses."),
    image: certificationImage,
    sections: [{ title: t("专业认证", "Professional Certifications"), text: t("CFtA和CFtP®为不同职业阶段提供清晰的发展路径。", "CFtA and CFtP® provide a clear progression for different career stages.") }, { title: t("高管项目", "Executive Programmes"), text: t("针对数字资产、合规和金融科技领导力等关键议题的集中学习。", "Focused learning on digital assets, compliance, and fintech leadership.") }],
  },
  {
    path: "programmes/cfta",
    eyebrow: t("专业认证", "Professional Certification"),
    title: t("特许金融科技助理（CFtA）", "Chartered Fintech Associate (CFtA)"),
    description: t("面向职业早期专业人士和金融服务新人的基础金融科技认证。全球在线，随时申请。", "A foundational fintech certification for early-career professionals and those new to financial services. Global, online, and open for ongoing applications."),
    image: certificationImage,
    sections: [{ title: t("课程概览", "Programme Overview"), text: t("通过24至30小时的灵活在线学习，系统掌握金融科技基础、数据、人工智能、区块链、网络安全、伦理与治理。", "Build a comprehensive grounding in fintech, data, AI, blockchain, cybersecurity, ethics, and governance through 24-30 hours of flexible online learning."), bullets: [t("全球在线课程", "Global online programme"), t("学习材料已包含", "Learning materials included"), t("基础级证书", "Foundational-level certificate")] }, { title: t("适合人群", "Who It's For"), text: t("职业早期专业人士、转行者、金融科技新人、学生和毕业生。", "Early-career professionals, career changers, fintech newcomers, students, and graduates.") }, { title: t("考试与费用", "Exam & Fees"), text: t("在线考试全天候开放，常规总费用为600美元。", "The online exam is available 24/7. Regular total fees are USD 600.") }],
  },
  {
    path: "programmes/cftp",
    eyebrow: t("旗舰专业认证", "Flagship Professional Certification"),
    title: t("特许金融科技专业人士（CFtP®）", "Chartered Fintech Professional (CFtP®)"),
    description: t("为职业中期及高级专业人士打造的高级认证，连接实用能力、全球标准与领导力。", "An advanced designation for mid-career and senior professionals, connecting practical capability, global standards, and leadership."),
    image: certificationImage,
    sections: [{ title: t("高级实践能力", "Advanced Practical Capability"), text: t("围绕金融科技战略、监管、风险、新兴技术和负责任创新建立深入能力。", "Build deep capability across fintech strategy, regulation, risk, emerging technologies, and responsible innovation.") }, { title: t("全球专业认可", "Global Professional Recognition"), text: t("展示您在复杂金融科技环境中的专业能力、道德判断和领导准备度。", "Demonstrate professional capability, ethical judgement, and leadership readiness in complex fintech environments.") }, common.network],
  },
  {
    path: "programmes/executive-program",
    eyebrow: t("高管项目", "Executive Programme"),
    title: t("加密货币监管与合规基础", "Foundation in Crypto Regulation and Compliance"),
    description: t("GFI与币安合作提供的16小时短期课程，深入理解数字资产监管、区块链基础设施和新兴合规框架。", "A rigorous 16-hour short course co-delivered by GFI and Binance on digital asset regulation, blockchain infrastructure, and emerging compliance frameworks."),
    image: `${assetBase}/3.C958NLmg_1yOg5O.webp`,
    sections: [{ title: t("课程重点", "Course Focus"), text: t("将政策、技术基础设施和真实合规实践结合在一个结构化学习体验中。", "Connect policy, technical infrastructure, and real compliance practice in one structured learning experience."), bullets: [t("数字资产监管", "Digital asset regulation"), t("区块链基础设施", "Blockchain infrastructure"), t("合规框架", "Compliance frameworks")] }, { title: t("适合专业人士", "Built for Professionals"), text: t("适合合规、风险、法律、金融服务和数字资产领域的从业者。", "Designed for practitioners across compliance, risk, legal, financial services, and digital assets.") }],
  },
  {
    path: "partnerships",
    eyebrow: t("生态系统", "Ecosystem"),
    title: t("与GFI合作", "Partner with GFI"),
    description: t("与监管机构、大学、企业和技术领导者共同推动金融科技专业标准。", "Advance fintech professional standards alongside regulators, universities, corporations, and technology leaders."),
    image: ecosystemImage,
    sections: [{ title: t("共同创造行业影响", "Create Industry Impact Together"), text: t("合作形式涵盖认证、课程、研究、活动与人才发展。", "Partnerships span certification, education, research, events, and talent development.") }, common.network],
  },
  {
    path: "membership/benefits",
    eyebrow: t("GFI会员", "Membership at GFI"),
    title: t("伴随整个职业生涯的专业社群", "A Professional Community for Your Entire Career"),
    description: t("分层会员体系支持从探索金融科技的学生到塑造行业未来的资深专业人士。", "Our tiered membership supports everyone from students exploring fintech to senior professionals shaping its future."),
    image: ecosystemImage,
    sections: [{ title: t("附属会员 · 每年49美元", "Affiliate Membership · USD 49/year"), text: t("适合学生、职业早期专业人士和转型人才，获得网络研讨会、报告、活动邀请和课程优惠。", "For students, early-career professionals, and career switchers, with access to webinars, reports, events, and course discounts.") }, { title: t("助理会员 · 每年129美元", "Associate Membership · USD 129/year"), text: t("适合CFtP候选人和CFtA持有者，提供更深入的资源、职业机会和学习优惠。", "For CFtP candidates and CFtA holders, with deeper resources, career opportunities, and enhanced learning benefits.") }, { title: t("特许会员 · 每年169美元", "Charterholder Membership · USD 169/year"), text: t("仅限CFtP®持证人，提供最高级别的专业认可、领导机会和行业参与。", "Reserved for CFtP® charterholders, offering the highest level of professional recognition, leadership opportunity, and industry engagement.") }],
  },
  {
    path: "membership/corporate",
    eyebrow: t("企业赞助人", "Corporate Patrons"),
    title: t("支持全球金融科技专业发展", "Support Global Fintech Professional Development"),
    description: t("企业赞助人计划帮助机构参与人才、研究、教育和行业标准建设。", "The Corporate Patron Programme enables organisations to contribute to talent, research, education, and industry standards."),
    image: ecosystemImage,
    sections: [{ title: t("机构参与", "Institutional Engagement"), text: t("与GFI共同开展专业项目、思想领导力和全球社群活动。", "Work with GFI on professional programmes, thought leadership, and global community initiatives.") }, common.standards],
  },
  ...[
    ["publications/reports", t("报告", "Reports"), t("以研究支持更好的金融科技决策", "Research for Better Fintech Decisions")],
    ["publications/insights", t("洞察", "Insights"), t("来自行业与政策前沿的观点", "Perspectives from the Edge of Industry and Policy")],
    ["news", t("新闻中心", "Press Room"), t("GFI最新新闻和公告", "Latest News and Announcements from GFI")],
    ["publications/journals", t("期刊", "Journals"), t("推动严谨的金融科技知识", "Advancing Rigorous Fintech Knowledge")],
  ].map(([path, eyebrow, title]) => ({
    path: path as string,
    eyebrow: eyebrow as LocalizedText,
    title: title as LocalizedText,
    description: t("探索GFI关于技术、监管、市场和专业实践的最新内容。", "Explore GFI's latest work across technology, regulation, markets, and professional practice."),
    image: publicationImage,
    sections: [{ title: t("值得信赖的专业内容", "Trusted Professional Content"), text: t("我们的内容连接研究证据、行业经验和政策视角，帮助专业人士应对快速变化的金融科技环境。", "Our content connects research evidence, industry experience, and policy perspectives to help professionals navigate a fast-changing fintech environment.") }, common.standards],
  })),
  ...[
    ["events/all-events", t("所有活动", "All Events"), t("连接全球金融科技社群", "Connecting the Global Fintech Community")],
    ["events/webinar-recordings", t("网络研讨会与录像", "Webinars & Recordings"), t("随时获取专家观点", "Expert Perspectives, Available Anytime")],
    ["events/conferences", t("会议与圆桌会议", "Conferences & Roundtables"), t("围绕关键议题展开深度对话", "Deep Dialogue on the Issues That Matter")],
  ].map(([path, eyebrow, title]) => ({
    path: path as string,
    eyebrow: eyebrow as LocalizedText,
    title: title as LocalizedText,
    description: t("通过会议、圆桌和网络研讨会与行业领袖、监管机构和研究人员交流。", "Engage with industry leaders, regulators, and researchers through conferences, roundtables, and webinars."),
    image: ecosystemImage,
    sections: [{ title: t("学习、交流与合作", "Learn, Connect, and Collaborate"), text: t("GFI活动关注真实挑战、可操作经验和跨市场合作。", "GFI events focus on real challenges, practical experience, and cross-market collaboration.") }, common.network],
  })),
  {
    path: "gfi-stories",
    eyebrow: t("校友故事", "Charterholder Stories"),
    title: t("GFI成功故事", "GFI Success Stories"),
    description: t("了解GFI专业人士如何在金融科技、政策、技术与创新领域创造影响。", "Meet GFI professionals creating impact across fintech, policy, technology, and innovation."),
    image: aboutImage,
    sections: [{ title: t("来自全球社群", "From Our Global Community"), text: t("这些经历展现专业认证、持续学习和可信社群如何支持长期职业发展。", "These journeys show how professional certification, continuous learning, and a trusted community support long-term career growth.") }, common.network],
  },
  ...[
    ["gfi-stories/jag-foo-cftp-cpp-psp-pci", "Jag Foo、CFtP、CPP、PSP、PCI", "Jag Foo, CFtP, CPP, PSP, PCI", "Safeheron", t("特许金融科技专业人士计划为认真驾驭金融科技监管、治理与长期发展的专业人士提供了严谨路径。", "The CFtP programme offers a rigorous pathway for professionals navigating fintech regulation, governance, and long-term relevance.")],
    ["gfi-stories/tat-yeen-yap-cftp", "达妍叶，CFtP", "Tat Yeen Yap, CFtP", "Maybank Singapore", t("从金融创新到银行领导力，Tat Yeen Yap将认证学习转化为跨境金融科技实践。", "From financial innovation to banking leadership, Tat Yeen Yap translates certification learning into cross-border fintech practice.")],
    ["gfi-stories/aaron-ting-cftp", "丁亚伦，CFtP", "Aaron Ting, CFtP", "ICP Hub Singapore", t("Aaron Ting结合去中心化基础设施、人工智能和生态系统建设，代表新一代金融科技领导力。", "Aaron Ting combines decentralised infrastructure, AI, and ecosystem building to represent a new generation of fintech leadership.")],
  ].map(([path, zhTitle, enTitle, organisation, summary]) => ({
    path: path as string,
    eyebrow: t("校友故事", "Charterholder Story"),
    title: t(zhTitle as string, enTitle as string),
    description: summary as LocalizedText,
    image: aboutImage,
    sections: [{ title: t("职业历程", "Professional Journey"), text: summary as LocalizedText }, { title: t("组织", "Organisation"), text: t(organisation as string, organisation as string) }, common.network],
  })),
  ...[
    ["news/global-fintech-institute-and-solusfutura-sign-mou-to-bring-chartered-fintech-professional-certification-to-hong-kong-and-the-greater-bay-area", t("全球金融科技学院与SolusFutura签署谅解备忘录", "Global Fintech Institute and SolusFutura Sign MOU"), t("双方建立战略合作伙伴关系，在香港和大湾区推进金融科技专业认证与教育。", "The organisations established a strategic partnership to advance fintech professional certification and education across Hong Kong and the Greater Bay Area.")],
    ["news/cfta-is-now-live-enhanced-curriculum-for-a-fast-moving-fintech-industry", t("CFtA现已上线：升级课程服务快速发展的行业", "CFtA is Now Live with an Enhanced Curriculum"), t("新版CFtA扩展课程与合作伙伴生态，继续提供可信的金融科技基础教育。", "The enhanced CFtA expands its curriculum and partner ecosystem while maintaining a commitment to credible foundational fintech education.")],
    ["news/introducing-the-digital-assets-security-and-compliance-subcommittee", t("数字资产安全与合规小组委员会正式成立", "Introducing the Digital Assets Security and Compliance Subcommittee"), t("新委员会将推进数字资产生态系统中的网络安全、合规和运营韧性研究。", "The new subcommittee advances institutional thinking on cybersecurity, compliance, and operational resilience in digital asset ecosystems.")],
  ].map(([path, title, summary]) => ({
    path: path as string,
    eyebrow: t("新闻中心", "Press Room"),
    title: title as LocalizedText,
    description: summary as LocalizedText,
    image: publicationImage,
    sections: [{ title: t("公告", "Announcement"), text: summary as LocalizedText }, common.standards],
  })),
  {
    path: "privacy-policy",
    eyebrow: t("法律", "Legal"),
    title: t("隐私政策", "Privacy Policy"),
    description: t("了解GFI如何收集、使用和保护个人信息。", "Learn how GFI collects, uses, and protects personal information."),
    image: aboutImage,
    sections: [{ title: t("信息处理", "Information Handling"), text: t("我们仅为提供服务、回应咨询、改进体验和履行法律义务而处理必要信息。", "We process necessary information to deliver services, respond to enquiries, improve experiences, and meet legal obligations.") }, { title: t("您的权利", "Your Rights"), text: t("您可以就个人信息访问、更正或删除请求联系我们。", "You may contact us to request access, correction, or deletion of your personal information.") }],
  },
  {
    path: "terms-conditions",
    eyebrow: t("法律", "Legal"),
    title: t("条款与条件", "Terms & Conditions"),
    description: t("使用GFI网站、课程、认证和会员服务时适用的条款。", "Terms applying to the use of GFI websites, programmes, certifications, and membership services."),
    image: aboutImage,
    sections: [{ title: t("服务使用", "Use of Services"), text: t("用户应合法、诚信地使用GFI服务，并遵守适用的注册、付款和知识产权规则。", "Users must use GFI services lawfully and in good faith, following applicable registration, payment, and intellectual property rules.") }, { title: t("专业认证", "Professional Certification"), text: t("认证资格取决于满足相应项目、考试和持续专业要求。", "Certification status depends on meeting the relevant programme, examination, and continuing professional requirements.") }],
  },
]

export function getGfiPage(path: string) {
  return gfiPages.find((page) => page.path === path)
}
