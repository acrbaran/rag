<template>
  <section class="conversation-feedback" :class="{ 'is-open': formOpen, 'is-collapsed': collapsed, 'is-ready': motionReady }" :style="{ '--feedback-form-height': formHeight + 'px' }" aria-label="Sohbet geri bildirimi">
    <!-- One surface card (background, rounded top, shadow) slides up from
         behind the bar; the form rides on it with the same transform, so the
         outline, the shadow and the content move as one piece. -->
    <div class="feedback-surface" aria-hidden="true" />
    <div class="feedback-note">
      <form id="feedback-comment-form" ref="formRef" class="feedback-form" :inert="!formOpen" @submit.prevent="submitComment" @keydown.esc.stop="cancelComment">
        <div class="feedback-types" role="group" aria-label="Geri bildirim türü">
          <button type="button" :aria-pressed="category === 'suggestion'" :class="{ selected: category === 'suggestion' }" @click="category = 'suggestion'"><t-icon name="lightbulb" aria-hidden="true" />Öneri</button>
          <button type="button" :aria-pressed="category === 'complaint'" :class="{ selected: category === 'complaint' }" @click="category = 'complaint'"><t-icon name="error-circle" aria-hidden="true" />Şikâyet</button>
        </div>
        <label for="feedback-comment">Yorumunuz</label>
        <textarea id="feedback-comment" ref="commentInput" v-model="comment" maxlength="2000" rows="3" placeholder="Neyin iyi gittiğini veya sorunu anlatın..." />
        <div class="feedback-meta"><span>Çalışma alanı yöneticileri bu geri bildirimi ve ilgili sohbeti inceleyebilir.</span><span>{{ comment.length }}/2000</span></div>
        <p v-if="error && formOpen" role="alert" class="feedback-error">{{ error }}</p>
        <div class="feedback-actions"><button type="button" @click="cancelComment">Vazgeç</button><button class="send" type="submit" :disabled="busy || !category || !comment.trim()">Geri bildirimi gönder</button></div>
      </form>
    </div>
    <div class="feedback-bar">
      <Transition name="feedback-bar-content" @after-enter="restoreBarFocus">
      <div v-if="collapsed" key="collapsed" class="feedback-bar-content"><button ref="reopenButton" class="reopen" type="button" @click="setCollapsed(false)"><span>Geri bildirim ver</span><t-icon name="chevron-up" size="14px" aria-hidden="true" /></button></div>
      <div v-else key="open" class="feedback-bar-content">
        <span class="feedback-question">Bu sohbet faydalı oldu mu?</span>
        <button type="button" aria-label="Faydalı" :aria-pressed="feedback?.helpful === true" :disabled="busy" @click="rate(true)"><t-icon name="thumb-up" size="14px" /></button>
        <button type="button" class="rate-down" aria-label="Faydalı değil" :aria-pressed="feedback?.helpful === false" :disabled="busy" @click="rate(false)"><t-icon name="thumb-down" size="14px" /></button>
        <button ref="commentTrigger" type="button" aria-label="Yorum ekle" :aria-expanded="expanded" aria-controls="feedback-comment-form" :disabled="busy" @click="expanded = !expanded"><t-icon name="chat-add" size="14px" /></button>
        <button type="button" aria-label="Geri bildirimi kapat" @click="setCollapsed(true)"><t-icon name="close" size="14px" /></button>
      </div>
      </Transition>
    </div>
    <p v-if="error && !formOpen" role="alert" class="feedback-error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { getFeedback, saveFeedback, type Feedback } from '@/api/feedback'

