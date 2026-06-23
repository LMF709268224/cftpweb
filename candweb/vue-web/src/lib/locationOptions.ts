type LocationApi = typeof import("country-state-city")

export type CountryOption = {
  code: string
  name: string
}

let locationApi: LocationApi | null = null
let locationApiPromise: Promise<LocationApi> | null = null
let allCountriesCache: any[] = []

const countryOptionsCache = new Map<string, CountryOption[]>()
const provinceOptionsCache = new Map<string, any[]>()
const stateCityOptionsCache = new Map<string, any[]>()
const countryCityOptionsCache = new Map<string, any[]>()

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
    .map((country) => ({ code: country.isoCode, name: displayNames.of(country.isoCode) || country.name }))
    .sort((a, b) => a.name.localeCompare(b.name, locale))

  countryOptionsCache.set(locale, options)
  return options
}

export function getProvinceOptions(countryCode: string) {
  if (!countryCode || !locationApi) return []

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
