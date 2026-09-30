# Rethra HTTP İstemcisi

Bu paket, Rethra hizmetiyle etkileşim kurmak için bir istemci kütüphanesi sağlar. HTTP tabanlı tüm arayüz çağrılarını destekler; böylece diğer modüller HTTP istek kodunu doğrudan yazmadan Rethra hizmetini kolayca entegre edebilir.

## Temel Özellikler

İstemci şu temel işlev modüllerini içerir:

1. **Oturum yönetimi**: oturum oluşturma, getirme, güncelleme ve silme
2. **Bilgi tabanı yönetimi**: bilgi tabanı oluşturma, getirme, güncelleme ve silme
3. **Bilgi yönetimi**: bilgi içeriği ekleme, getirme ve silme
4. **Alan yönetimi**: alanlar için CRUD işlemleri
5. **Bilgi tabanlı soru-cevap**: normal ve akışlı soru-cevap desteği
6. **Agent soru-cevap**: düşünme süreci, araç çağrıları ve yansıtma içeren Agent tabanlı akıllı soru-cevap desteği
7. **Parça yönetimi**: bilgi parçalarını sorgulama, güncelleme ve silme
8. **Mesaj yönetimi**: oturum mesajlarını getirme ve silme
9. **Model yönetimi**: model oluşturma, getirme, güncelleme ve silme
10. **Sanal alan becerileri**: sanal alan yapılandırmasına beceri kurma (zip yükleme veya ClawHub / SkillHub / GitHub gibi kaynaklardan) ve becerilerin ihtiyaç duyduğu ortam değişkenlerini yapılandırma
11. **Uzun süreli bellek**: geçerli kullanıcının oturumlar arası belleği (ayar anahtarı, kayıt ekleme/silme/düzenleme, onaylama/reddetme, konular, belge yakınlığı, dışa aktarma, anında düzenleme)
12. **Kimlik doğrulama**: giriş, token yenileme, etkin alanı değiştirme (`SwitchTenant` son etkin kiracı tercihini kaydeder)

## Kullanım

### İstemci örneği oluşturma

```go
import (
    "context"
    "github.com/acrbaran/rag/client"
    "time"
)

// İstemci örneği oluştur
apiClient := client.NewClient(
    "http://api.example.com", 
    client.WithToken("your-auth-token"),
    client.WithTimeout(30*time.Second),
)
```

### Alan yapılandırması

İstemci, `WithTenantID` ile varsayılan alan ayarlamayı destekler; istekler otomatik olarak `X-Tenant-ID` başlığını taşır:

```go
tenantID := uint64(10000)
apiClient := client.NewClient(
    "http://api.example.com",
    client.WithToken("your-auth-token"),
    client.WithTenantID(tenantID),
)
```

Bir isteğin alanı geçici olarak değiştirmesi gerekiyorsa `context` içinde `TenantID` ayarlanabilir. Değer `uint64`, `*uint64` veya sayı içeren bir dize olabilir; istemci öncelikle bu değeri kullanır:

```go
ctx := context.WithValue(context.Background(), "TenantID", uint64(10000))
// Herhangi bir istemci yöntemini çağırırken ctx verildiğinde 10000 numaralı alana geçilir
```

### Örnek: bilgi tabanı oluşturma ve dosya yükleme

```go
// Bilgi tabanı oluştur
kb := &client.KnowledgeBase{
    Name:        "Test bilgi tabanı",
    Description: "Bu bir test bilgi tabanıdır",
    ChunkingConfig: client.ChunkingConfig{
        ChunkSize:    500,
        ChunkOverlap: 50,
        Separators:   []string{"\n\n", "\n", ". ", "? ", "! "},
    },
    ImageProcessingConfig: client.ImageProcessingConfig{
        ModelID: "image_model_id",
    },
    EmbeddingModelID: "embedding_model_id",
    SummaryModelID:   "summary_model_id",
}

kb, err := apiClient.CreateKnowledgeBase(context.Background(), kb)
if err != nil {
    // Hatayı işle
}

// Bilgi dosyası yükle ve meta veri ekle
metadata := map[string]string{
    "source": "local",
    "type":   "document",
}
knowledge, err := apiClient.CreateKnowledgeFromFile(context.Background(), kb.ID, "path/to/file.pdf", metadata)
if err != nil {
    // Hatayı işle
}

// Aynı bilgi tabanındaki orijinal dosyaları ZIP olarak paketleyip indir (en fazla 200 ID, toplam 512 MiB)
err = apiClient.DownloadKnowledgeFiles(context.Background(), kb.ID, []string{knowledge.ID}, "knowledge-files.zip")
if err != nil {
    // Hatayı işle
}
```

