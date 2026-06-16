<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { RouterLink, useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, Send } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import { apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"

const route = useRoute()
const router = useRouter()
const { t, lang } = useTranslation()
const unitId = String(route.query.unitId || "")
const pipelineId = String(route.query.pipelineId || "")
const courseId = String(route.query.courseId || "")
const loading = ref(false)
const syncLoading = ref(false)
const selectedCountryCode = ref("")
const selectedProvinceCode = ref("")
const locationApi = ref<any>(null)
const allCountries = ref<any[]>([])
const provinceOptions = ref<any[]>([])
const cityOptions = ref<any[]>([])
const formData = reactive({
  first_name: "",
  middle_name: "",
  last_name: "",
  email: "",
  gender: "",
  birthdate: "",
  country: "",
  province: "",
  city: "",
  address: "",
  postal_code: "",
  home_phone: "",
  work_phone: "",
})
const backLink = computed(() => pipelineId ? `/courses/detail?id=${encodeURIComponent(pipelineId)}` : courseId ? `/courses/learn?courseId=${encodeURIComponent(courseId)}` : "/courses")
const countryOptions = computed(() => {
  const locale = lang.value === "zh" ? "zh-CN" : "en"
  const displayNames = new Intl.DisplayNames([locale], { type: "region" })
  return allCountries.value
    .map((country) => ({ code: country.isoCode, name: displayNames.of(country.isoCode) || country.name }))
    .sort((a, b) => a.name.localeCompare(b.name, locale))
})
const CN_STATE_LABELS: Record<string, string> = {
  AH: "安徽", BJ: "北京", CQ: "重庆", FJ: "福建", GS: "甘肃", GD: "广东", GX: "广西", GZ: "贵州",
  HI: "海南", HE: "河北", HL: "黑龙江", HA: "河南", HK: "香港", HB: "湖北", HN: "湖南", NM: "内蒙古",
  JS: "江苏", JX: "江西", JL: "吉林", LN: "辽宁", MO: "澳门", NX: "宁夏", QH: "青海", SN: "陕西",
  SD: "山东", SH: "上海", SX: "山西", SC: "四川", TJ: "天津", XJ: "新疆", XZ: "西藏", YN: "云南",
  ZJ: "浙江", TW: "台湾",
}
const CN_CITY_LABELS: Record<string, Record<string, string>> = {
  BJ: { Beijing: "北京", Changping: "昌平", Daxing: "大兴", Fangshan: "房山", Liangxiang: "良乡", Mentougou: "门头沟", Shunyi: "顺义", Tongzhou: "通州" },
  SH: { Shanghai: "上海", Baoshan: "宝山", Jiading: "嘉定", Minhang: "闵行", Pudong: "浦东", Songjiang: "松江" },
  GD: { Guangzhou: "广州", Shenzhen: "深圳", Dongguan: "东莞", Foshan: "佛山", Zhuhai: "珠海", Huizhou: "惠州" },
  ZJ: { Hangzhou: "杭州", Ningbo: "宁波", Wenzhou: "温州", Jiaxing: "嘉兴", Shaoxing: "绍兴", Jinhua: "金华" },
  JS: { Nanjing: "南京", Suzhou: "苏州", Wuxi: "无锡", Changzhou: "常州", Nantong: "南通", Xuzhou: "徐州" },
  SC: { Chengdu: "成都", Mianyang: "绵阳", Deyang: "德阳", Leshan: "乐山", Yibin: "宜宾" },
  CQ: { Chongqing: "重庆" },
  TJ: { Tianjin: "天津" },
}
const CN_CITY_OPTIONS_BY_STATE: Record<string, string[]> = {
  AH: ["合肥", "芜湖", "蚌埠", "淮南", "马鞍山", "淮北", "铜陵", "安庆", "黄山", "滁州", "阜阳", "宿州", "六安", "亳州", "池州", "宣城"],
  BJ: ["北京", "东城", "西城", "朝阳", "海淀", "丰台", "石景山", "通州", "昌平", "大兴", "顺义", "房山", "门头沟", "怀柔", "平谷", "密云", "延庆"],
  CQ: ["重庆", "万州", "涪陵", "渝中", "大渡口", "江北", "沙坪坝", "九龙坡", "南岸", "北碚", "渝北", "巴南", "长寿", "江津", "合川", "永川", "南川"],
  FJ: ["福州", "厦门", "莆田", "三明", "泉州", "漳州", "南平", "龙岩", "宁德"],
  GS: ["兰州", "嘉峪关", "金昌", "白银", "天水", "武威", "张掖", "平凉", "酒泉", "庆阳", "定西", "陇南", "临夏", "甘南"],
  GD: ["广州", "深圳", "珠海", "汕头", "佛山", "韶关", "湛江", "肇庆", "江门", "茂名", "惠州", "梅州", "汕尾", "河源", "阳江", "清远", "东莞", "中山", "潮州", "揭阳", "云浮"],
  GX: ["南宁", "柳州", "桂林", "梧州", "北海", "防城港", "钦州", "贵港", "玉林", "百色", "贺州", "河池", "来宾", "崇左"],
  GZ: ["贵阳", "六盘水", "遵义", "安顺", "毕节", "铜仁", "黔西南", "黔东南", "黔南"],
  HI: ["海口", "三亚", "三沙", "儋州", "五指山", "琼海", "文昌", "万宁", "东方"],
  HE: ["石家庄", "唐山", "秦皇岛", "邯郸", "邢台", "保定", "张家口", "承德", "沧州", "廊坊", "衡水"],
  HL: ["哈尔滨", "齐齐哈尔", "鸡西", "鹤岗", "双鸭山", "大庆", "伊春", "佳木斯", "七台河", "牡丹江", "黑河", "绥化", "大兴安岭"],
  HA: ["郑州", "开封", "洛阳", "平顶山", "安阳", "鹤壁", "新乡", "焦作", "濮阳", "许昌", "漯河", "三门峡", "南阳", "商丘", "信阳", "周口", "驻马店", "济源"],
  HB: ["武汉", "黄石", "十堰", "宜昌", "襄阳", "鄂州", "荆门", "孝感", "荆州", "黄冈", "咸宁", "随州", "恩施", "仙桃", "潜江", "天门", "神农架"],
  HN: ["长沙", "株洲", "湘潭", "衡阳", "邵阳", "岳阳", "常德", "张家界", "益阳", "郴州", "永州", "怀化", "娄底", "湘西"],
  NM: ["呼和浩特", "包头", "乌海", "赤峰", "通辽", "鄂尔多斯", "呼伦贝尔", "巴彦淖尔", "乌兰察布", "兴安", "锡林郭勒", "阿拉善"],
  JS: ["南京", "无锡", "徐州", "常州", "苏州", "南通", "连云港", "淮安", "盐城", "扬州", "镇江", "泰州", "宿迁"],
  JX: ["南昌", "景德镇", "萍乡", "九江", "新余", "鹰潭", "赣州", "吉安", "宜春", "抚州", "上饶"],
  JL: ["长春", "吉林", "四平", "辽源", "通化", "白山", "松原", "白城", "延边"],
  LN: ["沈阳", "大连", "鞍山", "抚顺", "本溪", "丹东", "锦州", "营口", "阜新", "辽阳", "盘锦", "铁岭", "朝阳", "葫芦岛"],
  NX: ["银川", "石嘴山", "吴忠", "固原", "中卫"],
  QH: ["西宁", "海东", "海北", "黄南", "海南", "果洛", "玉树", "海西"],
  SN: ["西安", "铜川", "宝鸡", "咸阳", "渭南", "延安", "汉中", "榆林", "安康", "商洛"],
  SD: ["济南", "青岛", "淄博", "枣庄", "东营", "烟台", "潍坊", "济宁", "泰安", "威海", "日照", "临沂", "德州", "聊城", "滨州", "菏泽"],
  SH: ["上海", "黄浦", "徐汇", "长宁", "静安", "普陀", "虹口", "杨浦", "闵行", "宝山", "嘉定", "浦东", "金山", "松江", "青浦", "奉贤", "崇明"],
  SX: ["太原", "大同", "阳泉", "长治", "晋城", "朔州", "晋中", "运城", "忻州", "临汾", "吕梁"],
  SC: ["成都", "自贡", "攀枝花", "泸州", "德阳", "绵阳", "广元", "遂宁", "内江", "乐山", "南充", "眉山", "宜宾", "广安", "达州", "雅安", "巴中", "资阳", "阿坝", "甘孜", "凉山"],
  TJ: ["天津", "和平", "河东", "河西", "南开", "河北", "红桥", "东丽", "西青", "津南", "北辰", "武清", "宝坻", "滨海新区", "宁河", "静海", "蓟州"],
  XJ: ["乌鲁木齐", "克拉玛依", "吐鲁番", "哈密", "昌吉", "博尔塔拉", "巴音郭楞", "阿克苏", "克孜勒苏", "喀什", "和田", "伊犁", "塔城", "阿勒泰", "石河子", "阿拉尔", "图木舒克", "五家渠", "北屯", "铁门关", "双河", "可克达拉", "昆玉"],
  XZ: ["拉萨", "日喀则", "昌都", "林芝", "山南", "那曲", "阿里"],
  YN: ["昆明", "曲靖", "玉溪", "保山", "昭通", "丽江", "普洱", "临沧", "楚雄", "红河", "文山", "西双版纳", "大理", "德宏", "怒江", "迪庆"],
  ZJ: ["杭州", "宁波", "温州", "嘉兴", "湖州", "绍兴", "金华", "衢州", "舟山", "台州", "丽水"],
  TW: ["台北", "新北", "桃园", "台中", "台南", "高雄", "基隆", "新竹", "嘉义"],
  HK: ["香港"],
  MO: ["澳门"],
}

function localizedProvinceName(province: any) {
  return lang.value === "zh" && selectedCountryCode.value === "CN" ? CN_STATE_LABELS[province.isoCode] || province.name : province.name
}

function localizedCityName(city: any) {
  if (typeof city?.localizedName === "string") return city.localizedName
  return lang.value === "zh" && selectedCountryCode.value === "CN" ? CN_CITY_LABELS[selectedProvinceCode.value]?.[city.name] || city.name : city.name
}

function normalizeLocationText(value: unknown) {
  return typeof value === "string" ? value.trim().toLowerCase() : ""
}

async function loadLocationData() {
  if (locationApi.value) return
  locationApi.value = await import("country-state-city")
  allCountries.value = locationApi.value.Country.getAllCountries()
}

function refreshProvinceOptions() {
  provinceOptions.value = selectedCountryCode.value ? locationApi.value?.State.getStatesOfCountry(selectedCountryCode.value) || [] : []
}

function refreshCityOptions() {
  if (!selectedCountryCode.value) {
    cityOptions.value = []
    return
  }
  if (selectedProvinceCode.value) {
    if (lang.value === "zh" && selectedCountryCode.value === "CN" && CN_CITY_OPTIONS_BY_STATE[selectedProvinceCode.value]) {
      cityOptions.value = CN_CITY_OPTIONS_BY_STATE[selectedProvinceCode.value].map((name) => ({ name, localizedName: name }))
      return
    }
    cityOptions.value = locationApi.value?.City.getCitiesOfState(selectedCountryCode.value, selectedProvinceCode.value) || []
    return
  }
  cityOptions.value = provinceOptions.value.length === 0 ? locationApi.value?.City.getCitiesOfCountry(selectedCountryCode.value) || [] : []
}

function syncLocationSelectionFromForm() {
  if (!locationApi.value) return
  const countryText = normalizeLocationText(formData.country)
  const zhRegionNames = new Intl.DisplayNames(["zh-CN"], { type: "region" })
  const matchedCountry = allCountries.value.find((country) =>
    [country.name, country.isoCode, country.phonecode].some((value) => normalizeLocationText(value) === countryText) ||
    normalizeLocationText(zhRegionNames.of(country.isoCode)) === countryText,
  )
  selectedCountryCode.value = matchedCountry?.isoCode || ""
  refreshProvinceOptions()

  const provinceText = normalizeLocationText(formData.province)
  const matchedProvince = selectedCountryCode.value
    ? provinceOptions.value.find((state) => [state.name, state.isoCode].some((value) => normalizeLocationText(value) === provinceText))
    : undefined
  selectedProvinceCode.value = matchedProvince?.isoCode || ""
  refreshCityOptions()
}

function handleCountryChange() {
  const country = countryOptions.value.find((item) => item.code === selectedCountryCode.value)
  formData.country = country?.name || ""
  formData.province = ""
  formData.city = ""
  selectedProvinceCode.value = ""
  refreshProvinceOptions()
  refreshCityOptions()
}

function handleProvinceChange() {
  const province = provinceOptions.value.find((item) => item.isoCode === selectedProvinceCode.value)
  formData.province = province ? localizedProvinceName(province) : ""
  formData.city = ""
  refreshCityOptions()
}

function normalizeInternationalPhone(value: string) {
  const trimmed = value.trim()
  const prefix = trimmed.includes("+") ? "+" : ""
  const digits = trimmed.replace(/\D/g, "").slice(0, 15)
  return `${prefix}${digits}`
}

function handlePhoneInput(field: "home_phone" | "work_phone", event: Event) {
  const target = event.target as HTMLInputElement
  formData[field] = normalizeInternationalPhone(target.value)
}

function normalizeDate(value: unknown) {
  return typeof value === "string" ? value.split("T")[0] : ""
}

function normalizeAddress(value: unknown, fallback: unknown) {
  if (typeof value === "string") return value
  if (Array.isArray(fallback)) return fallback.join(", ")
  if (typeof fallback === "string") return fallback
  return ""
}

function splitRealName(value: unknown) {
  if (typeof value !== "string") return { firstName: "", lastName: "" }
  const parts = value.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return { firstName: "", lastName: "" }
  return {
    firstName: parts[0] || "",
    lastName: parts.length > 1 ? parts.slice(1).join(" ") : "",
  }
}

function applyProfileToForm(profile: any) {
  const realName = splitRealName(profile.real_name)
  formData.email = profile.email || formData.email
  formData.gender = profile.gender || formData.gender
  formData.birthdate = normalizeDate(profile.birthday) || formData.birthdate
  formData.first_name = profile.first_name || realName.firstName || formData.first_name
  formData.last_name = profile.last_name || realName.lastName || formData.last_name
  formData.home_phone = profile.home_phone || profile.phone || formData.home_phone
  formData.work_phone = profile.work_phone || formData.work_phone
  formData.country = profile.country || profile.region || formData.country
  formData.province = profile.province || formData.province
  formData.city = profile.city || profile.location || formData.city
  formData.address = normalizeAddress(profile.address_text, profile.address) || formData.address
  formData.postal_code = profile.postal_code || formData.postal_code
  syncLocationSelectionFromForm()
}

function firstFilled(...values: unknown[]) {
  for (const value of values) {
    if (typeof value === "string" && value.trim()) return value.trim()
  }
  return ""
}

function fillOnlyWhenEmpty(current: unknown, next: unknown) {
  const currentValue = firstFilled(current)
  return currentValue || firstFilled(next)
}

function buildProfilePayload(current: any) {
  const currentAddress = normalizeAddress(current.address_text, current.address)
  return {
    display_name: current.display_name || "",
    email: fillOnlyWhenEmpty(current.email, formData.email),
    first_name: fillOnlyWhenEmpty(current.first_name, formData.first_name),
    last_name: fillOnlyWhenEmpty(current.last_name, formData.last_name),
    home_phone: fillOnlyWhenEmpty(current.home_phone || current.phone, formData.home_phone),
    work_phone: fillOnlyWhenEmpty(current.work_phone, formData.work_phone),
    gender: fillOnlyWhenEmpty(current.gender, formData.gender),
    birthday: fillOnlyWhenEmpty(normalizeDate(current.birthday), formData.birthdate),
    country: fillOnlyWhenEmpty(current.country || current.region, formData.country),
    province: fillOnlyWhenEmpty(current.province, formData.province),
    city: fillOnlyWhenEmpty(current.city || current.location, formData.city),
    address: fillOnlyWhenEmpty(currentAddress, formData.address),
    postal_code: fillOnlyWhenEmpty(current.postal_code, formData.postal_code),
    affiliation: current.affiliation || "",
    title: current.title || "",
    real_name: current.real_name || "",
    bio: current.bio || "",
    education: current.education || "",
  }
}

onMounted(async () => {
  try {
    await loadLocationData()
    const res = await apiClient("/api/user/me")
    if (res) {
      applyProfileToForm(res)
    }
  } catch (err) {
    console.error("Failed to load user profile", err)
  }
})

watch(lang, () => {
  const previousCity = formData.city
  const country = countryOptions.value.find((item) => item.code === selectedCountryCode.value)
  if (country) formData.country = country.name
  const province = provinceOptions.value.find((item) => item.isoCode === selectedProvinceCode.value)
  if (province) formData.province = localizedProvinceName(province)
  refreshCityOptions()
  const city = cityOptions.value.find((item) => [item.name, localizedCityName(item)].includes(formData.city))
  if (city) formData.city = localizedCityName(city)
  else if (lang.value === "zh" && selectedCountryCode.value === "CN" && selectedProvinceCode.value) {
    const mappedCity = Object.entries(CN_CITY_LABELS[selectedProvinceCode.value] || {}).find(([english, chinese]) => english === previousCity || chinese === previousCity)
    if (mappedCity) formData.city = mappedCity[1]
  }
})

async function handleSyncToProfile() {
  syncLoading.value = true
  try {
    const current = await apiClient("/api/user/me")
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify(buildProfilePayload(current || {})),
    })
    toast.success(t.value.examSignup.syncProfileSuccess)
  } finally {
    syncLoading.value = false
  }
}

