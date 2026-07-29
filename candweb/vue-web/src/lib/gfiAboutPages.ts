import type { LocalizedText } from "@/lib/gfiSite"

export const tx = (zh: string, en: string): LocalizedText => ({ zh, en })

export const aboutImages = [
  "gfi-about-1.BPJKq5hL_1NRqCB.webp",
  "gfi-about-2.CmNknN6S_1UfmRE.webp",
  "gfi-about-3.SguyxKbo_C4HVL.webp",
  "gfi-about-4.B6DxJnwE_4QDKn.webp",
  "gfi-about-5.DtgjLV6A_ZmcDuG.webp",
  "gfi-about-6.Bj9dUDyo_1Tip23.webp",
  "gfi-about-7.DBJyuoYa_17ylmJ.webp",
  "gfi-about-8.BinRJry9_Z2vLrrT.webp",
  "gfi-about-9.Ch5A14g__1zW7cu.webp",
].map((name) => `https://globalfintechinstitute.org/assets/${name}`)

export const peoplePages = {
  "about/board-of-directors": {
    title: tx("董事会", "Board of Directors"),
    introTitle: tx("管理、治理和战略方向", "Management, Governance and Strategic Direction"),
    intro: tx(
      "全球金融科技研究院董事会提供战略监督和治理，以推进专业标准、负责任的创新以及金融科技领域的全球合作。董事会由来自学术界、行业和公共政策的高级领导者组成，指导GFI的长期战略，维护其独立性和完整性，并确保其项目保持严谨、相关和全球认可。",
      "The Board of Directors provides strategic oversight and governance to advance professional standards, responsible innovation and global collaboration in fintech. Senior leaders from academia, industry and public policy guide GFI's long-term strategy, safeguard its independence and integrity, and ensure its programmes remain rigorous, relevant and globally recognised.",
    ),
    eyebrow: tx("董事会", "Board of Directors"),
    membersTitle: tx("董事会成员简介", "Meet the Board"),
    members: [
      {
        name: "Prof. David Lee Kuo Chen, CFtP",
        role: tx("董事会主席", "Chairman, Board of Directors"),
        image: "https://globalfintechinstitute.org/assets/David.Dg-Bjpyf_2rIYh4.webp",
        bio: tx(
          "David Lee教授是金融科技、数字资产和金融创新领域享有国际声誉的学者与思想领袖。他曾就新兴技术、市场结构和政策发展为政府、金融机构及国际组织提供咨询。作为董事会主席，他为GFI提供战略领导和治理监督，确保认证、研究与合作伙伴关系始终保持最高水平的学术严谨性、专业公信力与全球相关性。",
          "Prof David Lee is an internationally recognised academic and thought leader in fintech, digital assets, and financial innovation. He has advised governments, financial institutions, and international organisations on emerging technologies, market structure, and policy development. As Chairman of the Board, Prof David provides strategic leadership and governance oversight to ensure GFI's certifications, research, and partnerships uphold the highest standards of academic rigour, professional credibility, and global relevance.",
        ),
      },
      {
        name: "Dr. Joseph Lim, CFtP, CFA",
        role: tx("董事", "Board Director"),
        image: "https://globalfintechinstitute.org/assets/Joseph.xsu7TRxl_wpP7V.webp",
        bio: tx(
          "Joseph Lim是一位经验丰富的金融专业人士，在投资管理、金融市场和金融科技融合方面拥有深厚专长。他将传统金融与新兴金融技术的实践视角带入专业标准和能力建设，并确保GFI项目贴近真实行业需求、专业期望和从业者的长期职业发展。",
          "Joseph Lim is a seasoned finance professional with deep expertise across investment management, financial markets, and fintech integration. With experience spanning traditional finance and emerging financial technologies, he brings a practitioner's perspective to professional standards and capability-building. As a Board Director, Joseph contributes to GFI's strategic direction, ensuring its programmes remain aligned with real-world industry needs, professional expectations, and long-term career relevance for practitioners.",
        ),
      },
      {
        name: "Kenneth Oh",
        role: tx("董事", "Board Director"),
        image: "https://globalfintechinstitute.org/assets/Kenneth.DhJyCAPE_Z1GnMuV.webp",
        bio: tx(
          "Kenneth Oh是Dentons Rodyk公司业务的高级合伙人，也是该所中国和印度尼西亚业务的合伙人。他在企业融资和资本市场方面经验丰富，作为董事为GFI的专业标准、监管一致性和可信市场实践提供法律与治理专长。",
          "Kenneth Oh is a Senior Partner in Dentons Rodyk's Corporate practice and a Partner in the firm's China and Indonesia practices. With extensive experience in corporate finance and capital markets, he advises on IPOs, reverse takeovers, secondary listings, and post-listing compliance across Singapore and regional markets. As a Board Director, Kenneth brings legal and governance expertise to support GFI's commitment to professional standards, regulatory alignment, and credible market practice.",
        ),
      },
    ],
  },
  "about/team": {
    title: tx("认识团队", "Meet the Team"),
    introTitle: tx("在全球范围内执行全球金融科技研究院的使命", "Delivering GFI's Mission Globally"),
    intro: tx(
      "GFI的领导团队负责将战略转化为认证、合作伙伴关系、研究和社区参与方面的执行。与董事会、理事会和行业研究员密切合作，团队确保GFI的项目以一致性、严谨性和运营卓越性交付——支持跨市场的专业人士和组织。",
      "GFI's leadership team turns strategy into execution across certifications, partnerships, research and community engagement. Working closely with the Board, councils and Industry Fellows, the team delivers programmes with consistency, rigour and operational excellence for professionals and organisations across markets.",
    ),
    eyebrow: tx("领导团队", "Leadership Team"),
    membersTitle: tx("领导团队简介", "Meet the Leadership Team"),
    members: [
      { name: "Vincent Chan, CFtP, MBA", role: tx("首席运营官", "Chief Operations Officer"), image: "https://globalfintechinstitute.org/assets/Vincent.CBYQE9cc_3G1yD.webp", bio: tx("Vincent负责监督GFI在认证、项目、治理和内部流程方面的运营执行，确保项目交付的一致性、质量和严谨性。", "Vincent oversees GFI's operational execution across certifications, programmes, governance, and internal processes. He ensures consistency, quality, and rigour in programme delivery, supporting the Institute's commitment to professional standards and organisational excellence.") },
      { name: "Ho Hui Yi", role: tx("首席战略官", "Chief Strategy Officer"), image: "https://globalfintechinstitute.org/assets/HuiYi.zLK46reb_ZjeCW4.webp", bio: tx("Hui Yi负责GFI的战略规划、合作伙伴关系和机构倡议，与监管机构、企业、学术机构和生态系统伙伴密切合作。", "Hui Yi leads GFI's strategic planning, partnerships, and institutional initiatives. She works closely with regulators, corporates, academic institutions, and ecosystem partners to shape certification pathways, membership strategy, and long-term growth aligned with GFI's mission and global positioning.") },
      { name: "Cheryl Wang, PhD, CFtP, CFA", role: tx("中国区负责人 / 教育主管", "Head of China / Head of Education"), image: "https://globalfintechinstitute.org/assets/WangYu.hFNl9LRm_UR6GC.webp", bio: tx("Cheryl负责GFI教育产品战略，监督认证和高管教育项目的设计、开发与持续改进，同时推动GFI在中国的合作与发展。", "Cheryl leads GFI's education product strategy, overseeing the design, development, and continuous improvement of certifications and executive education programmes. She also drives GFI's growth in China, overseeing local partnerships, regulatory alignment, and the customisation of certifications and programmes.") },
      { name: "Fajri Abdillah", role: tx("工程主管", "Head of Engineering"), image: "https://globalfintechinstitute.org/assets/Fajri.jKDcwBt1_ZFJaBE.webp", bio: tx("Fajri负责GFI数字平台与基础设施的设计、开发和维护，确保会员门户、学习系统和FLEX平台安全、可扩展且可靠。", "Fajri heads GFI's engineering function, responsible for the design, development, and maintenance of GFI's digital platforms and infrastructure. He ensures that member portals, learning systems, and platforms such as FLEX are secure, scalable, and reliable to support GFI's growing global community.") },
    ],
  },
} as const