const props = defineProps<{ sessionId: string }>()
const feedback = ref<Feedback | null>(null)
const expanded = ref(false)
const collapsed = ref(false)
const formOpen = computed(() => expanded.value && !collapsed.value)
const category = ref<'suggestion' | 'complaint' | null>(null)
const comment = ref('')
const commentInput = ref<HTMLTextAreaElement | null>(null)
const formRef = ref<HTMLFormElement | null>(null)
const formHeight = ref(0)
// Transitions stay off until the form has been measured and painted once, so
// the card never animates from an unmeasured position on first render.
const motionReady = ref(false)
let formObserver: ResizeObserver | undefined
let readyFrame = 0
onMounted(() => {
  const form = formRef.value
  if (!form) return
  const measure = () => { formHeight.value = form.offsetHeight }
  measure()
  formObserver = new ResizeObserver(measure)
  formObserver.observe(form)
  readyFrame = requestAnimationFrame(() => { readyFrame = requestAnimationFrame(() => { motionReady.value = true }) })
})
onUnmounted(() => { formObserver?.disconnect(); cancelAnimationFrame(readyFrame) })
const commentTrigger = ref<HTMLButtonElement | null>(null)
const reopenButton = ref<HTMLButtonElement | null>(null)
let focusBarAfterSwap = false
function setCollapsed(value: boolean) {
  expanded.value = false
  collapsed.value = value
  focusBarAfterSwap = true
}
function restoreBarFocus() {
  if (!focusBarAfterSwap) return
  focusBarAfterSwap = false
  ;(collapsed.value ? reopenButton.value : commentTrigger.value)?.focus({ preventScroll: true })
}
function closeComment() { expanded.value = false; commentTrigger.value?.focus() }
function cancelComment() {
  category.value = feedback.value?.category ?? null
  comment.value = feedback.value?.comment ?? ''
  closeComment()
}
const busy = ref(false)
const error = ref('')
watch(expanded, async (open) => { if (open) { await nextTick(); commentInput.value?.focus({ preventScroll: true }) } })

watch(() => props.sessionId, async (id, _old, onCleanup) => {
  let active = true
  onCleanup(() => { active = false })
  expanded.value = false
  collapsed.value = false
  feedback.value = null
  category.value = null
  comment.value = ''
  error.value = ''
  if (!id) return
  try {
    const res = await getFeedback(id)
    if (!active) return
    feedback.value = res.data ?? null
    category.value = feedback.value?.category ?? null
    comment.value = feedback.value?.comment ?? ''
  } catch {
    if (active) error.value = 'Geri bildirim yüklenemedi.'
  }
}, { immediate: true })

async function rate(helpful: boolean) {
  if (busy.value) return
  const sessionId = props.sessionId
  busy.value = true
  error.value = ''
  try {
    const res = await saveFeedback(sessionId, { helpful: feedback.value?.helpful === helpful ? null : helpful })
    if (props.sessionId === sessionId) feedback.value = res.data
  } catch { if (props.sessionId === sessionId) error.value = 'Geri bildirim kaydedilemedi.' }
  finally { busy.value = false }
}

async function submitComment() {
  if (busy.value || !category.value || !comment.value.trim()) return
  const sessionId = props.sessionId
  busy.value = true
  error.value = ''
  try {
    const res = await saveFeedback(sessionId, { category: category.value, comment: comment.value.trim() })
    if (props.sessionId !== sessionId) return
    feedback.value = res.data
    closeComment()
    MessagePlugin.success('Geri bildiriminiz gönderildi.')
  } catch { if (props.sessionId === sessionId) error.value = 'Geri bildirim gönderilemedi.' }
  finally { busy.value = false }
}
</script>

<style scoped>
/*
 * One joined surface. The section keeps a fixed in-flow height (the bar), so
 * opening the form never resizes the composer or moves the messages.
 *
 * .feedback-surface is the visible card: composer background, rounded top and
 * shadow. It is as tall as form + bar and, when closed, is pushed down by the
 * form height so only its bar-sized top shows; the section's clip-path cuts
 * the rest at the seam with the composer. Opening slides the card up with
 * transform only. The form sits in .feedback-note above the bar, is clipped at
 * the bar's top edge and uses the very same transform, duration and curve, so
 * the rounded outline, shadow and content rise and fall together.
 */
