import assert from 'node:assert/strict'
import test from 'node:test'

import { resolveAttachmentParsingCounts } from './attachmentParsingDisplay'

test('attachment counts remain readable in Turkish, English and older events', () => {
  assert.deepEqual(resolveAttachmentParsingCounts({ output: '3 ek çözümlendi; tamamlanmamış 2 ek atlandı' }), { parsed: 3, skipped: 2 })
  assert.deepEqual(resolveAttachmentParsingCounts({ output: '3 attachments parsed; 2 unfinished attachments skipped' }), { parsed: 3, skipped: 2 })
  assert.deepEqual(resolveAttachmentParsingCounts({ output: '已解析 3 个附件，2 个未完成已跳过' }), { parsed: 3, skipped: 2 })
})
