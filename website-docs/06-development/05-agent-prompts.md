# Konuşma istemi birleştirme ve düzenlenebilir kapsam

Bu belge, istemleri, aracı düzenleyicisini ve model mesaj akışını sürdüren geliştiriciler içindir. Aracı yapılandırması ve yürütme akışı için bkz. [Agent motoru](../03-features/07-agent.md).

## Akıllı çıkarımın sistem bölümleri

Tek birleştirme giriş noktası `internal/agent/prompts.go` içindeki `BuildSystemPromptSections` işlevi olup hem motor hem de dize uyumluluk arayüzü `BuildSystemPromptWithOptions` bunu kullanır. Aşağıda gerçek birleştirme sırasıyla listelenmiştir; boş bölümler çıktıda atlanır.

| Bölüm adı | Kaynak ve sorumluluk |
| --- | --- |
| `base` | Açıkça belirtilen gövde önceliklidir; aksi durumda bağlı bilgi tabanı yoksa `pure` varsayılan şablonu, varsa `rag` varsayılan şablonu kullanılır |
| `steering` | Yanıt sürecinde kullanıcının eklediği mesajların nasıl işleneceği (ekleme, değiştirme veya iptal); tamamlanmamış görevler korunur; özel gövde de bunu içerir |
| `runtime_contract` | Veri ve talimat sınırları, geçerli belge kapsamı, tamamlama koşulları; kullanıcının dili biliniyorsa varsayılan yanıt dili eklenir |
| `sources` | Gerçekte kaydedilmiş araç kümesine göre kaynak kurallarını seçer; beceri kurulum modu özel kurulum doğrulama açıklamalarını kullanır |
| `tools` | Genel yürütme, dosya ve sanal alan kuralları; ayrıntılı parametreler yine araç tanımlarındadır |
| `output` | Ortak çıktı biçimi, görsel ekleme koşulları ve tamamlama denetimi |
| `skills` | Kullanılabilir beceri meta verileri; yalnızca beceri kurulum modu dışında, `read_file` mevcut olduğunda ve meta veri varsa eklenir |
| `memory` | Bu turda geri çağrılan bellek |
| `protocol` | Kaynak tanıtıcıları ve çıktı alıntılama protokolü |

Özel gövde yalnızca `base` bölümünü değiştirir; diğer çalışma zamanı bölümlerini kaldırmaz ve araç izinlerini artırmaz. Bölümler, model API'sinde farklı izin katmanları değildir; çağrılabilir araçlar yine arka uç kaydı ve yürütme yollarıyla denetlenir. Tarayıcı işlem sözleşmesi `local_browser` araç açıklamasında tutulur; bağlantı ve kullanım talimatları için bkz. [Yerel tarayıcı](../05-clients/09-local-browser.md).

`base` içindeki yer tutucular `renderPromptPlaceholdersWithStatus` tarafından genişletilir:

| Yer tutucu | Geçerli davranış |
| --- | --- |
| `{{knowledge_bases}}` | Kullanıcı mesajındaki `runtime_context` içindeki bilgi tabanı dizinine işaret eder; tam metni genişletmez |
| `{{web_search_status}}` | Gerçek `web_search` aracının kaydedilip kaydedilmediğine göre `Enabled` / `Disabled` olarak genişletilir |
| `{{current_time}}` | `YYYY-MM-DD` tarihi |
| `{{language}}` | Kullanıcının dil adı |
| `{{skills}}` | Temizlenir; beceri meta verileri ayrı bir bölüm tarafından sağlanır |

## Geçerli tur bağlamı ve ileti rollerı

`internal/agent/observe.go`, geçerli kullanıcı iletisine tarih, oturum ID'si, bağlı bilgi tabanı özeti, sabit belgeler ve soru kaynakları gibi bu tura ait bilgileri içeren `runtime_context` ekler; bunu geçmiş talimatlar olarak kalıcılaştırmaz. Bilgi tabanı adları ve açıklamaları uzunlukla sınırlanır ve kaçırılır; FAQ yanıtları ile tam belge metinleri arama aracıyla okunmalıdır.

