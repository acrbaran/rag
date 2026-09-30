# Go SDK

Go SDK, bilgi tabanları, belgeler ve oturumlar gibi temel kaynakların CRUD işlemlerini ve SSE akışlı soru-cevap işlevini kapsar. Kaynak kod `client/` dizinindedir, bağımsız bir Go module olarak sunulur; sunucu tarafındaki ilgili çağrılar bu SDK'yı yeniden kullanır.

## Kurulum

SDK'nin module yolu `client/go.mod` içinde tanımlanmıştır:

```
module github.com/acrbaran/rag/client

go 1.24.2
```

Kurulum yöntemi:

```bash
go get github.com/acrbaran/rag/client
```

İçe aktarma:

```go
import "github.com/acrbaran/rag/client"
```

## Başlatma ve kimlik doğrulama

Temel türler ve kurucu işlevler `client/client.go` içinde tanımlanmıştır.

### Client yapısı

```go
type Client struct {
    baseURL       string
    httpClient    *http.Client
    streamTimeout time.Duration
    apiKey        string
    bearerToken   string
    tenantID      *uint64
}
```

Bir örnek `NewClient(baseURL string, options ...ClientOption) *Client` ile oluşturulur. Varsayılan normal istek zaman aşımı 30 saniyedir; akışlı (SSE) isteklerde varsayılan olarak **zaman aşımı yoktur**, yaşam döngüsü `context` tarafından kontrol edilir (`WithTimeout` açıkça çağrılmadıkça).

### ClientOption özeti

| Option | Açıklama |
|---|---|
| `WithAPIKey(key string)` | Uzun süre geçerli API Key'i ayarlar, `X-API-Key` istek başlığıyla gönderilir |
| `WithBearerToken(token string)` | Kısa süreli JWT'yi ayarlar, `Authorization: Bearer <token>` istek başlığıyla gönderilir (genellikle `Login` başarılı olduktan sonra kullanılır) |
| `WithToken(token string)` | **Kullanımdan kaldırıldı**: `WithAPIKey` için v0.x uyumluluk takma adı; sonraki büyük sürümde kaldırılacaktır |
| `WithTimeout(timeout time.Duration)` | Normal ve akışlı istekler için zaman aşımı üst sınırını birlikte ayarlar |
| `WithTransport(rt http.RoundTripper)` | Alttaki `http.RoundTripper` bileşenini değiştirir (yeniden deneme/izleme/imzalama gibi ara yazılımlar için); `nil` verilmesi `http.DefaultTransport` değerini geri yükler |
| `WithTenantID(tenantID uint64)` | Her isteğe `X-Tenant-ID` istek başlığını ekler; yalnızca `CanAccessAllTenants` yetkisine sahip kiracılar arası açık erişim için kullanılır |

### Kimlik doğrulama yöntemleri

SDK, aynı anda yapılandırılabilen iki kimlik bilgisi destekler; HTTP katmanında `X-API-Key` önceliklidir:

- **API Key** (uzun süreli): `WithAPIKey`, istek başlığı `X-API-Key`;
- **Bearer JWT** (kısa süreli): `WithBearerToken`, istek başlığı `Authorization: Bearer <token>`; `client/auth.go` içindeki `Login` / `RefreshToken` / `GetCurrentUser` ile kullanılır.

Tipik JWT oturum açma akışı (`POST /api/v1/auth/login` ile eşleşir):

```go
c := client.NewClient("http://localhost:8080")
loginResp, err := c.Login(ctx, client.LoginRequest{ /* email + password */ })
// Ardından dönen access token ile kimlik doğrulamalı istemciyi yeniden oluştur
authed := client.NewClient("http://localhost:8080",
    client.WithBearerToken(loginResp.AccessToken))
```

### Kiracı (Tenant) ve istek başlığı ekleme

`applyAuthHeaders` (`client/client.go`), her isteğe otomatik olarak şunları ekler:

- `X-API-Key` / `Authorization` (yapılandırmaya göre);
- `X-Request-ID`: izleme için `ctx.Value("RequestID")` içinden (string türünde) okunur;
- `X-Tenant-ID`: öncelik context içindeki `"TenantID"` değeridir (`uint64`, `*uint64`, sayısal dize desteklenir) > `WithTenantID` istemci düzeyi varsayılan değeri.

Tek istek için kiracı geçersiz kılma örneği:

```go
tenantID := uint64(10000)
ctx := context.WithValue(context.Background(), "TenantID", &tenantID)
kb, err := apiClient.GetKnowledgeBase(ctx, kbID)
```

Not: JWT ve kiracı düzeyi API Key zaten kiracı kimliğini taşır; normal kullanıcılar `X-Tenant-ID` ayarlamamalıdır (sunucu auth ara yazılımı bu başlığı taşıyan bearer isteklerinde kiracılar arası doğrulama yapar, normal kullanıcılar 403 alır).

### Raw kaçış kapağı

