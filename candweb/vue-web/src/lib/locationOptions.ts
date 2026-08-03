type LocationApi = typeof import("country-state-city")

export type CountryOption = {
  code: string
  name: string
  displayName: string
}

export const CN_STATE_LABELS: Record<string, string> = {
  AH: "安徽", BJ: "北京", CQ: "重庆", FJ: "福建", GS: "甘肃", GD: "广东", GX: "广西", GZ: "贵州",
  HI: "海南", HE: "河北", HL: "黑龙江", HA: "河南", HK: "香港", HB: "湖北", HN: "湖南", NM: "内蒙古",
  JS: "江苏", JX: "江西", JL: "吉林", LN: "辽宁", MO: "澳门", NX: "宁夏", QH: "青海", SN: "陕西",
  SD: "山东", SH: "上海", SX: "山西", SC: "四川", TJ: "天津", XJ: "新疆", XZ: "西藏", YN: "云南",
  ZJ: "浙江", TW: "台湾",
}

export const CN_CITY_LABELS: Record<string, Record<string, string>> = {
  BJ: { Beijing: "北京", Changping: "昌平", Daxing: "大兴", Fangshan: "房山", Liangxiang: "良乡", Mentougou: "门头沟", Shunyi: "顺义", Tongzhou: "通州" },
  SH: { Shanghai: "上海", Baoshan: "宝山", Jiading: "嘉定", Minhang: "闵行", Pudong: "浦东", Songjiang: "松江" },
  GD: { Guangzhou: "广州", Shenzhen: "深圳", Dongguan: "东莞", Foshan: "佛山", Zhuhai: "珠海", Huizhou: "惠州" },
  ZJ: { Hangzhou: "杭州", Ningbo: "宁波", Wenzhou: "温州", Jiaxing: "嘉兴", Shaoxing: "绍兴", Jinhua: "金华" },
  JS: { Nanjing: "南京", Suzhou: "苏州", Wuxi: "无锡", Changzhou: "常州", Nantong: "南通", Xuzhou: "徐州" },
  SC: { Chengdu: "成都", Mianyang: "绵阳", Deyang: "德阳", Leshan: "乐山", Yibin: "宜宾" },
  CQ: { Chongqing: "重庆" },
  TJ: { Tianjin: "天津" },
}

export const CN_CITY_OPTIONS_BY_STATE: Record<string, string[]> = {
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

let locationApi: LocationApi | null = null
let locationApiPromise: Promise<LocationApi> | null = null
let allCountriesCache: any[] = []

const countryOptionsCache = new Map<string, CountryOption[]>()
const provinceOptionsCache = new Map<string, any[]>()
const stateCityOptionsCache = new Map<string, any[]>()
const countryCityOptionsCache = new Map<string, any[]>()

// These city-states have subdivision data in some datasets, but the product
// treats their postal address as country-level only.
const COUNTRY_LEVEL_ONLY_CODES = new Set(["SG", "MC", "VA"])

function rawProvinceOptions(countryCode: string) {
  if (!countryCode || !locationApi) return []

  const cached = provinceOptionsCache.get(countryCode)
  if (cached) return cached

  const options = locationApi.State.getStatesOfCountry(countryCode) || []
  provinceOptionsCache.set(countryCode, options)
  return options
}

export function countryUsesProvinceField(countryCode: string) {
  if (!countryCode || !locationApi) return true
  return !COUNTRY_LEVEL_ONLY_CODES.has(countryCode) && rawProvinceOptions(countryCode).length > 0
}

export function countryUsesCityField(countryCode: string) {
  return countryUsesProvinceField(countryCode)
}

export function normalizeLocationForSubmission(countryCode: string, country: string, province: string, city: string) {
  if (countryCode && !countryUsesProvinceField(countryCode)) {
    const canonicalCountryName = allCountriesCache.find((item) => item.isoCode === countryCode)?.name || country.trim()
    return {
      country: canonicalCountryName,
      province: canonicalCountryName,
      city: canonicalCountryName,
    }
  }
  return {
    country: country.trim(),
    province: province.trim(),
    city: city.trim(),
  }
}

export async function loadLocationData() {
  if (!locationApiPromise) {
    locationApiPromise = import("country-state-city")
      .then((api) => {
        locationApi = api
        allCountriesCache = api.Country.getAllCountries()
        countryOptionsCache.clear()
        return api
      })
      .catch((error) => {
        locationApi = null
        locationApiPromise = null
        allCountriesCache = []
        throw error
      })
  }

  await locationApiPromise
}

export function getCachedCountries() {
  return allCountriesCache
}

export function getCountryOptions(locale: string) {
  if (allCountriesCache.length === 0) return []

  const cached = countryOptionsCache.get(locale)
  if (cached) return cached

  const displayNames = new Intl.DisplayNames([locale], { type: "region" })
  const options = allCountriesCache
    .map((country) => {
      const localizedName = displayNames.of(country.isoCode) || country.name
      const shouldShowEnglishName = locale.toLowerCase().startsWith("zh")
        && country.isoCode !== "CN"
        && localizedName !== country.name
      return {
        code: country.isoCode,
        name: localizedName,
        displayName: shouldShowEnglishName ? `${localizedName} / ${country.name}` : localizedName,
      }
    })
    .sort((a, b) => a.name.localeCompare(b.name, locale))

  countryOptionsCache.set(locale, options)
  return options
}

export function getProvinceOptions(countryCode: string) {
  if (!countryCode || !locationApi) return []
  if (!countryUsesProvinceField(countryCode)) return []
  return rawProvinceOptions(countryCode)
}

export function getStateCityOptions(countryCode: string, provinceCode: string) {
  if (!countryCode || !provinceCode || !locationApi) return []

  const cacheKey = `${countryCode}:${provinceCode}`
  const cached = stateCityOptionsCache.get(cacheKey)
  if (cached) return cached

  const options = locationApi.City.getCitiesOfState(countryCode, provinceCode) || []
  stateCityOptionsCache.set(cacheKey, options)
  return options
}

export function getCountryCityOptions(countryCode: string) {
  if (!countryCode || !locationApi) return []

  const cached = countryCityOptionsCache.get(countryCode)
  if (cached) return cached

  const options = locationApi.City.getCitiesOfCountry(countryCode) || []
  countryCityOptionsCache.set(countryCode, options)
  return options
}