### Örnek: oturum oluşturma ve soru-cevap

```go
// Oturum oluştur
sessionRequest := &client.CreateSessionRequest{
    KnowledgeBaseID: knowledgeBaseID,
    SessionStrategy: &client.SessionStrategy{
        MaxRounds:        10,
        EnableRewrite:    true,
        FallbackStrategy: "fixed_answer",
        FallbackResponse: "Üzgünüm, bu soruyu yanıtlayamıyorum",
        EmbeddingTopK:    5,
        KeywordThreshold: 0.5,
        VectorThreshold:  0.7,
        RerankModelID:    "rerank_model_id",
        RerankTopK:       3,
        RerankThreshold:  0.8,
        SummaryModelID:   "summary_model_id",
    },
}

session, err := apiClient.CreateSession(context.Background(), sessionRequest)
if err != nil {
    // Hatayı işle
}

// Normal soru-cevap
answer, err := apiClient.KnowledgeQA(context.Background(), session.ID, &client.KnowledgeQARequest{
    Query: "Yapay zekâ nedir?",
})
if err != nil {
    // Hatayı işle
}

// Akışlı soru-cevap
err = apiClient.KnowledgeQAStream(context.Background(), session.ID, &client.KnowledgeQARequest{
    Query:            "Makine öğrenmesi nedir?",
    KnowledgeBaseIDs: []string{knowledgeBaseID}, // İsteğe bağlı: bilgi tabanı belirt
    WebSearchEnabled: false,                      // İsteğe bağlı: web aramasını etkinleştir
}, func(response *client.StreamResponse) error {
    // Her yanıt parçasını işle
    fmt.Print(response.Content)
    return nil
})
if err != nil {
    // Hatayı işle
}
```

### Örnek: Agent ile akıllı soru-cevap

Agent soru-cevap daha güçlü akıllı sohbet yetenekleri sunar; araç çağrısını, düşünme sürecinin gösterilmesini ve öz yansıtmayı destekler.

```go
// Agent oturumu oluştur
agentSession := apiClient.NewAgentSession(session.ID)

// Tüm olayları işleyerek Agent soru-cevap yap
err := agentSession.Ask(context.Background(), "Makine öğrenmesiyle ilgili bilgileri ara ve ana noktaları özetle", 
    func(resp *client.AgentStreamResponse) error {
        switch resp.ResponseType {
        case client.AgentResponseTypeThinking:
            // Agent düşünüyor
            if resp.Done {
                fmt.Printf("💭 Düşünme: %s\n", resp.Content)
            }
        
        case client.AgentResponseTypeToolCall:
            // Agent araç çağırıyor
            if resp.Data != nil {
                toolName := resp.Data["tool_name"]
                fmt.Printf("🔧 Araç çağrısı: %v\n", toolName)
            }
        
        case client.AgentResponseTypeToolResult:
            // Araç çalıştırma sonucu
            fmt.Printf("✓ Araç sonucu: %s\n", resp.Content)
        
        case client.AgentResponseTypeReferences:
            // Bilgi referansları
            if resp.KnowledgeReferences != nil {
                fmt.Printf("📚 %d ilgili bilgi bulundu\n", len(resp.KnowledgeReferences))
                for _, ref := range resp.KnowledgeReferences {
                    fmt.Printf("  - [%.3f] %s\n", ref.Score, ref.KnowledgeTitle)
                }
            }
        
        case client.AgentResponseTypeAnswer:
            // Nihai yanıt (akışlı çıktı)
            fmt.Print(resp.Content)
            if resp.Done {
                fmt.Println() // Bitince yeni satıra geç
            }
        
        case client.AgentResponseTypeReflection:
            // Agent'ın öz yansıtması
            if resp.Done {
                fmt.Printf("🤔 Yansıtma: %s\n", resp.Content)
            }
        
        case client.AgentResponseTypeError:
            // Hata bilgisi
            fmt.Printf("❌ Hata: %s\n", resp.Content)
        }
        return nil
    })

if err != nil {
    // Hatayı işle
}

// Basit sürüm: yalnızca nihai yanıtla ilgilen
var finalAnswer string
err = agentSession.Ask(context.Background(), "Derin öğrenme nedir?", 
    func(resp *client.AgentStreamResponse) error {
        if resp.ResponseType == client.AgentResponseTypeAnswer {
            finalAnswer += resp.Content
        }
        return nil
    })
```

