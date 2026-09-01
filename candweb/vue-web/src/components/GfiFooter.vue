<script setup lang="ts">
</script>

<template>
  <div aria-hidden="true"></div>
</template>

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
