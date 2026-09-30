import assert from 'node:assert/strict'
import { test } from 'node:test'
import en from '../i18n/locales/en-US'
import tr from '../i18n/locales/tr-TR'
import { agentDisplayDescription, agentDisplayName } from './agent-mode'

test('built-in cards follow the UI language while custom agents keep their names', () => {
  const translate = (messages: typeof en) => (key: string): string =>
    key.split('.').reduce<any>((value, part) => value[part], messages)
  const analyst = { id: 'builtin-data-analyst', name: 'Data Analyst', description: 'English description' }
  const wiki = { id: 'builtin-wiki-researcher', name: 'Wiki Questioner', description: 'English description' }
  const custom = { id: 'my-agent', name: 'My Agent', description: 'My description' }

  assert.equal(agentDisplayName(analyst, translate(tr)), 'Veri Analisti')
  assert.equal(agentDisplayDescription(analyst, translate(tr)), tr.agent.builtinLabels.dataAnalystDesc)
  assert.equal(agentDisplayName(wiki, translate(tr)), 'Wiki Soru-Cevap')
  assert.equal(agentDisplayDescription(wiki, translate(tr)), tr.agent.builtinLabels.wikiQuestionerDesc)
  assert.equal(agentDisplayName(analyst, translate(en)), 'Data Analyst')
  assert.equal(agentDisplayDescription(wiki, translate(en)), en.agent.builtinLabels.wikiQuestionerDesc)
  assert.equal(agentDisplayName(custom, translate(tr)), custom.name)
  assert.equal(agentDisplayDescription(custom, translate(tr)), custom.description)
  assert.equal(agentDisplayName({ ...custom, id: 'toString' }, translate(tr)), custom.name)
})