### Agent olay türleri

| Olay türü | Açıklama | Ne zaman tetiklenir |
|---------|------|---------|
| `AgentResponseTypeThinking` | Agent düşünme süreci | Agent soruyu analiz edip plan yaparken |
| `AgentResponseTypeToolCall` | Araç çağrısı | Agent bir aracı kullanmaya karar verdiğinde |
| `AgentResponseTypeToolResult` | Araç çalıştırma sonucu | Araç çalışması tamamlandıktan sonra |
| `AgentResponseTypeReferences` | Bilgi referansları | İlgili bilgi bulunduğunda |
| `AgentResponseTypeAnswer` | Nihai yanıt | Agent yanıt üretirken (akışlı) |
| `AgentResponseTypeArtifactsPending` | Oluşturulan dosyalar yükleniyor | Yanıt bittikten sonra, dosyalar nesne depolamaya yazılmadan önce |
| `AgentResponseTypeReflection` | Öz yansıtma | Agent kendi yanıtını değerlendirirken |
| `AgentResponseTypeError` | Hata | Hata oluştuğunda |

### Agent soru-cevap test aracı

Agent işlevlerini test etmek için etkileşimli bir komut satırı aracı sunuyoruz:

```bash
cd client/cmd/agent_test
go build -o agent_test
./agent_test -url http://localhost:8080 -kb <knowledge_base_id>
```

Araç şunları destekler:
- Oturum oluşturma ve yönetme
- Etkileşimli Agent soru-cevap
- Tüm Agent olaylarını gerçek zamanlı gösterme
- Performans istatistikleri ve hata ayıklama bilgileri

Ayrıntılı kullanım için `client/cmd/agent_test/README.md` dosyasına bakın.

### Agent soru-cevabın gelişmiş kullanımı

Daha fazla gelişmiş kullanım örneği için `agent_example.go` dosyasına bakın. Şunları içerir:
- Temel Agent soru-cevap
- Araç çağrısı takibi
- Bilgi referanslarını yakalama
- Tüm olayların takibi
- Özel hata işleme
- Akış iptal denetimi
- Çoklu oturum yönetimi

```

### Örnek: model yönetimi

```go
// Model oluştur
modelRequest := &client.CreateModelRequest{
    Name:        "Test modeli",
    Type:        client.ModelTypeChat,
    Source:      client.ModelSourceInternal,
    Description: "Bu bir test modelidir",
    Parameters: client.ModelParameters{
        "temperature": 0.7,
        "top_p":       0.9,
    },
    IsDefault: true,
}
model, err := apiClient.CreateModel(context.Background(), modelRequest)
if err != nil {
    // Hatayı işle
}

// Tüm modelleri listele
models, err := apiClient.ListModels(context.Background())
if err != nil {
    // Hatayı işle
}
```

### Örnek: bilgi parçalarını yönetme

```go
// Bilgi parçalarını listele
chunks, total, err := apiClient.ListKnowledgeChunks(context.Background(), knowledgeID, 1, 10)
if err != nil {
    // Hatayı işle
}

