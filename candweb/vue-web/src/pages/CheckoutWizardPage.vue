<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { toast } from "vue-sonner"
import { ArrowLeft, ArrowRight, ClipboardList, Loader2, Send, Check, CheckCircle2, CircleAlert, Clock, UploadCloud } from "lucide-vue-next"
import AppShell from "@/components/AppShell.vue"
import LoadingState from "@/components/LoadingState.vue"
import PaymentSessionPanel from "@/components/PaymentSessionPanel.vue"
import { ApiClientError, apiClient } from "@/lib/apiClient"
import { useTranslation } from "@/lib/language"
import { useUser } from "@/lib/user"
import { getCachedCountries, getCountryCityOptions, getCountryOptions, getProvinceOptions, getStateCityOptions, loadLocationData, type CountryOption } from "@/lib/locationOptions"
import { GENDER_OPTIONS, PROFILE_TEXT_LIMITS, isValidEmail, isValidInternationalPhone, isValidPostalCode, normalizeGender, normalizeInternationalPhone, normalizePostalCode, trimToMax } from "@/lib/profileFormValidation"
import { CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, statusEnumNameForStatus } from "@/lib/status-labels"
import { getFileConstraintInfo } from "@/lib/fileConstraints"
import { sha256Hex, uploadWithTimeout } from "@/lib/upload"

const route = useRoute()
const router = useRouter()
const { t, lang } = useTranslation()
const { currentUser, fetchUser } = useUser()
const bundleId = String(route.params.bundleId || route.query.bundleId || "")
const TEMPORARY_IMPLICIT_UNLOCK_BUNDLE_GPATH = "/gcc/pipeline/core/cftp"
const currentStep = ref(1)
const bundleData = ref<any>(null)
const pricingDetail = ref<any>(null)
const paymentMode = ref("FULL_PIPELINE")
const paymentPreview = ref<any>(null)
const exemptionStages = ref<any[]>([])
const selectedExemptionUnitIds = ref<Record<string, boolean>>({})
const activeOrderId = ref("")
const activeOrderAction = ref<"purchase" | "unlock" | "credential_application">("purchase")
const activeCredentialQualIds = ref<string[]>([])
const activeCredentialUnitId = ref("")
const credentialApplicationLoadingUnitId = ref("")
const qualificationApplications = ref<Record<string, any>>({})
const qualificationDefinitions = ref<Record<string, any>>({})
const expandedQualificationUnitIds = ref<Record<string, boolean>>({})
const qualificationUploadedFiles = ref<Record<string, Record<string, { name: string; url: string; ext: string; hash: string; size: number }>>>({})
const qualificationUploadingKey = ref("")
const qualificationSubmittingUnitId = ref("")
const levelPlaceholder = "{" + "{level}}"
const exemptionDeclarationChecked = ref(false)
const isExemptionSelected = computed(() => Object.values(selectedExemptionUnitIds.value).some(Boolean))

const dynamicPaymentPreview = computed(() => {
  if (!pricingDetail.value) {
    return paymentPreview.value
  }

  try {
    const detail = typeof pricingDetail.value === "string" ? JSON.parse(pricingDetail.value) : pricingDetail.value

    let total = 0
    let currency = "USD"
    let subtotal = 0

    // 1. Unlocks
    if (detail.unlocks) {
      for (const priceObj of Object.values(detail.unlocks)) {
        const p = priceObj as { amount: number, currency: string }
        total += p.amount
        subtotal += p.amount
        if (p.currency) currency = p.currency
      }
    }

    // 2. Units (or Exemption qual reviews)
    if (Array.isArray(detail.units)) {
      for (const u of detail.units) {
        if (selectedExemptionUnitIds.value[u.unit_id]) {
          // Add qual_review prices
          const unitData = exemptionStages.value.flatMap((s: any) => s.units || []).find((x: any) => x.unit_id === u.unit_id)
          if (unitData && Array.isArray(unitData.exemption_quals)) {
             for (const q of unitData.exemption_quals) {
                const qualId = String(q.qual_id || "").trim()
                const qualReview = Array.isArray(detail.qual_reviews) ? detail.qual_reviews.find((qr: any) => qr.qual_id === qualId) : null
                if (qualReview && qualReview.price) {
                  total += qualReview.price.amount
                  subtotal += qualReview.price.amount
                }
             }
          }
        } else {
          // Add unit access price
          if (u.access) {
             total += u.access.amount
             subtotal += u.access.amount
             if (u.access.currency) currency = u.access.currency
          }
        }
      }
    }

    // 3. Memberships
    if (Array.isArray(detail.memberships)) {
       for (const m of detail.memberships) {
          if (m.price) {
             total += m.price.amount
             subtotal += m.price.amount
             if (m.price.currency) currency = m.price.currency
          }
       }
    }

    return {
      total,
      subtotal,
      currency,
      pay_amount_label: "",
      amount_label: "",
      discount_total: 0 // Assume no complex discounts for dynamic preview on this page if not provided
    }

  } catch (err) {
    console.error("Failed to calculate dynamic pricing", err)
    return paymentPreview.value
  }
})

const unitPriceDisplay = computed<Record<string, { accessAmount?: number, exemptionAmount?: number, currency: string }>>(() => {
  if (!pricingDetail.value) return {}

  try {
    const detail = typeof pricingDetail.value === "string" ? JSON.parse(pricingDetail.value) : pricingDetail.value
    const display: Record<string, { accessAmount?: number, exemptionAmount?: number, currency: string }> = {}

    for (const stage of exemptionStages.value) {
      for (const unit of stage.units || []) {
        const pricingUnit = Array.isArray(detail.units)
          ? detail.units.find((item: any) => item.unit_id === unit.unit_id)
          : null
        let exemptionAmount = 0
        let hasExemptionAmount = false
        let currency = pricingUnit?.access?.currency || "USD"

        for (const qualification of unit.exemption_quals || []) {
          const qualificationId = String(qualification.qual_id || "").trim()
          const review = Array.isArray(detail.qual_reviews)
            ? detail.qual_reviews.find((item: any) => item.qual_id === qualificationId)
            : null
          if (typeof review?.price?.amount === "number") {
            exemptionAmount += review.price.amount
            hasExemptionAmount = true
            currency = review.price.currency || currency
          }
        }

        display[unit.unit_id] = {
          accessAmount: typeof pricingUnit?.access?.amount === "number" ? pricingUnit.access.amount : undefined,
          exemptionAmount: hasExemptionAmount ? exemptionAmount : undefined,
          currency,
        }
      }
    }

    return display
  } catch {
    return {}
  }
})

const isMultiStage = computed(() => {
  return (bundleData.value?.stages?.length || 0) > 1
})
const hasExpandedQualificationEditors = computed(() =>
  Object.values(expandedQualificationUnitIds.value).some(Boolean)
)

const isMembershipBundle = computed(() => {
  if (!bundleData.value) return false
  const itemTypes = bundleData.value.bundle_item_types || bundleData.value.item_types || []
  return itemTypes.some((type: string) => String(type).includes("membership"))
})

const loading = ref(false)
const initialLoading = ref(true)
const pipelineId = computed(() =>
  String(bundleData.value?.pipeline_id || bundleData.value?.pipeline_cc_ulid || "").trim()
)
const paymentBizType = computed(() => {
  if (activeOrderAction.value === "unlock") return "PIPELINE_UNLOCK"
  if (activeOrderAction.value === "credential_application") return "CREDENTIAL_APPLICATION"
  return "BUNDLE_PURCHASE"
})
const paymentReturnPath = computed(() => {
  if (activeOrderAction.value === "unlock") return "/my-certifications"
  if (activeOrderAction.value === "credential_application") return route.path
  return `/checkout/success/${activeOrderId.value}`
})
const paymentReturnParams = computed(() => {
  if (activeOrderAction.value === "credential_application") {
    return {
      qual_ulids: activeCredentialQualIds.value.join(","),
      qualification_unit_id: activeCredentialUnitId.value,
    }
  }
  return {
    bundle_id: bundleId,
    pipeline_id: pipelineId.value,
  }
})
const selectedCountryCode = ref("")
const selectedProvinceCode = ref("")
const countryOptions = ref<CountryOption[]>([])
const provinceOptions = ref<any[]>([])
const cityOptions = ref<any[]>([])
const orgPhonePrefixes = ref<{ code: string, dialCode: string, name: string }[]>([])
const genderOptions = GENDER_OPTIONS
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
  phone_country_code: "",
  phone: "",
  agreement: false,
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

function normalizeProvinceText(value: unknown) {
  return normalizeLocationText(value)
    .replace(/\s+(province|state|autonomous region|special administrative region)$/i, "")
    .replace(/(壮族自治区|回族自治区|维吾尔自治区|特别行政区|自治区|省|市)$/u, "")
}

function provinceMatchValues(province: any) {
  const values = [province.name, province.isoCode, localizedProvinceName(province)]
  if (selectedCountryCode.value === "CN") {
    values.push(CN_STATE_LABELS[province.isoCode] || "")
  }
  return values
}

function ensureCurrentCityOption() {
  const cityText = normalizeLocationText(formData.city)
  if (!cityText) return
  const exists = cityOptions.value.some((city) =>
    [city.name, localizedCityName(city)].some((value) => normalizeLocationText(value) === cityText),
  )
  if (!exists) {
    cityOptions.value = [{ name: formData.city, localizedName: formData.city }, ...cityOptions.value]
  }
}

