# Rethra olay sistemi özeti

## Genel bakış

Rethra projesi için eksiksiz bir olay gönderme ve dinleme mekanizması oluşturuldu; kullanıcı sorgusu işleme akışındaki her adım için olay işlemeyi destekler.

## Temel özellikler

### ✅ Uygulanan özellikler

1. **Olay veri yolu (EventBus)**
   - `Emit(ctx, event)` - olay gönderir
   - `On(eventType, handler)` - olay dinleyicisi kaydeder
   - `Off(eventType)` - olay dinleyicisini kaldırır
   - `EmitAndWait(ctx, event)` - olay gönderir ve tüm işleyicilerin bitmesini bekler
   - Eşzamanlı ve eşzamansız iki mod

2. **Olay türleri**
   - Sorgu işleme olayları (alma, doğrulama, ön işleme, yeniden yazma)
   - Arama olayları (başlangıç, vektör araması, anahtar sözcük araması, varlık araması, tamamlanma)
   - Sıralama olayları (başlangıç, tamamlanma)
   - Birleştirme olayları (başlangıç, tamamlanma)
   - Sohbet üretimi olayları (başlangıç, tamamlanma, akışlı çıktı)
   - Hata olayları

3. **Olay veri yapıları**
   - `QueryData` - sorgu verisi
   - `RetrievalData` - arama verisi
   - `RerankData` - sıralama verisi
   - `MergeData` - birleştirme verisi
   - `ChatData` - sohbet verisi
   - `ErrorData` - hata verisi

4. **Ara katman desteği**
   - `WithLogging()` - günlük kaydı ara katmanı
   - `WithTiming()` - zamanlama ara katmanı
   - `WithRecovery()` - hata kurtarma ara katmanı
   - `Chain()` - ara katmanları birleştirme

5. **Genel olay veri yolu**
   - Singleton modelinde genel olay veri yolu
   - Genel kolaylık fonksiyonları (`On`, `Emit`, `EmitAndWait` vb.)

6. **Örnekler ve testler**
   - Eksiksiz birim testleri
   - Performans kıyaslama testleri
   - Eksiksiz kullanım örnekleri
   - Gerçek senaryo gösterimleri

## Dosya yapısı

```
internal/event/
├── event.go                    # Temel olay veri yolu uygulaması
├── event_data.go              # Olay veri yapısı tanımları
├── middleware.go              # Ara katman uygulaması
├── global.go                  # Genel olay veri yolu
├── integration_example.go     # Entegrasyon örneği (izleme, analiz işleyicileri)
├── example_test.go            # Testler ve örnekler
├── demo/
│   └── main.go               # Eksiksiz RAG akışı gösterimi
├── README.md                 # Ayrıntılı belge
├── usage_example.md          # Kullanım örnekleri belgesi
└── SUMMARY.md                # Bu belge
```

## Performans ölçümleri

- **Olay gönderme performansı**: ~9 nanosaniye/çağrı (kıyaslama testi)
- **Eşzamanlılık güvenliği**: İş parçacığı güvenliği `sync.RWMutex` ile sağlanır
- **Bellek yükü**: Çok düşük; yalnızca olay işleyici fonksiyon referansları saklanır

## Kullanım senaryoları

### 1. İzleme ve ölçüm toplama

```go
bus.On(event.EventRetrievalComplete, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.RetrievalData)
    // Prometheus'a ya da başka bir izleme sistemine gönder
    metricsCollector.RecordRetrievalDuration(data.Duration)
    return nil
})
```

### 2. Günlük kaydı

```go
bus.On(event.EventQueryRewritten, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.QueryData)
    logger.Infof(ctx, "Query rewritten: %s -> %s", 
        data.OriginalQuery, data.RewrittenQuery)
    return nil
})
```

### 3. Kullanıcı davranışı analizi

```go
bus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.QueryData)
    // Analiz platformuna gönder
    analytics.TrackQuery(data.UserID, data.OriginalQuery)
    return nil
})
```

### 4. Hata izleme

```go
bus.On(event.EventError, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.ErrorData)
    // Hata izleme sistemine gönder
    sentry.CaptureException(data.Error)
    return nil
})
```

## Entegrasyon yöntemi

### Adım 1: Olay sistemini başlatma

Uygulama başlarken (örneğin `main.go` ya da `container.go` içinde):

```go
import "github.com/acrbaran/rag/internal/event"

func Initialize() {
    // Genel olay veri yolunu al
    bus := event.GetGlobalEventBus()
    
    // İzleme ve analizi ayarla
    event.NewMonitoringHandler(bus)
    event.NewAnalyticsHandler(bus)
}
```

### Adım 2: Her işleme aşamasında olay gönderme

Sorgu işleme akışındaki eklentilere olay gönderimi ekleyin:

