type LocationApi = typeof import("country-state-city")

export type CountryOption = {
  code: string
  name: string
  displayName: string
}

let locationApi: LocationApi | null = null
let locationApiPromise: Promise<LocationApi> | null = null
let allCountriesCache: any[] = []

const countryOptionsCache = new Map<string, CountryOption[]>()
const provinceOptionsCache = new Map<string, any[]>()
const stateCityOptionsCache = new Map<string, any[]>()
const countryCityOptionsCache = new Map<string, any[]>()

// Singapore is a city-state; these address forms do not require separate
// State / Province or City values for it.
const COUNTRIES_WITHOUT_PROVINCE_FIELD = new Set(["SG"])
const COUNTRIES_WITHOUT_CITY_FIELD = new Set(["SG"])

export function countryUsesCityField(countryCode: string) {
  return !COUNTRIES_WITHOUT_CITY_FIELD.has(countryCode)
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
  if (COUNTRIES_WITHOUT_PROVINCE_FIELD.has(countryCode)) return []

  const cached = provinceOptionsCache.get(countryCode)
  if (cached) return cached

  const options = locationApi.State.getStatesOfCountry(countryCode) || []
  provinceOptionsCache.set(countryCode, options)
  return options
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
