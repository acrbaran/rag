# API referansı: korumalı alan, beceriler ve kişisel değişkenler

Korumalı alan yapılandırmalarını, beceri kataloglarını, kurulum görevlerini, oturum etkileşimli terminallerini ve kişisel ortam değişkenlerini yönetin. Tüm yollar `/api/v1` önekiyle başlar; örneklerdeki `$BASE` hizmet adresi, `$TOKEN` ise mevcut kullanıcının Bearer belirtecidir. Arayüz işlem adımları için bkz. [Beceri kataloğu ve korumalı alan](../03-features/22-skills-sandbox.md).

## Yetkiler

| Kaynak | Okuma | Yazma / denetleme |
| --- | --- | --- |
| Korumalı alan yapılandırma listesi/ayrıntısı | Viewer+; API Key full-access olmalıdır | Admin+; API Key full-access olmalıdır |
| Korumalı alan örnekleri listesi, yüklü beceriler ve dosyalar | Admin+ | Admin+; API Key full-access olmalıdır |
| Beceri kataloğu listesi | Viewer+, JWT | Katalog ekleme/silme, kurulum ve paket dosyası gezintisi Admin+ gerektirir; yazma grubunun API Key'i full-access olmalıdır |
| `/skills` kullanılabilir becerileri | Viewer+, JWT | Salt okunur, sorgu parametresi sandbox_config_id; paylaşılan ajan için ayrıca agent_id, agent_source_tenant_id gönderilir |
| Oturum etkileşimli terminali | Oturum sahibi, yalnızca oturum açılmış JWT | Belirteç alındıktan sonra WebSocket kurulur |
| `/me/env-vars*` | Oturum açmış mevcut çağıran | Yalnızca kendini değiştirebilir, kimlik sunucu tarafından türetilir; ek yönetici eşiği yoktur, Bearer JWT kullanılır |

Alanlar arası ID erişim hakkı vermez. Kişisel değişken arayüzü, alan beceri yönetimi arayüzünün yerine geçemez.

## Korumalı alan yapılandırması

| Yöntem | Yol | İstek / yanıt |
| --- | --- | --- |
| GET | `/sandbox-configs` | 200 `{success,data:[ConfigResponse],workspace_scripts_disabled}` |
| POST | `/sandbox-configs` | `{name,description?,config}`; 201 `{success,data:ConfigResponse}` |
| GET | `/sandbox-configs/:id` | 200 `{success,data:ConfigResponse}` |
| PUT | `/sandbox-configs/:id` | `{name,description?,config}`, name zorunludur; 200 ayrıntıyla aynıdır |
| DELETE | `/sandbox-configs/:id` | İsteğe bağlı `force=true`; 200 `{success:true}` |
| GET | `/sandbox-configs/:id/sandboxes` | 200 `{success,data:SandboxInventory}`, kullanım durumunu ve ilişkili ajanları içerir |
| PUT | `/sandbox-configs/workspace-policy` | `{"scripts_disabled":true}`; success/workspace_scripts_disabled döndürür |
| POST | `/sandbox-configs/templates/query` | `{config,config_id?,ensure_standard?,replace_standard?,ensure_desktop?,replace_desktop?}`; uzak şablonları sorgular, standart veya masaüstü şablonu oluşturabilir/değiştirebilir; replace için config_id gerekir |

ConfigResponse `{id,name,description,sandbox_type,config,created_at,updated_at}` şeklindedir, kimlik bilgileri maskelenir. Düzenleme sırasında kayıtlı yapılandırma temel alınarak güncellenir; maskelenmiş yer tutucu değerler eski kimlik bilgilerini korur, `skill_image` kurulum hizmeti tarafından yönetilir ve istemci tarafından değiştirilemez.

### config alanları

