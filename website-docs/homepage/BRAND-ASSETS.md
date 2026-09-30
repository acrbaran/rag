# Brand asset sources

Logos identify supported integrations; their names and marks belong to their respective owners.

Paths below written as `public/brand/…`, `public/brands/…` and `public/product/…` live under `public/docs/_home/`, so the site serves them from `/docs/_home/` next to the rest of the documentation.

- Rethra logo: `frontend/src/assets/img/rethra-logo.png`, copied to `docs/images/logo.png` and `public/brand/rethra-original.png`. The transparent wordmark is displayed at its native proportions.
- Rethra browser icon: generated from `frontend/src/assets/img/rethra-icon.png` and copied to `frontend/public/favicon.ico` and `website-docs/public/favicon.ico`.
- Feishu, GitLab, Tencent IMA, Notion, Yuque, RSS, Slack, Telegram: copied unchanged from the Rethra frontend asset library.
- OpenAI, DeepSeek, Qwen, Hunyuan, Gemini, MCP: static SVGs from [Lobe Icons](https://github.com/lobehub/lobe-icons), `@lobehub/icons-static-svg@1.95.0`. Upstream MIT license included with assets.
- Confluence, DingTalk: copied unchanged from the Rethra frontend asset library (`frontend/src/assets/img/datasource-confluence.svg`, `website-docs/homepage/public/docs/_home/brands/dingtalk.svg`).
- Claude (Anthropic), Zhipu, Kimi (Moonshot), Volcengine, MiniMax: copied unchanged from `internal/models/providers/assets/`, which carries the same Lobe Icons static SVGs; covered by the included Lobe Icons MIT license.
- BrowserSkill: `frontend/src/assets/browserskill/logo.png` from the Rethra frontend (the extension's own icon, [Tencent/BrowserSkill](https://github.com/Tencent/BrowserSkill)), copied unchanged to `public/brands/browserskill.png` with its MIT license as `public/brands/BROWSERSKILL-LICENSE`.
- Chrome: [official Chrome website asset](https://www.google.com/chrome/static/images/chrome-logo-m100.svg).
- File formats, REST API, CLI: generic interface icons, not brand logos.
- LiteLLM: [official documentation favicon](https://docs.litellm.ai/img/favicon.ico).
- WeChat Dialog Open Platform: [official platform icon](https://res.wx.qq.com/mmspraiweb_node/dist/static/logo/logo180.png), referenced by the platform login page and saved unchanged as `public/brands/wechat-dialog.png`. The platform icon retains its original colors in light and dark mode.

## Product screenshots

- Skill catalog and sandbox terminal: copied unchanged from `../public/screenshots/skill-catalog.png` and `sandbox-panel-terminal.png` to `public/product/skill-catalog.png` and `sandbox-terminal.png`, for the skills & sandbox gallery.
- Sandbox masaustu (`public/product/sandbox-desktop.png`): sohbetin masaustu sekmesindeki sandbox paneli, bir masaustu şablonunun XFCE masaustunu gösterir. Terminal ekran görüntüsüyle terminal ve grafik masaustu kartını paylaşır; kart içinde geçiş yapılır.
- v0.8.2 galerisi: `mcp-server-endpoint.png`, `chat-steer-queue.png` ve `browser-connection.png`, `../public/screenshots/` konumundan `public/product/` konumuna değiştirilmeden kopyalanmıştır. Yerel tarayıcı kartı, aynı yeteneğin iki görünümü arasında geçiş yapar: görev (aşağıda) ve bağlantı (tarayıcı bağlantı sayfası).
- Browser task (`public/product/local-browser-task.png`): the default view of the local browser card, a smart-reasoning chat driving the connected Chrome through BrowserSkill (GitHub release lookup, then filling the httpbin.org sample order form without submitting), with the in-chat task preview and pause / end controls.

- Wiki browser: copied unchanged from `../public/screenshots/wiki-browser.png` to `public/product/wiki-browser.png`. Shows the actual Wiki directory, linked page and source references using sample company policies.
- Wiki graph and revision history: copied unchanged from `../public/screenshots/wiki-graph.png` and `wiki-revision-history.png` to matching files in `public/product/`, for the horizontal Wiki gallery.