function refreshCountryOptions() {
  countryOptions.value = getCountryOptions(lang.value === "zh" ? "zh-CN" : "en")
}

function refreshProvinceOptions() {
  provinceOptions.value = selectedCountryCode.value ? getProvinceOptions(selectedCountryCode.value) : []
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
    cityOptions.value = getStateCityOptions(selectedCountryCode.value, selectedProvinceCode.value)
    return
  }
  cityOptions.value = provinceOptions.value.length === 0 ? getCountryCityOptions(selectedCountryCode.value) : []
}

function syncLocationSelectionFromForm() {
  const allCountries = getCachedCountries()
  if (allCountries.length === 0) return
  const countryText = normalizeLocationText(formData.country)
  const zhRegionNames = new Intl.DisplayNames(["zh-CN"], { type: "region" })
  const matchedCountry = allCountries.find((country: any) =>
    [country.name, country.isoCode, country.phonecode].some((value) => normalizeLocationText(value) === countryText) ||
    normalizeLocationText(zhRegionNames.of(country.isoCode)) === countryText,
  )
  selectedCountryCode.value = matchedCountry?.isoCode || ""
  refreshProvinceOptions()

  const provinceText = normalizeLocationText(formData.province)
  const matchedProvince = selectedCountryCode.value
    ? provinceOptions.value.find((state) => provinceMatchValues(state).some((value) => normalizeProvinceText(value) === normalizeProvinceText(provinceText)))
    : undefined
  selectedProvinceCode.value = matchedProvince?.isoCode || ""
  refreshCityOptions()
  ensureCurrentCityOption()
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

function sanitizeSignupForm() {
  formData.first_name = trimToMax(formData.first_name, PROFILE_TEXT_LIMITS.name)
  formData.middle_name = trimToMax(formData.middle_name, PROFILE_TEXT_LIMITS.name)
  formData.last_name = trimToMax(formData.last_name, PROFILE_TEXT_LIMITS.name)
  formData.email = trimToMax(formData.email, PROFILE_TEXT_LIMITS.short)
  formData.gender = normalizeGender(formData.gender)
  formData.country = trimToMax(formData.country, PROFILE_TEXT_LIMITS.short)
  formData.province = trimToMax(formData.province, PROFILE_TEXT_LIMITS.short)
  formData.city = trimToMax(formData.city, PROFILE_TEXT_LIMITS.short)
  formData.address = trimToMax(formData.address, PROFILE_TEXT_LIMITS.address)
  formData.postal_code = normalizePostalCode(formData.postal_code)
  formData.phone = normalizeInternationalPhone(formData.phone)
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
  formData.gender = normalizeGender(profile.gender) || formData.gender
  formData.birthdate = normalizeDate(profile.birthday) || formData.birthdate
  formData.first_name = profile.first_name || realName.firstName || formData.first_name
  formData.middle_name = profile.middle_name || formData.middle_name
  formData.last_name = profile.last_name || realName.lastName || formData.last_name
  formData.phone_country_code = profile.phone_country_code || formData.phone_country_code
  formData.phone = profile.phone || formData.phone
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

function takeFormValue(current: unknown, formValue: unknown) {
  const fv = firstFilled(formValue)
  return fv || firstFilled(current)
}

function buildProfilePayload(current: any) {
  const currentAddress = normalizeAddress(current.address_text, current.address)
  return {
    display_name: firstFilled(current.display_name),
    email: takeFormValue(current.email, formData.email),
    first_name: takeFormValue(current.first_name, formData.first_name),
    last_name: takeFormValue(current.last_name, formData.last_name),
    home_phone: current.home_phone || current.phone || "",
    phone_country_code: takeFormValue(current.phone_country_code, formData.phone_country_code),
    phone: takeFormValue(current.phone, formData.phone),
    gender: takeFormValue(current.gender, formData.gender),
    birthday: takeFormValue(normalizeDate(current.birthday), formData.birthdate),
    country: takeFormValue(current.country || current.region, formData.country),
    province: takeFormValue(current.province, formData.province),
    city: takeFormValue(current.city || current.location, formData.city),
    address: takeFormValue(currentAddress, formData.address),
    postal_code: takeFormValue(current.postal_code, formData.postal_code),
    affiliation: current.affiliation || "",
    title: current.title || "",
    real_name: current.real_name || "",
    bio: current.bio || "",
    education: current.education || "",
  }
}

async function loadProfile() {
  try {
    const res = currentUser.value || await fetchUser()
    if (res) {
      applyProfileToForm(res)
    }
  } catch (err) {
    console.error("Failed to load user profile", err)
  }
}

async function fetchOrgConfig() {
  try {
    const configRes = await apiClient("/api/public/config/organization")
    if (configRes && configRes.country_codes) {
      const allCountries = getCachedCountries()
      orgPhonePrefixes.value = configRes.country_codes.map((code: string) => {
        const country = allCountries.find((c) => c.isoCode === code)
        return {
          code,
          dialCode: country ? `+${country.phonecode}` : code,
          name: country ? country.name : code,
        }
      })
      if (!formData.phone_country_code && orgPhonePrefixes.value.length > 0) {
        formData.phone_country_code = orgPhonePrefixes.value[0].code
      }
    }
  } catch (err) {
    console.error("Failed to load organization config", err)
  }
}

onMounted(() => {
  void (async () => {
    await fetchBundleInfo()
    await resumeQualificationUploadAfterPayment()
  })()
  void loadProfile()
  void loadLocationData()
    .then(() => {
      refreshCountryOptions()
      syncLocationSelectionFromForm()
      void fetchOrgConfig()
    })
    .catch((err: any) => console.error("Failed to load location data", err))
})

watch(lang, () => {
  const previousCity = formData.city
  refreshCountryOptions()
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
  ensureCurrentCityOption()
})

async function syncSignupToProfile() {
  try {
    const current = await fetchUser(true)
    if (!current) return
    await apiClient("/api/user/profile", {
      method: "PUT",
      body: JSON.stringify(buildProfilePayload(current || {})),
      suppressErrorToast: true,
    })
    await fetchUser(true)
  } catch (err) {
    console.warn("Failed to sync signup form to profile", err)
  }
}

function applyBundleInfo(response: any) {
  bundleData.value = response
  const purchaseState = response?.purchase_state || response
  paymentPreview.value = purchaseState?.payment_preview || null

  const stages = purchaseState?.exemption_options?.stages || []
  exemptionStages.value = stages.filter((stage: any) => (stage.units?.length || 0) > 0)

  if (exemptionStages.value.length === 0 && currentStep.value === 1) {
    currentStep.value = 2
  }
}

async function fetchBundlePayload() {
  return apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}`, {
    suppressErrorToast: true,
  })
}

function bundlePipelineId(response: any) {
  return String(response?.pipeline_id || response?.pipeline_cc_ulid || "").trim()
}

function shouldImplicitlyUnlockCftp(response: any) {
  return String(response?.bundle_gpath || "").trim() === TEMPORARY_IMPLICIT_UNLOCK_BUNDLE_GPATH
    && Boolean(getEligibility(response)?.can_unlock)
    && Boolean(bundlePipelineId(response))
}

async function completeTemporaryCftpUnlock(response: any) {
  if (!shouldImplicitlyUnlockCftp(response)) return response

  // TEMP: Remove after gmall makes qualification-only CFtP bundles directly purchasable.
  const unlockResponse = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/unlock`, {
    method: "POST",
    suppressErrorToast: true,
    body: JSON.stringify({
      pipeline_cc_ulid: bundlePipelineId(response),
    }),
  })
  const orderStatus = unlockResponse?.order_status || unlockResponse?.status
  if (!isCompletedStatus(orderStatus)) {
    throw new Error(t.value.checkoutWizard.implicitUnlockFailed)
  }

  const refreshedBundle = await fetchBundlePayload()
  if (!getEligibility(refreshedBundle)?.can_purchase) {
    throw new Error(t.value.checkoutWizard.implicitUnlockFailed)
  }
  return refreshedBundle
}

async function loadBundleInfo() {
  const response = await fetchBundlePayload()
  applyBundleInfo(response)
  await refreshQualificationApplications()
  return response
}

