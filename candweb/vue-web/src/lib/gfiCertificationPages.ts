import type { LocalizedText } from "@/lib/gfiSite"

export const certText = (zh: string, en: string): LocalizedText => ({ zh, en })

export type CertificationContent = {
  heading: LocalizedText
  paragraphs?: LocalizedText[]
  bullets?: LocalizedText[]
  groups?: Array<{ heading: LocalizedText; bullets: LocalizedText[] }>
}

export type CertificationTab = {
  key: string
  label: LocalizedText
  content: CertificationContent[]
}

export const certificationAssets = {
  hero: "https://globalfintechinstitute.org/assets/gfi-banner-2.COpPUN29_Z1eroN0.webp",
  globalStandard: "https://globalfintechinstitute.org/images/certifications/global-standard-graphics.png",
  professional: "https://globalfintechinstitute.org/assets/professional-posing.5FkZgM1T_ZRku9E.webp",
  pathwayPattern: "https://globalfintechinstitute.org/assets/bg.CZWEzqel_Z5lu0T.svg",
  pathwayLines: "https://globalfintechinstitute.org/assets/line-pattern.d60pUyQJ_AYMiN.svg",
  cftaLogo: "https://globalfintechinstitute.org/images/certifications/cfta-stacked-logo.png",
  cftpLogo: "https://globalfintechinstitute.org/images/certifications/cftp-stacked-logo.png",
  charterholder: "https://globalfintechinstitute.org/assets/3.ByAwyiz4_Z1pXris.webp",
  cftaBanner: "https://globalfintechinstitute.org/images/CftA-banner-img.jpg",
  cftpBanner: "https://globalfintechinstitute.org/images/CftP-banner-img.jpg",
} as const

export const pathwayStages = [
  {
    number: "01",
    title: certText("特许金融科技助理 (CFtA)", "Chartered Fintech Associate (CFtA)"),
    level: certText("基础级别", "Foundation Level"),
    paragraphs: [
      certText("CFtA专为正在建立金融科技基础知识理解的人士设计。它提供金融、技术、监管和道德的结构化介绍，帮助候选人全面了解金融科技如何在市场中运作。", "The CFtA is designed for individuals building their understanding of fintech fundamentals. It provides a structured introduction to finance, technology, regulation, and ethics, helping candidates develop a holistic view of how fintech operates across markets."),
      certText("这一步非常适合进入金融科技领域、转换角色或寻求共同知识基础的人士，然后再进入更专业或高级的职责。", "This step is ideal for those entering fintech, transitioning roles, or seeking a common knowledge baseline before advancing into more specialised or senior responsibilities."),
    ],
    link: "/gfi/programmes/cfta",
    linkLabel: certText("从CFtA开始", "Start with CFtA"),
    image: certificationAssets.cftaLogo,
  },
  {
    number: "02",
    title: certText("特许金融科技专业人士 (CFtP®)", "Chartered Fintech Professional (CFtP®)"),
    level: certText("高级认证", "Advanced Certification"),
    paragraphs: [
      certText("CFtP®建立在基础知识之上，专注于高级应用、治理、技术风险和全球金融科技趋势。它专为在受监管、高影响环境中运营的专业人士设计，这些环境中的决策需要技术深度和良好的判断力。", "The CFtP® builds on foundational knowledge to focus on advanced applications, governance, technology risk, and global fintech trends. It is designed for professionals operating in regulated, high-impact environments where decisions require technical depth and sound judgement."),
      certText("进入此阶段的候选人不仅展示了专业知识，还展示了在涉及机构、监管机构和跨境运营的复杂现实环境中应用框架的能力。", "Candidates who progress to this stage demonstrate not only subject-matter expertise, but the ability to apply frameworks across complex, real-world contexts involving institutions, regulators, and cross-border operations."),
    ],
    link: "/gfi/programmes/cftp",
    linkLabel: certText("探索直接CFtP®入学", "Explore Direct CFtP® Entry"),
    image: certificationAssets.cftpLogo,
  },
  {
    number: "03",
    title: certText("CFtP®特许持有人身份", "CFtP® Charterholder Status"),
    level: certText("认可的专业人士", "Recognized Professional"),
    paragraphs: [
      certText("CFtP®特许持有人身份授予已完成CFtP®计划、满足所需工作经验标准并承诺遵守GFI道德和专业标准的专业人士。它代表从认证到认可的专业地位的转变。", "CFtP® Charterholder Status is awarded to professionals who have completed the CFtP® programme, met the required work experience criteria, and committed to GFI's ethical and professional standards. It represents a transition from certification to recognised professional standing."),
      certText("特许持有人因其可信度、责任感和领导准备而受到认可。此身份可访问高级网络、领导论坛和职业机会，向雇主、合作伙伴和政策制定者表明值得信赖的能力。", "Charterholders are recognised for their credibility, accountability, and readiness to lead. This status grants access to senior networks, leadership forums, and career opportunities, signalling trusted capability to employers, partners, and policymakers."),
    ],
    link: "",
    linkLabel: certText("", ""),
    image: certificationAssets.charterholder,
  },
] as const

