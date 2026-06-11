"use client"

import React from "react"

import { useState } from "react"
import { Sidebar } from "@/components/sidebar"
import { cn } from "@/lib/utils"
import { Check, Crown, Zap, Star, Shield, Download, Video, Percent, HelpCircle } from "lucide-react"

import { useTranslation } from "@/lib/useLanguage"



export default function MembershipPage() {
  const { t, lang } = useTranslation()
  const [activeTab, setActiveTab] = useState("benefits")

  const tabs = [
    { id: "intro", label: t.membership.tabs.intro },
    { id: "benefits", label: t.membership.tabs.benefits },
    { id: "levels", label: t.membership.tabs.levels },
    { id: "settings", label: t.membership.tabs.settings },
    { id: "orders", label: t.membership.tabs.orders },
  ]

  const benefits = [
    {
      icon: Zap,
      title: t.membership.benefitsList.b1Title,
      description: t.membership.benefitsList.b1Desc,
    },
    {
      icon: Video,
      title: t.membership.benefitsList.b2Title,
      description: t.membership.benefitsList.b2Desc,
    },
    {
      icon: Download,
      title: t.membership.benefitsList.b3Title,
      description: t.membership.benefitsList.b3Desc,
    },
    {
      icon: Shield,
      title: t.membership.benefitsList.b4Title,
      description: t.membership.benefitsList.b4Desc,
    },
    {
      icon: Percent,
      title: t.membership.benefitsList.b5Title,
      description: t.membership.benefitsList.b5Desc,
    },
    {
      icon: HelpCircle,
      title: t.membership.benefitsList.b6Title,
      description: t.membership.benefitsList.b6Desc,
    },
  ]

  const membershipLevels = [
    {
      id: "basic",
      name: t.membership.levelsTitle.basic,
      englishName: t.membership.levelsEnglishName.basic,
      price: t.membership.priceFree,
      features: t.membership.basicBenefits,
      current: false,
    },
    {
      id: "certified",
      name: t.membership.levelsTitle.certified,
      englishName: t.membership.levelsEnglishName.certified,
      price: t.membership.priceYearly1999,
      features: t.membership.certifiedBenefits,
      highlight: true,
    },
    {
      id: "premium",
      name: t.membership.levelsTitle.premium,
      englishName: t.membership.levelsEnglishName.premium,
      price: t.membership.priceYearly4999,
      features: t.membership.premiumBenefits,
      current: false,
    },
  ]

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      
      <main className="pl-64 transition-all duration-300">
        <div className="px-8 py-8">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold tracking-tight text-foreground">{t.membership.title}</h1>
            <p className="mt-1 text-muted-foreground">{t.membership.subtitle}</p>
          </div>

          <div className="mb-8 max-w-2xl rounded-2xl border border-border bg-card p-6 shadow-sm">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                <Crown className="h-5 w-5 text-muted-foreground" />
              </div>
              <div>
                <h2 className="font-semibold text-card-foreground">{t.membership.currentMember}</h2>
                <p className="text-sm text-muted-foreground">{t.membership.devNotice}</p>
              </div>
            </div>
          </div>

          {/* Tabs */}
          <div className="mb-8 flex gap-1 rounded-xl bg-muted p-1 w-fit overflow-x-auto">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={cn(
                  "px-4 py-2 text-sm font-medium rounded-lg transition-all duration-200 whitespace-nowrap",
                  activeTab === tab.id
                    ? "bg-card text-card-foreground shadow-sm"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          {activeTab === "benefits" && (
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
              <h2 className="text-lg font-semibold text-card-foreground mb-6">{t.membership.currentBenefits}</h2>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {benefits.map((benefit, index) => (
                  <div
                    key={index}
                    className="group flex gap-4 rounded-xl border border-border p-4 transition-all hover:border-primary/20 hover:shadow-sm"
                  >
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary transition-transform group-hover:scale-105">
                      <benefit.icon className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="font-medium text-card-foreground mb-1">{benefit.title}</h3>
                      <p className="text-sm text-muted-foreground">{benefit.description}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {activeTab === "levels" && (
            <div className="grid gap-6 md:grid-cols-3">
              {membershipLevels.map((level) => (
                <div
                  key={level.id}
                  className={cn(
                    "relative rounded-2xl border p-6 transition-all",
                    level.highlight
                      ? "border-primary bg-primary/5 shadow-lg"
                      : "border-border bg-card"
                  )}
                >
                  <div className="mb-4 text-center">
                    <div className={cn(
                      "mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-full",
                      level.id === "basic" && "bg-slate-100 text-slate-600",
                      level.id === "certified" && "bg-primary/10 text-primary",
                      level.id === "premium" && "bg-amber-100 text-amber-600"
                    )}>
                      {level.id === "basic" && <Star className="h-7 w-7" />}
                      {level.id === "certified" && <Crown className="h-7 w-7" />}
                      {level.id === "premium" && <Crown className="h-7 w-7" />}
                    </div>
                    <h3 className="text-lg font-semibold text-card-foreground">{level.name}</h3>
                    <p className="text-sm text-muted-foreground">{level.englishName}</p>
                  </div>
                  
                  <div className="mb-6 text-center">
                    <span className="text-2xl font-bold text-card-foreground">{level.price}</span>
                  </div>
                  
                  <ul className="space-y-3">
                    {level.features.map((feature, idx) => (
                      <li key={idx} className="flex items-center gap-2 text-sm">
                        <Check className={cn(
                          "h-4 w-4 shrink-0",
                          level.highlight ? "text-primary" : "text-emerald-500"
                        )} />
                        <span className="text-card-foreground">{feature}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}

          {activeTab === "intro" && (
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
              <h2 className="text-lg font-semibold text-card-foreground mb-4">{t.membership.introTitle}</h2>
              <p className="text-muted-foreground leading-relaxed">
                {t.membership.introDesc}
              </p>
            </div>
          )}

          {(activeTab === "settings" || activeTab === "orders") && (
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <div className="mb-4 h-16 w-16 rounded-full bg-muted flex items-center justify-center">
                  <Crown className="h-8 w-8 text-muted-foreground" />
                </div>
                <h3 className="text-lg font-semibold text-foreground mb-2">
                  {activeTab === "settings" ? t.membership.tabs.settings : t.membership.tabs.orders}
                </h3>
                <p className="text-muted-foreground">{t.membership.devNotice}</p>
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  )
}
