import { nextTick, reactive, watch, type Ref } from 'vue'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'

type RuleMap = Record<string, FormRule[]>

const hasError = (result: unknown, name: string) =>
  typeof result === 'object' && result !== null && name in result

// TDesign runs rules without an explicit trigger on every keystroke.
// Tagging them with 'submit' keeps the built-in change/blur hooks quiet
// so useDeferredValidation can decide when a field is checked.
export function deferRules<T extends RuleMap>(rules: T): T {
  return Object.fromEntries(
    Object.entries(rules).map(([name, list]) => [name, list.map(rule => ({ ...rule, trigger: 'submit' as const }))]),
  ) as T
}

interface Options {
  // Fields to re-check when another field changes, e.g. password -> confirmPassword.
  dependents?: Record<string, string[]>
}

// Validation timing: a field is first checked when the user leaves it after
// typing, or when the form is submitted. Once a field shows an error it is
// re-checked on every change, so the message disappears as soon as it is fixed.
export function useDeferredValidation(
  formRef: Ref<FormInstanceFunctions | undefined>,
  data: Record<string, unknown>,
  options: Options = {},
) {
  const dirty = reactive(new Set<string>())
  const validated = reactive(new Set<string>())
  const invalid = reactive(new Set<string>())

  const validateField = async (name: string) => {
    const result = await formRef.value?.validate({ fields: [name], trigger: 'submit' })
    validated.add(name)
    if (hasError(result, name)) invalid.add(name)
    else invalid.delete(name)
  }

  const revalidate = (name: string) => {
    if (validated.has(name)) void validateField(name)
  }

  // Bound to the form item's focusout so moving focus inside the same field
  // (password visibility toggle, phone country picker) does not count as leaving it.
  const onFocusOut = (name: string, event: FocusEvent) => {
    const field = event.currentTarget as HTMLElement | null
    if (field?.contains(event.relatedTarget as Node | null)) return
    if (dirty.has(name)) void validateField(name)
  }

  const validateAll = async () => {
    const result = await formRef.value?.validate()
    Object.keys(data).forEach(name => {
      validated.add(name)
      if (hasError(result, name)) invalid.add(name)
      else invalid.delete(name)
    })
    return result === true
  }

  // Call after clearing the form data; waits for the data watcher so the
  // cleared values are not treated as user edits.
  const reset = async () => {
    await nextTick()
    dirty.clear()
    validated.clear()
    invalid.clear()
    formRef.value?.clearValidate()
  }

  watch(
    () => ({ ...data }),
    (next, prev) => {
      Object.keys(next).forEach(name => {
        if (next[name] === prev[name]) return
        dirty.add(name)
        if (invalid.has(name)) void validateField(name)
        options.dependents?.[name]?.forEach(revalidate)
      })
    },
  )

  return { onFocusOut, revalidate, validateAll, reset }
}