.conversation-feedback {
  --feedback-radius: var(--app-radius-composer, 32px);
  --feedback-surface-bg: var(--rethra-surface, var(--td-bg-color-container));
  --feedback-divider: color-mix(in srgb, var(--rethra-border, var(--td-component-stroke)) 70%, transparent);
  --feedback-ease: cubic-bezier(.32, .72, 0, 1);
  --feedback-ease-close: cubic-bezier(.4, 0, .6, 1);
  --feedback-ease-out: cubic-bezier(.22, 1, .36, 1);
  --feedback-open: 420ms;
  --feedback-close: 280ms;
  --feedback-shift: var(--feedback-form-height, 0px);
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: var(--rethra-column, 960px);
  box-sizing: border-box;
  margin: 0 auto;
  /* Keep everything above the seam (card, form, side shadows) and cut below it. */
  clip-path: inset(-100vh -24px 0 -24px);
  color: var(--td-text-color-primary);
}

.feedback-surface {
  position: absolute;
  top: calc(-1 * var(--feedback-shift));
  right: 0;
  bottom: 0;
  left: 0;
  border-radius: var(--feedback-radius) var(--feedback-radius) 0 0;
  background: var(--feedback-surface-bg);
  box-shadow: var(--rethra-composer-shadow, 0 2px 8px -2px rgba(0, 0, 0, .16));
  pointer-events: none;
}

.feedback-note {
  position: absolute;
  right: 0;
  bottom: 100%;
  left: 0;
  overflow: clip;
  border-radius: var(--feedback-radius) var(--feedback-radius) 0 0;
  pointer-events: none;
}

.feedback-surface,
.feedback-form {
  transform: translate3d(0, var(--feedback-shift), 0);
  transition: transform var(--feedback-close) var(--feedback-ease-close);
}
.is-open .feedback-surface,
.is-open .feedback-form {
  transform: translate3d(0, 0, 0);
  transition: transform var(--feedback-open) var(--feedback-ease);
}

.feedback-form {
  display: grid;
  gap: 10px;
  box-sizing: border-box;
  padding: 14px 16px 16px;
  opacity: 0;
  visibility: hidden;
  transition:
    transform var(--feedback-close) var(--feedback-ease-close),
    opacity var(--feedback-close) var(--feedback-ease-close),
    visibility 0s linear var(--feedback-close);
}
.is-open .feedback-form {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
  transition:
    transform var(--feedback-open) var(--feedback-ease),
    opacity var(--feedback-open) var(--feedback-ease),
    visibility 0s linear 0s;
}

.feedback-bar {
  position: relative;
  z-index: 1;
  display: grid;
  padding: 0 14px 0 16px;
}
.feedback-bar::before,
.feedback-bar::after {
  position: absolute;
  right: 0;
  left: 0;
  height: 1px;
  background: var(--feedback-divider);
  content: '';
}
.feedback-bar::after { bottom: 0; }
.feedback-bar::before { top: 0; opacity: 0; transition: opacity var(--feedback-close) var(--feedback-ease-close); }
.is-open .feedback-bar::before { opacity: 1; transition: opacity var(--feedback-open) var(--feedback-ease); }

/* Both bar states share one grid cell and crossfade in place. */
.feedback-bar-content { display: flex; grid-area: 1 / 1; width: 100%; min-width: 0; min-height: 40px; align-items: center; gap: 4px; }
.feedback-bar-content-enter-active { transition: opacity 220ms var(--feedback-ease-out) 60ms; }
.feedback-bar-content-leave-active { transition: opacity 140ms ease-in; pointer-events: none; }
.feedback-bar-content-enter-from,
.feedback-bar-content-leave-to { opacity: 0; }