// Parçayı güncelle
updateRequest := &client.UpdateChunkRequest{
    Content:   "Güncellenmiş parça içeriği",
    IsEnabled: true,
}
updatedChunk, err := apiClient.UpdateChunk(context.Background(), knowledgeID, chunkID, updateRequest)
if err != nil {
    // Hatayı işle
}
```

### Örnek: bilgiyi yeniden ayrıştırma

```go
// Bilgiyi yeniden ayrıştır (mevcut içeriği silip yeniden ayrıştırır)
// Uygun senaryolar:
// 1. İlk ayrıştırma başarısız oldu, yeniden denenmesi gerekiyor
// 2. Ayrıştırma yapılandırması güncellendi (ör. parçalama stratejisi, çok kipli ayarlar), yeniden ayrıştırma gerekiyor
// 3. Bilgi içeriği güncellendi, ayrıştırma sonucunun yenilenmesi gerekiyor

knowledge, err := apiClient.ReparseKnowledge(context.Background(), knowledgeID)
if err != nil {
    // Hatayı işle
}

// Bilgi "pending" durumuna geçer ve asenkron olarak yeniden ayrıştırılır
fmt.Printf("Knowledge ID: %s\n", knowledge.ID)
fmt.Printf("Parse Status: %s\n", knowledge.ParseStatus)      // "pending"
fmt.Printf("Enable Status: %s\n", knowledge.EnableStatus)    // "disabled"

// Ayrıştırma durumu yoklanarak kontrol edilebilir
for {
    time.Sleep(5 * time.Second)
    knowledge, err := apiClient.GetKnowledge(context.Background(), knowledgeID)
    if err != nil {
        // Hatayı işle
    }
    
    if knowledge.ParseStatus == "completed" {
        fmt.Println("Knowledge re-parsing completed!")
        break
    } else if knowledge.ParseStatus == "failed" {
        fmt.Printf("Knowledge re-parsing failed: %s\n", knowledge.ErrorMessage)
        break
    }
}
```

### Örnek: ayrıştırmayı iptal etme

```go
// Devam eden ayrıştırma görevini iptal et (kaynaklar kısıtlıyken / yanlış dosya yüklendiğinde kullanılır)
// - completed / failed durumundaki bilgi iptal edilemez
// - Yazılmış parçalar/dizinler korunur; daha sonra ReparseKnowledge ile yeniden ayrıştırılabilir

knowledge, err := apiClient.CancelKnowledgeParse(context.Background(), knowledgeID)
if err != nil {
    // Hatayı işle
}
fmt.Printf("Parse Status: %s\n", knowledge.ParseStatus) // "cancelled"
```

### Örnek: belge ayrıştırma izini görüntüleme (Span ağacı)

```go
// Belge ayrıştırma hattının Span ağacını getir (root → stage → subspan)
// - attempt için 0 verilirse en son ayrıştırma denemesi getirilir
// - Her zaman 5 standart aşama döner: docreader / chunking / embedding / multimodal / postprocess
trace, err := apiClient.GetKnowledgeProcessingSpans(context.Background(), knowledgeID, 0)
if err != nil {
    // Hatayı işle
}
fmt.Printf("ParseStatus=%s CurrentStage=%s\n", trace.ParseStatus, trace.CurrentStage)
for _, stage := range trace.Trace.Children {
    fmt.Printf("- %s: %s (%dms)\n", stage.Name, stage.Status, stage.DurationMs)
}
```

### Örnek: oturum mesajlarını getirme

```go
// Son mesajları getir
messages, err := apiClient.GetRecentMessages(context.Background(), sessionID, 10)
if err != nil {
    // Hatayı işle
}

// Belirtilen zamandan önceki mesajları getir
beforeTime := time.Now().Add(-24 * time.Hour)
olderMessages, err := apiClient.GetMessagesBefore(context.Background(), sessionID, beforeTime, 10)
if err != nil {
    // Hatayı işle
}
```

### Örnek: barındırılan platformdan sanal alan becerisi kurma

`source` açıkça yazılmalıdır: ClawHub için `@owner/slug`, ClawHub üzerindeki skills.sh kayıtları için tam `https://clawhub.ai/skills-sh/owner/repo/slug` veya `skills-sh:owner/repo/slug`, GitHub / SkillHub için tam URL yapıştırın. Yalın `owner/slug` göndermeyin.

