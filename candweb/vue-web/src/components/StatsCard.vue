<script setup lang="ts">
import type { Component } from "vue"
import { RouterLink } from "vue-router"

const props = withDefaults(defineProps<{
  title: string
  value: string | number
  icon: Component
  description?: string
  href?: string
  actionLabel?: string
  variant?: "default" | "primary" | "success" | "warning" | "info"
}>(), {
  variant: "default",
})

const variantStyles = {
  default: { iconBg: "bg-muted", iconColor: "text-muted-foreground", panel: "bg-muted/40", border: "border-border" },
  primary: { iconBg: "bg-primary/10", iconColor: "text-primary", panel: "bg-blue-50/90", border: "border-primary/10" },
  success: { iconBg: "bg-emerald-500/10", iconColor: "text-emerald-600", panel: "bg-emerald-50/90", border: "border-emerald-500/10" },
  warning: { iconBg: "bg-amber-500/10", iconColor: "text-amber-600", panel: "bg-amber-50/90", border: "border-amber-500/10" },
  info: { iconBg: "bg-sky-500/10", iconColor: "text-sky-600", panel: "bg-sky-50/90", border: "border-sky-500/10" },
}
</script>

<template>
  <component
    :is="props.href ? RouterLink : 'div'"
    :to="props.href"
    :class="['group relative block overflow-hidden rounded-[14px] border p-5 transition-all duration-300 hover:-translate-y-0.5 hover:bg-white hover:shadow-[0_12px_26px_rgba(16,30,67,0.08)]', variantStyles[variant].panel, variantStyles[variant].border, props.href && 'cursor-pointer']"
  >
    <div class="absolute -right-8 -top-8 h-28 w-28 rounded-full bg-white/50 opacity-70 transition-transform duration-300 group-hover:scale-110" />
    <div class="relative flex items-start justify-between">
      <div class="min-w-0 space-y-2">
        <p class="text-sm font-semibold" :class="variantStyles[variant].iconColor">{{ title }}</p>
        <p class="text-4xl font-bold tracking-tight text-card-foreground">{{ value }}</p>
        <p v-if="description" class="text-xs text-muted-foreground">{{ description }}</p>
        <p v-if="actionLabel" class="pt-1 text-xs font-medium opacity-0 transition-opacity duration-300 group-hover:opacity-100" :class="variantStyles[variant].iconColor">
          {{ actionLabel }}
        </p>
      </div>
      <div :class="['flex h-14 w-14 shrink-0 items-center justify-center rounded-xl transition-transform duration-300 group-hover:scale-105', variantStyles[variant].iconBg]">
        <component :is="icon" :class="['h-7 w-7', variantStyles[variant].iconColor]" />
      </div>
    </div>
  </component>
</template>
