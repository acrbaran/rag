import { readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig, type DefaultTheme } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'
import './theme-config.d.ts'
import { repoVersionLabel } from './version'

const root = resolve(import.meta.dirname, '..')

const sections: { dir: string; label: string; newestFirst?: boolean }[] = [
  { dir: '01-getting-started', label: 'Hızlı başlangıç' },
  { dir: '02-architecture', label: 'Mimari' },
  { dir: '03-features', label: 'Özellik modülleri' },
  { dir: '04-api', label: 'API referansı' },
  { dir: '05-clients', label: 'İstemci' },
  { dir: '06-development', label: 'Geliştirme kılavuzu' },
  // Her sürüm için bir sayfa (v0.8.2.md); en yeni sürüm en başta yer alır
  { dir: '07-releases', label: 'Sürüm yayınları', newestFirst: true },
]

/** Kenar çubuğu öğe metni: ana metindeki birinci seviye başlığı al, gereksiz önek ve sonekleri kaldır */
function itemText(dir: string, file: string): string {
  const raw = readFileSync(resolve(root, dir, file), 'utf-8')
  const heading = raw.match(/^#\s+(.+)$/m)?.[1] ?? file.replace(/\.md$/, '')
  return heading
    .replace(/^API (?:başvurusu|referansı)\s*[:：]\s*/i, '')
    .replace(/\s*[（(][^（()）]*[)）]\s*/g, ' ')
    .replace(/\s{2,}/g, ' ')
    .trim()
}

/** Sürüm numaralarını sayısal olarak karşılaştır; v0.10.0, v0.9.0'dan sonra gelir */
function compareVersions(a: string, b: string): number {
  const parts = (f: string) => f.replace(/^v|\.md$/g, '').split('.').map(Number)
  const [x, y] = [parts(a), parts(b)]
  for (let i = 0; i < Math.max(x.length, y.length); i++) {
    if ((x[i] ?? 0) !== (y[i] ?? 0)) return (x[i] ?? 0) - (y[i] ?? 0)
  }
  return 0
}

function itemsOf(dir: string, newestFirst = false): DefaultTheme.SidebarItem[] {
  const files = readdirSync(resolve(root, dir)).filter((f) => f.endsWith('.md'))
  return (newestFirst ? files.sort(compareVersions).reverse() : files.sort())
    .map((f) => ({
      text: itemText(dir, f),
      link: `/${dir}/${f.replace(/\.md$/, '')}`,
    }))
}

const sidebar: DefaultTheme.SidebarItem[] = sections.map((s) => ({
  text: s.label,
  collapsed: false,
  items: itemsOf(s.dir, s.newestFirst),
}))

/** Yerel arama varsayılan olarak boşluklara göre sözcüklere ayırır; boşluksuz CJK (Çince) metin tek sözcük sayılacağından karakter düzeyinde bölmeye geri düşülür. Ayırıcı listesi Türkçe/Latin ve tam genişlikli CJK noktalamasını kapsar. */
function tokenize(text: string): string[] {
  const tokens: string[] = []
  for (const part of text.split(/[\s\n\r#%*,=/:;?[\]{}()&+\-!'"$·、，。：；？！（）【】《》…—]+/)) {
    if (!part) continue
    if (/[\u4e00-\u9fa5]/.test(part)) {
      tokens.push(part, ...part.split(''))
    } else {
      tokens.push(part)
    }
  }
  return tokens
}

const repo = 'https://github.com/acrbaran/rag'

export default withMermaid(
  defineConfig({
    title: 'Rethra',
    titleTemplate: ':title · Rethra Belgeleri',
    description: 'Rethra resmi belgeleri: dağıtım, yapılandırma, özellik açıklamaları, API referansı ve ikincil geliştirme',
    lang: 'tr-TR',
    base: '/docs/',
    cleanUrls: true,
    appearance: { storageKey: 'vitepress-theme-appearance' },
    lastUpdated: true,
    srcExclude: ['README.md', 'MIGRATION.md', 'homepage/**', 'shared/**', 'scripts/**', 'deploy/**', 'static-site/**', 'releases/**'],
    metaChunk: true,
    transformPageData(pageData) {
      // The shared masthead replaces the default documentation navbar.
      pageData.frontmatter.navbar = false
    },

    head: [
      ['link', { rel: 'icon', href: '/docs/favicon.ico', type: 'image/x-icon' }],
      ['meta', { name: 'theme-color', content: '#101f38' }],
      ['meta', { property: 'og:type', content: 'website' }],
      ['meta', { property: 'og:title', content: 'Rethra Belgeleri' }],
      [
        'meta',
        {
          property: 'og:description',
          content: 'PDF, Word, web sayfaları ve Feishu / Notion / Yuque kaynaklarını bilgi tabanına ekleyin; kaynak gösteren yanıtlarla bir soru-cevap sistemi oluşturun',
        },
      ],
    ],

    markdown: {
      theme: { light: 'github-light', dark: 'github-dark' },
      lineNumbers: false,
      toc: { level: [2, 3] },
      config(md) {
        // Tablonun dışını kaydırılabilir bir kapsayıcıyla sar. Varsayılan tema bunu yapmak için <table> öğesinin kendisini display:block olarak ayarlar
        // yatay kaydırma; yan etkisi, tablonun içeriğe göre daralmasıdır—az sütunlu tablolar ana metin sütunundan belirgin biçimde daha dar olur ve sayfadaki genişlikler tutarsızlaşır.
        // Kaydırma sarmalayıcıya devredildikten sonra tablolar display:table + width:100% değerine dönebilir ve metin sütununu tamamen doldurur.
        md.renderer.rules.table_open = () => '<div class="wk-table">\n<table>\n'
        md.renderer.rules.table_close = () => '</table>\n</div>\n'
      },
    },

    themeConfig: {
      logoLink: { link: '/', target: '_self' },
      siteTitle: 'Rethra',

      nav: [],

      rethraVersion: repoVersionLabel,

      sidebar,

      socialLinks: [{ icon: 'github', link: repo }],

      outline: { level: [2, 3], label: 'Bu sayfanın içeriği' },

      docFooter: { prev: 'Önceki', next: 'Sonraki' },
      returnToTopLabel: 'Başa dön',
      sidebarMenuLabel: 'İçindekiler',
      darkModeSwitchLabel: 'Görünüm',
      lightModeSwitchTitle: 'Açık temaya geç',
      darkModeSwitchTitle: 'Koyu temaya geç',

      lastUpdated: {
        text: 'Son güncelleme',
        formatOptions: { dateStyle: 'medium', timeStyle: undefined },
      },

      editLink: {
        pattern: `${repo}/edit/main/website-docs/:path`,
        text: 'Bu sayfayı GitHub üzerinde düzenle',
      },

      search: {
        provider: 'local',
        options: {
          translations: {
            button: { buttonText: 'Belgelerde ara', buttonAriaLabel: 'Belgelerde ara' },
            modal: {
              displayDetails: 'Ayrıntıları genişlet',
              resetButtonTitle: 'Temizle',
              backButtonTitle: 'Geri',
              noResultsText: 'Sonuç bulunamadı',
              footer: {
                selectText: 'Seç',
                navigateText: 'Değiştir',
                closeText: 'Kapat',
              },
            },
          },
          miniSearch: {
            options: { tokenize },
            searchOptions: {
              fuzzy: 0.2,
              prefix: true,
              boost: { title: 4, text: 2, titles: 1 },
            },
          },
        },
      },

      footer: {
        message: `Rethra ${repoVersionLabel} kaynak kodu temel alınarak derlenmiştir · MIT License`,
        copyright: '© Rethra',
      },
    },

    mermaid: {
      theme: 'base',
      fontFamily:
        '"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", ui-sans-serif, sans-serif',
      themeVariables: {
        primaryColor: '#eef1f6',
        primaryTextColor: '#101f38',
        primaryBorderColor: '#9aa8bd',
        lineColor: '#7d8ba1',
        secondaryColor: '#faf7ef',
        tertiaryColor: '#f6f7f9',
        fontSize: '14px',
      },
    },
  }),
)