`@MCP` / `@Skill`, geçerli tur için `must_use` ipuçları üretir. MCP bahsi yalnızca yetkilendirilmiş hizmetlere öncelik ipucu verir, yapılandırılmış diğer hizmetleri kaldırmaz; henüz sunulmayan hizmetler `discover_mcp_tools` ile keşfedilir. Beceri bahsi, önce ilgili `SKILL.md` dosyasının okunmasını önerir. Bu seçimler ilgisiz işlemlere yetki veremez; yine kullanıcının açık kaynak kısıtlarına ve arka uç izin sınırlamalarına tabidir.

Yanıt sırasında kullanıcının ekleyip hemen eklemeyi seçtiği iletiler, sonraki yinelemeden önce ileti listesinin sonuna user iletisi olarak eklenir: modele gönderilen içerik `<steer_message>` ile sarılır ve `<continue_task>` açıklaması eklenir (`types.SteerMessageContent`); oturum geçmişi ve arayüz kullanıcının özgün metnini korur; `steering` bölümü modelin bu tür iletileri nasıl ele alacağını belirtir.

Normal araç sonuçları `tool` rolünü ve çağrı ID'sini korur; ileti onarımı sırasında eşleştirilemeyen sonuçlar, kaçırılmış `untrusted_tool_result` veri blokları olarak saklanır ve sistem talimatına yükseltilemez. Agent, normal soru-cevap ve model geri dönüşü aynı kaynak verisi sınır kurallarını paylaşır, ancak üçünün de istem birleştirme akışı tamamen aynı değildir.

Yineleme sınırına ulaşıldığında veya hata sonlandırmasında, `internal/agent/finalize.go` geçerli ileti listesini kullanmaya devam eder; geçmişi, görselleri ve araç çağrısı eşleştirmelerini korur, ardından sonlandırma isteğini ekler. Son çağrıda araç sunulmaz, `tool_choice=none` ayarlanır ve thinking kapatılır.

## Düzenleme ve kaydetme

Ön uç düzenleyicisi geçerli gövde metnini gösterir; `frontend/src/utils/agentPromptTemplates.ts`, kaydetme sırasında şablon başvurularını özel içerikten ayırır:

- Gövde, bilinen bir şablonla tamamen aynıysa: `system_prompt_id` / `context_template_id` kaydedilir, gövde boş bırakılır.
- Gövde gerçekten değiştirildiyse: gövde kaydedilir, eski şablon ID'si temizlenir.
- Yeniden yazım, geri dönüş alanları ve geçerli varsayılan şablon aynıysa: boş değer kaydedilir ve varsayılan değer devralınır; niyet istemleri yalnızca varsayılandan sapmış geçersiz kılmaları kaydeder.

Örneğin, değiştirilmemiş Wiki şablonu şu şekilde kaydedilir:

```json
{"agent_mode":"smart-reasoning","system_prompt_id":"wiki_researcher","system_prompt":""}
```

`internal/config/agent_prompts.go` içindeki `ResolveCustomAgentPrompts`, istek sırasında başvuruları çözümler; açık gövde her zaman önceliklidir, başvurular yalnızca ait oldukları alan ve modun şablon kümesinde aranır. Çözümleme sonucu kaydedilen nesneye geri yazılmaz; bilinmeyen başvurular boş gövde döndürür ve çağıran varsayılan yolu kullanır.

Varsayılan şablon güncellemeleri mevcut özel gövdelerin üzerine yazmaz, ayrıca geçmiş varsayılan kopyalarını toplu olarak taşımaz. Eski yapılandırmanın yeniden şablonu izlemesi gerektiğinde, düzenleyicide varsayılanı geri yükleyip kaydedin.

## Bakımda ne denetlenmeli

Yeni kurallar için önce ait olduğu yeri belirleyin: rol ve alan yöntemleri `base` içine, kaynak seçimi `sources` içine, çağrı sözleşmeleri araç tanımına veya `tools` içine, çıktı biçimi `output` içine, başvuru kodlaması `protocol` içine konur. Aynı kuralın birden çok yerde yinelenerek bakımının yapılmasından kaçının.

`[Agent][Prompt] section=... bytes=...` günlüğü her bölümün boyutunu belirlemeye yardımcı olur; davranışı doğrulamak için ayrıca son model iletileri, gerçek araç kayıt defteri ve yürütme izinleri denetlenmeli, kaynak seçimi, hata sonrası toparlanma ve çıktı biçimi görevlerle doğrulanmalıdır.