`Client.Raw(ctx, method, path, body)` (Deneysel), istemcide yapılandırılmış kimlik doğrulama başlıklarıyla doğrudan rastgele HTTP istekleri başlatır ve tek seferlik entegrasyonlar için kullanılır; türlenmiş yöntemler mevcutsa öncelikle türlenmiş yöntemler kullanılmalıdır.

## Kaynaklar ve yöntemlere genel bakış

Aşağıdakilerin tümü `Client` sınıfının herkese açık yöntemleridir; dahili yöntemler (`buildRequest`, `doRequest`, `doRequestStream`, `processAgentSSEStream` vb.) dahil edilmemiştir.

### Kimlik doğrulama Auth — `client/auth.go`

| Yöntem | Açıklama |
|---|---|
| `Login` | E-posta ve parolayla giriş yapar, JWT access/refresh token döndürür |
| `GetCurrentUser` | Mevcut oturum sahibi ve kiracı bilgilerini alır (`GET /api/v1/auth/me`) |
| `RefreshToken` | Refresh token kullanarak yeni access token alır |
| `SwitchTenant` | Belirtilen alana geçer ve token çiftini yeniden oluşturur; ayrıca bunu sonraki giriş için varsayılan alan olarak kaydeder (`POST /api/v1/auth/switch-tenant`) |
| `ChangePassword` | Mevcut kullanıcının parolasını değiştirir; başarılı olduğunda sunucu bu kullanıcının tüm oturumlarını iptal eder, çağıran taraf yerel token'ları atmalıdır |
| `GetAuthConfig` | Herkese açık kimlik doğrulama yapılandırmasını okur: kayıt modu, karmaşık parola etkin mi (`GET /api/v1/auth/config`, kimlik doğrulama gerekmez) |

### Bilgi tabanı KnowledgeBase — `client/knowledgebase.go`

| Yöntem | Açıklama |
|---|---|
| `CreateKnowledgeBase` | Bilgi tabanı oluşturur |
| `GetKnowledgeBase` | Bilgi tabanı ayrıntılarını alır |
| `ListKnowledgeBases` | Bilgi tabanlarını listeler |
| `UpdateKnowledgeBase` | Bilgi tabanını günceller |
| `DeleteKnowledgeBase` | Bilgi tabanını siler |
| `ClearKnowledgeBaseContents` | Bilgi tabanı içeriğini temizler |
| `HybridSearch` | Bilgi tabanında karma arama yapar (vektör + anahtar kelime); dosya doğrudan bağlantısını döndürmek için `ResourceURLOptions` iletilebilir |
| `TogglePinKnowledgeBase` | Sabitler/sabitlemeyi kaldırır |
| `ListMoveTargets` | Bilginin taşınabileceği hedef bilgi tabanlarını listeler |
| `CopyKnowledgeBase` | Bilgi tabanını kopyalar |
| `DuplicateKnowledgeBase` | Bilgi tabanını çoğaltır (`duplicate`) |
| `GetKBCloneProgress` | Klonlama görevi ilerlemesini sorgular |

### Bilgi Knowledge — `client/knowledge.go`

| Yöntem | Açıklama |
|---|---|
| `CreateKnowledgeFromFile` | Yerel dosya yükleyerek bilgi oluşturur (`multipart`; `metadata`, çok modlu anahtar, özel dosya adı, `channel`, ayrıştırma yapılandırması geçersiz kılmalarını destekler) |
| `CreateKnowledgeFromURL` | URL'den bilgi oluşturur |
| `GetKnowledge` | Bilgi ayrıntılarını alır |
| `GetKnowledgeBatch` | Bilgileri toplu olarak alır |
| `ListKnowledge` | Bilgileri sayfalı olarak listeler |
| `ListKnowledgeWithFilter` | Bilgileri filtre koşullarıyla listeler (`KnowledgeListFilter`: etiketler, anahtar sözcükler, dosya türü, ayrıştırma durumu, kaynak, zaman aralığı, klasör) |
| `DeleteKnowledge` | Bilgiyi siler |
| `DownloadKnowledgeFile` | Bilginin özgün dosyasını yerel yola indirir |
| `OpenKnowledgeFile` | Bilginin özgün dosyasını akış olarak açar (dosya adı + `io.ReadCloser` döndürür) |
| `DownloadKnowledgeFiles` / `OpenKnowledgeFilesArchive` | Aynı bilgi tabanındaki birden çok belgenin özgün dosyalarını ZIP olarak paketleyip yerel ortama indirir / akış olarak okur (`POST /api/v1/knowledge-bases/{id}/knowledge/batch-download`; tek seferde en fazla 200 ID, toplam 512 MiB; Contributor ve ilgili bilgi tabanında yazma izni gerektirir); akışlı HTTP istemcisi kullanır ve 30 saniyelik varsayılan zaman aşımına tabi değildir |
| `ListKnowledgeFolders` / `MoveKnowledgeToFolder` / `RenameKnowledgeFolder` | Bilgi tabanı klasör ağacı; belgeleri klasöre taşır (`FolderPath` boşsa kök dizine geri taşınır); klasörü ve tüm alt yollarını yeniden adlandırır |
| `UpdateKnowledge` | Bilgiyi günceller |
| `ReparseKnowledge` | Bilgiyi yeniden ayrıştırır |
| `CancelKnowledgeParse` | Ayrıştırma görevini iptal eder |
| `GetKnowledgeProcessingSpans` | Bilgi işleme zinciri span'lerini alır |
| `UpdateImageInfo` | Görsel bilgilerini günceller |
| `CreateManualKnowledge` | Elle oluşturulan (`manual`) bilgi oluşturur |
| `UpdateManualKnowledge` | Elle oluşturulan bilgiyi günceller |
| `FilterKnowledge` | Bilgiyi anahtar sözcüklere/dosya türüne/agent'a göre filtreler |
| `MoveKnowledge` | Bilgiyi bilgi tabanları arasında taşı |
| `GetKnowledgeMoveProgress` | Taşıma görevinin ilerlemesini sorgula |
| `PreviewKnowledgeFile` | Bilgi dosyasını önizle (ham `*http.Response` döndürür) |
| `BatchUpdateKnowledgeTags` | Bilgi etiketlerini toplu olarak güncelle |

