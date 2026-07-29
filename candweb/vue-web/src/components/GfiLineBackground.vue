<script setup lang="ts">
const lines = Array.from({ length: 25 }, (_, index) => index)
const path = "M2061.59 -593.948C2002.02 -573.676 1884.54 -485.377 1891.15 -294.364C1899.41 -55.5983 2079.61 93.8177 2124.66 319.069C2157.91 485.321 2153.19 901.719 1950.47 1076.66"
</script>

<template>
  <div class="gfi-line-background" aria-hidden="true">
    <svg viewBox="0 0 1920 819" preserveAspectRatio="xMidYMid slice">
      <defs>
        <linearGradient id="gfi-line-background-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stop-color="#f7f9fb" />
          <stop offset="1" stop-color="#ffffff" />
        </linearGradient>
        <linearGradient id="gfi-line-highlight" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stop-color="#ffffff" />
          <stop offset=".45" stop-color="#88a9ff" />
          <stop offset="1" stop-color="#ffffff" />
        </linearGradient>
      </defs>
      <rect width="1920" height="819" fill="url(#gfi-line-background-fill)" />
      <g class="gfi-line-paths">
        <path v-for="line in lines" :key="line" :d="path" :transform="`translate(${-line * 91} 0)`" />
        <path class="gfi-line-highlight" :d="path" transform="translate(-1092 0)" />
      </g>
    </svg>
  </div>
</template>

<style scoped>
.gfi-line-background { position:absolute; inset:0 0 auto; z-index:0; height:819px; overflow:hidden; pointer-events:none; background:#fff; }
.gfi-line-background svg { display:block; width:100%; height:100%; }
.gfi-line-paths path { fill:none; stroke:#fff; stroke-width:1.5; }
.gfi-line-paths .gfi-line-highlight { stroke:url(#gfi-line-highlight); stroke-width:2; stroke-dasharray:180 1500; animation:gfi-line-flow 5s linear infinite; }
@keyframes gfi-line-flow { from { stroke-dashoffset:0; } to { stroke-dashoffset:-1680; } }
@media (prefers-reduced-motion:reduce) { .gfi-line-paths .gfi-line-highlight { animation:none; } }
</style>
