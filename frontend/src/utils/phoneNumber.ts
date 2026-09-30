import { parsePhoneNumberFromString, type CountryCode } from 'libphonenumber-js/max'

export function normalizePhoneNumber(input: string, country: CountryCode): string | null {
  const value = input.trim()
  const parsed = parsePhoneNumberFromString(value, value.startsWith('+') ? undefined : country)
  return parsed?.isValid() ? parsed.number : null
}