| Alan | Tür | Açıklama |
| --- | --- | --- |
| `sandbox_type` | string | docker/cube/e2b; local kaldırıldı; Lite masaüstünün host değeri yapılandırma olarak kaydedilemez |
| `default_timeout_sec` | int | Çalıştırma zaman aşımı, 0 yerleşik varsayılanı kullanır |
| `terminal_idle_disconnect_sec` | int | Terminal/masaüstü işlem yapılmadan ne kadar süre sonra bağlantının kesileceği; 0 varsayılan 900 saniyedir, geçerli aralık 60 saniye–24 saattir |
| `desktop_enabled` | bool | Seçilen şablonun Cube/E2B masaüstü şablonu olduğunu belirtir; beceri yüklendikten sonra değiştirilemez |
| `allow_private_endpoints` | bool | Özel ağ kümesi adreslerine izin verir, link-local/bulut meta verilerine izin vermez |
| `env_vars` | map[string]string | Bu korumalı alan yapılandırmasının ortamı; değerler şifrelenerek saklanır |
| `skill_rollout` | string | next_turn (varsayılan)/new_session |
| `network` | object | Cube/E2B ağ politikası, aşağıya bakın |
| `cube` / `e2b` / `docker` | object | sandbox_type ile eşleşen bağlantı yapılandırması |
| `volume_mount` | object | İsteğe bağlı birim yapılandırması; etkili olup olmayacağını arka uç yetenekleri belirler, yalnızca alanın varlığı desteği göstermez |
| `skill_image` | object | Yükleme hizmetinin yönettiği anlık görüntü bilgileri, salt okunur |

volume_mount alanları enabled, mount_path, provider, volume_id, volume_name içerir; volume_owner_fingerprint sunucu tarafından yönetilir. Birim yapılandırması ve arka uç destek yetenekleri birlikte bağlanıp bağlanmayacağını belirler; düzenlerken sunucunun döndürdüğü sahiplik bilgilerini koruyun.

Arka uç yapılandırması:

| Arka uç | Alanlar |
| --- | --- |
| cube | api_url, proxy_url, sandbox_domain, template_id; api_key küme gereksinimine bağlıdır; http_timeout_sec, cube_sandbox_ttl_seconds, dns_servers |
| e2b | api_key, template_id zorunludur; api_url, sandbox_domain, proxy_url kendi barındırılan kurulumlar içindir; http_timeout_sec, e2b_sandbox_ttl_seconds |
| docker | image zorunludur; host (boşsa yerel socket), tls_cert_path (TCP için zorunlu), cpu_limit, memory_limit_mb, pids_limit, network_mode (bridge/none), runtime, idle_ttl_seconds, http_timeout_sec |

```bash
curl -X POST "$BASE/api/v1/sandbox-configs" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"E2B çalışma ortamı","config":{"sandbox_type":"e2b","e2b":{"api_key":"<e2b-key>","template_id":"<template-id>"}}}'

curl "$BASE/api/v1/sandbox-configs/cfg-1/sandboxes" \
  -H "Authorization: Bearer $TOKEN"
```

Arka uç kimliğini değiştirmek ve yapılandırmayı silmek uzak örnekleri denetler. 409 için error.code, `sandboxes_still_live`, `sandbox_inventory_unverifiable`, `skill_snapshot_release_failed` vb. olabilir; 423, yapılandırmanın başka bir istek tarafından değiştirildiğini belirtir. `force=true` yalnızca envanter doğrulanamadığında silmeye izin verir; doğrulanmış etkin/duraklatılmış örnekleri atlamaz. Yanıttaki kullanım verileri, ilgili oturumları ve akıllı etmenleri bulmak için kullanılır.

### Ağ politikası

`network`:

| Alan | Açıklama |
| --- | --- |
| `deny_egress_by_default` | false varsayılan olarak çıkışa izin verir; true varsayılan olarak reddeder |
| `allow_out` | IPv4/CIDR/alan adı veya tek etiketli joker alan adı; alan adları varsayılan reddetme modunda kullanılır |
| `deny_out` | IPv4/CIDR reddetme listesi |
| `cube_rules` | Cube için name/scheme/sni/host/methods/path/deny/audit/inject kuralları; sıra önemlidir |
| `e2b_host_rules` | E2B için host/headers kuralları; host ayrıca allow_out içinde listelenmelidir |
| `allow_public_inbound` | Eski girdilerle uyumludur, kaydedilirken temizlenir; gelen bağlantılar her zaman kimlik bilgileri gerektirir |

`cube_rules[].inject`, `[{header,secret,format}]` biçimindedir; `e2b_host_rules[].headers`, header→secret biçimindedir; gizli alanlar şifrelenerek saklanır ve maskelenir. Docker, `docker.network_mode` kullanır ve bu L7 kurallarını desteklemez.

```json
{
  "deny_egress_by_default": true,
  "allow_out": ["pypi.org", "files.pythonhosted.org"],
  "deny_out": []
}
```