Sunucu tarafındaki bilgi listesi arayüzü, v0.8.2 sürümünden itibaren `sort_by` (`updated_at` / `created_at` / `file_name`) ve `sort_order` (`asc` / `desc`) parametrelerini destekler; varsayılan ayar hâlâ `created_at desc` şeklindedir. `KnowledgeListFilter` henüz karşılık gelen alanları sunmaz; özel sıralama gerektiğinde `Raw` çağrısı kullanılabilir. Parametre açıklamaları için [Bilgi API'sine](../04-api/02-api-knowledge.md) bakın.

### Parça Chunk — `client/chunk.go`

| Yöntem | Açıklama |
|---|---|
| `ListKnowledgeChunks` | Belirli bir bilginin parçalarını sayfalı olarak listele |
| `UpdateChunk` | Parça içeriğini/etkin durumunu güncelle |
| `DeleteChunk` | Parçayı sil |
| `GetChunkByIDOnly` | Parçayı yalnızca parça kimliğiyle al |
| `DeleteGeneratedQuestion` | Parçanın oluşturduğu soruyu sil |
| `DeleteChunksByKnowledgeID` | Belirli bir bilginin tüm parçalarını sil |

### Oturum Session — `client/session.go`

| Yöntem | Açıklama |
|---|---|
| `CreateSession` | Oturum oluştur |
| `GetSession` | Oturumu al |
| `GetSessionsByTenant` | Kiracı oturumlarını sayfalı olarak listele |
| `UpdateSession` | Oturumu güncelle |
| `DeleteSession` | Oturumu sil |
| `BatchDeleteSessions` | Oturumları toplu olarak sil |
| `GenerateTitle` | Oturum başlığı oluştur |
| `KnowledgeQAStream` | Bilgi soru-cevap işlemi (SSE akışı, aşağıya bakın) |
| `ContinueStream` | Devam eden akışı sürdür (bağlantı kesilip yeniden bağlanma durumu) |
| `StopSession` | Belirli bir assistant mesajının oluşturulmasını durdur |
| `SearchKnowledge` | Bilgi arama |

### Mesaj Message — `client/message.go`, `client/message_suggestion.go`

| Yöntem | Açıklama | Kaynak dosya |
|---|---|---|
| `LoadMessages` | Mesajları zamana göre yükler | `client/message.go` |
| `GetRecentMessages` | En son N mesajı alır | `client/message.go` |
| `GetMessagesBefore` | Belirli bir zamandan önceki mesajları alır | `client/message.go` |
| `SearchMessages` | Geçmiş mesajları arar | `client/message.go` |
| `GetChatHistoryKBStats` | Sohbet geçmişini bilgi tabanına göre istatistiksel olarak sunar | `client/message.go` |
| `DeleteMessage` | Mesajı siler | `client/message.go` |
| `EnsureMessageSuggestions` | Önerilen soruların oluşturulmasını garanti eder (zorla yeniden oluşturabilir) | `client/message_suggestion.go` |
| `GetMessageSuggestions` | Mesaj için önerilen soruları alır | `client/message_suggestion.go` |
| `RecordMessageSuggestionEvent` | Önerilen soru tıklama/görüntülenme olayını bildirir | `client/message_suggestion.go` |

### Agent konuşması (akışlı) — `client/agent.go`

| Yöntem | Açıklama |
|---|---|
| `AgentQAStream` | Agent modunda akışlı soru-cevap (kullanımdan kaldırıldı, basitleştirilmiş giriş noktası) |
| `AgentQAStreamWithRequest` | Agent modunda akışlı soru-cevap (tam `AgentQARequest` yükü) |
| `NewAgentSession` | `AgentSession` sarmalayıcısı oluşturur (üzerinde `Ask` / `AskWithRequest` / `GetSessionID` bulunur) |

### Agent yönetimi — `client/agent_manage.go`

| Yöntem | Açıklama |
|---|---|
| `CreateAgent` | Özel Agent oluşturur |
| `ListAgents` | Agent'ları listeler |
| `GetAgent` | Agent alır |
| `UpdateAgent` | Agent günceller |
| `DeleteAgent` | Agent siler |
| `CopyAgent` | Agent kopyalar |
| `GetAgentPlaceholders` | Agent yapılandırma yer tutucularını alır |
| `GetSuggestedQuestions` | Agent öneri sorularını alır |

v0.8.2'den itibaren iki istek alanı değişikliği:

- `UpdateAgentRequest.Avatar`, `string` yerine `*string` oldu: `nil` olduğunda bu alan gönderilmez ve mevcut avatar korunur; boş bir dizgeyi işaret etmesi avatarın temizleneceği anlamına gelir. SDK yükseltildikten sonra atama kodunun ayarlanması gerekir (örneğin `Avatar: &avatar`).
- Öneri sorusu `SuggestedQuestion` için `KnowledgeID` (kaynak belge) eklendi. Kullanıcı öneri sorusunu seçerek sorduğunda, kaynak `KnowledgeQARequest` / `AgentQARequest` içindeki `QuestionOrigin` (`KnowledgeBaseID`, `KnowledgeID`) aracılığıyla geri iletilebilir; sunucu yanıttan önce bu kaynağı öncelikli olarak arar. Bu yalnızca bir ipucudur ve isteğin kendi arama kapsamını genişletmez.

### Model Model — `client/model.go`

| Yöntem | Açıklama |
|---|---|
| `CreateModel` | Model oluşturur |
| `GetModel` | Model alır |
| `ListModels` | Modelleri listeler |
| `UpdateModel` | Modeli günceller |
| `DeleteModel` | Modeli siler |
| `ListModelProviders` | Model sağlayıcılarını model türüne göre listeler |

### Kiracı Tenant — `client/tenant.go`

| Yöntem | Açıklama |
|---|---|
| `CreateTenant` | Kiracı oluşturur |
| `GetTenant` | Kiracı alır |
| `UpdateTenant` | Kiracıyı günceller |
| `DeleteTenant` | Kiracıyı siler |
| `ListTenants` | Kiracıları listeler |
| `ListAllTenants` | Tüm kiracıları listeler (yönetici) |
| `SearchTenants` | Kiracıları arar (sayfalı) |
| `ListTenantAPIKeys` | Kiracı API anahtarlarını listeler |
| `CreateTenantAPIKey` | Kiracı API anahtarı oluşturur |
| `DeleteTenantAPIKey` | Kiracı API anahtarını siler |
| `GetTenantKV` | Kiracı düzeyindeki KV yapılandırmasını okur |
| `UpdateTenantKV` | Kiracı düzeyindeki KV yapılandırmasını günceller |
| `GetAPIPrincipalConfig` | API sorumlusu yapılandırmasını alır |
| `UpdateAPIPrincipalConfig` | API sorumlusu yapılandırmasını günceller |
| `CreateAPIPrincipalTestToken` | API sorumlusu için test tokenı oluşturur |

### Organizasyon ve paylaşım Organization — `client/organization.go`

| Yöntem | Açıklama |
|---|---|
| `CreateOrganization` / `ListMyOrganizations` / `GetOrganization` / `UpdateOrganization` / `DeleteOrganization` | Organizasyon CRUD işlemleri |
| `SearchOrganizations` / `PreviewOrganizationByInviteCode` | Organizasyon arama/davet koduyla önizleme |
| `JoinOrganizationByInviteCode` / `SubmitJoinRequest` / `JoinByOrganizationID` / `LeaveOrganization` / `RequestRoleUpgrade` | Katılma/ayrılma/rol yükseltme |
| `GenerateInviteCode` / `SearchUsersForInvite` / `InviteMember` | Üye davet etme (`SearchUsersForInvite` sınırlamaları için aşağıdaki açıklamaya bakın) |
| `ListOrgMembers` / `UpdateMemberRole` / `RemoveMember` | Üye yönetimi |
| `ListJoinRequests` / `ReviewJoinRequest` | Katılım başvurusu onayı |
| `ShareKnowledgeBase` / `ListKBShares` / `UpdateSharePermission` / `RemoveKBShare` | Bilgi tabanı paylaşımı |
| `ShareAgent` / `ListAgentShares` / `RemoveAgentShare` | Agent paylaşımı |
| `ListOrgShares` / `ListOrgAgentShares` / `ListSharedKnowledgeBases` / `ListSharedAgents` | Paylaşılan kaynak sorgulama |

v0.8.2 sürümünden itibaren sunucudaki davet adayı yalnızca **tam alan ID'si** ile kesin olarak çözümlenir (`GET /api/v1/organizations/{id}/search-tenants?q=<alanID>`); artık alan adı, kullanıcı adı veya e-posta ile arama yapılmaz. `SearchUsersForInvite`, kullanımdan kaldırılmış `search-users` diğer adını çağırır ve değeri `keyword` parametresiyle iletir; sunucu ise `q` değerini okur, bu nedenle bu yöntem şu anda adayları getiremez. Aday aramanız gerekiyorsa, `Raw` kullanarak doğrudan `search-tenants` çağrısı yapın.

### FAQ — `client/faq.go`

| Yöntem | Açıklama |
|---|---|
| `ListFAQEntries` | FAQ girdilerini sayfalı olarak listeler |
| `UpsertFAQEntries` | FAQ girdilerini toplu olarak ekler/günceller |
| `CreateFAQEntry` | Tek bir FAQ oluşturur |
| `GetFAQEntry` | Tek bir FAQ alır |
| `UpdateFAQEntry` | Tek bir FAQ günceller |
| `AddSimilarQuestions` | Benzer sorular ekler |
| `UpdateFAQEntryFieldsBatch` | Alanları toplu olarak günceller |
| `UpdateFAQEntryTagBatch` | Etiketleri toplu güncelle |
| `DeleteFAQEntries` | Toplu sil |
| `SearchFAQEntries` | FAQ ara |
| `ExportFAQEntries` | CSV olarak dışa aktar (`[]byte` döndürür) |
| `GetFAQImportProgress` | Eşzamansız içe aktarma görevi ilerlemesini sorgula (dry run dahil) |
| `UpdateLastFAQImportResultDisplayStatus` | En son içe aktarma sonucunun görüntüleme durumunu güncelle |

### Etiket Tag — `client/tag.go`

| Yöntem | Açıklama |
|---|---|
| `ListTags` | Etiketleri listele |
| `CreateTag` | Etiket oluştur |
| `UpdateTag` / `UpdateTagBySeqID` | Etiketi güncelle (ID ile / seq ID ile) |
| `DeleteTag` / `DeleteTagBySeqID` | Etiketi sil (ID ile / seq ID ile) |

### MCP Hizmeti — `client/mcp_service.go`

| Yöntem | Açıklama |
|---|---|
| `CreateMCPService` / `ListMCPServices` / `GetMCPService` / `UpdateMCPService` / `DeleteMCPService` | MCP hizmeti CRUD |
| `TestMCPService` | Bağlantı testi |
| `GetMCPServiceTools` / `GetMCPServiceResources` | MCP araçlarını/kaynaklarını listele |
| `GetMCPMetadata` | Kaydedilmiş araç kataloğunu okur, üst kaynağa bağlanmaz; hiç eşitlenmemişse `nil` döner, `Stale=true` kaydedilmiş bağlantının geçerli yapılandırmayla tutarsız olduğunu belirtir |
| `RefreshMCPMetadata` | Üst kaynağa bağlanır ve kaydedilmiş araç kataloğunu bütünüyle değiştirir; OAuth hizmetleri çağırana göre anlık görüntü kaydeder (Viewer+), statik kimlik doğrulama hizmetleri çalışma alanı düzeyinde anlık görüntü yazar (Admin veya MCP hizmetlerini yönetebilen API Key gerektirir) |
| `ResolveToolApproval` | Araç çağrısı onayını işle |

`MCPService` liste öğeleri `Catalog` (araç sayısı, güncel olup olmadığı, eşitleme zamanı) ve `UsageInstructions` alanlarını içerir.

### MCP Uç Noktası (MCP Server olarak Rethra) — `client/mcp_endpoint.go`

| Yöntem | Açıklama |
|---|---|
| `ListMCPEndpoints` / `GetMCPEndpoint` / `CreateMCPEndpoint` / `UpdateMCPEndpoint` / `DeleteMCPEndpoint` | Çalışma alanının dışarıya sunduğu MCP uç noktaları için CRUD; oluşturma yanıtındaki `Token` yalnızca bir kez döner |
| `RotateMCPEndpointToken` | Uç nokta belirtecini yenile, eski belirteç hemen geçersiz olur |
| `GetMCPEndpointToolCatalog` | Uç noktanın seçebileceği araç kataloğu, grupları ve varsayılan seçimler |

### Başlatma ve model denetimi — `client/initialization.go`

| Yöntem | Açıklama |
|---|---|
| `GetInitializationConfig` / `InitializeByKB` / `UpdateKBConfig` / `SetKBModelConfig` | Bilgi tabanı başlatma ve model yapılandırması |
| `CheckRemoteModel` / `TestEmbeddingModel` / `CheckRerankModel` / `TestMultimodalFunction` | Uzak LLM / Embedding / Rerank / çok modlu bağlantı denetimi |
| `ExtractTextRelations` | Metin ilişkisi çıkarma testi |

### Sistem System — `client/system.go`

| Yöntem | Açıklama |
|---|---|
| `GetSystemInfo` | Sistem bilgilerini alır (sürüm vb.) |
| `GetDeploymentCapabilities` | Dağıtım yetenekleri anlık görüntüsünü alır (`GET /api/v1/system/capabilities`, edition ve her yeteneğin supported / reason değerleri) |
| `ListParserEngines` / `CheckParserEngines` | Belge ayrıştırma motoru listesi/denetimi |
| `ReconnectDocReader` | DocReader hizmetine yeniden bağlanır |
| `GetStorageEngineStatus` / `CheckStorageEngine` | Depolama motoru durumu/denetimi |

### Diğer

| Yöntem | Açıklama | Kaynak dosya |
|---|---|---|
| `StartEvaluation` / `GetEvaluationResult` | Değerlendirme görevi başlatır / değerlendirme sonucunu sorgular | `client/evaluation.go` |
| `ListSkills(ctx, sandboxConfigID)` | Belirtilen korumalı alan yapılandırmasının çağırabileceği becerileri listeler; beceri listesini ve kullanılabilirlik işaretini döndürür | `client/skill.go` |
| `GetWebSearchProviders` | Kullanılabilir Web arama sağlayıcılarını listeler | `client/web_search.go` |
| `Raw` | Ham HTTP kaçış kapsülü (Deneysel) | `client/client.go` |

Toplamda 20'den fazla kaynak türünü kapsayan 220'den fazla açık yöntem vardır.

### Bellek, korumalı alan becerileri ve kişisel değişkenler

| Dosya | Yöntemler ve amaçları |
| --- | --- |
| `client/memory.go` | GetMemorySettings / UpdateMemorySettings; List/Create/Update/DeleteMemoryItem; Confirm/RejectMemoryItem; ClearMemoryItems |
| `client/memory.go` | ListMemoryTopics / PromoteMemoryTopic / DeleteMemoryTopic; ListMemoryDocuments / DeleteMemoryDocument; ExportMemory / ConsolidateMemory |
| `client/skill.go` | InstallSandboxSkillFromSource / UploadSandboxSkill / ReinstallSandboxSkill / StopSandboxSkill, kurulum sürecini yönetir |
| `client/skill.go` | UpdateSandboxSkill / SetSandboxSkillEnabled / SetSandboxSkillEnvValues; ListSandboxSkillFiles / GetSandboxSkillFile |
| `client/env_var.go` | ListMyEnvVars; SetMySkillEnvVar / DeleteMySkillEnvVar; SetMySandboxEnvVar / DeleteMySandboxEnvVar |
| `client/tenant.go` | UpdateTenantAPIKey, token yenilemeden mevcut Key'in tam yetkilendirme yapılandırmasını değiştirir |

Alan beceri değişkenleri yönetici tarafından ayarlanır; kişisel değişkenler yalnızca çağıranın kendisi tarafından kullanılır. Listeler düz metni döndürmez. Bellek ve beceri yöntemlerinin izinleri yine arka uç API'si tarafından doğrulanır; SDK bu kısıtlamaları atlamaz.

```go
items, total, err := c.ListMemoryItems(ctx, "active", 50, 0)
_ = items
_ = total
_ = err

skills, available, err := c.ListSkills(ctx, "sandbox-config-id")
_ = skills
_ = available
_ = err
```

Sistem yöneticileri kullanıcıları `POST /system/admin/users/create` ile oluşturur; mevcut SDK'da bu uç nokta için özel bir yöntem yoktur, önceki Raw kaçış kapağı mekanizmasını veya HTTP istemcisini kullanabilirsiniz; yanıt için bkz. [Sistem API](../04-api/02-api-system.md). Tam ekleme sözleşmeleri için bkz. [Uzun Süreli Bellek API](../04-api/02-api-memory.md), [Korumalı Alan ve Yetenekler API](../04-api/02-api-sandbox-skills.md).

## Akışlı sohbet (SSE)

SDK'nın akış arayüzü channel yerine **geri çağırım (callback) mekanizması** kullanır: SDK, SSE'yi dahili olarak `bufio.Scanner` ile satır satır ayrıştırır (`event:` / `data:` önekleri, boş satırlarla çerçeveleme) ve ayrıştırılan her çerçeve için geri çağırımı bir kez çağırır; geri çağırım nil olmayan bir error döndürürse akış sonlandırılır. SSE satır arabellek üst sınırı 4 MiB'a yükseltilmiştir (`scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)`); böylece büyük references çerçevelerinin "token too long" hatasını tetiklemesi önlenir.

Akışlı istekler `doRequestStream` (`client/client.go`) üzerinden gider, varsayılan olarak 30 saniyelik zaman aşımına tabi değildir ve akış yaşam döngüsü geçirilen `ctx` tarafından kontrol edilir.

### Bilgi soru-cevap akışı: `KnowledgeQAStream` (`client/session.go`)

```go
func (c *Client) KnowledgeQAStream(
    ctx context.Context,
    sessionID string,
    request *KnowledgeQARequest,
    callback func(*StreamResponse) error,
) error
```

Her `StreamResponse` çerçevesi `ResponseType` (`answer`, `references`, `thinking`, `tool_call`, `tool_result`, `error`, `reflection`, `session_title`, `agent_query`, `complete`), artımlı `Content`, bitiş işareti `Done` ve `Done` çerçevesindeki `KnowledgeReferences` (atıf kaynakları) alanlarını taşır.

### Agent soru-cevap akışı: `AgentQAStreamWithRequest` (`client/agent.go`)

```go
type AgentEventCallback func(*AgentStreamResponse) error

func (c *Client) AgentQAStreamWithRequest(ctx context.Context,
    sessionID string, request *AgentQARequest, callback AgentEventCallback,
) error
```

`AgentQARequest`; `KnowledgeBaseIDs`, `AgentID`, `WebSearchEnabled`, `MentionedItems` (@ ile anılan bilgi tabanı/dosya/etiket/MCP/skill), `Images` (çok modlu görseller) ve benzeri alanları destekler. Kolaylık sarmalayıcıları da kullanılabilir:

```go
as := apiClient.NewAgentSession(session.ID)
err := as.Ask(ctx, "Rethra'yı tanıt", func(ev *client.AgentStreamResponse) error {
    if ev.ResponseType == client.AgentResponseTypeAnswer {
        fmt.Print(ev.Content)
    }
    return nil
})
```

### Bağlantı kesilince devam etme: `ContinueStream` (`client/session.go`)

`ContinueStream(ctx, sessionID, messageID, callback)`, sunucuda hâlâ üretilmekte olan akışa `GET /api/v1/sessions/continue-stream/{sessionID}?message_id=...` ile yeniden bağlanır; geri çağırım mekanizması `KnowledgeQAStream` ile aynıdır. `StopSession(ctx, sessionID, messageID)` ile birlikte üretim durdurulabilir.

## Hata işleme

### HTTP katmanı: `APIError` (`client/client.go`)

2xx olmayan tüm yanıtlar `*APIError` olarak sarmalanır; HTTP durum koduna veya sunucu tarafındaki yapılandırılmış hata koduna göre dallanmak için `errors.As` kullanın:

```go
var apiErr *client.APIError
if errors.As(err, &apiErr) {
    switch {
    case apiErr.StatusCode == 404:
        // Kaynak mevcut değil
    case apiErr.Code == client.ServerErrUnauthorized: // 1001
        // Yeniden girişi tetikle
    }
}
```

`Code`, yanıt gövdesindeki `{"code":N}` yapılandırılmış hata kodudur; paket, `ServerErrBadRequest`(1000) ile `ServerErrValidation`(1010) arasında sabitler sağlar. `Error()`, dize eşleştirmesi kullanan tüketicilerle uyumluluk için `"HTTP error <status>: <body>"` eski biçimini korur.

### Akış katmanı: `SSEStreamError` (`client/stream_errors.go`)

Sunucu SSE akışı üzerinden sonlandırıcı bir hata çerçevesi (`response_type=error, done=true`) gönderdiğinde, SDK **önce bu çerçeveyi geri çağırıma iletir**, ardından `*SSEStreamError` döndürür:

```go
type SSEStreamError struct {
    Content string // Hata çerçevesi içeriği
}
```

Kontrol yöntemi (ikisi eşdeğerdir, ilki önerilir):

```go
// Yöntem 1: nöbetçi hata (SSEStreamError.Unwrap() bunu döndürür)
if errors.Is(err, client.ErrSSEStreamTerminal) { ... }

// Yöntem 2: yardımcı fonksiyon (eski fmt.Errorf("SSE stream error: ...") zinciriyle uyumlu)
if client.IsSSEStreamError(err) { ... }
```

## Günlükler ve izleme

`client/log.go`, `log/slog` tabanlı SDK dahili hata ayıklama günlükleri sağlar; varsayılan olarak `io.Discard` içine yazılır (kullanıcı için tamamen sessizdir). Gömülü kullanımda başlangıç sırasında şunu çağırabilirsiniz:

```go
client.SetDebugLevel("debug") // "debug"/"info"/"warn"; diğer değerler ("error", "" dahil) sessizdir
```

Günlükler stderr'e yazılır ve SSE'nin satır satır ayrıştırılması, istek hataları gibi trace bilgilerini içerir. Bu işlev **eşzamanlı kullanıma güvenli değildir**; herhangi bir SDK çağrısı başlatılmadan önce bir kez çağrılmalıdır.

İzleme için context içine `"RequestID"` (string) koyun; SDK bunu otomatik olarak `X-Request-ID` istek başlığı olarak gönderir (`client/client.go` içindeki `applyAuthHeaders` bölümüne bakın):

```go
ctx := context.WithValue(context.Background(), "RequestID", "req-20260727-0001")
```

## Tam örnek

Aşağıdaki örnek, `client/example.go` içindeki gerçek koddan uyarlanmıştır.

### Örnek bir: Bilgi tabanı oluşturma ve dosya yükleme

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/acrbaran/rag/client"
)