const cftaTabs: CertificationTab[] = [
  {
    key: "overview",
    label: certText("概述", "Overview"),
    content: [
      {
        heading: certText("关于特许金融科技助理 (CFtA)", "About Chartered Fintech Associate (CFtA)"),
        paragraphs: [certText("特许金融科技助理(CFtA)是我们为早期职业专业人士和金融服务新手设计的基础金融科技认证项目。这个综合项目提供在快速发展的金融科技行业中成功所需的基本知识和技能。", "The Chartered Fintech Associate (CFtA) is our foundational fintech certification program designed for early-career professionals and those new to financial services. This comprehensive program provides essential knowledge and skills needed to succeed in the rapidly evolving fintech industry.")],
      },
      {
        heading: certText("项目优势", "Programme Strengths"),
        bullets: [
          certText("基础且全面: 完整介绍金融科技基础知识和金融系统", "Foundational & Comprehensive: Complete introduction to fintech fundamentals and financial systems"),
          certText("行业认可认证: 在金融服务和金融科技领域受到尊重的证书", "Industry-Recognized Certification: Credential respected across financial services and fintech sectors"),
          certText("易于接受的入学要求: 最低N级教育，无需金融科技经验", "Accessible Entry Requirements: Designed for candidates starting their fintech learning journey"),
          certText("灵活的在线形式: 全天候24/7可用，自主学习方式", "Flexible Online Format: Available 24/7 with self-paced learning approach"),
          certText("职业发展基础: 通向CFtP等高级认证的完美跳板", "Career Pathway Foundation: Perfect stepping stone to advanced certifications like CFtP"),
        ],
      },
      {
        heading: certText("适合人群", "Who It's For"),
        bullets: [
          certText("在金融服务领域开始职业生涯的早期职业专业人士", "Early-Career Professionals starting their journey in financial services"),
          certText("从其他行业转入金融科技的职业转换者", "Career Changers transitioning into fintech from other industries"),
          certText("寻求金融科技基础知识的金融新手", "Finance Newcomers seeking foundational knowledge in financial technology"),
          certText("希望专门从事金融科技和数字金融的学生和毕业生", "Students and Graduates looking to specialize in fintech and digital finance"),
        ],
      },
    ],
  },
  {
    key: "curriculum",
    label: certText("课程大纲", "Curriculum"),
    content: [
      {
        heading: certText("课程设置", "Curriculum"),
        groups: [
          { heading: certText("模块1: 金融系统概述", "Chapter 1: Overview of Fintech"), bullets: [certText("金融系统的核心功能", "What Is Fintech"), certText("金融系统的监管和监督", "Evolution of Fintech"), certText("传统金融和痛点", "Impact and Benefits: Reshaping Financial Industries")] },
          { heading: certText("模块2: 金融科技的演进", "Chapter 2: The Traditional Financial System"), bullets: [certText("金融科技的定义", "Core Functions of the Financial System"), certText("金融科技1.0、2.0、3.0及其未来", "Key Institutions and Division of Roles"), certText("金融科技的现状", "A Classic Transaction Walkthrough: How Traditional Finance Operates"), certText("金融科技创新的潜在风险", "The Banking Achilles' Heel: Maturity Transformation and Trust")] },
          { heading: certText("模块3: 金融科技生态系统", "Chapter 3: Data in Fintech: How Information Becomes Products, Decisions, and Trust"), bullets: [certText("金融科技领域的关键参与者", "Why Data Is the Backbone of Fintech"), certText("金融科技子领域", "The Fintech Data Lifecycle"), certText("技术和基础设施", "Sources of Fintech Data"), certText("金融科技法规", "Data Access and Sharing: APIs and Open Banking")] },
          { heading: certText("模块4: 金融科技对传统金融的影响", "Chapter 4: Artificial Intelligence in Financial Services"), bullets: [certText("颠覆与合作", "Foundations of AI"), certText("客户期望的变化", "Applications of AI in Fintech"), certText("效率和成本降低", "Language AI and Text Processing"), certText("金融包容性", "Generative AI Tools and Working Methods")] },
          { heading: certText("模块5: 全球金融科技中心和市场趋势", "Chapter 5: Blockchain in Financial Services"), bullets: [certText("主要金融科技中心", "Why Blockchain Matters in Fintech"), certText("市场趋势", "How Blockchain Works"), certText("投资趋势", "Why Crypto Exists and Why Bitcoin and Ethereum Became Important"), certText("监管趋势", "Tokenization and Real-World Use Cases"), certText("新兴市场和机会", "Risks, Limits, and Reality Checks")] },
          { heading: certText("模块6: 职业道德、行为准则和负责任的创新", "Chapter 6: Cybersecurity in Fintech"), bullets: [certText("金融科技中道德的重要性", "Why Cybersecurity Matters in Fintech"), certText("金融科技中的职业行为", "Core Concepts in Cybersecurity"), certText("负责任的创新", "The Main Cybersecurity Threats in Fintech"), certText("道德决策框架", "What Good Security Looks Like in Practice"), certText("金融科技道德案例研究", "How Safe Is Fintech Today?"), certText("在金融科技组织中建立道德文化", "Blockchain, Cryptography, and Trust"), certText("金融科技道德的未来", "Trends, Opportunities, and Future Challenges"), certText("总结", "Applying the Knowledge")] },
        ],
      },
    ],
  },
  {
    key: "eligibility",
    label: certText("入学要求", "Eligibility"),
    content: [{
      heading: certText("注册要求", "Registration Requirements"),
      paragraphs: [certText("要成为CFtA候选人，您必须注册CFtA项目并报名考试。要注册，您必须满足以下所有标准：", "To become a CFtA candidate, you must enrol in the CFtA Program and register for the exam. To enrol, you must meet the criteria below:")],
      bullets: [certText("最低N级或同等学历", "Candidates must be at least 18 years old to enrol in the CFtA certification exam"), certText("候选人必须年满18岁才能注册CFtA认证考试", "Must be able to read and write in English (exam language)"), certText("必须能够用英语读写(考试语言)", "")],
    }],
  },
  {
    key: "fees",
    label: certText("考试与费用", "Exam & Fees"),
    content: [
      { heading: certText("考试和费用", "Exam & Fees") },
      { heading: certText("考试注册指南", "Exam Registration Guidelines"), bullets: [certText("在GFI网站或学习平台上完成注册", "Complete registration on the GFI website or Learn platform"), certText("提交所需文件并确认详细信息", "Submit required documents and confirm details"), certText("通过信用卡或银行转账支付费用", "Pay fees via credit card or bank transfer"), certText("收到包含考试详情的确认电子邮件", "Receive confirmation email with exam details")] },
      { heading: certText("考试详情", "Exam Details"), bullets: [certText("在线考试: 每天24/7可用", "Online Exam: Available daily 24/7")] },
      { heading: certText("费用结构", "Fees Structure"), groups: [
        { heading: certText("常规", "Regular"), bullets: [certText("注册费: $100", "Total Fees Payable: USD 600"), certText("考试注册: $600", ""), certText("应付总费用: $700", "")] },
        { heading: certText("附属会员资格", "Associate Membership"), bullets: [certText("第1年: 免费", "1st Year: Waived"), certText("后续年份: $129(每年计费)", "Subsequent Year(s): $129 (Billed annually)")] },
      ] },
      { heading: certText("取消和退款", "Cancellations & Refunds"), bullets: [certText("截止日期前允许退款(适用管理费)", "Refunds allowed before deadline (admin fee applies)"), certText("截止日期后不退款", "No refunds after deadline")] },
    ],
  },
]