.feedback-question { flex: 1; min-width: 0; overflow: hidden; color: var(--rethra-muted-fg, var(--td-text-color-placeholder)); font-family: var(--rethra-font, inherit); font-size: 12.5px; font-weight: 400; letter-spacing: .01em; line-height: 1.4; text-overflow: ellipsis; white-space: nowrap; }
button { border: 0; background: transparent; color: inherit; cursor: pointer; font: inherit; }
.feedback-bar button { display: grid; width: 32px; height: 32px; flex: 0 0 32px; place-items: center; border-radius: 50%; color: var(--td-text-color-secondary); transition: color 150ms ease, background-color 150ms ease, transform 150ms var(--feedback-ease-out); }
.feedback-bar button:hover { color: var(--td-text-color-primary); background: var(--td-bg-color-container-hover); }
.feedback-bar button:active:not(:disabled) { transform: scale(.92); }
.feedback-bar button[aria-pressed=true] { color: var(--td-brand-color); }
.feedback-bar .rate-down[aria-pressed=true] { color: var(--td-error-color-6); }
.feedback-bar button[aria-expanded=true] { color: var(--td-text-color-primary); background: var(--td-bg-color-container-hover); }
.feedback-bar .reopen { display: flex; width: 100%; height: 32px; flex: 1; align-items: center; justify-content: space-between; padding: 0 6px; border-radius: 8px; font-size: 13px; }
.feedback-bar .reopen:hover { background: transparent; }
.feedback-bar .reopen:active:not(:disabled) { transform: none; }
.feedback-bar .reopen .t-icon { transition: transform 200ms var(--feedback-ease-out); }
.feedback-bar .reopen:hover .t-icon { transform: translateY(-2px); }
button:focus-visible, textarea:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }
button:disabled { opacity: .55; cursor: not-allowed; }

.feedback-types { display: flex; width: min(100%, 208px); height: 32px; box-sizing: border-box; margin: 0 auto 2px; padding: 3px; border-radius: 99px; background: var(--td-bg-color-container-hover); }
.feedback-types button { display: inline-flex; min-width: 0; flex: 1; align-items: center; justify-content: center; gap: 6px; padding: 0 8px; border-radius: 99px; color: var(--td-text-color-secondary); font-size: 13px; transition: color 150ms ease, background-color 150ms ease, box-shadow 150ms ease; }
.feedback-types button:first-child .t-icon { color: #059669; }
.feedback-types button:nth-child(2) .t-icon { color: #e11d48; }
.feedback-types .selected { color: var(--td-text-color-primary); background: var(--td-bg-color-container); box-shadow: 0 1px 4px #0002; }
.feedback-form label { font-size: 13px; font-weight: 500; }
.feedback-form textarea { box-sizing: border-box; width: 100%; height: 80px; min-height: 80px; resize: vertical; padding: 12px 14px; border: 1px solid var(--td-component-stroke); border-radius: 20px; background: var(--feedback-surface-bg); color: inherit; font: inherit; font-size: 14px; transition: border-color 150ms ease, box-shadow 150ms ease; }
.feedback-form textarea:focus-visible { outline: none; border-color: #9dcdbd; box-shadow: 0 0 0 1px #9dcdbd; }
.feedback-meta { display: flex; justify-content: space-between; gap: 12px; color: var(--td-text-color-secondary); font-size: 12px; }
.feedback-meta span:last-child { flex: none; }
.feedback-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 2px; }
.feedback-actions button { min-height: 32px; padding: 6px 14px; border-radius: 99px; transition: background-color 150ms ease, color 150ms ease; }
.feedback-actions button:not(.send):hover { background: var(--td-bg-color-container-hover); }
.feedback-actions .send { color: #fff; background: #10b981; }
.feedback-actions .send:hover:not(:disabled) { background: #059669; }
.feedback-actions .send:disabled { color: #fff; background: #9cddc8; opacity: 1; }
.feedback-error { margin: 0; padding: 0 16px 8px; color: var(--td-error-color-6); font-size: 12px; }
.feedback-form .feedback-error { padding: 0; }
@media (max-width: 600px) { .feedback-form { padding: 12px; } .feedback-meta { font-size: 11px; } .feedback-question { font-size: 12px; } .feedback-bar { gap: 2px; padding: 0 8px; } }
.conversation-feedback:not(.is-ready) .feedback-surface,
.conversation-feedback:not(.is-ready) .feedback-form { transition: none; }
@media (prefers-reduced-motion: reduce) {
  .feedback-surface, .is-open .feedback-surface, .feedback-form, .is-open .feedback-form,
  .feedback-bar::before, .is-open .feedback-bar::before, .feedback-bar button, .feedback-bar .reopen .t-icon,
  .feedback-bar-content-enter-active, .feedback-bar-content-leave-active { transition: none; }
}
</style>