async function loadPurchaseReadyBundleInfo() {
  const response = await fetchBundlePayload()
  const purchaseReadyBundle = await completeTemporaryCftpUnlock(response)
  applyBundleInfo(purchaseReadyBundle)

  try {
    const pricingRes = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/pricing-detail`, { suppressErrorToast: true })
    if (pricingRes && pricingRes.pricing_detail_json) {
      pricingDetail.value = pricingRes.pricing_detail_json
    }
  } catch (e) {
    console.error("Failed to load pricing detail", e)
  }

  await refreshQualificationApplications()
  return purchaseReadyBundle
}

async function fetchBundleInfo() {
  if (!bundleId) {
    initialLoading.value = false
    return
  }
  loading.value = true
  try {
    await loadPurchaseReadyBundleInfo()
  } catch (err) {
    console.error(err)
    toast.error(err instanceof Error && err.message
      ? err.message
      : t.value.common.error)
  } finally {
    loading.value = false
    initialLoading.value = false
  }
}

function buildSelectedExemptionsJson() {
  const stages = exemptionStages.value
    .map((stage) => {
      const exemptedUnitIds = (stage.units || [])
        .filter((unit: any) => unit.qualified && unit.unit_id && selectedExemptionUnitIds.value[unit.unit_id])
        .map((unit: any) => unit.unit_id)
      return {
        index: stage.index,
        stage_cc_ulid: stage.stage_id,
        exempted_unit_cc_ulids: exemptedUnitIds,
      }
    })
    .filter((stage) => stage.exempted_unit_cc_ulids.length > 0)

  const pipelineId = bundleData.value?.pipeline_id || bundleData.value?.pipeline_cc_ulid || ""
  return JSON.stringify({
    [pipelineId]: {
      stages
    }
  })
}



function normalizedCredentialApplicationStatus(status: unknown) {
  const enumName = statusEnumNameForStatus(CANDIDATE_APPLICATION_STATUS_ENUM_NAMES, status as string)
  return String(enumName || status || "").trim().toUpperCase()
}

function isApplicationPendingStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_PENDING"
}

function isApplicationApprovedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_APPROVED"
}

function isApplicationRejectedStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_REJECTED"
}

function isApplicationResubmitStatus(status: unknown) {
  return normalizedCredentialApplicationStatus(status) === "APPLICATION_STATUS_RESUBMIT"
}

function qualificationIdsForUnit(unit: any) {
  return (unit?.exemption_quals || [])
    .map((qual: any) => String(qual?.qual_id || "").trim())
    .filter(Boolean)
}

function qualificationOrderQualIds(primaryQualId: string) {
  const allQualIds = Array.from(new Set(
    exemptionStages.value
      .flatMap((stage: any) => stage.units || [])
      .filter((unit: any) => !unit?.qualified)
      .flatMap((unit: any) => qualificationIdsForUnit(unit)),
  ))
  return [
    primaryQualId,
    ...allQualIds.filter((qualId) => qualId !== primaryQualId),
  ].filter(Boolean)
}

function qualificationApplicationForUnit(unit: any) {
  const applications = qualificationIdsForUnit(unit)
    .map((qualId: string) => qualificationApplications.value[qualId])
    .filter(Boolean)
  return applications.find((application: any) => isApplicationPendingStatus(application?.status))
    || applications.find((application: any) => isApplicationResubmitStatus(application?.status))
    || applications.find((application: any) => isApplicationRejectedStatus(application?.status))
    || applications.find((application: any) => isApplicationApprovedStatus(application?.status))
    || applications[0]
    || null
}

async function latestCredentialApplication(qualId: string) {
  const response = await apiClient(`/api/credentials/applications?cred_def_ulid=${encodeURIComponent(qualId)}`, {
    suppressErrorToast: true,
  })
  return (response?.applications || [])[0] || null
}

async function refreshQualificationApplications() {
  const qualIds = Array.from(new Set(
    exemptionStages.value
      .flatMap((stage: any) => stage.units || [])
      .flatMap((unit: any) => qualificationIdsForUnit(unit)),
  ))
  const next: Record<string, any> = {}
  await Promise.all(qualIds.map(async (qualId) => {
    try {
      const application = await latestCredentialApplication(qualId)
      if (application) next[qualId] = application
    } catch (error) {
      console.warn(`Failed to load credential application ${qualId}`, error)
    }
  }))
  qualificationApplications.value = next
}

let pollingTimer: ReturnType<typeof setInterval> | null = null

function checkPolling() {
  const needsPolling = exemptionStages.value.some((stage: any) =>
    (stage.units || []).some((unit: any) => exemptionCredentialState(unit) === "pending")
  )

  if (needsPolling && !pollingTimer && currentStep.value === 1) {
    pollingTimer = setInterval(async () => {
      if (currentStep.value !== 1) {
        stopPolling()
        return
      }
      try {
        await loadPurchaseReadyBundleInfo()
      } catch (e) {
        // ignore polling errors
      }
    }, 5000)
  } else if (!needsPolling && pollingTimer) {
    stopPolling()
  }
}

function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer)
    pollingTimer = null
  }
}

onUnmounted(() => {
  stopPolling()
})

watch([qualificationApplications, currentStep], checkPolling, { deep: true })

function qualificationDefinitionId(definition: any) {
  return String(definition?.cred_def_id || definition?.cred_def_ulid || "").trim()
}

function qualificationApplicationId(application: any) {
  return String(application?.app_id || application?.app_ulid || "").trim()
}

function exemptionUnitById(unitId: string) {
  return exemptionStages.value
    .flatMap((stage: any) => stage.units || [])
    .find((unit: any) => String(unit?.unit_id || "") === unitId)
}

function exemptionUnitByQualId(qualId: string) {
  return exemptionStages.value
    .flatMap((stage: any) => stage.units || [])
    .find((unit: any) => qualificationIdsForUnit(unit).includes(qualId))
}

function qualificationDefinitionForUnit(unit: any) {
  const qualId = qualificationIdsForUnit(unit)[0] || ""
  return qualificationDefinitions.value[qualId] || null
}

function qualificationFilesForUnit(unitId: string) {
  return qualificationUploadedFiles.value[unitId] || {}
}

async function loadQualificationDefinition(qualId: string) {
  if (qualificationDefinitions.value[qualId]) return qualificationDefinitions.value[qualId]
  const response = await apiClient(`/api/credentials/definitions?qual_ulids=${encodeURIComponent(qualId)}`)
  const definitions = Array.isArray(response?.definitions) ? response.definitions : []
  const definition = definitions.find((item: any) => qualificationDefinitionId(item) === qualId) || definitions[0]
  if (!definition) {
    throw new Error(t.value.credentialsPage.materialRequirementsUnavailable)
  }
  qualificationDefinitions.value = {
    ...qualificationDefinitions.value,
    [qualId]: definition,
  }
  return definition
}

async function openQualificationEditor(unit: any, qualId = qualificationIdsForUnit(unit)[0] || "") {
  if (!unit?.unit_id || !qualId) return
  await loadQualificationDefinition(qualId)
  expandedQualificationUnitIds.value = {
    ...expandedQualificationUnitIds.value,
    [unit.unit_id]: true,
  }
}

function closeQualificationEditor(unitId: string) {
  const next = { ...expandedQualificationUnitIds.value }
  delete next[unitId]
  expandedQualificationUnitIds.value = next
}

function isQualificationEditorExpanded(unitId: string) {
  return Boolean(expandedQualificationUnitIds.value[unitId])
}

function qualificationConstraintInputId(unitId: string, constraintName: string) {
  return `qualification-file-${unitId}-${constraintName}`
}

function triggerQualificationFileInput(unitId: string, constraintName: string) {
  document.getElementById(qualificationConstraintInputId(unitId, constraintName))?.click()
}

function qualificationFormatHint(constraint: any) {
  const info = getFileConstraintInfo(constraint?.type)
  const extText = info.extLabel === "Any" ? t.value.credentialsPage.anyFileType : info.extLabel
  return t.value.credentialsPage.supportedFormats
    .replace("{{exts}}", extText)
    .replace("{{limit}}", info.maxLabel)
}

function qualificationUploadSuccessText(fileName: string) {
  return t.value.credentialsPage.uploadSuccess.replace("{{fileName}}", fileName)
}

async function onQualificationFileChange(event: Event, unit: any, constraint: any) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) await uploadQualificationFile(unit, constraint, file)
  input.value = ""
}

async function uploadQualificationFile(unit: any, constraint: any, file: File) {
  const unitId = String(unit?.unit_id || "")
  const qualId = qualificationIdsForUnit(unit)[0] || ""
  const constraintName = String(constraint?.name || "").trim()
  const uploadingKey = `${unitId}:${constraintName}`
  if (!unitId || !qualId || !constraintName || qualificationUploadingKey.value) return

  const info = getFileConstraintInfo(constraint?.type)
  const fileExt = file.name.includes(".") ? `.${file.name.split(".").pop()?.toLowerCase()}` : ""
  if (info.maxSize && file.size > info.maxSize) {
    toast.error(t.value.credentialsPage.fileSizeLimitError.replace("{{limit}}", info.maxLabel))
    return
  }
  if (info.exts.length > 0 && !info.exts.includes(fileExt)) {
    toast.error(t.value.credentialsPage.fileTypeError.replace("{{exts}}", info.extLabel))
    return
  }

  qualificationUploadingKey.value = uploadingKey
  try {
    const fileHash = await sha256Hex(file)
    const contentType = file.type || "application/octet-stream"
    const upload = await apiClient("/api/credentials/upload-url", {
      method: "POST",
      body: JSON.stringify({
        cred_def_ulid: qualId,
        file_name: file.name,
        file_ext: fileExt,
        file_hash: fileHash,
        content_type: contentType,
        file_usage: constraintName,
      }),
    })
    const uploadResponse = await uploadWithTimeout(upload.upload_url, {
      method: "PUT",
      headers: new Headers(upload.signed_headers || {}),
      body: file,
    })
    if (!uploadResponse.ok) {
      throw new Error(`S3 upload failed: ${uploadResponse.status} ${uploadResponse.statusText}`)
    }
    qualificationUploadedFiles.value = {
      ...qualificationUploadedFiles.value,
      [unitId]: {
        ...qualificationFilesForUnit(unitId),
        [constraintName]: {
          name: file.name,
          url: upload.file_key,
          ext: fileExt,
          hash: fileHash,
          size: file.size,
        },
      },
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error || "")
    toast.error(`${t.value.credentialsPage.uploadFailed}: ${message}`)
  } finally {
    qualificationUploadingKey.value = ""
  }
}

async function submitQualificationApplication(unit: any) {
  const unitId = String(unit?.unit_id || "")
  const qualId = qualificationIdsForUnit(unit)[0] || ""
  const definition = qualificationDefinitionForUnit(unit)
  const constraints = definition?.file_constraints
  const uploadedFiles = qualificationFilesForUnit(unitId)
  if (!unitId || !qualId || !Array.isArray(constraints)) {
    toast.error(t.value.credentialsPage.materialRequirementsUnavailable)
    return
  }
  if (Object.keys(uploadedFiles).length === 0
    || constraints.some((constraint: any) => constraint.is_required && !uploadedFiles[constraint.name])) {
    toast.error(t.value.credentialsPage.requiredMaterialsMissing)
    return
  }

  const evidenceFiles = Object.keys(uploadedFiles).map((constraintName) => ({
    file_name: uploadedFiles[constraintName].name,
    file_url: uploadedFiles[constraintName].url,
    file_hash: uploadedFiles[constraintName].hash,
    file_ext: uploadedFiles[constraintName].ext,
    file_size: uploadedFiles[constraintName].size,
    file_usage: constraintName,
    file_type: constraints.find((constraint: any) => constraint.name === constraintName)?.type || 1,
  }))
  const existingApplication = qualificationApplications.value[qualId]
  qualificationSubmittingUnitId.value = unitId
  try {
    if (isApplicationResubmitStatus(existingApplication?.status)) {
      const appId = qualificationApplicationId(existingApplication)
      if (!appId) throw new Error(t.value.credentialsPage.submitFailed)
      await apiClient("/api/credentials/update", {
        method: "PUT",
        body: JSON.stringify({ app_id: appId, files: evidenceFiles }),
      })
    } else {
      await apiClient("/api/credentials/submit", {
        method: "POST",
        body: JSON.stringify({ cred_def_ulid: qualId, files: evidenceFiles }),
      })
    }
    toast.success(t.value.credentialsPage.submitSuccess)
    closeQualificationEditor(unitId)
    qualificationUploadedFiles.value = {
      ...qualificationUploadedFiles.value,
      [unitId]: {},
    }
    await refreshQualificationApplications()
  } catch (error) {
    console.error(error)
    toast.error(t.value.credentialsPage.submitFailed)
  } finally {
    qualificationSubmittingUnitId.value = ""
  }
}

async function resumeQualificationUploadAfterPayment() {
  const paymentAction = String(route.query.payment_action || "")
  const paymentStatus = String(route.query.payment_status || "")
  if (paymentAction !== "credential_application" || paymentStatus !== "success") return
  const qualId = String(route.query.qual_ulids || "").split(",")[0]?.trim() || ""
  const unitId = String(route.query.qualification_unit_id || "").trim()
  const unit = exemptionUnitById(unitId) || exemptionUnitByQualId(qualId)
  if (unit && qualId) {
    currentStep.value = 1
    try {
      await openQualificationEditor(unit, qualId)
    } catch (error) {
      console.error(error)
      toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
    }
  }
  const nextQuery = { ...route.query }
  delete nextQuery.payment_status
  delete nextQuery.payment_action
  delete nextQuery.order_id
  delete nextQuery.qual_ulids
  delete nextQuery.qualification_unit_id
  await router.replace({ path: route.path, query: nextQuery })
}

function isUploadReadyStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("UPLOAD_READY")
}

function isCredentialApplicationPaymentStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("WAIT_REVIEW_FEE_PAYMENT")
}

function isCredentialApplicationUnderReviewStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("UNDER_REVIEW")
}

function isCredentialApplicationResolvedStatus(status: unknown) {
  return String(status || "").trim().toUpperCase().includes("RESOLVED")
}

async function startQualificationApplication(unit: any) {
  const qualId = qualificationIdsForUnit(unit)[0] || ""
  const orderQualIds = qualificationOrderQualIds(qualId)
  if (!unit?.unit_id || !qualId || orderQualIds.length === 0 || !pipelineId.value || !bundleId) {
    toast.error(t.value.checkoutWizard.qualificationApplicationFailed)
    return
  }

  credentialApplicationLoadingUnitId.value = unit.unit_id
  try {
    const existingApplication = qualificationApplications.value[qualId] || await latestCredentialApplication(qualId)
    if (existingApplication) {
      qualificationApplications.value = {
        ...qualificationApplications.value,
        [qualId]: existingApplication,
      }
      if (isApplicationPendingStatus(existingApplication.status)) {
        toast.info(t.value.checkoutWizard.qualificationUnderReview)
        return
      }
      if (isApplicationApprovedStatus(existingApplication.status)) {
        toast.success(t.value.checkoutWizard.qualificationAlreadyApproved)
        await loadPurchaseReadyBundleInfo()
        return
      }
      if (isApplicationResubmitStatus(existingApplication.status)) {
        await openQualificationEditor(unit, qualId)
        return
      }
    }

    let order
    try {
      order = await apiClient("/api/credentials/application-orders", {
        method: "POST",
        suppressErrorToast: true,
        body: JSON.stringify({
          pipeline_cc_ulid: pipelineId.value,
          bundle_ulid: bundleId,
          qual_ulids: orderQualIds,
        }),
      })
    } catch (error) {
      const message = error instanceof ApiClientError
        ? error.rawMessage || error.errorCode || ""
        : error instanceof Error ? error.message : ""
      if (message.includes("in-progress credential application") || message.includes("进行中") || message.includes("请先处理")) {
        await refreshQualificationApplications()
        toast.info(t.value.checkoutWizard.qualificationUnderReview)
        return
      }
      throw error
    }

    const orderId = String(order?.application_order_ulid || "").trim()
    const orderStatus = String(order?.order_status || "")
    if (isUploadReadyStatus(orderStatus)) {
      toast.info(t.value.checkoutWizard.qualificationUploadReady)
      await openQualificationEditor(unit, qualId)
      return
    }
    if (isCredentialApplicationUnderReviewStatus(orderStatus)) {
      await refreshQualificationApplications()
      toast.info(t.value.checkoutWizard.qualificationUnderReview)
      return
    }
    if (isCredentialApplicationResolvedStatus(orderStatus)) {
      await loadPurchaseReadyBundleInfo()
      return
    }
    if (isCredentialApplicationPaymentStatus(orderStatus) || order?.payment_key) {
      if (!orderId) {
        throw new Error(t.value.checkoutWizard.qualificationApplicationFailed)
      }
      activeCredentialQualIds.value = orderQualIds
      activeCredentialUnitId.value = unit.unit_id
      activeOrderAction.value = "credential_application"
      activeOrderId.value = orderId
      currentStep.value = 4
      return
    }
    toast.info(t.value.checkoutWizard.qualificationApplicationCreated)
  } catch (error) {
    console.error(error)
    toast.error(error instanceof Error && error.message
      ? error.message
      : t.value.checkoutWizard.qualificationApplicationFailed)
  } finally {
    credentialApplicationLoadingUnitId.value = ""
  }
}

async function onExemptionToggle(unit: any, event: Event) {
  const input = event.target as HTMLInputElement | null
  if (!unit?.unit_id) return
  if (!unit.qualified) {
    if (input?.checked) await startQualificationApplication(unit)
    else closeQualificationEditor(unit.unit_id)
    return
  }
  selectedExemptionUnitIds.value = {
    ...selectedExemptionUnitIds.value,
    [unit.unit_id]: Boolean(input?.checked),
  }
}

async function nextFromStep1() {
  currentStep.value = 2
}

function formatMoney(amount?: number, currency = "usd") {
  if (typeof amount !== "number") return "-"
  return new Intl.NumberFormat(undefined, { style: "currency", currency: currency || "usd" }).format(amount / 100)
}

type ExemptionCredentialState = "active" | "pending" | "resubmit" | "rejected" | "expired" | "revoked" | "missing" | "unavailable"

function exemptionCredentialState(unit: any): ExemptionCredentialState {
  const qualifications = unit?.exemption_quals || []
  if (unit?.qualified || qualifications.some((qual: any) =>
    qual?.eligible || String(qual?.credential_status || "").toUpperCase() === "CREDENTIAL_STATUS_ACTIVE"
  )) {
    return "active"
  }

  const application = qualificationApplicationForUnit(unit)
  if (isApplicationPendingStatus(application?.status)) return "pending"
  if (isApplicationResubmitStatus(application?.status)) return "resubmit"
  if (isApplicationRejectedStatus(application?.status)) return "rejected"
  if (isApplicationApprovedStatus(application?.status)) return "active"

  const statuses = qualifications
    .map((qual: any) => String(qual?.credential_status || "").trim().toUpperCase())
    .filter(Boolean)
  if (statuses.includes("CREDENTIAL_STATUS_EXPIRED")) return "expired"
  if (statuses.includes("CREDENTIAL_STATUS_REVOKED")) return "revoked"
  if (statuses.includes("CREDENTIAL_STATUS_UNSPECIFIED")) return "missing"
  return "unavailable"
}

function exemptionCredentialLabel(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "active":
      return t.value.checkoutWizard.statusApproved
    case "pending":
      return t.value.checkoutWizard.statusPending
    case "resubmit":
      return t.value.checkoutWizard.statusResubmit
    case "rejected":
      return t.value.checkoutWizard.statusRejected
    case "expired":
      return t.value.checkoutWizard.statusExpired
    case "revoked":
      return t.value.checkoutWizard.statusRevoked
    case "missing":
      return t.value.checkoutWizard.statusMissing
    default:
      return t.value.checkoutWizard.statusUnavailable
  }
}

function exemptionCredentialBadgeClass(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "active":
      return "bg-emerald-100 text-emerald-800"
    case "pending":
      return "bg-blue-100 text-blue-800"
    case "resubmit":
      return "bg-amber-100 text-amber-800"
    case "rejected":
      return "bg-rose-100 text-rose-800"
    case "expired":
      return "bg-amber-100 text-amber-800"
    case "revoked":
      return "bg-rose-100 text-rose-800"
    default:
      return "bg-slate-100 text-slate-700"
  }
}

function qualificationActionLabel(unit: any) {
  switch (exemptionCredentialState(unit)) {
    case "pending":
      return t.value.checkoutWizard.statusPending
    case "resubmit":
      return t.value.checkoutWizard.resubmitQualification
    default:
      return t.value.checkoutWizard.applyQualification
  }
}

async function nextFromStep2() {
  if (!isMembershipBundle.value && !formData.agreement) {
    toast.error(t.value.examSignup.agreementRequired)
    return
  }
  sanitizeSignupForm()
  const requiredFields = [
    ["first_name", t.value.examSignup.formFirstName],
    ["last_name", t.value.examSignup.formLastName],
    ["email", t.value.examSignup.formEmail],
    ["gender", t.value.examSignup.formGender],
    ["birthdate", t.value.examSignup.formBirthdate],
    ["country", t.value.examSignup.formCountry],
    ["province", t.value.examSignup.formProvince],
    ["city", t.value.examSignup.formCity],
    ["address", t.value.examSignup.formAddress],
    ["postal_code", t.value.examSignup.formPostalCode],
  ] as const
  for (const [key, label] of requiredFields) {
    if (!String(formData[key as keyof typeof formData]).trim()) {
      toast.error(t.value.examSignup.validationRequired.replace("{{field}}", label))
      return
    }
  }
  if (!isValidEmail(formData.email)) {
    toast.error(t.value.examSignup.validationInvalidEmail)
    return
  }
  if (!isValidInternationalPhone(formData.phone)) {
    toast.error(t.value.examSignup.validationInvalidPhone.replace("{{field}}", t.value.examSignup.formWorkPhone))
    return
  }
  if (!isValidPostalCode(formData.postal_code, true)) {
    toast.error(t.value.examSignup.validationInvalidPostalCode)
    return
  }
  loading.value = true
  try {
    await syncSignupToProfile()
    currentStep.value = 3
  } catch (err) {
    console.error(err)
  } finally {
    loading.value = false
  }
}

function normalizedOrderStatus(status: unknown) {
  const value = String(status || "").trim().toUpperCase()
  switch (value) {
    case "1":
      return "PENDING_CREATE"
    case "2":
      return "PENDING_PAYMENT"
    case "3":
      return "COMPLETED"
    case "4":
      return "CANCELLED"
    case "5":
      return "FAILED"
    default:
      return value
  }
}

function isCompletedStatus(status: unknown) {
  return normalizedOrderStatus(status).includes("COMPLETED")
}

function isFailedStatus(status: unknown) {
  const value = normalizedOrderStatus(status)
  return value.includes("FAILED") || value.includes("CANCEL") || value.includes("REJECT")
}

function getEligibility(response: any) {
  return response?.purchase_state?.eligibility || response?.eligibility || {}
}

async function createPurchaseOrder() {
  const response = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/purchase`, {
    method: "POST",
    suppressErrorToast: true,
    body: JSON.stringify({
      payment_mode: paymentMode.value,
      selected_exemptions_json: buildSelectedExemptionsJson(),
    }),
  })
  const orderId = String(response?.bundle_order_ulid || response?.order_id || "").trim()
  const orderStatus = response?.order_status || response?.status

  if (isFailedStatus(orderStatus)) {
    throw new Error(response?.message || t.value.checkoutWizard.orderCreationFailed)
  }
  if (!orderId) {
    throw new Error(t.value.checkoutWizard.orderCreationFailed)
  }

  if (isCompletedStatus(orderStatus)) {
    toast.success(t.value.checkoutWizard.purchaseCompleted)
    await router.push(`/checkout/success/${encodeURIComponent(orderId)}`)
    return
  }

  activeOrderAction.value = "purchase"
  activeOrderId.value = orderId
  currentStep.value = 4
}