export const fellowBenefits = [
  [tx("指导和培养人才", "Mentor and Develop Talent"), tx("通过指导和实用的行业视角，支持下一代金融科技专业人士。", "Support the next generation of fintech professionals through mentorship and practical industry perspectives.")],
  [tx("塑造政策和最佳实践", "Shape Policy and Best Practice"), tx("参与知情对话，帮助定义治理、风险、合规和市场实践方面的负责任标准。", "Contribute to informed dialogue that helps define responsible standards in governance, risk, compliance and market practice.")],
  [tx("建立跨部门合作", "Build Cross-Sector Collaboration"), tx("加强利益相关者之间的联系，促进整个生态系统的合作", "Strengthen connections between stakeholders and foster collaboration across the ecosystem")],
] as const

export const fellows = [
  { name: tx("张恩尼博士，CFtP®", "Dr. Ernie Teo, CFtP®"), role: tx("项目主任", "Programme Director"), org: tx("南洋商学院，新加坡南洋理工大学", "Nanyang Business School, Nanyang Technological University"), tags: [tx("金融科技教育", "Fintech Education"), tx("区块链技术", "Blockchain Technology")], url: "https://www.linkedin.com/in/ernieteo/" },
  { name: tx("Gary Loh, CFtP®", "Gary Loh, CFtP®"), role: tx("创始人兼首席执行官", "Founder and CEO"), org: tx("DiMuto", "DiMuto"), tags: [tx("贸易与供应链金融", "Trade & Supply Chain Finance"), tx("区块链与数字资产", "Blockchain & Digital Assets")], url: "https://www.linkedin.com/in/gary-loh-43541a37/" },
  { name: tx("Tat Yeen Yap, CFtP®", "Tat Yeen Yap, CFtP®"), role: tx("供应链解决方案主管", "Head of Supply Chain Solutions"), org: tx("马来亚银行新加坡", "Maybank Singapore"), tags: [tx("银行与金融基础设施", "Banking & Financial Infrastructure"), tx("贸易与供应链金融", "Trade & Supply Chain Finance")], url: "https://www.linkedin.com/in/tat-yeen-yap/" },
  { name: tx("Jag Foo, CFtP®", "Jag Foo, CFtP®"), role: tx("合伙人", "Partner"), org: tx("Safeheron", "Safeheron"), tags: [tx("数字资产安全", "Digital Asset Security"), tx("合规与治理", "Compliance & Governance")], url: "https://www.linkedin.com/in/jag-foo-cftp-cpp-psp-pci-3bb85175/" },
  { name: tx("Aaron Ting", "Aaron Ting"), role: tx("生态系统与媒体关系主管", "Head of Ecosystem and Media Relations"), org: tx("DFINITY Foundation", "DFINITY Foundation"), tags: [tx("Web3生态系统", "Web3 Ecosystem"), tx("人工智能与新兴技术", "AI & Emerging Technology")], url: "https://www.linkedin.com/in/aaronting8/" },
] as const

