<script setup lang="ts">
import type { Component } from "vue"
import { RouterLink } from "vue-router"

const props = withDefaults(defineProps<{
  title: string
  value: string | number
  icon: Component
  description?: string
  href?: string
  variant?: "default" | "primary" | "success" | "warning" | "info"
}>(), {
  variant: "default",
})

const variantStyles = {
  default: { iconBg: "bg-muted", iconColor: "text-muted-foreground", accent: "bg-muted" },
  primary: { iconBg: "bg-primary/10", iconColor: "text-primary", accent: "bg-primary" },
  success: { iconBg: "bg-emerald-500/10", iconColor: "text-emerald-600", accent: "bg-emerald-500" },
  warning: { iconBg: "bg-amber-500/10", iconColor: "text-amber-600", accent: "bg-amber-500" },
  info: { iconBg: "bg-sky-500/10", iconColor: "text-sky-600", accent: "bg-sky-500" },
}
</script>

<template>
  <component
    :is="props.href ? RouterLink : 'div'"
    :to="props.href"
    :class="['group relative block overflow-hidden rounded-[14px] bg-[#f8fcfc] p-4 transition-all duration-300 hover:-translate-y-0.5 hover:bg-white hover:shadow-[0_10px_22px_rgba(15,74,82,0.08)]', props.href && 'cursor-pointer']"
  >
    <div :class="['absolute bottom-0 left-4 right-4 h-1 rounded-full opacity-70', variantStyles[variant].accent]" />
    <div class="absolute -right-8 -top-8 h-28 w-28 rounded-full bg-primary/5 opacity-0 transition-opacity duration-300 group-hover:opacity-100" />
    <div class="relative flex items-start justify-between">
      <div class="space-y-2">
        <p class="text-sm font-medium text-muted-foreground">{{ title }}</p>
        <p class="text-3xl font-bold tracking-tight text-card-foreground">{{ value }}</p>
        <p v-if="description" class="text-xs text-muted-foreground">{{ description }}</p>
      </div>
      <div :class="['flex h-12 w-12 shrink-0 items-center justify-center rounded-lg transition-transform duration-300 group-hover:scale-110', variantStyles[variant].iconBg]">
        <component :is="icon" :class="['h-6 w-6', variantStyles[variant].iconColor]" />
      </div>
    </div>
  </component>
</template>
