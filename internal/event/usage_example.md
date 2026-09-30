# Olay sistemi kullanım örnekleri

## Olay sistemini Chat Pipeline'a entegre etme

### 1. Servis başlatılırken olay veri yolunu ayarlama

```go
// internal/container/container.go veya main.go

import (
    "github.com/acrbaran/rag/internal/event"
)

func InitializeEventSystem() {
    // Genel olay veri yolunu al
    bus := event.GetGlobalEventBus()
    
    // İzleme işleyicisini kaydet
    event.NewMonitoringHandler(bus)
    
    // Analiz işleyicisini kaydet
    event.NewAnalyticsHandler(bus)
    
    // Veya özel işleyici kaydet
    bus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        // Özel işleme mantığı
        return nil
    })
}
```

### 2. Sorgu işleme servisinde olay gönderme

#### Örnek: search.go içine olay ekleme

```go
// internal/application/service/chat_pipline/search.go

import (
    "github.com/acrbaran/rag/internal/event"
    "time"
)

func (p *PluginSearch) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Arama başladı olayını gönder
    startTime := time.Now()
    event.Emit(ctx, event.NewEvent(event.EventRetrievalStart, event.RetrievalData{
        Query:           chatManage.ProcessedQuery,
        KnowledgeBaseID: chatManage.KnowledgeBaseID,
        TopK:            chatManage.EmbeddingTopK,
        RetrievalType:   "vector",
    }).WithSessionID(chatManage.SessionID))
    
    // Arama mantığını çalıştır
    results, err := p.performSearch(ctx, chatManage)
    if err != nil {
        // Hata olayını gönder
        event.Emit(ctx, event.NewEvent(event.EventError, event.ErrorData{
            Error:     err.Error(),
            Stage:     "retrieval",
            SessionID: chatManage.SessionID,
            Query:     chatManage.ProcessedQuery,
        }).WithSessionID(chatManage.SessionID))
        return ErrSearch.WithError(err)
    }
    
    // Arama tamamlandı olayını gönder
    event.Emit(ctx, event.NewEvent(event.EventRetrievalComplete, event.RetrievalData{
        Query:           chatManage.ProcessedQuery,
        KnowledgeBaseID: chatManage.KnowledgeBaseID,
        TopK:            chatManage.EmbeddingTopK,
        RetrievalType:   "vector",
        ResultCount:     len(results),
        Duration:        time.Since(startTime).Milliseconds(),
        Results:         results,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.SearchResult = results
    return next()
}
```

#### Örnek: rewrite.go içine olay ekleme

```go
// internal/application/service/chat_pipline/rewrite.go

func (p *PluginRewriteQuery) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Yeniden yazma başladı olayını gönder
    event.Emit(ctx, event.NewEvent(event.EventQueryRewrite, event.QueryData{
        OriginalQuery: chatManage.Query,
        SessionID:     chatManage.SessionID,
    }).WithSessionID(chatManage.SessionID))
    
    // Sorgu yeniden yazmayı çalıştır
    rewrittenQuery, err := p.rewriteQuery(ctx, chatManage)
    if err != nil {
        return ErrRewrite.WithError(err)
    }
    
    // Yeniden yazma tamamlandı olayını gönder
    event.Emit(ctx, event.NewEvent(event.EventQueryRewritten, event.QueryData{
        OriginalQuery:  chatManage.Query,
        RewrittenQuery: rewrittenQuery,
        SessionID:      chatManage.SessionID,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.RewriteQuery = rewrittenQuery
    return next()
}
```

#### Örnek: rerank.go içine olay ekleme