```go
skillID, err := apiClient.InstallSandboxSkillFromSource(
    context.Background(), sandboxConfigID, "@owner/slug")
if err != nil {
    // Hatayı işle
}
_ = skillID // skillID ile /sandbox-configs/{id}/skills/{skillID}/install-events olaylarına abone olun
```

### Örnek: takılan kurulumu durdurma

Hizmet yeniden başlatıldıktan sonra kurulum satırı `installing` durumunda takılı kalabilir ve arayüzden yeniden denenemez veya kaldırılamaz. Durdurma bu satırı hemen günceller (süreç içinde hâlâ goroutine varsa o da iptal edilir); ardından yeniden deneme veya kaldırma çağrılabilir.

```go
skill, err := apiClient.StopSandboxSkill(context.Background(), sandboxConfigID, skillID)
if err != nil {
    // Hatayı işle
}
_ = skill
```

### Örnek: başarısız kurulumu yeniden deneme

Kurulum hatalarının nedeni çoğu zaman kurulum paketiyle ilgisizdir (sanal alana ulaşılamaması, bağımlılık kaynağında zaman aşımı). Sunucu orijinal kurulum paketini sakladığı için yeniden denerken paketi tekrar göndermek gerekmez.

```go
skillID, err := apiClient.ReinstallSandboxSkill(context.Background(), sandboxConfigID, skillID)
if err != nil {
    // Hatayı işle
}
```

### Örnek: kurulu becerinin dosyalarını görüntüleme

```go
files, err := apiClient.ListSandboxSkillFiles(context.Background(), sandboxConfigID, skillID)
if err != nil {
    // Hatayı işle
}
content, err := apiClient.GetSandboxSkillFile(context.Background(), sandboxConfigID, skillID, "SKILL.md")
if err != nil {
    // Hatayı işle
}
_ = files
_ = content
```

### Örnek: beceri ortam değişkenlerini yapılandırma

Beceri kurulurken hangi ortam değişkenlerine ihtiyaç duyduğunu bildirir. Değerler iki katmanlıdır: alan düzeyindeki değerler yönetici tarafından ayarlanır ve herkes için geçerlidir; kişisel düzeydeki değerler yalnızca **geçerli çağıran kimlik** için geçerlidir ve alan düzeyini geçersiz kılar. Hiçbir arayüz kaydedilmiş değeri geri okumaz, yalnızca ayarlanıp ayarlanmadığını bildirir.

API Key ile çağırmak ve web üzerinden giriş yapmak iki farklı kimliktir: web arayüzünde girilen kişisel değerler API Key ile başlatılan çalıştırmalara uygulanmaz. Entegrasyon senaryolarında alan düzeyindeki değerleri tercih edin.

```go
// Alan düzeyi: bu alandaki herkes için geçerlidir, Admin veya üzeri yetki gerekir
skill, err := apiClient.SetSandboxSkillEnvValues(
    context.Background(), sandboxConfigID, skillID,
    map[string]string{"TAVILY_API_KEY": "tvly-xxxxx"})
if err != nil {
    // Hatayı işle
}

// Kişisel düzey: yalnızca geçerli çağıran kimlik için geçerlidir
err = apiClient.SetMySkillEnvVar(
    context.Background(), skillID, "TAVILY_API_KEY", "tvly-yyyyy")
if err != nil {
    // Hatayı işle
}

// Hangi değişkenlerin henüz doldurulmadığını görüntüle. Bir değeri temizlemek için boş dize yazmak yerine Delete kullanın
groups, err := apiClient.ListMyEnvVars(context.Background())
if err != nil {
    // Hatayı işle
}
_ = skill
_ = groups
```

## Eksiksiz Örnek

İstemcinin tüm kullanım akışını gösteren `example.go` dosyasındaki `ExampleUsage` fonksiyonuna bakın.