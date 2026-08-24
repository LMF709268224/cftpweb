<script setup lang="ts">
/*
import { Facebook, Instagram, Link2, Linkedin, MessageCircle } from "lucide-vue-next"
import { useTranslation } from "@/lib/language"

const { lang } = useTranslation()
const logo = "/gfi/gfi-logo-blue.svg"

const groups = [
  {
    sections: [
      {
        zh: "认证", en: "Certifications", links: [
          ["认证概览", "Overview", "/gfi/certifications"],
          ["CFtA", "CFtA", "/gfi/programmes/cfta"],
          ["CFtP®", "CFtP®", "/gfi/programmes/cftp"],
          ["认证路径", "Certification Pathway", "/gfi/certifications/pathway"],
        ],
      },
      {
        zh: "会员与治理", en: "Membership & Governance", links: [
          ["个人会员", "Individual Memberships", "/gfi/membership/benefits"],
          ["行业研究员", "Industry Fellows", "/gfi/industry-fellow"],
          ["子委员会", "SubCommittees", "/gfi/subcommittees"],
          ["董事会", "Board of Directors", "/gfi/about/board-of-directors"],
        ],
      },
    ],
  },
  {
    sections: [
      {
        zh: "出版物与活动", en: "Publications & Events", links: [
          ["洞察", "Insights", "/gfi/publications/insights"],
          ["报告", "Reports", "/gfi/publications/reports"],
          ["会议与圆桌", "Conferences & Roundtables", "/gfi/events/conferences"],
          ["网络研讨会洞察", "Webinar Insights", "/gfi/events/webinar-recordings"],
        ],
      },
      {
        zh: "职业与合作伙伴", en: "Careers & Partnerships", links: [
          ["FLEX职业门户", "FLEX Career Portal", "/gfi/career"],
          ["企业赞助人计划", "Corporate Patron Programme", "/gfi/membership/corporate"],
          ["成为合作伙伴", "Become a Partner", "/gfi/partnerships"],
          ["联系我们", "Contact Us", "/gfi/contact"],
        ],
      },
    ],
  },
] as const

const bottomLinks = [
  ["关于我们", "About Us", "/gfi/about"],
  ["我们的项目", "Our Programmes", "/gfi/programmes"],
  ["活动", "Events", "/gfi/events/all-events"],
  ["新闻中心", "Newsroom", "/gfi/publications/news"],
  ["联系我们", "Contact Us", "/gfi/contact"],
] as const
*/
</script>

<template>
  <!-- <footer class="gfi-footer">
    <div class="footer-corner footer-corner-left" aria-hidden="true"></div>
    <div class="footer-corner footer-corner-right" aria-hidden="true"></div>
    <div class="gfi-footer-inner gfi-footer-grid">
      <div class="gfi-footer-about">
        <RouterLink to="/gfi"><img :src="logo" alt="Global Fintech Institute" /></RouterLink>
        <p v-if="lang === 'zh'">全球金融科技学院（GFI）是一个非营利智库，致力于促进监管机构、企业和学术机构之间的合作。GFI总部位于新加坡，通过研究、白皮书和思想领导力讨论来应对金融科技领域的挑战，塑造政策和行业标准。</p>
        <p v-else>The Global Fintech Institute (GFI) is a non-profit think tank fostering collaboration among regulators, corporations and academia. Headquartered in Singapore, GFI addresses fintech challenges through research, white papers and thought leadership to shape policy and industry standards.</p>
        <div class="gfi-socials">
          <a href="https://sg.linkedin.com/company/globalfintechinstitute" target="_blank" rel="noopener noreferrer" aria-label="LinkedIn"><Linkedin /></a>
          <span class="social-static" aria-label="WeChat"><MessageCircle /></span>
          <a href="https://linktr.ee/GlobalFintechInstitute" target="_blank" rel="noopener noreferrer" aria-label="Linktree"><Link2 /></a>
          <a href="https://x.com/GFI_charter" target="_blank" rel="noopener noreferrer" aria-label="X"><span class="x-mark">X</span></a>
          <a href="https://www.facebook.com/people/The-Global-Fintech-Institute-GFI/61578996897834/" target="_blank" rel="noopener noreferrer" aria-label="Facebook"><Facebook /></a>
          <a href="https://www.instagram.com/globalfintechinstitute" target="_blank" rel="noopener noreferrer" aria-label="Instagram"><Instagram /></a>
        </div>
      </div>
      <div v-for="(group, groupIndex) in groups" :key="groupIndex" class="gfi-footer-group">
        <section v-for="section in group.sections" :key="section.en" class="gfi-footer-column">
          <h3>{{ lang === "zh" ? section.zh : section.en }}</h3>
          <RouterLink v-for="link in section.links" :key="link[2]" :to="link[2]">{{ lang === "zh" ? link[0] : link[1] }}</RouterLink>
        </section>
      </div>
      <div class="gfi-footer-office">
        <section class="gfi-footer-column">
          <h3>{{ lang === "zh" ? "法律" : "Legal" }}</h3>
          <RouterLink to="/gfi/terms-conditions">{{ lang === "zh" ? "条款与条件" : "Terms & Conditions" }}</RouterLink>
          <RouterLink to="/gfi/privacy-policy">{{ lang === "zh" ? "隐私政策" : "Privacy Policy" }}</RouterLink>
        </section>
        <section class="office-address">
          <h3>{{ lang === "zh" ? "我们的办公室" : "Our Office" }}</h3>
          <p>80 Robinson Road,<br>#08-01,<br>80RR Fintech Hub SG<br>Singapore</p>
        </section>
      </div>
    </div>
    <div class="gfi-footer-bottom">
      <div class="gfi-footer-inner gfi-legal-row">
        <nav><template v-for="(link, index) in bottomLinks" :key="link[2]"><span v-if="index" aria-hidden="true">•</span><RouterLink :to="link[2]">{{ lang === "zh" ? link[0] : link[1] }}</RouterLink></template></nav>
        <p>{{ lang === "zh" ? "版权所有 2026" : "Copyright 2026" }} <i aria-hidden="true"></i> {{ lang === "zh" ? "全球金融科技学院保留所有权利" : "All Rights Reserved by Global Fintech Institute" }}</p>
      </div>
    </div>
  </footer> -->