```go
// internal/application/service/chat_pipline/rerank.go

func (p *PluginRerank) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Sıralama başladı olayını gönder
    startTime := time.Now()
    inputCount := len(chatManage.SearchResult)
    
    event.Emit(ctx, event.NewEvent(event.EventRerankStart, event.RerankData{
        Query:      chatManage.ProcessedQuery,
        InputCount: inputCount,
        ModelID:    chatManage.RerankModelID,
    }).WithSessionID(chatManage.SessionID))
    
    // Sıralamayı çalıştır
    rerankResults, err := p.performRerank(ctx, chatManage)
    if err != nil {
        return ErrRerank.WithError(err)
    }
    
    // Sıralama tamamlandı olayını gönder
    event.Emit(ctx, event.NewEvent(event.EventRerankComplete, event.RerankData{
        Query:       chatManage.ProcessedQuery,
        InputCount:  inputCount,
        OutputCount: len(rerankResults),
        ModelID:     chatManage.RerankModelID,
        Duration:    time.Since(startTime).Milliseconds(),
        Results:     rerankResults,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.RerankResult = rerankResults
    return next()
}
```

#### Örnek: chat_completion.go içine olay ekleme

```go
// internal/application/service/chat_pipline/chat_completion.go

func (p *PluginChatCompletion) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // Sohbet başladı olayını gönder
    startTime := time.Now()
    event.Emit(ctx, event.NewEvent(event.EventChatStart, event.ChatData{
        Query:    chatManage.Query,
        ModelID:  chatManage.ChatModelID,
        IsStream: false,
    }).WithSessionID(chatManage.SessionID))
    
    // Modeli ve mesajları hazırla
    chatModel, opt, err := prepareChatModel(ctx, p.modelService, chatManage)
    if err != nil {
        return ErrGetChatModel.WithError(err)
    }
    
    chatMessages := prepareMessagesWithHistory(chatManage)
    
    // Modeli çağır
    chatResponse, err := chatModel.Chat(ctx, chatMessages, opt)
    if err != nil {
        event.Emit(ctx, event.NewEvent(event.EventError, event.ErrorData{
            Error:     err.Error(),
            Stage:     "chat_completion",
            SessionID: chatManage.SessionID,
            Query:     chatManage.Query,
        }).WithSessionID(chatManage.SessionID))
        return ErrModelCall.WithError(err)
    }
    
    // Sohbet tamamlandı olayını gönder
    event.Emit(ctx, event.NewEvent(event.EventChatComplete, event.ChatData{
        Query:      chatManage.Query,
        ModelID:    chatManage.ChatModelID,
        Response:   chatResponse.Content,
        TokenCount: chatResponse.TokenCount,
        Duration:   time.Since(startTime).Milliseconds(),
        IsStream:   false,
    }).WithSessionID(chatManage.SessionID))
    
    chatManage.ChatResponse = chatResponse
    return next()
}
```

### 3. Handler katmanında istek alındı olayı gönderme

```go
// internal/handler/message.go

func (h *MessageHandler) SendMessage(c *gin.Context) {
    ctx := c.Request.Context()
    
    // İsteği ayrıştır
    var req types.SendMessageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Sorgu alındı olayını gönder
    event.Emit(ctx, event.NewEvent(event.EventQueryReceived, event.QueryData{
        OriginalQuery: req.Content,
        SessionID:     req.SessionID,
        UserID:        c.GetString("user_id"),
    }).WithSessionID(req.SessionID).WithRequestID(c.GetString("request_id")))
    
    // Mesajı işle...
}
```

### 4. Özel izleme işleyicisi

```go
// internal/monitoring/event_monitor.go

package monitoring

import (
    "context"
    "github.com/acrbaran/rag/internal/event"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    retrievalDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "retrieval_duration_milliseconds",
            Help: "Duration of retrieval operations",
        },
        []string{"knowledge_base_id", "retrieval_type"},
    )
    
    rerankDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "rerank_duration_milliseconds",
            Help: "Duration of rerank operations",
        },
        []string{"model_id"},
    )
)

func init() {
    prometheus.MustRegister(retrievalDuration)
    prometheus.MustRegister(rerankDuration)
}

func SetupEventMonitoring() {
    bus := event.GetGlobalEventBus()
    
    // Arama performansını izle
    bus.On(event.EventRetrievalComplete, func(ctx context.Context, e event.Event) error {
        data := e.Data.(event.RetrievalData)
        retrievalDuration.WithLabelValues(
            data.KnowledgeBaseID,
            data.RetrievalType,
        ).Observe(float64(data.Duration))
        return nil
    })
    
    // Sıralama performansını izle
    bus.On(event.EventRerankComplete, func(ctx context.Context, e event.Event) error {
        data := e.Data.(event.RerankData)
        rerankDuration.WithLabelValues(data.ModelID).Observe(float64(data.Duration))
        return nil
    })
}
```