const cftpTabs: CertificationTab[] = [
  {
    key: "overview",
    label: certText("概述", "Overview"),
    content: [
      {
        heading: certText("项目优势", "Programme Strengths"),
        bullets: [
          certText("高级且全面：深入涵盖金融、金融科技应用和新兴技术", "Advanced & Comprehensive: In-depth coverage of finance, fintech applications, and emerging technologies"),
          certText("双途径入学：通过CFtA认证或具有相关资格的直接入学", "Dual Pathway Entry: Accessible via CFtA certification or direct entry with relevant qualifications"),
          certText("行业领先课程：人工智能/机器学习、区块链、量子计算和网络安全的前沿内容", "Industry-Leading Curriculum: Cutting-edge content in AI/ML, blockchain, quantum computing, and cybersecurity"),
          certText("全球认可：在金融市场、央行和金融科技领域受到尊重的证书", "Global Recognition: Credential respected across financial markets, central banks, and fintech sectors"),
          certText("专业卓越：为寻求职业发展的中高级专业人士设计", "Professional Excellence: Designed for mid to senior-level professionals seeking career advancement"),
        ],
      },
      {
        heading: certText("适合人群", "Who It's For"),
        bullets: [
          certText("职业中期专业人士寻求专门从事高级金融科技应用", "Mid-Career Professionals seeking to specialize in advanced fintech applications"),
          certText("高级领导者希望深化其金融科技专业知识和信誉", "Senior Leaders wanting to deepen their fintech expertise and credibility"),
          certText("金融和科技高管在传统金融和创新之间架起桥梁", "Finance & Tech Executives bridging gaps between traditional finance and innovation"),
          certText("CFtA持有者准备晋升到专业级认证", "CFtA Holders ready to advance to professional-level certification"),
          certText("合格专业人士具有相关学位或认证（CFA、CAIA等）", "Qualified Professionals with relevant degrees or certifications (CFA, CAIA, etc.)"),
        ],
      },
      {
        heading: certText("关于特许金融科技专业人士 (CFtP®)", "About Chartered Fintech Professional (CFtP®)"),
        paragraphs: [certText("特许金融科技专业人士(CFtP®)是我们的顶级金融科技认证项目，专为寻求金融技术高级专业知识的经验丰富的专业人士设计。这个严格的项目提供金融、技术、道德和全球金融科技趋势的综合知识，具有灵活的入学途径。", "The Chartered Fintech Professional (CFtP®) is our premier fintech certification program designed for experienced professionals seeking advanced expertise in financial technology. This rigorous program is developed by Global Fintech Institute (GFI) in collaboration with its academic partners, council members, industry leaders and institutions worldwide. The aims of the qualification are the promotion of the field of Fintech and professionalism in the Fintech industry."), certText("", "In addition, the CFtP provides a pathway for professionals in other industries who aspire to make a career in Fintech. The CFtP consists of two levels, Level 1 and Level 2. Upon successful completion of all the exams for the two levels as well as satisfying experience requirements, candidates will be awarded the Charter.")],
      },
    ],
  },
  {
    key: "curriculum",
    label: certText("课程大纲", "Curriculum"),
    content: [{
      heading: certText("课程设置", "Curriculum"),
      groups: [
        { heading: certText("第一级：基础", "Level 1: Foundations"), bullets: [] },
        { heading: certText("1AB 基础（仅直接途径）", "1AB Foundation (Direct Pathway Only)"), bullets: [certText("道德与治理（I）：基础原则和框架", "Ethics & Governance (I): Foundational principles and frameworks"), certText("定量方法：统计分析和金融建模", "Quantitative Methods: Statistical analysis and financial modeling"), certText("区块链与金融创新：核心分布式账本技术", "Blockchain & Financial Innovations: Core distributed ledger technologies")] },
        { heading: certText("1A 金融", "1A Finance"), bullets: [certText("经济学：宏观经济原理和金融市场", "Economics: Macroeconomic principles and financial markets"), certText("财务报表分析：高级分析技术", "Financial Statement Analysis: Advanced analytical techniques"), certText("财务管理：公司金融和资本配置", "Financial Management: Corporate finance and capital allocation"), certText("投资管理：投资组合理论和资产管理", "Investment Management: Portfolio theory and asset management")] },
        { heading: certText("1B 金融科技", "1B Fintech"), bullets: [certText("数据结构与Python：金融编程基础", "Data Structures & Python: Programming fundamentals for finance"), certText("大数据与数据科学：分析和数据处理", "Big Data & Data Science: Analytics and data processing"), certText("人工智能与机器学习：算法交易和风险建模", "AI & Machine Learning: Algorithmic trading and risk modeling"), certText("计算机网络与安全：基础设施和网络安全", "Computer Networks & Security: Infrastructure and cybersecurity")] },
        { heading: certText("第二级：应用与趋势", "Level 2: Applications & Trends"), bullets: [] },
        { heading: certText("核心高级模块", "Core Advanced Modules"), bullets: [certText("道德与治理（II）：高级道德框架和监管合规", "Ethics & Governance (II): Advanced ethical frameworks and regulatory compliance"), certText("金融中的人工智能、机器学习和深度学习：高级应用和实施", "AI, Machine Learning & Deep Learning in Finance: Advanced applications and implementation"), certText("区块链编程与数字货币：智能合约和加密货币技术", "Blockchain Programming & Digital Currency: Smart contracts and cryptocurrency technologies"), certText("云计算、网络安全与量子计算：基础设施和新兴技术", "Cloud Computing, Cybersecurity & Quantum Computing: Infrastructure and emerging technologies"), certText("合规与技术风险管理：监管框架和风险缓解", "Compliance & Technology Risk Management: Regulatory frameworks and risk mitigation"), certText("全球金融科技趋势：市场分析和未来发展", "Global Fintech Trends: Market analysis and future developments")] },
      ],
    }],
  },
  {
    key: "eligibility",
    label: certText("入学要求", "Eligibility"),
    content: [{
      heading: certText("入学要求", "Eligibility"),
      groups: [
        { heading: certText("CFtA到CFtP®途径", "CFtA to CFtP® Pathway"), bullets: [certText("通过CFtA认证入学（无需学位）", "Entry via CFtA certification (no degree required)"), certText("必须年满21岁", "Must be above 21 years of age"), certText("2年相关工作经验", "2 years relevant work experience"), certText("2个专业推荐人", "2 professional references"), certText("所需考试：1AB级（基础）、1A级和1B级（金融、金融科技）、2A级和2B级（应用）", "Exams Required: Level 1AB (Foundation), Level 1A & 1B (Finance, Fintech), Level 2A & 2B (Applications)")] },
        { heading: certText("直接到CFtP®途径（需要学位）", "Direct to CFtP® Pathway (Degree Required)"), bullets: [certText("认可机构的学士学位，或", "Bachelor's degree from recognized institution, OR"), certText("相关专业资格（CFA、CAIA等），或", "Relevant professional qualification (CFA, CAIA, etc.), OR"), certText("认可大学的最后一年学习", "Final-year studies in recognized universities"), certText("必须年满21岁", "Must be above 21 years of age"), certText("2年相关工作经验", "2 years relevant work experience"), certText("2个专业推荐人", "2 professional references"), certText("所需考试：基础（1AB）、1A级和1B级、2A级和2B级", "Exams Required: Foundation (1AB), Level 1A & 1B, Level 2A & 2B")] },
      ],
    }],
  },
  {
    key: "exemptions",
    label: certText("豁免", "Exemption"),
    content: [{
      heading: certText("豁免", "Exemptions"),
      paragraphs: [certText("CFtP 1级考试的豁免基于之前修读的项目或课程。考生可通过豁免申请表提交豁免申请。1AB级必修基础和2级考试不授予豁免。以下认可项目可授予豁免：", "Exemptions from CFtP Level 1 exams are based on prior programs or courses taken. Candidates can submit his/her exemption request through the exemption application form. No exemptions will be granted for Level 1AB Compulsory Foundation and Level 2 exams. Exemptions are granted for accredited programs, as follow:")],
      groups: [
        { heading: certText("新加坡社会科学大学", "Singapore University of Social Sciences"), bullets: [certText("金融学士 – 根据所修课程，可豁免1A级金融和/或1B级金融科技。", "Bachelor of Finance – Exempt from Level 1A Finance and/or Level 1B Fintech depending on courses taken."), certText("金融硕士 – 根据所修课程，可豁免1A级金融和/或1B级金融科技。", "Master of Finance – Exempt from Level 1A Finance and/or Level 1B Fintech depending on courses taken."), certText("金融科技硕士 – 根据所修课程，可豁免1A级金融和/或1B级金融科技。", "Master of FinTech – Exempt from Level 1A Finance and/or Level 1B Fintech depending on courses taken.")] },
        { heading: certText("上海财经大学金融学院", "School of Finance, Shanghai University of Finance and Economics"), bullets: [certText("金融学士 – 可豁免1A级金融。", "Bachelor of Finance – Exempt from Level 1A Finance."), certText("双学位项目（金融科技基础班） – 可豁免1A级金融和1B级金融科技。", "Double degree program (FinTech Base Class) – Exempt from Level 1A Finance and Level 1B Fintech."), certText("金融科技理学硕士 – 可豁免1A级金融和1B级金融科技。", "MSc of Fintech – Exempt from Level 1A Finance and Level 1B Fintech.")] },
        { heading: certText("新加坡国立大学", "National University of Singapore"), bullets: [certText("工商管理学士（BBA） – 根据所修课程，可豁免1A级金融和/或1B级金融科技。", "Bachelor of Business Administration (BBA) – Exempt from Level 1A Finance and/or Level 1B Fintech depending on courses taken."), certText("金融理学硕士（MFIN） – 根据所修课程，可豁免1A级。", "Master of Science in Finance (MFIN) – Exempt from Level 1A depending on courses taken.")] },
        { heading: certText("新加坡管理大学", "Singapore Management University"), bullets: [certText("应用金融理学硕士（MAF） – 根据所修课程，可豁免1A级金融和1B级金融科技。", "Master of Science in Applied Finance (MAF) – Exempt from Level 1A Finance and Level 1B Fintech depending on courses taken.")] },
        { heading: certText("新加坡南洋理工大学", "Nanyang Technological University Singapore"), bullets: [certText("商业学士（银行与金融） – 根据所修课程，可豁免1A级金融和1B级金融科技。", "Bachelor of Business (Banking & Finance) – Exempt from Level 1A Finance and Level 1B Fintech depending on courses taken."), certText("理学硕士（金融科技） – 根据所修课程，可豁免1A级金融和1B级金融科技。", "Master of Science (Financial Technology) – Exempt from Level 1A Finance and Level 1B Fintech depending on courses taken.")] },
        { heading: certText("美国印第安纳大学凯利商学院", "Indiana University Kelley Business School, USA"), bullets: [certText("金融理学硕士（MSF） – 根据所修课程，可豁免1A级金融和/或1B级金融科技。", "Master of Science in Finance (MSF) – Exempt from Level 1A Finance and/or Level 1B Fintech depending on courses taken.")] },
        { heading: certText("澳大利亚默多克大学", "Murdoch University, Australia"), bullets: [certText("银行与金融商业学士（双专业） – 根据所修课程，可豁免1A级金融。", "Bachelor of Business in Banking and Finance (Double Major) – Exempt from Level 1A Finance."), certText("金融科技数据分析学士（主修）及信息与系统安全（辅修） – 根据所修课程，可豁免1B级金融科技。", "Bachelor of Data Analytics in Fintech (Major) and Information and Systems Security (Minor) – Exempt from Level 1B Fintech.")] },
        { heading: certText("专业认证", "Professional certification"), bullets: [certText("CFA 1级及以上", "CFA Level 1 and above"), certText("CAIA 1级及以上", "CAIA Level 1 and above")] },
      ],
    }],
  },
  {
    key: "fees",
    label: certText("考试与费用", "Exam & Fees"),
    content: [
      { heading: certText("考试与费用", "Exam & Fees") },
      { heading: certText("考试窗口", "Exam Windows"), bullets: [certText("7月和12月：通过Prometric考试中心或远程监考", "July and December: Via Prometric test centres or remote proctoring"), certText("在线考试：基础级每天24/7可用", "Online Exam: Foundation level available daily 24/7")] },
      { heading: certText("考试详情", "Exam Details"), bullets: [certText("基础（1AB）：120分钟内60道多选题（仅直接途径）", "Foundation (1AB): 60 MCQs in 120 minutes (direct pathway only)"), certText("1A级金融/1B级金融科技：每次考试180分钟内90道多选题", "Level 1A Finance / 1B Fintech: 90 MCQs in 180 minutes per exam"), certText("2级（2A和2B）：每次180分钟内75道多选题加3道简答题", "Level 2 (2A and 2B): 75 MCQs plus 3 short-answer questions in 180 minutes each")] },
      { heading: certText("注册规则", "Registration Rules"), bullets: [certText("每个考试窗口一个级别", "One level per examination window"), certText("必须通过1级才能参加2级", "Must pass Level 1 before attempting Level 2"), certText("每次注册2次考试机会", "Two examination attempts per registration"), certText("CFtA持有者可绕过学位要求", "CFtA holders may bypass the degree requirement")] },
      { heading: certText("费用结构", "Fees Structure"), groups: [
        { heading: certText("CFtA到CFtP®途径", "CFtA to CFtP® Pathway"), bullets: [certText("项目注册：$300", "Programme registration: $300"), certText("CFtA费用：$600", "CFtA fee: $600"), certText("考试注册：每级别$1,500（1级和2级）", "Exam registration: $1,500 per level (Level 1 and Level 2)"), certText("应付总费用：$3,900", "Total fees payable: $3,900")] },
        { heading: certText("直接到CFtP®途径", "Direct to CFtP® Pathway"), bullets: [certText("项目注册：$300", "Programme registration: $300"), certText("考试注册：每级别$1,500（1级和2级）", "Exam registration: $1,500 per level (Level 1 and Level 2)"), certText("应付总费用：$3,300", "Total fees payable: $3,300")] },
        { heading: certText("附属会员资格", "Associate Membership"), bullets: [certText("第1年：免费", "1st Year: Waived"), certText("后续年份：$129（每年计费）", "Subsequent Year(s): $129 (Billed annually)")] },
      ] },
      { heading: certText("取消与退款", "Cancellations & Refunds"), bullets: [certText("截止日期前允许退款（适用管理费）", "Refunds allowed before deadline (admin fee applies)"), certText("截止日期后不退款", "No refunds after deadline"), certText("所有价格以美元计算，2025年有效", "All prices are in USD and valid for 2025")] },
    ],
  },
]

export const certificationPrograms = {
  cfta: {
    key: "cfta",
    banner: certificationAssets.cftaBanner,
    eyebrow: certText("专业认证", "Professional Certification"),
    title: certText("特许金融科技助理 (CFtA)", "Chartered Fintech Associate (CFtA)"),
    type: certText("在线项目", "Online Program"),
    price: "600",
    registerUrl: "https://portal.globalfintechinstitute.org/certifications/881fd904-683f-400f-b261-83c80fdd5137",
    tabs: cftaTabs,
  },
  cftp: {
    key: "cftp",
    banner: certificationAssets.cftpBanner,
    eyebrow: certText("专业认证", "Professional Certification"),
    title: certText("特许金融科技专业人士 (CFtP®)", "Chartered Fintech Professional (CFtP®)"),
    type: certText("高级项目", "Advanced Program"),
    price: "3,300",
    registerUrl: "https://airtable.com/app3CG0QruzsNSzMI/pagG1uk0ML2BmgDMv/form",
    handbookUrl: "https://app-na1.hubspotdocuments.com/documents/9495468/view/1284480747?accessId=4d3fe2",
    tabs: cftpTabs,
  },
} as const
