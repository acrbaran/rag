import assert from 'node:assert/strict'
import test from 'node:test'

import { normalizeAPIKeyKnowledgeBaseIDs } from './apiKeyScope.ts'

/**
 * Tam yetkili Key icin null/undefined kapsamini bos diziye donusturerek, liste render edilirken length okunmasindan kaynaklanan beyaz ekrani onle.
 * Sunucunun donebilecegi bos degeri aktar; tum bilgi tabanlarini ifade eden bos bir dizi donmesi beklenir.
 */
test('normalizes missing API key knowledge base scope to an empty array', () => {
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs(null), [])
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs(undefined), [])
})

/**
 * scoped Key bilgi tabani ID'lerinin eksiksiz kopyalandigini ve duzenleme formunun liste verisini kirletmesini onlemek icin donen degerin kaynak dizi olmadigini dogrula.
 * Iki bilgi tabani ID'si aktar; ayni sirada yeni bir dizi donmesi beklenir.
 */
test('copies configured API key knowledge base scope', () => {
  const ids = ['kb-1', 'kb-2']
  const normalized = normalizeAPIKeyKnowledgeBaseIDs(ids)

  assert.deepEqual(normalized, ids)
  assert.notEqual(normalized, ids)
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs([]), [])
})
