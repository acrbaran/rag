import DOMPurify from 'dompurify'
import { Marked, Renderer } from 'marked'
import { escapeHTML } from '@/utils/security'

const ANNOUNCEMENT_BUTTON_RE = /(?:^|\n+)\[([^\]\n]+)]\((https?:\/\/[^\s)]+)\)\s*$/i

export function splitAnnouncementBody(body: string) {
  const match = body.match(ANNOUNCEMENT_BUTTON_RE)
  return match
    ? { body: body.slice(0, match.index).trimEnd(), buttonLabel: match[1], buttonUrl: match[2] }
    : { body, buttonLabel: '', buttonUrl: '' }
}

export function isAnnouncementButtonUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

export function withAnnouncementButton(body: string, label: string, url: string): string {
  const text = body.trim()
  const buttonLabel = label.trim().replaceAll('[', '').replaceAll(']', '')
  const buttonUrl = url.trim().replaceAll(')', '%29')
  return buttonLabel && buttonUrl ? `${text}\n\n[${buttonLabel}](${buttonUrl})` : text
}

export function announcementPlainText(body: string): string {
  return body
    .replace(/\[([^\]]+)]\(https?:\/\/[^\s)]+\)/gi, '$1')
    .replace(/(\*\*|__)(.*?)\1/g, '$2')
    .replace(/([*_])(.*?)\1/g, '$2')
}

export function announcementExcerpt(body: string) {
  const excerpt = Array.from(body).slice(0, 240).join('').split('\n').slice(0, 3).join('\n')
  return { excerpt, shortened: excerpt !== body }
}

const HTTP_URL = /^https?:\/\//i

function createRenderer(): Renderer {
  const renderer = new Renderer()
  // Duyuru metni ham HTML içeremez; yazılan etiketler metin olarak gösterilir.
  renderer.html = ({ text }) => escapeHTML(text)
  renderer.image = ({ text }) => escapeHTML(text || '')
  renderer.link = function ({ href, tokens }) {
    const label = this.parser.parseInline(tokens)
    if (!href || !HTTP_URL.test(href)) return `<span>${label}</span>`
    const safeHref = href.replace(/"/g, '&quot;')
    return `<a class="announcement-button" href="${safeHref}" target="_blank" rel="noopener noreferrer">${label}</a>`
  }
  return renderer
}

const announcementMarked = new Marked({ gfm: true, breaks: true, renderer: createRenderer() })

export function renderAnnouncementMarkdown(body: string): string {
  const source = body.trim() || '_Empty note_'
  const html = announcementMarked.parse(source, { async: false }) as string
  return DOMPurify.sanitize(html, {
    ALLOWED_URI_REGEXP: /^https?:\/\//i,
    ADD_ATTR: ['target'],
    FORBID_TAGS: ['style', 'script', 'img', 'iframe', 'form', 'input'],
  })
}