async function createUnlockOrder() {
  if (!pipelineId.value) {
    throw new Error(t.value.checkoutWizard.missingPipeline)
  }

  const hadExemptionOptions = exemptionStages.value.length > 0
  const response = await apiClient(`/api/mall/bundles/${encodeURIComponent(bundleId)}/unlock`, {
    method: "POST",
    suppressErrorToast: true,
    body: JSON.stringify({
      pipeline_cc_ulid: pipelineId.value,
    }),
  })
  const orderId = String(response?.pipeline_unlock_order_ulid || response?.order_id || "").trim()
  const orderStatus = response?.order_status || response?.status

  if (isFailedStatus(orderStatus)) {
    throw new Error(response?.message || t.value.checkoutWizard.orderCreationFailed)
  }

  if (isCompletedStatus(orderStatus)) {
    toast.success(t.value.checkoutWizard.unlockCompleted)
    const refreshedBundle = await loadBundleInfo()
    if (!getEligibility(refreshedBundle)?.can_purchase) {
      return
    }

    if (!hadExemptionOptions && exemptionStages.value.length > 0) {
      currentStep.value = 1
      return
    }

    await createPurchaseOrder()
    return
  }

  if (!orderId) {
    throw new Error(t.value.checkoutWizard.orderCreationFailed)
  }

  activeOrderAction.value = "unlock"
  activeOrderId.value = orderId
  currentStep.value = 4
}

