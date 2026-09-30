import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  EMBED_MESSAGES,
  normalizeEmbedLocale,
  SUPPORTED_LOCALES,
} from './embed.ts'

const EXPECTED_CONVERSATION_TIME_KEYS = [
  'today',
  'yesterday',
  'thisYear',
  'otherYear',
] as const

const EXPECTED_REFERENCES_DRAWER_KEYS = [
  'referencesDrawerTitle',
  'referencesDrawerTitleWeb',
  'referencesDrawerTitleDocs',
  'referencesDrawerTitleTools',
  'referencesDrawerTitleMixed',
  'referencesDrawerWebSection',
  'referencesDrawerDocsSection',
  'referencesDrawerToolsSection',
  'referencesDrawerEmpty',
] as const

test('supported embed locales are English and Turkish', () => {
  assert.deepEqual([...SUPPORTED_LOCALES], ['en-US', 'tr-TR'])
})

test('every supported locale defines conversationTime and referencesDrawer in chat', () => {
  for (const locale of SUPPORTED_LOCALES) {
    const bundle = EMBED_MESSAGES[locale] as {
      chat?: Record<string, unknown>
      common?: Record<string, unknown>
    }
    assert.ok(bundle, `Locale bundle for ${locale} must exist`)
    assert.ok(bundle.chat, `Locale bundle for ${locale} must have chat section`)

    // conversationTime checks
    const conversationTime = bundle.chat.conversationTime as Record<string, string> | undefined
    assert.ok(
      conversationTime && typeof conversationTime === 'object',
      `Locale ${locale} is missing chat.conversationTime`,
    )
    for (const key of EXPECTED_CONVERSATION_TIME_KEYS) {
      assert.equal(
        typeof conversationTime[key],
        'string',
        `Locale ${locale} is missing chat.conversationTime.${key}`,
      )
      assert.ok(
        conversationTime[key].trim().length > 0,
        `chat.conversationTime.${key} in ${locale} should not be empty`,
      )
      assert.ok(
        conversationTime[key].includes('{time}'),
        `chat.conversationTime.${key} in ${locale} must include {time} placeholder`,
      )
    }

    // referencesDrawer checks
    const chatBag = bundle.chat as Record<string, unknown>
    for (const key of EXPECTED_REFERENCES_DRAWER_KEYS) {
      const val: unknown = chatBag[key]
      assert.equal(
        typeof val,
        'string',
        `Locale ${locale} is missing chat.${key}`,
      )
      assert.ok(
        (val as string).trim().length > 0,
        `chat.${key} in ${locale} should not be empty`,
      )
    }

    // common.close check
    assert.ok(bundle.common, `Locale bundle for ${locale} must have common section`)
    assert.equal(
      typeof bundle.common.close,
      'string',
      `Locale ${locale} is missing common.close`,
    )
    assert.ok(
      (bundle.common.close as string).trim().length > 0,
      `common.close in ${locale} should not be empty`,
    )
  }
})

test('Turkish embed messages are available', () => {
  assert.equal(EMBED_MESSAGES['tr-TR'].common.close, 'Kapat')
  assert.ok(EMBED_MESSAGES['tr-TR'].chat.conversationTime.today)
})

test('normalizeEmbedLocale maps English and defaults unsupported tags to Turkish', () => {
  assert.equal(normalizeEmbedLocale('tr-TR'), 'tr-TR')
  assert.equal(normalizeEmbedLocale('tr'), 'tr-TR')
  assert.equal(normalizeEmbedLocale('EN-gb'), 'en-US')
  assert.equal(normalizeEmbedLocale('zh-CN'), 'tr-TR')
  assert.equal(normalizeEmbedLocale('unknown-locale'), 'tr-TR')
})