</template>

<style scoped>
.gfi-footer { position: relative; padding: 106px 0 0; overflow: hidden; background: #101f47; color: rgba(255,255,255,.9); font-family: Inter, "PingFang SC", "Microsoft YaHei", Arial, sans-serif; }
.gfi-footer * { box-sizing: border-box; }
.gfi-footer a { color: inherit; text-decoration: none; }
.gfi-footer-inner { width: min(1288px, calc(100% - 64px)); margin: 0 auto; }
.gfi-footer-grid { position: relative; display: grid; grid-template-columns: 1.25fr 1fr 1.08fr .95fr; min-height: 456px; }
.gfi-footer-about { padding-right: 58px; }
.gfi-footer-about img { width: 128px; margin-bottom: 25px; filter: brightness(0) invert(1); }
.gfi-footer-about p { max-width: 310px; margin: 0; font-size: 14px; line-height: 1.65; }
.gfi-footer-group, .gfi-footer-office { padding: 0 24px; border-left: 1px solid rgba(255,255,255,.13); }
.gfi-footer-group { display: flex; flex-direction: column; gap: 24px; }
.gfi-footer-column { display: flex; flex-direction: column; gap: 13px; }
.gfi-footer-column h3, .office-address h3 { margin: 0 0 14px; color: #fff; font-size: 20px; font-weight: 600; }
.gfi-footer-column a { font-size: 14px; line-height: 1.35; }
.gfi-footer-column a:hover, .gfi-legal-row a:hover { color: #fff; }
.gfi-socials { display: flex; flex-wrap: wrap; gap: 12px; max-width: 260px; margin-top: 34px; }
.gfi-socials a, .social-static { display: flex; width: 48px; height: 48px; align-items: center; justify-content: center; background: rgba(255,255,255,.025); color: #fff; }
.gfi-socials svg { width: 22px; height: 22px; }
.x-mark { font-size: 23px; font-weight: 300; }
.office-address { margin-top: 36px; padding-top: 20px; border-top: 1px solid rgba(255,255,255,.14); }
.office-address p { margin: 0; font-size: 14px; line-height: 1.45; }
.gfi-footer-bottom { background: rgba(255,255,255,.055); }
.gfi-legal-row { display: flex; min-height: 86px; align-items: center; justify-content: space-between; gap: 30px; }
.gfi-legal-row nav { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; font-size: 14px; }
.gfi-legal-row nav span { color: #3670ff; }
.gfi-legal-row p { display: flex; align-items: center; gap: 16px; margin: 0; font-size: 13px; }
.gfi-legal-row i { display: block; width: 1px; height: 18px; background: rgba(255,255,255,.2); }
.footer-corner { position: absolute; top: 4px; width: 176px; height: 155px; opacity: .35; background-image: linear-gradient(rgba(52,100,213,.7) 1px, transparent 1px), linear-gradient(90deg, rgba(52,100,213,.7) 1px, transparent 1px); background-size: 72px 72px; }
.footer-corner-left { left: 0; }
.footer-corner-right { right: 0; transform: scaleX(-1); }

@media (max-width: 1000px) {
  .gfi-footer-grid { grid-template-columns: 1.3fr 1fr 1fr; }
  .gfi-footer-office { grid-column: 2 / 4; margin-top: 40px; padding-top: 36px; }
}
@media (max-width: 700px) {
  .gfi-footer { padding-top: 64px; }
  .gfi-footer-inner { width: calc(100% - 32px); }
  .gfi-footer-grid { grid-template-columns: 1fr; gap: 34px; }
  .gfi-footer-about { padding-right: 0; }
  .gfi-footer-group, .gfi-footer-office { padding: 32px 0 0; border-top: 1px solid rgba(255,255,255,.13); border-left: 0; }
  .gfi-footer-office { grid-column: auto; margin-top: 0; }
  .gfi-legal-row { min-height: 0; align-items: flex-start; flex-direction: column; margin-top: 44px; padding: 25px 0; }
  .gfi-legal-row p { flex-wrap: wrap; }
}
</style>