async function confirmAndPay() {
  loading.value = true
  try {
    const latestBundle = await loadPurchaseReadyBundleInfo()
    const eligibility = getEligibility(latestBundle)

    if (eligibility?.can_unlock) {
      await createUnlockOrder()
      return
    }
    if (eligibility?.can_purchase) {
      await createPurchaseOrder()
      return
    }
    if (latestBundle?.purchase_state?.active_order) {
      toast.info(t.value.checkoutWizard.continueExistingOrder)
      await router.push("/orders")
      return
    }

    throw new Error(t.value.checkoutWizard.purchaseUnavailable)
  } catch (err) {
    console.error(err)
    toast.error(err instanceof Error && err.message
      ? err.message
      : t.value.checkoutWizard.orderCreationFailed)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <AppShell content-class="p-0">
    <div class="checkout-page page-panel">
      <header class="flex h-16 items-center border-b border-border bg-white px-5">
        <ClipboardList class="mr-4 h-4 w-4 text-slate-700" />
        <span class="text-sm font-medium text-foreground">{{ t.checkoutWizard.checkoutTitle }}</span>
      </header>

      <main class="checkout-content px-5 py-8 md:px-8 lg:px-10">
        <div class="checkout-heading mb-8 max-w-5xl">
          <h1 class="text-3xl font-bold tracking-tight text-foreground">{{ t.checkoutWizard.checkoutTitle }}</h1>
          <div class="checkout-progress" aria-label="Checkout progress">
            <div class="checkout-progress-step" :class="{ active: currentStep === 1 }" :aria-current="currentStep === 1 ? 'step' : undefined">
              <span class="checkout-progress-node">1</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step1.replace(/^\d+\s*/, "") }}</span>
            </div>
            <div class="checkout-progress-step" :class="{ active: currentStep === 2 }" :aria-current="currentStep === 2 ? 'step' : undefined">
              <span class="checkout-progress-node">2</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step2.replace(/^\d+\s*/, "") }}</span>
            </div>
            <div class="checkout-progress-step" :class="{ active: currentStep === 3 }" :aria-current="currentStep === 3 ? 'step' : undefined">
              <span class="checkout-progress-node">3</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step3.replace(/^\d+\s*/, "") }}</span>
            </div>
            <div class="checkout-progress-step" :class="{ active: currentStep === 4 }" :aria-current="currentStep === 4 ? 'step' : undefined">
              <span class="checkout-progress-node">4</span>
              <span class="checkout-progress-label">{{ t.checkoutWizard.step4.replace(/^\d+\s*/, "") }}</span>
            </div>
          </div>
        </div>
        
        <LoadingState
          v-if="initialLoading"
          class="checkout-loading-state"
          :label="t.common.loading"
          variant="section"
          :rows="4"
        />

        <template v-else>
        <div class="checkout-card max-w-5xl rounded-[16px] bg-white p-6 shadow-[0_10px_24px_rgba(15,74,82,0.05)]">
          <!-- Step 1: Selection -->
          <div v-if="currentStep === 1" class="checkout-step-one space-y-8">
            <div class="checkout-step-one-title mb-4">
              <h2 class="text-2xl font-bold">{{ t.checkoutWizard.yourLevel1Paper.replace(levelPlaceholder, "1") }}</h2>
            </div>
            
            <div v-if="exemptionStages.length > 0" class="checkout-stage-list space-y-6">
              <div v-for="stage in exemptionStages" :key="stage.stage_id || stage.index" class="checkout-stage space-y-6">
                <div class="checkout-unit-grid grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div
                    v-for="unit in stage.units"
                    :key="unit.unit_id"
                    :class="[
                      'checkout-unit-card group relative flex flex-col justify-between overflow-hidden rounded-2xl border p-5 transition-all duration-300',
                      isQualificationEditorExpanded(unit.unit_id) ? 'md:col-span-2' : '',
                      selectedExemptionUnitIds[unit.unit_id]
                        ? 'border-emerald-400 bg-emerald-50/40 shadow-md ring-1 ring-emerald-400'
                        : unit.qualified
                          ? 'cursor-pointer border-border hover:border-emerald-200 hover:shadow-sm'
                          : 'cursor-pointer border-border bg-slate-50/70 hover:border-blue-200 hover:shadow-sm',
                    ]"
                  >
                    <div class="checkout-unit-main mb-4">
                      <div class="checkout-unit-id mb-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">{{ unit.unit_id }}</div>
                      <h3 class="checkout-unit-title text-xl font-bold text-slate-800">{{ unit.unit_name || unit.unit_id }}</h3>
                      <p v-if="unit.exemption_quals?.[0]?.description" class="checkout-unit-description mt-2 text-sm text-slate-500">{{ unit.exemption_quals[0].description }}</p>
                      
                      <div :class="['checkout-unit-badge mt-3 inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium', exemptionCredentialBadgeClass(unit)]">
                        <CheckCircle2 v-if="exemptionCredentialState(unit) === 'active'" class="mr-1 h-3.5 w-3.5" />
                        <Clock v-else-if="['pending', 'expired'].includes(exemptionCredentialState(unit))" class="mr-1 h-3.5 w-3.5" />
                        <CircleAlert v-else class="mr-1 h-3.5 w-3.5" />
                        {{ exemptionCredentialLabel(unit) }}
                      </div>
                    </div>
                    
                    <div class="checkout-unit-footer mt-auto pt-4 flex items-center justify-between border-t border-slate-100">
                      <label class="checkout-unit-option cursor-pointer">
                        <div class="relative flex items-center justify-center">
                          <input
                            type="checkbox"
                            class="peer sr-only"
                            :checked="unit.qualified ? Boolean(selectedExemptionUnitIds[unit.unit_id]) : isQualificationEditorExpanded(unit.unit_id)"
                            :disabled="credentialApplicationLoadingUnitId === unit.unit_id || (!unit.qualified && exemptionCredentialState(unit) === 'pending')"
                            @change="onExemptionToggle(unit, $event)"
                          />
                          <div class="checkout-unit-checkbox h-6 w-6 rounded-md border-2 border-slate-300 bg-white transition-all peer-checked:border-emerald-500 peer-checked:bg-emerald-500"></div>
                          <Loader2 v-if="credentialApplicationLoadingUnitId === unit.unit_id" class="pointer-events-none absolute h-4 w-4 animate-spin text-blue-600" />
                          <Check v-else class="pointer-events-none absolute h-4 w-4 text-white opacity-0 transition-opacity peer-checked:opacity-100" />
                        </div>
                        <span class="checkout-unit-action font-medium text-slate-700">
                          {{ unit.qualified ? t.checkoutWizard.applyForExemption : qualificationActionLabel(unit) }}
                        </span>
                        <span
                          v-if="selectedExemptionUnitIds[unit.unit_id] && (unitPriceDisplay[unit.unit_id]?.exemptionAmount !== undefined || unitPriceDisplay[unit.unit_id]?.accessAmount !== undefined)"
                          class="checkout-unit-selected-price"
                        >
                          {{ formatMoney(unitPriceDisplay[unit.unit_id]?.exemptionAmount ?? unitPriceDisplay[unit.unit_id]?.accessAmount, unitPriceDisplay[unit.unit_id]?.currency) }}
                        </span>
                        <strong
                          v-else-if="unitPriceDisplay[unit.unit_id]?.accessAmount !== undefined"
                          class="checkout-unit-default-price"
                        >
                          {{ formatMoney(unitPriceDisplay[unit.unit_id]?.accessAmount, unitPriceDisplay[unit.unit_id]?.currency) }}
                        </strong>
                      </label>
                    </div>

                    <div
                      v-if="isQualificationEditorExpanded(unit.unit_id) && !unit.qualified"
                      class="mt-5 border-t border-blue-100 pt-5"
                    >
                      <div class="rounded-2xl border border-blue-100 bg-blue-50/70 p-4 sm:p-5">
                        <div class="flex items-start gap-3">
                          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-white text-blue-600 shadow-sm">
                            <UploadCloud class="h-5 w-5" />
                          </div>
                          <div>
                            <h4 class="font-semibold text-slate-900">
                              {{ qualificationDefinitionForUnit(unit)?.name || t.credentialsPage.uploadMaterials }}
                            </h4>
                            <p class="mt-1 text-sm leading-6 text-slate-600">
                              {{ qualificationDefinitionForUnit(unit)?.description || t.credentialsPage?.description }}
                            </p>
                          </div>
                        </div>

                        <div
                          v-if="Array.isArray(qualificationDefinitionForUnit(unit)?.file_constraints)"
                          class="mt-5 grid gap-4 sm:grid-cols-2"
                        >
                          <div
                            v-for="constraint in qualificationDefinitionForUnit(unit)?.file_constraints || []"
                            :key="constraint.name"
                            class="rounded-xl border border-white bg-white p-4 shadow-sm"
                          >
                            <div class="flex items-center gap-1 text-sm font-semibold text-slate-800">
                              <span v-if="constraint.is_required" class="text-rose-500">*</span>
                              <span>{{ constraint.name }}</span>
                            </div>
                            <p class="mt-1 text-xs text-slate-500">{{ qualificationFormatHint(constraint) }}</p>
                            <div class="mt-3 flex flex-wrap items-center gap-3">
                              <button
                                type="button"
                                class="btn btn-outline h-9 rounded-lg px-3 text-xs"
                                :disabled="Boolean(qualificationUploadingKey) || qualificationSubmittingUnitId === unit.unit_id"
                                @click="triggerQualificationFileInput(unit.unit_id, constraint.name)"
                              >
                                <Loader2
                                  v-if="qualificationUploadingKey === `${unit.unit_id}:${constraint.name}`"
                                  class="h-4 w-4 animate-spin"
                                />
                                <UploadCloud v-else class="h-4 w-4" />
                                {{ t.credentialsPage.chooseFile }}
                              </button>
                              <span
                                class="max-w-[260px] truncate text-sm text-slate-500"
                                :title="qualificationFilesForUnit(unit.unit_id)[constraint.name]?.name || ''"
                              >
                                {{ qualificationFilesForUnit(unit.unit_id)[constraint.name]?.name || t.credentialsPage.noFileChosen }}
                              </span>
                              <input
                                :id="qualificationConstraintInputId(unit.unit_id, constraint.name)"
                                type="file"
                                class="hidden"
                                :accept="getFileConstraintInfo(constraint.type).acceptStr"
                                @change="onQualificationFileChange($event, unit, constraint)"
                              />
                            </div>
                            <p
                              v-if="qualificationFilesForUnit(unit.unit_id)[constraint.name]"
                              class="mt-3 flex items-center gap-1 text-xs font-medium text-emerald-600"
                            >
                              <CheckCircle2 class="h-3.5 w-3.5" />
                              {{ qualificationUploadSuccessText(qualificationFilesForUnit(unit.unit_id)[constraint.name].name) }}
                            </p>
                          </div>
                        </div>

                        <div class="mt-5 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
                          <button
                            type="button"
                            class="btn btn-outline"
                            :disabled="qualificationSubmittingUnitId === unit.unit_id"
                            @click="closeQualificationEditor(unit.unit_id)"
                          >
                            {{ t.common.cancel }}
                          </button>
                          <button
                            type="button"
                            class="btn bg-emerald-600 text-white hover:bg-emerald-700"
                            :disabled="Boolean(qualificationUploadingKey) || qualificationSubmittingUnitId === unit.unit_id"
                            @click="submitQualificationApplication(unit)"
                          >
                            <Loader2 v-if="qualificationSubmittingUnitId === unit.unit_id" class="h-4 w-4 animate-spin" />
                            {{ qualificationSubmittingUnitId === unit.unit_id
                              ? t.credentialsPage.submitting
                              : t.credentialsPage.submitApplication }}
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div v-if="isExemptionSelected" class="checkout-declaration mt-8 rounded-xl border border-blue-200 bg-blue-50/50 p-5 transition-all">
                <label class="flex cursor-pointer items-start gap-3">
                  <div class="relative mt-0.5 flex shrink-0 items-center justify-center">
                    <input
                      v-model="exemptionDeclarationChecked"
                      type="checkbox"
                      class="peer sr-only"
                    />
                    <div class="h-5 w-5 rounded border border-slate-300 bg-white transition-all peer-checked:border-emerald-500 peer-checked:bg-emerald-500"></div>
                    <Check class="pointer-events-none absolute h-3.5 w-3.5 text-white opacity-0 transition-opacity peer-checked:opacity-100" />
                  </div>
                  <span class="text-sm font-medium leading-relaxed text-slate-700">
                    {{ t.checkoutWizard.declarationText }}
                  </span>
                </label>
              </div>

              <div v-if="bundleData" class="checkout-step-actions mt-6 flex items-center justify-end">
                <div class="checkout-total text-lg font-bold text-slate-900">
                  <template v-if="dynamicPaymentPreview">
                    {{ t.checkoutWizard.baseTotal }} {{ formatMoney(dynamicPaymentPreview.total, dynamicPaymentPreview.currency) }}
                  </template>
                </div>
              </div>
            </div>
          </div>

          <!-- Step 2: Registration -->
          <form id="checkout-registration-form" v-if="currentStep === 2" class="space-y-6" novalidate @submit.prevent="nextFromStep2">
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formFirstName }}</span><input v-model="formData.first_name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" required /></label>
              <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formLastName }}</span><input v-model="formData.last_name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" required /></label>
            </div>
            <label class="block space-y-2"><span class="text-sm font-medium">{{ t.examSignup.formMiddleName }}</span><input v-model="formData.middle_name" class="input" :maxlength="PROFILE_TEXT_LIMITS.name" /></label>
            <div class="grid gap-4 sm:grid-cols-2">
              <label class="space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formEmail }}</span><input v-model="formData.email" class="input" type="email" :maxlength="PROFILE_TEXT_LIMITS.short" required /></label>
              <label class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formGender }}</span>
                <select v-model="formData.gender" class="input cursor-pointer" required>
                  <option value="" disabled>{{ t.examSignup.formGender }}</option>
                  <option v-for="option in genderOptions" :key="option" :value="option">{{ t.common.genderOptions[option] }}</option>
                </select>
              </label>
            </div>
            <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formBirthdate }}</span><input v-model="formData.birthdate" class="input" type="date" required /></label>
            <div class="grid gap-4 sm:grid-cols-3">
              <label class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formCountry }}</span>
                <select v-model="selectedCountryCode" class="input cursor-pointer" required @change="handleCountryChange">
                  <option value="" disabled>{{ t.examSignup.formCountry }}</option>
                  <option v-for="country in countryOptions" :key="country.code" :value="country.code">{{ country.displayName }}</option>
                </select>
              </label>
              <label class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formProvince }}</span>
                <select v-if="provinceOptions.length > 0" v-model="selectedProvinceCode" class="input cursor-pointer" required @change="handleProvinceChange">
                  <option value="" disabled>{{ t.examSignup.formProvince }}</option>
                  <option v-for="province in provinceOptions" :key="province.isoCode" :value="province.isoCode">{{ localizedProvinceName(province) }}</option>
                </select>
                <input v-else v-model="formData.province" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" required />
              </label>
              <label class="space-y-2">
                <span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formCity }}</span>
                <select v-if="cityOptions.length > 0" v-model="formData.city" class="input cursor-pointer" required>
                  <option value="" disabled>{{ t.examSignup.formCity }}</option>
                  <option v-for="city in cityOptions" :key="`${city.name}-${city.latitude}-${city.longitude}`" :value="localizedCityName(city)">{{ localizedCityName(city) }}</option>
                </select>
                <input v-else v-model="formData.city" class="input" :maxlength="PROFILE_TEXT_LIMITS.short" required />
              </label>
            </div>
            <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formAddress }}</span><input v-model="formData.address" class="input" :maxlength="PROFILE_TEXT_LIMITS.address" required /></label>
            <label class="block space-y-2"><span class="text-sm font-medium"><span class="text-red-500">*</span> {{ t.examSignup.formPostalCode }}</span><input v-model="formData.postal_code" class="input" :maxlength="PROFILE_TEXT_LIMITS.postalCode" pattern="[A-Za-z0-9][A-Za-z0-9 -]*[A-Za-z0-9]" required @blur="formData.postal_code = normalizePostalCode(formData.postal_code)" /></label>
            
            <div class="grid gap-4 sm:grid-cols-1">
              <label class="space-y-2">
                <span class="text-sm font-medium">{{ t.examSignup.formWorkPhone }}</span>
                <div class="flex gap-2">
                  <select v-if="orgPhonePrefixes.length > 0" v-model="formData.phone_country_code" class="input cursor-pointer w-28 shrink-0">
                    <option v-for="prefix in orgPhonePrefixes" :key="prefix.code" :value="prefix.code">{{ prefix.dialCode }}</option>
                  </select>
                  <input
                    id="exam-signup-work-phone"
                    v-model="formData.phone"
                    class="input flex-1"
                    type="tel"
                    inputmode="tel"
                    autocomplete="tel"
                    maxlength="24"
                    :placeholder="t.examSignup.formWorkPhonePlaceholder"
                  />
                </div>
              </label>
            </div>

            <div v-if="!isMembershipBundle" class="mt-6 border-t border-border pt-6">
              <label class="flex items-center gap-3">
                <input v-model="formData.agreement" type="checkbox" class="h-4 w-4 shrink-0 rounded border-gray-300 text-emerald-600 focus:ring-emerald-500" :required="!isMembershipBundle" />
                <span class="text-sm font-medium text-foreground">{{ t.examSignup.agreement }}</span>
              </label>
            </div>

          </form>

          <!-- Step 3: Review -->
          <div v-if="currentStep === 3" class="space-y-6">
            <h2 class="text-xl font-semibold">{{ t.checkoutWizard.review }}</h2>
            <div class="rounded-lg border border-border p-4 text-sm space-y-2">
              <div class="grid grid-cols-3 gap-2">
                <div class="text-muted-foreground">{{ t.checkoutWizard.reviewName }}</div>
                <div class="col-span-2 font-medium">{{ formData.first_name }} {{ formData.last_name }}</div>
                
                <div class="text-muted-foreground">{{ t.checkoutWizard.reviewEmail }}</div>
                <div class="col-span-2 font-medium">{{ formData.email }}</div>
                
                <div class="text-muted-foreground">{{ t.checkoutWizard.reviewLocation }}</div>
                <div class="col-span-2 font-medium">{{ formData.city }}, {{ formData.province }}, {{ formData.country }}</div>
              </div>
            </div>

            <!-- PAYMENT MODE SELECTION -->
            <div v-if="isMultiStage" class="rounded-lg border border-border p-4 text-sm space-y-4">
              <div class="mb-2 text-sm font-semibold">{{ t.checkoutWizard.paymentModeTitle }}</div>
              
              <label class="flex items-start gap-3 rounded-lg border p-4 transition-colors hover:bg-slate-50 cursor-pointer" :class="{ 'border-emerald-500 bg-emerald-50/30': paymentMode === 'FULL_PIPELINE', 'border-border': paymentMode !== 'FULL_PIPELINE' }">
                <input type="radio" v-model="paymentMode" value="FULL_PIPELINE" class="mt-1 h-4 w-4 text-emerald-600 focus:ring-emerald-500" />
                <div>
                  <div class="font-medium text-slate-900">{{ t.checkoutWizard.modeFullPipeline }}</div>
                  <div class="text-xs text-slate-500 mt-1">{{ t.checkoutWizard.modeFullPipelineDesc }}</div>
                </div>
              </label>

              <label class="flex items-start gap-3 rounded-lg border p-4 transition-colors hover:bg-slate-50 cursor-pointer" :class="{ 'border-emerald-500 bg-emerald-50/30': paymentMode === 'BY_STAGE', 'border-border': paymentMode !== 'BY_STAGE' }">
                <input type="radio" v-model="paymentMode" value="BY_STAGE" class="mt-1 h-4 w-4 text-emerald-600 focus:ring-emerald-500" />
                <div>
                  <div class="font-medium text-slate-900">{{ t.checkoutWizard.modeByStage }}</div>
                  <div class="text-xs text-slate-500 mt-1">{{ t.checkoutWizard.modeByStageDesc }}</div>
                </div>
              </label>
            </div>

            <div v-if="dynamicPaymentPreview && paymentMode === 'FULL_PIPELINE'" class="rounded-lg bg-muted/30 p-4 border border-border">
              <div class="mb-3 text-sm font-semibold">{{ t.checkoutWizard.priceSummary }}</div>
              <div class="space-y-2 text-sm">
                <div class="flex justify-between">
                  <span class="text-muted-foreground">{{ t.checkoutWizard.subtotal }}</span>
                  <span class="font-medium">{{ dynamicPaymentPreview.amount_label || formatMoney(dynamicPaymentPreview.subtotal, dynamicPaymentPreview.currency) }}</span>
                </div>
                <div v-if="dynamicPaymentPreview.discount_total" class="flex justify-between">
                  <span class="text-muted-foreground">{{ t.checkoutWizard.discount }}</span>
                  <span class="font-medium">-{{ formatMoney(dynamicPaymentPreview.discount_total, dynamicPaymentPreview.currency) }}</span>
                </div>
                <div class="mt-2 flex justify-between border-t border-border pt-2">
                  <span class="font-semibold text-foreground">{{ t.checkoutWizard.total }}</span>
                  <span class="text-lg font-bold text-foreground">{{ dynamicPaymentPreview.pay_amount_label || formatMoney(dynamicPaymentPreview.total, dynamicPaymentPreview.currency) }}</span>
                </div>
              </div>
            </div>

          </div>

          <!-- Step 4: Payment -->
          <div v-if="currentStep === 4" class="space-y-6">
            <div>
              <h2 class="text-xl font-semibold">
                {{ activeOrderAction === "credential_application"
                  ? t.checkoutWizard.qualificationPaymentTitle
                  : t.checkoutWizard.payment }}
              </h2>
              <p v-if="activeOrderAction === 'credential_application'" class="mt-2 text-sm text-muted-foreground">
                {{ t.checkoutWizard.qualificationPaymentDesc }}
              </p>
            </div>
            <PaymentSessionPanel
              v-if="activeOrderId"
              :biz-type="paymentBizType"
              :biz-ref-ulid="activeOrderId"
              :order-id="activeOrderId"
              :source="activeOrderAction"
              :return-path="paymentReturnPath"
              :extra-return-params="paymentReturnParams"
              min-height-class="min-h-[420px]"
            />
          </div>
        </div>

        <div
          v-if="currentStep === 1 && exemptionStages.length > 0 && bundleData"
          class="checkout-step-footer"
        >
          <button
            class="checkout-next-button btn rounded-full px-8 py-3 text-white disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="hasExpandedQualificationEditors || Boolean(qualificationSubmittingUnitId) || (isExemptionSelected && !exemptionDeclarationChecked)"
            @click="nextFromStep1"
          >
            {{ t.checkoutWizard.saveAndContinue }}
          </button>
        </div>

        <div v-else-if="currentStep === 2" class="checkout-step-footer checkout-form-actions flex items-center">
          <button v-if="exemptionStages.length > 0" type="button" class="checkout-back-button btn btn-outline" @click="currentStep = 1">
            <ArrowLeft class="h-4 w-4" />
            {{ t.checkoutWizard.back }}
          </button>
          <button form="checkout-registration-form" type="submit" class="checkout-form-next-button btn rounded-full px-8 text-white" :disabled="loading">
            <template v-if="loading"><Loader2 class="h-4 w-4 animate-spin" /> {{ t.examSignup.submitting }}</template>
            <template v-else>{{ t.checkoutWizard.next }} <Send class="ml-2 h-4 w-4" /></template>
          </button>
        </div>

        <div v-else-if="currentStep === 3" class="checkout-step-footer checkout-review-actions flex items-center">
          <button type="button" class="checkout-back-button btn btn-outline" @click="currentStep = 2" :disabled="loading">
            <ArrowLeft class="h-4 w-4" />
            {{ t.checkoutWizard.back }}
          </button>
          <button class="checkout-form-next-button btn btn-primary" :disabled="loading" @click="confirmAndPay">
            <template v-if="loading"><Loader2 class="h-4 w-4 animate-spin" /> {{ t.checkoutWizard.processing }}</template>
            <template v-else>{{ t.checkoutWizard.confirmAndPay }} <ArrowRight class="h-4 w-4" /></template>
          </button>
        </div>
        </template>
      </main>
    </div>
  </AppShell>