async function handleSubmit() {
  if (!unitId) {
    toast.error(t.value.common.error)
    return
  }
  const requiredFields = [
    ["first_name", t.value.examSignup.formFirstName],
    ["middle_name", t.value.examSignup.formMiddleName],
    ["last_name", t.value.examSignup.formLastName],
    ["email", t.value.examSignup.formEmail],
    ["gender", t.value.examSignup.formGender],
    ["birthdate", t.value.examSignup.formBirthdate],
    ["country", t.value.examSignup.formCountry],
    ["province", t.value.examSignup.formProvince],
    ["city", t.value.examSignup.formCity],
    ["address", t.value.examSignup.formAddress],
    ["postal_code", t.value.examSignup.formPostalCode],
    ["home_phone", t.value.examSignup.formHomePhone],
  ] as const
  for (const [key, label] of requiredFields) {
    if (!String(formData[key]).trim()) {
      toast.error(t.value.examSignup.validationRequired.replace("{{field}}", label))
      return
    }
  }
  loading.value = true
  try {
    await apiClient(`/api/exams/units/${encodeURIComponent(unitId)}/signup`, { method: "POST", body: JSON.stringify(formData) })
    toast.success(t.value.examSignup.success)
    router.push("/exams")
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AppShell content-class="p-4">
    <RouterLink :to="backLink" class="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
      <ArrowLeft class="h-4 w-4" /> {{ t.examSignup.backToCourse }}
    </RouterLink>
    <div class="mb-8 max-w-2xl">
      <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.examSignup.title }}</h1>
      <p class="mt-2 text-muted-foreground">{{ t.examSignup.subtitle }}</p>
    </div>
    <div class="max-w-2xl rounded-[16px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
      <form class="space-y-6" @submit.prevent="handleSubmit">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formFirstName }} *</span><input v-model="formData.first_name" class="input" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formLastName }} *</span><input v-model="formData.last_name" class="input" required /></label>
        </div>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formMiddleName }} *</span><input v-model="formData.middle_name" class="input" /></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formEmail }} *</span><input v-model="formData.email" class="input" type="email" required /></label>
          <label class="space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formGender }} *</span><input v-model="formData.gender" class="input" placeholder="Male / Female" required /></label>
        </div>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formBirthdate }} *</span><input v-model="formData.birthdate" class="input" type="date" required /></label>
        <div class="grid gap-4 sm:grid-cols-3">
          <label class="space-y-2">
            <span class="text-sm font-medium">{{ t.examSignup.formCountry }} *</span>
            <select v-model="selectedCountryCode" class="input cursor-pointer" required @change="handleCountryChange">
              <option value="" disabled>{{ t.examSignup.formCountry }}</option>
              <option v-for="country in countryOptions" :key="country.code" :value="country.code">{{ country.name }}</option>
            </select>
          </label>
          <label class="space-y-2">
            <span class="text-sm font-medium">{{ t.examSignup.formProvince }} *</span>
            <select v-if="provinceOptions.length > 0" v-model="selectedProvinceCode" class="input cursor-pointer" required @change="handleProvinceChange">
              <option value="" disabled>{{ t.examSignup.formProvince }}</option>
              <option v-for="province in provinceOptions" :key="province.isoCode" :value="province.isoCode">{{ localizedProvinceName(province) }}</option>
            </select>
            <input v-else v-model="formData.province" class="input" required />
          </label>
          <label class="space-y-2">
            <span class="text-sm font-medium">{{ t.examSignup.formCity }} *</span>
            <select v-if="cityOptions.length > 0" v-model="formData.city" class="input cursor-pointer" required>
              <option value="" disabled>{{ t.examSignup.formCity }}</option>
              <option v-for="city in cityOptions" :key="`${city.name}-${city.latitude}-${city.longitude}`" :value="localizedCityName(city)">{{ localizedCityName(city) }}</option>
            </select>
            <input v-else v-model="formData.city" class="input" required />
          </label>
        </div>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formAddress }} *</span><input v-model="formData.address" class="input" required /></label>
        <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formPostalCode }} *</span><input v-model="formData.postal_code" class="input" required /></label>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="space-y-2">
            <span class="text-sm font-medium">{{ t.examSignup.formWorkPhone }}</span>
            <input
              v-model="formData.work_phone"
              class="input"
              type="tel"
              inputmode="tel"
              maxlength="16"
              @input="handlePhoneInput('work_phone', $event)"
            />
          </label>
          <label class="space-y-2">
            <span class="text-sm font-medium">{{ t.examSignup.formHomePhone }} *</span>
            <input
              v-model="formData.home_phone"
              class="input"
              type="tel"
              inputmode="tel"
              maxlength="16"
              required
              @input="handlePhoneInput('home_phone', $event)"
            />
          </label>
        </div>
        <div class="rounded-xl border border-emerald-100 bg-emerald-50/60 p-4 text-sm text-muted-foreground">
          <p>{{ t.examSignup.syncToProfileHint }}</p>
          <button type="button" class="btn btn-outline mt-3" :disabled="syncLoading || loading" @click="handleSyncToProfile">
            <template v-if="syncLoading">{{ t.examSignup.syncingToProfile }}</template>
            <template v-else>{{ t.examSignup.syncToProfile }}</template>
          </button>
        </div>
        <div class="flex justify-end pt-2">
          <button class="btn btn-primary w-full sm:w-auto" :disabled="loading">
            <template v-if="loading">{{ t.examSignup.submitting }}</template>
            <template v-else><Send class="mr-2 h-4 w-4" /> {{ t.examSignup.submit }}</template>
          </button>
        </div>
      </form>
    </div>
  </AppShell>
</template>