### 5. Günlük kaydı işleyicisi

```go
// internal/logging/event_logger.go

package logging

import (
    "context"
    "encoding/json"
    "github.com/acrbaran/rag/internal/event"
    "github.com/acrbaran/rag/internal/logger"
)

func SetupEventLogging() {
    bus := event.GetGlobalEventBus()
    
    // Tüm olaylar için yapılandırılmış günlük kaydı
    logHandler := event.ApplyMiddleware(
        func(ctx context.Context, e event.Event) error {
            data, _ := json.Marshal(e.Data)
            logger.Infof(ctx, "Event: type=%s, session=%s, request=%s, data=%s",
                e.Type, e.SessionID, e.RequestID, string(data))
            return nil
        },
        event.WithTiming(),
    )
    
    // Tüm kritik olaylara kaydet
    bus.On(event.EventQueryReceived, logHandler)
    bus.On(event.EventQueryRewritten, logHandler)
    bus.On(event.EventRetrievalComplete, logHandler)
    bus.On(event.EventRerankComplete, logHandler)
    bus.On(event.EventChatComplete, logHandler)
    bus.On(event.EventError, logHandler)
}
```

### 6. Tam başlatma akışı

```go
// cmd/server/main.go veya internal/container/container.go

func Initialize() {
    // 1. Olay sistemini başlat
    eventBus := event.GetGlobalEventBus()
    
    // 2. İzlemeyi ayarla
    event.NewMonitoringHandler(eventBus)
    
    // 3. Analizi ayarla
    event.NewAnalyticsHandler(eventBus)
    
    // 4. Prometheus izlemesini ayarla (gerekirse)
    // monitoring.SetupEventMonitoring()
    
    // 5. Yapılandırılmış günlüğü ayarla (gerekirse)
    // logging.SetupEventLogging()
    
    // 6. Diğer başlatma işlemleri...
}
```

## Olay sistemini test etme

```go
// Testlerde bağımsız bir olay veri yolu kullanın
func TestMyService(t *testing.T) {
    ctx := context.Background()
    
    // Teste özel olay veri yolu oluştur
    testBus := event.NewEventBus()
    
    // Test dinleyicisini kaydet
    var receivedEvents []event.Event
    testBus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        receivedEvents = append(receivedEvents, e)
        return nil
    })
    
    // Testi çalıştır...
    testBus.Emit(ctx, event.NewEvent(event.EventQueryReceived, event.QueryData{
        OriginalQuery: "test",
    }))
    
    // Olayları doğrula
    if len(receivedEvents) != 1 {
        t.Errorf("Expected 1 event, got %d", len(receivedEvents))
    }
}
```

## Asenkron işleme örneği

```go
// Ana akışı etkilemeyen olaylar için asenkron mod kullanılabilir
func SetupAsyncAnalytics() {
    asyncBus := event.NewAsyncEventBus()
    
    asyncBus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        // Ana akışı engellemeden analiz platformuna asenkron gönder
        // sendToAnalyticsPlatform(e)
        return nil
    })
    
    // Olayı asenkron veri yoluyla gönder
    // asyncBus.Emit(ctx, event)
}
```

## Performans iyileştirme önerileri

1. **Kritik yolda senkron olay veri yolundan kaçının**: İş mantığını etkilemeyen izleme, günlük vb. için asenkron mod kullanın
2. **Ara katmanları yerinde kullanın**: Gereksiz yükten kaçınmak için ara katmanları yalnızca gereken yerlerde kullanın
3. **Olay veri boyutunu denetleyin**: Özellikle asenkron modda olaylarla büyük miktarda veri taşımaktan kaçının
4. **Özel dinleyiciler kullanın**: Tek bir dinleyicide çok fazla iş yapmayın, tek sorumluluk ilkesini koruyun