```go
// search.go içinde
event.Emit(ctx, event.NewEvent(event.EventRetrievalStart, event.RetrievalData{
    Query:           chatManage.ProcessedQuery,
    KnowledgeBaseID: chatManage.KnowledgeBaseID,
    TopK:            chatManage.EmbeddingTopK,
}).WithSessionID(chatManage.SessionID))

// rerank.go içinde
event.Emit(ctx, event.NewEvent(event.EventRerankComplete, event.RerankData{
    Query:       chatManage.ProcessedQuery,
    InputCount:  len(chatManage.SearchResult),
    OutputCount: len(rerankResults),
    Duration:    time.Since(startTime).Milliseconds(),
}).WithSessionID(chatManage.SessionID))
```

### Adım 3: Özel olay işleyicileri kaydetme

Gerektiğinde özel işleyiciler kaydedin:

```go
event.On(event.EventQueryRewritten, func(ctx context.Context, e event.Event) error {
    // Özel işleme mantığı
    return nil
})
```

## Avantajlar

1. **Gevşek bağlılık**: Olay gönderenler ve dinleyiciler tamamen ayrıştırılmıştır; bakım ve genişletme kolaydır
2. **Yüksek performans**: Çok düşük performans yükü (~9 nanosaniye/çağrı)
3. **Esneklik**: Eşzamanlı/eşzamansız, tek/çoklu dinleyici desteği
4. **Genişletilebilirlik**: Yeni olay türleri ve işleyiciler kolayca eklenir
5. **Tür güvenliği**: Önceden tanımlanmış olay veri yapıları
6. **Ara katman desteği**: Kesişen kaygıları (günlük, zamanlama, hata işleme vb.) eklemeyi kolaylaştırır
7. **Test dostu**: Olay davranışı testlerde kolayca doğrulanır

## Test sonuçları

✅ Tüm birim testleri geçti
✅ Performans testi geçti (~9 nanosaniye/çağrı)
✅ Eşzamansız işleme testi geçti
✅ Çoklu işleyici testi geçti
✅ Tam akış gösterimi başarılı

## Sonraki öneriler

### İsteğe bağlı iyileştirmeler

1. **Olay kalıcılığı**: Önemli olayları veritabanına ya da mesaj kuyruğuna kaydetme
2. **Olay yeniden oynatma**: Hata ayıklama ya da analiz için olayları yeniden oynatma desteği
3. **Olay filtreleme**: Daha karmaşık olay filtreleme ve yönlendirme desteği
4. **Öncelik kuyruğu**: Olay önceliğine göre işleme desteği
5. **Dağıtık olaylar**: Mesaj kuyruğu üzerinden servisler arası olay desteği

### Entegrasyon önerileri

1. **İzleme entegrasyonu**: Ölçüm toplamak için Prometheus entegrasyonu
2. **Günlük entegrasyonu**: Birleşik yapılandırılmış günlük kaydı
3. **İzleme (tracing) entegrasyonu**: Mevcut tracing sistemiyle entegrasyon
4. **Uyarı entegrasyonu**: Olay tabanlı uyarı mekanizması

## Örnek çıktı

`go run ./internal/event/demo/main.go` çalıştırıldığında tam RAG akışının olay çıktısı görülebilir:

```
Step 1: Query Received
[MONITOR] Query received - Session: session-xxx, Query: RAG teknolojisi nedir?
[ANALYTICS] Query tracked - User: user-123, Session: session-xxx

Step 2: Query Rewriting
[MONITOR] Query rewrite started
[MONITOR] Query rewritten - Original: RAG teknolojisi nedir?, Rewritten: Erişimle zenginleştirilmiş üretim teknolojisi...
[CUSTOM] Query Transformation: ...

Step 3: Vector Retrieval
[MONITOR] Retrieval started - Type: vector, TopK: 20
[MONITOR] Retrieval completed - Results: 18, Duration: 301ms
[CUSTOM] Retrieval Efficiency: Rate: 90.00%

Step 4: Result Reranking
[MONITOR] Rerank started - Input: 18
[MONITOR] Rerank completed - Output: 5, Duration: 201ms
[CUSTOM] Rerank Statistics: Reduction: 72.22%

Step 5: Chat Completion
[MONITOR] Chat generation started
[MONITOR] Chat generation completed - Tokens: 256, Duration: 801ms
[ANALYTICS] Chat metrics - Model: gpt-4, Tokens: 256
```

## Özet

Olay sistemi tamamen uygulanmış ve testlerle doğrulanmıştır; sorgu işleme akışının her aşamasını izlemek, günlüğe kaydetmek, analiz etmek ve hata ayıklamak için hemen Rethra projesine entegre edilebilir. Sistem sade tasarımlı, yüksek performanslı, kullanımı ve genişletmesi kolaydır.