func main() {
    apiClient := client.NewClient(
        "http://localhost:8080",
        client.WithAPIKey("your-api-key"),
        client.WithTimeout(30*time.Second),
    )

    // Bilgi tabanı oluştur
    kb := &client.KnowledgeBase{
        Name:        "Test Knowledge Base",
        Description: "This is a test knowledge base",
        ChunkingConfig: client.ChunkingConfig{
            ChunkSize:    500,
            ChunkOverlap: 50,
            Separators:   []string{"\n\n", "\n", ". ", "? ", "! "},
        },
        EmbeddingModelID: "embedding_model_id",
        SummaryModelID:   "summary_model_id",
    }
    createdKB, err := apiClient.CreateKnowledgeBase(context.Background(), kb)
    if err != nil {
        fmt.Printf("Failed to create knowledge base: %v\n", err)
        return
    }
    fmt.Printf("Knowledge base created: ID=%s, Name=%s\n", createdKB.ID, createdKB.Name)

    // Dosya yükleyerek bilgi oluştur
    metadata := map[string]string{"source": "local", "type": "document"}
    knowledge, err := apiClient.CreateKnowledgeFromFile(
        context.Background(), createdKB.ID, "path/to/sample.pdf",
        metadata, nil, "", "", nil)
    if err != nil {
        fmt.Printf("Failed to upload knowledge file: %v\n", err)
        return
    }
    fmt.Printf("File uploaded: Knowledge ID=%s, Title=%s\n", knowledge.ID, knowledge.Title)
}
```

### Örnek iki: Oturum oluşturma ve akışlı bilgi soru-cevap

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "strings"

    "github.com/acrbaran/rag/client"
)

func main() {
    apiClient := client.NewClient("http://localhost:8080",
        client.WithAPIKey("your-api-key"))

    // Oturum oluştur
    session, err := apiClient.CreateSession(context.Background(), &client.CreateSessionRequest{
        Title:       "Test Session",
        Description: "A test session for knowledge Q&A",
    })
    if err != nil {
        fmt.Printf("Failed to create session: %v\n", err)
        return
    }

    // Akışlı soru-cevap: yanıtı ve alıntıları biriktir
    question := "What is artificial intelligence?"
    var answer strings.Builder
    var references []*client.SearchResult

    err = apiClient.KnowledgeQAStream(context.Background(),
        session.ID,
        &client.KnowledgeQARequest{Query: question},
        func(response *client.StreamResponse) error {
            if response.ResponseType == client.ResponseTypeAnswer {
                answer.WriteString(response.Content)
            }
            if response.Done && len(response.KnowledgeReferences) > 0 {
                references = response.KnowledgeReferences
            }
            return nil
        })
    if err != nil {
        // SSE sonlandırma hata çerçevesini diğer hatalardan ayır
        if errors.Is(err, client.ErrSSEStreamTerminal) {
            fmt.Printf("Stream terminated by server error: %v\n", err)
        } else {
            fmt.Printf("Q&A failed: %v\n", err)
        }
        return
    }
    fmt.Printf("Answer: %s\n", answer.String())
    for i, ref := range references {
        fmt.Printf("Reference %d: %s\n", i+1, ref.Content)
    }
}
```