export const jobs = [
  [tx("人力资源  财务与会计", "Human Resources  Finance & Accounting"), tx("人力资源、行政与财务执行员（全职 / 兼职）", "HR, Administration & Finance Executive (Full-time / Part-time)"), tx("全职 / 兼职", "Full-time / Part-time"), tx("远程 / 混合办公 / 现场办公", "Remote / Hybrid / On-site")],
  [tx("活动与社区", "Events & Community"), tx("活动与社区实习生", "Events & Community Intern"), tx("实习", "Internship"), tx("新加坡罗敏申路80号(80RR)现场办公", "On-site at 80 Robinson Road (80RR), Singapore")],
  [tx("人力资源  财务与会计", "Human Resources  Finance & Accounting"), tx("人力资源、行政与财务实习生", "HR, Administration & Finance Intern"), tx("实习", "Internship"), tx("新加坡罗敏申路80号(80RR)现场办公", "On-site at 80 Robinson Road (80RR), Singapore")],
  [tx("营销与传播", "Marketing & Communications"), tx("营销与内容创作实习生", "Marketing & Content Creation Intern"), tx("实习", "Internship"), tx("新加坡罗敏申路80号(80RR)现场办公", "On-site at 80 Robinson Road (80RR), Singapore")],
  [tx("业务运营  财务", "Business Operations  Finance"), tx("运营与财务执行（人事、薪资、开票、审计）", "Operations & Finance Executive (HR, Payroll, Billing, Audit)"), tx("全职", "Full-time"), tx("远程", "Remote")],
  [tx("业务运营", "Business Operations"), tx("运营负责人 / 运营总监", "Head / Director of Operations"), tx("全职", "Full-time"), tx("新加坡 / 马来西亚", "Singapore / Malaysia")],
  [tx("合作伙伴关系  市场营销", "Partnerships  Marketing"), tx("合作伙伴关系与市场营销助理", "Partnerships & Marketing Associate"), tx("全职", "Full-time"), tx("新加坡 80 Robinson Road（80RR）现场办公", "On-site at 80 Robinson Road (80RR), Singapore")],
  [tx("合作伙伴与业务拓展", "Partnerships & Business Development"), tx("合作伙伴与拓展实习生", "Partnerships & Outreach Intern"), tx("实习", "Internship"), tx("新加坡罗敏申路80号(80RR)现场办公", "On-site at 80 Robinson Road (80RR), Singapore")],
  [tx("运营与管理", "Operations & Management"), tx("项目运营实习生", "Programme Operations Intern"), tx("实习", "Internship"), tx("新加坡罗敏申路80号(80RR)现场办公", "On-site at 80 Robinson Road (80RR), Singapore")],
  [tx("项目运营  课程与项目", "Programme Operations  Courses & Programmes"), tx("项目运营执行（考试与学员支持）", "Programme Operations Executive (Exams & Learner Support)"), tx("全职", "Full-time"), tx("远程", "Remote")],
] as const

