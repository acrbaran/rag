// Backend task type ids (internal/types/task.go) → leaf key under
// system.globalSettings.runtime.tasks.taskTypes. Shared by the runtime
// queue page and the platform audit log.
export const RUNTIME_TASK_TYPE_KEYS: Record<string, string> = {
  'document:process': 'documentProcess',
  'manual:process': 'manualProcess',
  'temporary_document:process': 'temporaryDocumentProcess',
  'knowledge:post_process': 'postProcess',
  'summary:generation': 'summary',
  'datatable:summary': 'tableSummary',
  'knowledge:auto_tag': 'autoTag',
  'kb:profile': 'kbProfile',
  'question:generation': 'question',
  'image:multimodal': 'multimodal',
  'chunk:extract': 'graph',
  'memory:extract': 'memoryExtract',
  'datasource:sync': 'sync',
  'faq:import': 'faqImport',
  'knowledge:list_reparse': 'batchReparse',
  'knowledge:list_delete': 'batchDelete',
  'knowledge:move': 'move',
  'index:delete': 'indexDelete',
  'kb:clone': 'kbClone',
  'kb:delete': 'kbDelete',
  'wiki:ingest': 'wikiIngest',
  'wiki:finalize': 'wikiFinalize',
}