### Örnek Üç: Geçmiş Mesajlar ile Chunk Yönetimi ve Kaynak Temizleme

```go
package main

import (
    "context"
    "fmt"

    "github.com/acrbaran/rag/client"
)

func main() {
    apiClient := client.NewClient("http://localhost:8080",
        client.WithAPIKey("your-api-key"))
    ctx := context.Background()

    // Son 10 oturum mesajını al
    sessionID := "your-session-id"
    messages, err := apiClient.GetRecentMessages(ctx, sessionID, 10)
    if err != nil {
        fmt.Printf("Failed to get session messages: %v\n", err)
    } else {
        for i, msg := range messages {
            fmt.Printf("%d. Role: %s, Content: %s\n", i+1, msg.Role, msg.Content)
        }
    }

    // Bilgi chunk'larını yönet: sayfalı listele ve ilkini güncelle
    knowledgeID := "your-knowledge-id"
    chunks, total, err := apiClient.ListKnowledgeChunks(ctx, knowledgeID, 1, 10)
    if err != nil {
        fmt.Printf("Failed to get knowledge chunks: %v\n", err)
    } else {
        fmt.Printf("Knowledge has %d chunks, retrieved %d\n", total, len(chunks))
        if len(chunks) > 0 {
            updated, err := apiClient.UpdateChunk(ctx, knowledgeID, chunks[0].ID,
                &client.UpdateChunkRequest{
                    Content:   "Updated chunk content - " + chunks[0].Content,
                    IsEnabled: true,
                })
            if err != nil {
                fmt.Printf("Failed to update chunk: %v\n", err)
            } else {
                fmt.Printf("Chunk updated: ID=%s\n", updated.ID)
            }
        }
    }

    // Kaynakları temizle
    if err := apiClient.DeleteSession(ctx, sessionID); err != nil {
        fmt.Printf("Failed to delete session: %v\n", err)
    }
    if err := apiClient.DeleteKnowledge(ctx, knowledgeID); err != nil {
        fmt.Printf("Failed to delete knowledge: %v\n", err)
    }
}
```

## Kaynak Kod Referansı

- İstemci çekirdeği ve hata türleri: `client/client.go`
- Kimlik doğrulama: `client/auth.go`
- Akışlı soru-cevap: `client/session.go`, `client/agent.go`
- Akış hataları: `client/stream_errors.go`
- Günlükler: `client/log.go`
- Tam kullanım örneği: `client/example.go`