export const jobLinks = [
  "hr-admin-finance-executive",
  "events-community-intern",
  "hr-admin-finance-intern",
  "marketing-content-intern",
  "operations-finance-executive",
  "head-of-operations",
  "partnerships-marketing-associate",
  "partnerships-outreach-intern",
  "program-operations-intern",
  "programme-operations-executive",
] as const

export const contacts = [
  [tx("一般查询", "General Enquiries"), tx("关于GFI、会员、活动或使用我们平台的问题。", "Questions about GFI, membership, events or using our platforms."), "info@globalfintechinstitute.org"],
  [tx("认证与项目", "Certifications & Programmes"), tx("关于CFtP®、CFtA、认证路径、资格和注册支持。", "Support for CFtP®, CFtA, certification pathways, eligibility and enrolment."), "cftp@globalfintechinstitute.org"],
  [tx("合作伙伴关系与协作", "Partnerships & Collaboration"), tx("关于企业赞助人计划、学术合作伙伴关系、共同主办的活动和生态系统协作。", "Corporate patron programmes, academic partnerships, co-hosted events and ecosystem collaboration."), "partnerships@globalfintechinstitute.org"],
  [tx("市场营销与传播", "Marketing & Communications"), tx("关于媒体查询、传播、活动推广和品牌相关事宜。", "Media enquiries, communications, event promotion and brand-related matters."), "marketing@globalfintechinstitute.org"],
] as const
