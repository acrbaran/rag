import { strict as assert } from 'node:assert'
import { test } from 'node:test'
import { normalizePhoneNumber } from './phoneNumber'

test('normalizes local and international phone numbers for registration', () => {
  assert.equal(normalizePhoneNumber('0555 123 45 67', 'TR'), '+905551234567')
  assert.equal(normalizePhoneNumber('+1 650 253 0000', 'TR'), '+16502530000')
  assert.equal(normalizePhoneNumber('123', 'TR'), null)
})