### POST /system/sandbox-check

İstek `{config,config_id?,deep?}` şeklindedir; config_id, kaydedilmiş maskeli kimlik bilgilerini geri yüklemek için kullanılabilir. `deep=false` bağlantı denetimi yapar; true ayrıca geçici bir betik çalıştırır. Uzak arka uç, sanal alanı gerçekten oluşturup yok eder ve arka uç kullanımı doğurabilir.

Başarılı yoklama yürütmesi `{success:true,data:{ok,provider,checks,capabilities}}` döndürür; yoklama başarısız olsa bile HTTP 200 olabilir, bu nedenle data.ok denetlenmelidir. Parametre hataları 400 ile code/msg döndürür. Her check, name/ok/message/reason/latency_ms içerir; `ok=null` atlandığını belirtir. Örneğin, varsayılan olarak giden ağ erişimi reddedildiğinde dış ağ yoklaması politika nedeniyle atlanır; bu, dış ağ erişiminin doğrulandığı anlamına gelmez.

```bash
curl -X POST "$BASE/api/v1/system/sandbox-check" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"config_id":"cfg-1","deep":true}'
```

## Yetenek kataloğu

| Yöntem | Yol | İstek / Yanıt |
| --- | --- | --- |
| GET | `/skills/catalog` | 200 `{success,data:[CatalogItem]}`; CatalogItem, `installations:[{sandbox_config_id,status,enabled,version,bundle_sha256,served?,...}]` içerir |
| POST | `/skills/catalog` | multipart `file` veya JSON `{"source":"@owner/slug"}`; 201 `{success,data:{id,name,version,description}}` |
| POST | `/skills/catalog/:id/install` | `{"sandbox_config_ids":["cfg-1","cfg-2"]}`; 202 `{success,data:{installs,errors?}}` |
| GET | `/skills/catalog/:id/files` | 200 `{success,data:[FileEntry]}` |
| GET | `/skills/catalog/:id/files/content?path=SKILL.md` | 200 `{success,data:FileContent}` |
| DELETE | `/skills/catalog/:id` | Kurulum başvurusu olmadığında siler; 200 `{success:true}` |

Birden çok sanal alana kurulum kısmen kabul edilebilir: `data.installs` ve `data.errors` denetlenmelidir; HTTP 202, tüm kurulumların tamamlandığı anlamına gelmez. Katalog silme işlemi, her sanal alandaki kurulumu örtük olarak kaldırmaz.

```bash
curl -X POST "$BASE/api/v1/skills/catalog" \
  -H "Authorization: Bearer $TOKEN" -F 'file=@skill.zip'
curl -X POST "$BASE/api/v1/skills/catalog/catalog-1/install" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"sandbox_config_ids":["cfg-1"]}'
```

