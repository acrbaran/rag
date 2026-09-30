package event

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/acrbaran/rag/internal/logger"
)

// EventType represents the type of event in the system
type EventType string

const (
	// Query processing events
	EventQueryReceived   EventType = "query.received"   // Kullanıcı sorgusu ulaştı
	EventQueryValidated  EventType = "query.validated"  // Sorgu doğrulaması tamamlandı
	EventQueryPreprocess EventType = "query.preprocess" // Sorgu ön işleme
	EventQueryRewrite    EventType = "query.rewrite"    // Sorgu yeniden yazma
	EventQueryRewritten  EventType = "query.rewritten"  // Sorgu yeniden yazma tamamlandı

	// Retrieval events
	EventRetrievalStart    EventType = "retrieval.start"    // Getirme başladı
	EventRetrievalVector   EventType = "retrieval.vector"   // Vektör getirme
	EventRetrievalKeyword  EventType = "retrieval.keyword"  // Anahtar kelime getirme
	EventRetrievalEntity   EventType = "retrieval.entity"   // Varlık getirme
	EventRetrievalComplete EventType = "retrieval.complete" // Getirme tamamlandı

	// Rerank events
	EventRerankStart    EventType = "rerank.start"    // Sıralama başladı
	EventRerankComplete EventType = "rerank.complete" // Sıralama tamamlandı

	// Merge events
	EventMergeStart    EventType = "merge.start"    // Birleştirme başladı
	EventMergeComplete EventType = "merge.complete" // Birleştirme tamamlandı

	// Chat completion events
	EventChatStart    EventType = "chat.start"    // Sohbet oluşturma başladı
	EventChatComplete EventType = "chat.complete" // Sohbet oluşturma tamamlandı
	EventChatStream   EventType = "chat.stream"   // Sohbet akışlı çıktısı

	// Agent events
	EventAgentQuery    EventType = "agent.query"    // Agent sorgusu başladı
	EventAgentPlan     EventType = "agent.plan"     // Agent plan oluşturma
	EventAgentStep     EventType = "agent.step"     // Agent adım yürütme
	EventAgentTool     EventType = "agent.tool"     // Agent araç çağrısı
	EventAgentComplete EventType = "agent.complete" // Agent tamamlandı

	// Agent streaming events (for real-time feedback)
	EventAgentThought       EventType = "thought"        // Agent düşünme süreci
	EventAgentCommandOutput EventType = "command_output" // bounded command output
	EventAgentToolCall      EventType = "tool_call"      // Araç çağrısı bildirimi
	EventAgentToolResult    EventType = "tool_result"    // Araç sonucu
	EventAgentReflection    EventType = "reflection"     // Agent değerlendirmesi
	EventAgentReferences    EventType = "references"     // Bilgi alıntısı
	EventAgentFinalAnswer   EventType = "final_answer"   // Nihai yanıt

	// MCP tool human approval (issue #1173)
	EventToolApprovalRequired EventType = "tool_approval_required"
	EventToolApprovalResolved EventType = "tool_approval_resolved"

	// MCP OAuth in-conversation authorization prompt: emitted when an
	// OAuth-enabled MCP service is invoked but the current user has not
	// authorized it yet. The agent pauses until the user authorizes (or the
	// wait times out / is canceled).
	EventMCPOAuthRequired EventType = "mcp_oauth_required"
	EventMCPOAuthResolved EventType = "mcp_oauth_resolved"

	// Error events
	EventError EventType = "error" // Hata olayı

	// Long-term memory recalled for this turn. Emitted once, before the answer
	// streams, so the UI can show which memories the answer saw.
	EventMemoryRecalled EventType = "memory_recalled"

	// EventContextCompacted is emitted when older conversation was summarized
	// away to fit the context window.
	EventContextCompacted EventType = "context_compacted"

	// EventUserMessageInjected is emitted when a message the user appended
	// while the run was in flight was accepted into the running turn (see
	// agent drainSteerMessages).
	EventUserMessageInjected EventType = "user_message_injected"

	// Session events
	EventSessionTitle EventType = "session_title" // Oturum başlığı güncellemesi

	// Control events
	EventStop EventType = "stop" // Diyalog oluşturmayı durdur
)

// Event represents an event in the system
type Event struct {
	ID        string                 // Olay kimliği (akış güncellemesi takibi için otomatik oluşturulan UUID)
	Type      EventType              // Olay türü
	SessionID string                 // Oturum kimliği
	Data      interface{}            // Olay verileri
	Metadata  map[string]interface{} // Olay meta verileri
	RequestID string                 // İstek kimliği
}

// EventHandler is a function that handles events
type EventHandler func(ctx context.Context, event Event) error

// EventBus manages event publishing and subscription
type EventBus struct {
	mu        sync.RWMutex
	handlers  map[EventType][]EventHandler
	asyncMode bool // Olayın eşzamansız olarak işlenip işlenmeyeceği
}

// NewEventBus creates a new EventBus instance
func NewEventBus() *EventBus {
	return &EventBus{
		handlers:  make(map[EventType][]EventHandler),
		asyncMode: false,
	}
}

// NewAsyncEventBus creates a new EventBus with async mode enabled
func NewAsyncEventBus() *EventBus {
	return &EventBus{
		handlers:  make(map[EventType][]EventHandler),
		asyncMode: true,
	}
}

// On registers an event handler for a specific event type
// Multiple handlers can be registered for the same event type
func (eb *EventBus) On(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// Off removes all handlers for a specific event type
func (eb *EventBus) Off(eventType EventType) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.handlers, eventType)
}

// Emit publishes an event to all registered handlers
// Returns error if any handler fails (in sync mode)
// Automatically generates an ID for the event if not provided (from source)
func (eb *EventBus) Emit(ctx context.Context, event Event) error {
	// Auto-generate ID if not provided (from source)
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	eb.mu.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		// No handlers registered for this event type
		return nil
	}

	if eb.asyncMode {
		// Async mode: fire and forget
		for _, handler := range handlers {
			h := handler // capture loop variable
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Errorf(ctx, "event handler panic recovered (type=%s): %v", event.Type, r)
					}
				}()
				_ = h(ctx, event)
			}()
		}
		return nil
	}

	// Sync mode: execute handlers sequentially
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("event handler failed for %s: %w", event.Type, err)
		}
	}

	return nil
}

// EmitAndWait publishes an event and waits for all handlers to complete
// This method works in both sync and async mode
// Automatically generates an ID for the event if not provided (from source)
func (eb *EventBus) EmitAndWait(ctx context.Context, event Event) error {
	// Auto-generate ID if not provided (from source)
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	eb.mu.RLock()
	handlers, exists := eb.handlers[event.Type]
	eb.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		h := handler // capture loop variable

		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("event handler panic (type=%s): %v", event.Type, r)
				}
			}()
			if err := h(ctx, event); err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	for err := range errChan {
		if err != nil {
			return fmt.Errorf("event handler failed for %s: %w", event.Type, err)
		}
	}

	return nil
}

// HasHandlers checks if there are any handlers registered for an event type
func (eb *EventBus) HasHandlers(eventType EventType) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlers, exists := eb.handlers[eventType]
	return exists && len(handlers) > 0
}

// GetHandlerCount returns the number of handlers for a specific event type
func (eb *EventBus) GetHandlerCount(eventType EventType) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if handlers, exists := eb.handlers[eventType]; exists {
		return len(handlers)
	}
	return 0
}

// Clear removes all event handlers
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers = make(map[EventType][]EventHandler)
}
