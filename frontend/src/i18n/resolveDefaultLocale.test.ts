import assert from 'node:assert/strict'
import { test } from 'node:test'

import { BUILT_IN_DEFAULT, resolveDefaultLocale } from './resolveDefaultLocale.ts'

test('resolveDefaultLocale prefers runtime over build-time default', () => {
  assert.equal(resolveDefaultLocale('en-US', 'tr-TR'), 'en-US')
})

test('resolveDefaultLocale falls back to build-time then built-in default', () => {
  assert.equal(resolveDefaultLocale('', 'tr-TR'), 'tr-TR')
  assert.equal(BUILT_IN_DEFAULT, 'tr-TR')
  assert.equal(resolveDefaultLocale(undefined, undefined), 'tr-TR')
})

test('resolveDefaultLocale trims whitespace around supported tags', () => {
  assert.equal(resolveDefaultLocale(' en-US '), 'en-US')
})

test('resolveDefaultLocale rejects unknown values', () => {
  assert.equal(resolveDefaultLocale('fr-FR'), BUILT_IN_DEFAULT)
  assert.equal(resolveDefaultLocale('zh-CN'), BUILT_IN_DEFAULT)
  assert.equal(resolveDefaultLocale('en-US"};alert(1);//'), BUILT_IN_DEFAULT)
  assert.equal(resolveDefaultLocale('   '), BUILT_IN_DEFAULT)
})