Kaynak yazımı, anonim indirme gereksinimleri ve ZIP sınırları için bkz. [Beceri kaynakları](../03-features/22-skills-sandbox.md#desteklenen-kaynaklar).

## Sanal alandaki yetenekler {#sanal-alandaki-yetenekler}

Aşağıdaki yol öneki `/sandbox-configs/:id/skills` şeklindedir; okuma dahil tüm işlemler Admin+ gerektirir.

| Yöntem | Sonek | Davranış |
| --- | --- | --- |
| GET | Boş | `{success,data:[SkillResponse]}` |
| POST | Boş | ZIP file veya source JSON ile kurulum; 202 `{success,data:{skill_id}}` |
| GET | `/:skillId` | `{success,data:SkillResponse}` |
| POST | `/:skillId/reinstall` | Paketi yeniden kullanarak kurar; isteğe bağlı `{"instructions":"..."}` (≤10000 karakter) kurulum açıklaması olarak verilebilir; 202 `{success,data:{skill_id}}` |
| POST | `/:skillId/stop` | Kurulumu durdurur; 200 yetenek durumunu döndürür, failed durumunda tekrar çağrılabilir; installing dışındaki diğer durumlar reddedilir |
| GET | `/:skillId/guidance` | Geçerli kurulum çalıştırma açıklamaları: `{success,data:{accepting,messages:[{id,content,status}]}}` |
| POST | `/:skillId/guidance` | Kurulum sürerken `{expected_message_id,steer_id,content}` ile ek açıklama ekler: `expected_message_id`, SkillResponse içindeki `install_message_id` değeridir; `steer_id` istemci tarafından üretilen UUID'dir (yeniden denemeler yinelenen ekleme yapmaz); `content` 1–10000 karakterdir; 202 `{success:true}`; kurulum yeni tura geçtiyse 409 döner |
| PATCH | `/:skillId` | `enabled?`, `envs?`; atlanırsa değiştirilmez |
| DELETE | `/:skillId` | Bu sanal alandan kaldırır, katalog paketini korur |
| GET | `/:skillId/files` | Dosya listesi |
| GET | `/:skillId/files/content?path=SKILL.md` | Göreli yol dosyasını okur, dizin geçişini reddeder |
| GET | `/:skillId/install-events` | SSE kurulum ilerlemesi |
| GET | `/:skillId/transcript` | SSE kurulum süreci olayları |

SkillResponse; id/name/version/description/enabled/status/error/bundle_sha256/installed_snapshot_id/install_session_id/install_message_id/created_at/updated_at ile `envs:[{name,description,required,is_set}]` alanlarını içerir. Yükseltme veya yeniden kurulum sürerken ya da başarısız olduktan sonra, `served:{version}` sanal alanın hâlâ sunduğu önceki hazır sürümü belirtir; aracının kullanabileceği beceriler bu sürüme göre belirlenir. Dosya yanıtları UTF-8 metin, küçük görsellerin base64 verisi veya ikili meta veri olabilir; tüm dosyaların metin içeriği olduğu varsayılamaz.

PATCH içindeki envs yalnızca bildirilmiş değişkenleri günceller; bildirilmemiş adlar yok sayılır, boş dizge değeri temizler ve bildirimi korur.

```bash
curl -X PATCH "$BASE/api/v1/sandbox-configs/cfg-1/skills/skill-1" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"enabled":true,"envs":{"API_TOKEN":"<workspace-token>"}}'

curl -N "$BASE/api/v1/sandbox-configs/cfg-1/skills/skill-1/install-events" \
  -H "Authorization: Bearer $TOKEN"
```

İlerleme yükü `{percent,stage,log?,status?,done}` biçimindedir. Bağlantının kesilmesi kurulumu iptal etmez; done yalnızca ilerlemenin artık izlenemediğini gösterebilir, nihai sonuç için beceri status değeri esas alınır. Redis yoksa akış ipucu durum yoklamasına dönüşür. transcript için 204, kurulumun henüz başladığını ve olay konumunun hazır olmadığını belirtir; 404 günlük olmadığını veya günlüğün süresinin dolduğunu gösterebilir, buna dayanarak kurulumun başarısız olduğu sonucuna varılamaz.

`GET /skills?sandbox_config_id=cfg-1`, aracının kullanabileceği beceri meta verilerini `{success,data:[{name,description}],skills_available}` olarak döndürür; bu, yöneticilerin gördüğü tüm kurulum kayıtlarından farklıdır.

`agent_id` ile `agent_source_tenant_id` birlikte verildiğinde (paylaşılan aracının kaynak alanı ID'si), bu paylaşılan aracının @ kullanarak erişebileceği beceriler listelenir: sanal alan yapılandırması aracının kendisinden alınır ve sorgudaki `sandbox_config_id` yok sayılır; aracının beceri kapsamı `selected` ise yalnızca belirtilen beceriler döner, devre dışıysa boş liste ve `skills_available:false` döner. Çağıranın bu aracıyı kullanma yetkisi yoksa 403 döner.

## Oturum etkileşimli terminali {#oturum-etkilesimli-terminali}

Sohbet kenar çubuğundaki terminal, bu oturumun uzak sandbox'ına WebSocket ile bağlanır; yalnızca Cube/E2B desteklenir. Rota `internal/router/routes_chat.go` tarafından kaydedilir ve henüz Swagger'da yoktur; masaüstü arayüzü için bkz. [Oturum API'si](02-api-chat.md#sandbox-desktop), proxy gereksinimleri için bkz. [Sandbox dağıtımı](../06-development/04-sandbox-deployment.md#etkilesimli-terminal-ve-grafik-masaustu).

### POST /api/v1/sessions/:session_id/sandbox/terminal-ticket

İki dakika geçerli bir el sıkışma bileti düzenler. Oturum sahibinin giriş Bearer access token'ı gerekir; API Key ile çağrı yapılamaz.

```bash
curl -X POST "$BASE/api/v1/sessions/$SESSION_ID/sandbox/terminal-ticket" \
  -H "Authorization: Bearer $TOKEN"
```

Yanıt: 200 `{"success":true,"data":{"ticket":"<ticket>","expires_in":120}}`.

### GET /api/v1/sessions/:id/sandbox/terminal

WebSocket el sıkışması normal kimlik doğrulama ara yazılımından geçmez:

| Sorgu parametresi | Zorunlu | Açıklama |
| --- | --- | --- |
| `ticket` | Evet | Önceki adımda alınan bilet; kullanıcıya, alana, oturuma ve düzenleme anındaki access token'a bağlıdır |
| `provision` | Hayır | `1`, sanal alan oluşturulmasına veya uyandırılmasına izin verildiğini belirtir; ücretlendirme olabilir. Atlanırsa yalnızca çalışan sanal alana bağlanır |
| `agent_id` / `agent_source_tenant_id` | Hayır | `provision=1` ile birlikte kullanılır; hangi aracının, paylaşılan aracı dahil, sanal alan yapılandırmasına göre oluşturulacağını belirtir |
| `cols` / `rows` | Hayır | Başlangıç terminal boyutu |
| `pty_id` | Hayır | Önceki `ready` karesinin döndürdüğü Shell sürecine yeniden bağlanır |

Bağlantıdan sonra ikili kareler terminal girdi ve çıktısını çift yönlü aktarır; metin kareleri JSON denetim iletileridir: sunucu `ready` (`pty_id`, `backend` içerir), `exited` (`exit_code` içerir) ve `error` gönderir; istemci `{"type":"resize","cols":..,"rows":..}` gönderebilir. Hata kodları arasında `SANDBOX_NOT_BOUND`, `SANDBOX_PAUSED`, `TERMINAL_UNSUPPORTED`, `IDLE_DISCONNECTED`, `AUTH_REVOKED`, `INTERNAL` bulunur. Her oturumda aynı anda en fazla 5 terminal olabilir; aşılırsa 429 döner.


## Kişisel ortam değişkenleri {#kisisel-ortam-degiskenleri}

Çağıran kimliği mevcut kimlik doğrulama bağlamıyla belirlenir, user_id kabul edilmez. Kişisel arayüz değişken adını, kaynağını ve güncellenme zamanını döndürür; açık metin değer döndürmez.

| Yöntem | Yol | İstek |
| --- | --- | --- |
| GET | `/me/env-vars` | sandbox_config_id bazında gruplandırılmış yapılandırma değişkenlerini ve beceri değişkenlerini döndürür |
| PUT | `/me/env-vars/skill` | `{skill_id,name,value}`; ad, becerinin bildirilmiş bir öğesi olmalıdır |
| DELETE | `/me/env-vars/skill` | `{skill_id,name}` |
| PUT | `/me/env-vars/sandbox` | `{sandbox_config_id,name,value}` |
| DELETE | `/me/env-vars/sandbox` | `{sandbox_config_id,name}` |

Okuma işlemi `{success,data:[{sandbox_config_id,sandbox_config_name,description,vars,skills}]}` döndürür; source, user/workspace/unset gibi durumları belirtir. Ayarlama veya silme başarılı olduğunda success döner; ayarlanmamış bir öğeyi silmek 404 döndürür. Kişisel sanal alan değişkenleri RETHRA_ ön ekini ve PATH gibi ayrılmış adları reddeder; beceri bildirimli değişkenleri gerekli RETHRA_* kimlik bilgisi adlarını kullanabilir.

```bash
curl -X PUT "$BASE/api/v1/me/env-vars/skill" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"skill_id":"skill-1","name":"API_TOKEN","value":"<my-token>"}'

curl -X DELETE "$BASE/api/v1/me/env-vars/skill" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"skill_id":"skill-1","name":"API_TOKEN"}'
```

Geçersiz kılma sırası kişisel beceri değeri > kişisel sandbox değeri > alan beceri değeridir. Kişisel değer yazılması yönetici yapılandırmasını değiştirmez. Uygulama referansları: `internal/router/routes_infra.go`, `routes_agent.go`, `routes_auth_tenant.go` ile `internal/handler/sandbox_config.go`, `sandbox_skill.go`, `skill_catalog.go`, `me_env_var.go`, `host_project_picker.go`.
