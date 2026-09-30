const BUILTIN_QUICK_ANSWER_ID = 'builtin-quick-answer'
const BUILTIN_SMART_REASONING_ID = 'builtin-smart-reasoning'

const builtinLabels: Record<string, [string, string]> = {
  [BUILTIN_QUICK_ANSWER_ID]: ['input.normalMode', 'input.normalModeDesc'],
  [BUILTIN_SMART_REASONING_ID]: ['input.agentMode', 'input.agentModeDesc'],
  'builtin-data-analyst': ['agent.builtinLabels.dataAnalyst', 'agent.builtinLabels.dataAnalystDesc'],
  'builtin-wiki-researcher': ['agent.builtinLabels.wikiQuestioner', 'agent.builtinLabels.wikiQuestionerDesc'],
  'builtin-wiki-fixer': ['agent.builtinLabels.wikiFixer', 'agent.builtinLabels.wikiFixerDesc'],
  'builtin-skill-installer': ['agent.builtinLabels.skillInstaller', 'agent.builtinLabels.skillInstallerDesc'],
}

export function agentDisplayName(agent: { id: string; name?: string }, t: (key: string) => string): string {
  return Object.hasOwn(builtinLabels, agent.id) ? t(builtinLabels[agent.id][0]) : agent.name || ''
}

export function agentDisplayDescription(agent: { id: string; description?: string }, t: (key: string) => string): string {
  return Object.hasOwn(builtinLabels, agent.id) ? t(builtinLabels[agent.id][1]) : agent.description || ''
}

/** Whether the selected agent id is the builtin quick-answer (RAG) mode. */
export function isQuickAnswerAgentId(agentId: string | null | undefined): boolean {
  return (agentId || BUILTIN_QUICK_ANSWER_ID) === BUILTIN_QUICK_ANSWER_ID
}

/** Whether requests should use the Agent stream pipeline (not quick-answer RAG). */
export function isAgentStreamAgentId(
  agentId: string | null | undefined,
  isAgentEnabled: boolean,
): boolean {
  const id = agentId || BUILTIN_QUICK_ANSWER_ID
  if (id === BUILTIN_QUICK_ANSWER_ID) return false
  if (id === BUILTIN_SMART_REASONING_ID) return true
  return isAgentEnabled
}

/** Reconcile builtin agent id with isAgentEnabled after localStorage reload. */
export function reconcileBuiltinAgentMode(settings: {
  selectedAgentId?: string
  isAgentEnabled: boolean
}): boolean {
  const agentId = settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID
  if (agentId === BUILTIN_QUICK_ANSWER_ID && settings.isAgentEnabled) {
    settings.isAgentEnabled = false
    return true
  }
  if (agentId === BUILTIN_SMART_REASONING_ID && !settings.isAgentEnabled) {
    settings.isAgentEnabled = true
    return true
  }
  return false
}