</template>

<style scoped>
.checkout-page {
  background-color: #f2f6fc !important;
}

.checkout-content {
  width: 100%;
  max-width: 1080px;
  margin: 0 auto;
  padding: 24px 32px 64px !important;
}

.checkout-heading,
.checkout-card,
.checkout-loading-state,
.checkout-step-footer {
  width: 100%;
  max-width: none;
}

.checkout-loading-state {
  min-height: 320px;
}

.checkout-heading {
  margin-bottom: 18px;
}

.checkout-heading h1 {
  font-size: 28px;
  line-height: 1.25;
  letter-spacing: 0;
}

.checkout-progress {
  display: grid;
  width: min(460px, 100%);
  margin: 18px auto 0;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.checkout-progress-step {
  position: relative;
  display: flex;
  min-width: 0;
  align-items: center;
  flex-direction: column;
  gap: 7px;
  color: #52617a;
}

.checkout-progress-step:not(:last-child)::after {
  position: absolute;
  z-index: 0;
  top: 14px;
  left: calc(50% + 19px);
  width: calc(100% - 38px);
  border-top: 2px dotted #cbd8e9;
  content: "";
}

.checkout-progress-node {
  position: relative;
  z-index: 1;
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #52617a;
  background: #e4ebf5;
  font-size: 13px;
  font-weight: 700;
}

.checkout-progress-label {
  overflow-wrap: anywhere;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.35;
  text-align: center;
}

.checkout-progress-step.active {
  color: #0d3f72;
}

.checkout-progress-step.active .checkout-progress-node {
  color: #fff;
  background: #104d84;
}

.checkout-card {
  padding: 26px;
  border: 1px solid #d5e0ef;
  border-radius: 14px;
  box-shadow: none;
}

.checkout-step-one > :not([hidden]) ~ :not([hidden]),
.checkout-stage-list > :not([hidden]) ~ :not([hidden]),
.checkout-stage > :not([hidden]) ~ :not([hidden]) {
  margin-top: 18px;
}

.checkout-step-one-title {
  margin-bottom: 0;
}

.checkout-step-one-title h2 {
  color: #0b2347;
  font-size: 20px;
  line-height: 1.35;
}

.checkout-unit-grid {
  gap: 12px;
}

.checkout-unit-card {
  min-height: 132px;
  padding: 14px;
  border-color: #cfdaea;
  border-radius: 12px;
  background: #fff;
  box-shadow: none;
}

.checkout-unit-card:hover {
  border-color: #9eb5d3;
  box-shadow: none;
}

.checkout-unit-card.ring-1 {
  border-color: #cfdaea;
  background: #fff;
  box-shadow: none;
}

.checkout-unit-main {
  margin-bottom: 7px;
}

.checkout-unit-id {
  display: none;
}

.checkout-unit-title {
  color: #0b2347;
  font-size: 15px;
  line-height: 1.4;
}

.checkout-unit-description {
  display: none;
}

.checkout-unit-badge {
  margin-top: 7px;
  padding: 2px 9px;
  border-color: transparent;
  font-size: 11px;
}

.checkout-unit-badge.bg-emerald-100 {
  color: #9a6500;
  background: #fff3d8;
}

.checkout-unit-badge.bg-emerald-100 svg {
  display: none;
}

.checkout-unit-footer {
  padding-top: 5px;
  border-top: 0;
}

.checkout-unit-option {
  display: grid;
  align-items: center;
  grid-template-columns: 17px minmax(0, 1fr);
  column-gap: 8px;
  row-gap: 6px;
}

.checkout-unit-checkbox {
  width: 17px;
  height: 17px;
  border-width: 1px;
  border-radius: 3px;
}

.checkout-unit-footer input.peer:checked + .checkout-unit-checkbox {
  border-color: #2d6cdf;
  background: #2d6cdf;
}

.checkout-unit-action {
  color: #243b60;
  font-size: 13px;
}

.checkout-unit-default-price,
.checkout-unit-selected-price {
  grid-column: 1 / -1;
  width: fit-content;
  line-height: 1.35;
}

.checkout-unit-default-price {
  color: #0b2347;
  font-size: 15px;
  font-weight: 700;
}

.checkout-unit-selected-price {
  padding: 3px 11px;
  border-radius: 999px;
  color: #078653;
  background: #e2f5eb;
  font-size: 12px;
  font-weight: 700;
}

.checkout-declaration {
  margin-top: 16px;
  padding: 14px 16px;
  border-radius: 8px;
}

.checkout-declaration input.peer:checked + div {
  border-color: #2d6cdf;
  background: #2d6cdf;
}

.checkout-step-actions {
  margin-top: 20px;
  padding: 13px 0 0;
  border: 0;
  border-top: 2px solid #102f59;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.checkout-total:empty {
  display: none;
}

.checkout-total {
  color: #0b2347;
  font-size: 18px;
  line-height: 1.4;
}

.checkout-step-footer {
  margin-top: 18px;
}

.checkout-next-button {
  min-width: 136px;
  min-height: 44px;
  padding: 10px 24px;
  background: #0d4c83;
  box-shadow: none;
}

.checkout-next-button:hover {
  background: #083b68;
}

.checkout-form-actions {
  margin-top: 22px;
  padding-top: 0;
  justify-content: flex-start;
  gap: 10px;
  border-top: 0;
}

.checkout-review-actions {
  margin-top: 22px;
  padding-top: 0;
  justify-content: flex-start;
  gap: 10px;
}

.checkout-back-button,
.checkout-form-next-button {
  min-height: 44px;
  padding: 10px 24px;
  border-radius: 999px;
  box-shadow: none;
  font-weight: 600;
}

.checkout-back-button {
  min-width: 96px;
  border-color: #0d4c83;
  color: #0d4c83;
  background: #fff;
}

.checkout-back-button:hover {
  border-color: #083b68;
  color: #083b68;
  background: #f3f7fc;
}

.checkout-form-next-button {
  min-width: 136px;
  background: #0d4c83;
}

.checkout-form-next-button:hover {
  background: #083b68;
}

@media (max-width: 767px) {
  .checkout-content {
    padding: 20px 16px 48px !important;
  }

  .checkout-card {
    padding: 18px;
  }

  .checkout-heading h1 {
    font-size: 25px;
  }

  .checkout-progress {
    width: 100%;
    margin-top: 16px;
  }

  .checkout-progress-step:not(:last-child)::after {
    left: calc(50% + 17px);
    width: calc(100% - 34px);
  }

  .checkout-progress-label {
    font-size: 11px;
  }

  .checkout-step-actions {
    justify-content: flex-start;
  }

  .checkout-form-actions,
  .checkout-review-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .checkout-back-button,
  .checkout-form-next-button {
    width: 100%;
  }

  .checkout-next-button {
    width: auto;
  }
}
</style>
