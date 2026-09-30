package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/acrbaran/rag/internal/application/access"
	"github.com/acrbaran/rag/internal/application/repository"
	"github.com/acrbaran/rag/internal/application/service/retriever"
	werrors "github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/models/embedding"
	"github.com/acrbaran/rag/internal/tracing/langfuse"
	"github.com/acrbaran/rag/internal/types"
	secutils "github.com/acrbaran/rag/internal/utils"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// UpsertFAQEntries imports or appends FAQ entries asynchronously.
// Returns task ID (UUID) for tracking import progress.
func (s *knowledgeService) UpsertFAQEntries(ctx context.Context,
	kbID string, payload *types.FAQBatchUpsertPayload,
) (string, error) {
	if payload == nil || len(payload.Entries) == 0 {
		return "", werrors.NewBadRequestError(types.LocalizedText(ctx, "SSS kayıtları boş olamaz", "FAQ entries cannot be empty"))
	}
	if payload.Mode == "" {
		payload.Mode = types.FAQBatchModeAppend
	}
	if payload.Mode != types.FAQBatchModeAppend && payload.Mode != types.FAQBatchModeReplace {
		return "", werrors.NewBadRequestError(types.LocalizedText(ctx, "Mod yalnızca append veya replace olabilir", "Mode must be append or replace"))
	}

	// Bilgi tabanının mevcut ve geçerli olduğunu doğrula
	kb, ctx, err := s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return "", err
	}
	if err := s.validateFAQImportTags(ctx, kb, payload.Entries); err != nil {
		return "", err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)

	// İletilen TaskID'yi kullan; iletilmemişse geliştirilmiş bir TaskID oluştur.
	// İstemcinin ilettiği task_id dosya adına ve Redis key'e girer; yol ayırıcıları içermeyen bir tanımlayıcı olmalıdır.
	taskID := strings.TrimSpace(payload.TaskID)
	if taskID == "" {
		taskID = secutils.GenerateTaskID("faq_import", tenantID, kbID)
	} else if err := secutils.ValidateTaskID(taskID); err != nil {
		return "", werrors.NewBadRequestError(types.LocalizedText(ctx, "task_id biçimi geçersiz", "Invalid task_id format"))
	}

	var knowledgeID string

	// Devam eden içe aktarma görevi olup olmadığını kontrol et (Redis aracılığıyla)
	runningTaskID, err := s.getRunningFAQImportTaskID(ctx, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to check running import task: %v", err)
		// Kontrolün başarısız olması içe aktarmayı etkilemez, yürütmeye devam et
	} else if runningTaskID != "" {
		logger.Warnf(ctx, "Import task already running for KB %s: %s", kbID, runningTaskID)
		return "", werrors.NewBadRequestError(fmt.Sprintf(types.LocalizedText(ctx,
			"Bu bilgi tabanında bir içe aktarma görevi sürüyor (görev kimliği: %s); tamamlanmasını bekleyin",
			"An import task is already running for this knowledge base (task ID: %s); wait for it to finish"), runningTaskID))
	}

	// FAQ knowledge'ın mevcut olduğundan emin ol
	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return "", fmt.Errorf("failed to ensure FAQ knowledge: %w", err)
	}
	knowledgeID = faqKnowledge.ID

	// Görevin kuyruğa alınma zamanını kaydet
	enqueuedAt := time.Now().Unix()
	instanceID := uuid.NewString()
	runningInfoSet := false
	enqueueSucceeded := false

	// KB'nin çalışan görev bilgilerini ayarla
	if err := s.setRunningFAQImportInfo(ctx, kbID, &runningFAQImportInfo{
		TaskID:     taskID,
		EnqueuedAt: enqueuedAt,
		InstanceID: instanceID,
	}); err != nil {
		logger.Errorf(ctx, "Failed to set running FAQ import task info: %v", err)
		// Görev yürütmesini etkilemez, devam et
	} else {
		runningInfoSet = true
	}
	defer func() {
		if !runningInfoSet || enqueueSucceeded {
			return
		}
		if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, kbID, taskID, instanceID, enqueuedAt); clearErr != nil {
			logger.Warnf(ctx, "Failed to clear FAQ import running info after setup failure: %v", clearErr)
		}
	}()

	// İçe aktarma görevi durumunu Redis'e başlat
	progress := &types.FAQImportProgress{
		TaskID:        taskID,
		KBID:          kbID,
		KnowledgeID:   knowledgeID,
		Status:        types.FAQImportStatusPending,
		Progress:      0,
		Total:         len(payload.Entries),
		Processed:     0,
		SuccessCount:  0,
		FailedCount:   0,
		FailedEntries: make([]types.FAQFailedEntry, 0),
		Message:       types.LocalizedText(ctx, "Görev oluşturuldu, işlenmeyi bekliyor", "Task created, awaiting processing"),
		CreatedAt:     time.Now().Unix(),
		UpdatedAt:     time.Now().Unix(),
		DryRun:        payload.DryRun,
	}
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Errorf(ctx, "Failed to initialize FAQ import task status: %v", err)
		return "", fmt.Errorf("failed to initialize task: %w", err)
	}

	logger.Infof(ctx, "FAQ import task initialized: %s, kb_id: %s, total entries: %d, dry_run: %v",
		taskID, kbID, len(payload.Entries), payload.DryRun)

	// Enqueue FAQ import task to Asynq
	logger.Info(ctx, "Enqueuing FAQ import task to Asynq")

	// Görev payload'ını oluştur
	taskPayload := types.FAQImportPayload{
		Language:    types.LanguageFromContextOrDefault(ctx),
		TenantID:    tenantID,
		TaskID:      taskID,
		KBID:        kbID,
		KnowledgeID: knowledgeID,
		Mode:        payload.Mode,
		DryRun:      payload.DryRun,
		EnqueuedAt:  enqueuedAt,
		InstanceID:  instanceID,
		Initiator:   types.TaskInitiatorFromContext(ctx),
	}

	// Eşik: 200 kaydı aşarsa veya serileştirildikten sonra 50KB'ı aşarsa nesne depolama kullan
	const (
		entryCountThreshold  = 200
		payloadSizeThreshold = 50 * 1024 // 50KB
	)

	entryCount := len(payload.Entries)
	if entryCount > entryCountThreshold {
		// Veri miktarı büyük, nesne depolamaya yükle
		entriesData, err := json.Marshal(payload.Entries)
		if err != nil {
			logger.Errorf(ctx, "Failed to marshal FAQ entries: %v", err)
			return "", fmt.Errorf("failed to marshal entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries size: %d bytes, uploading to object storage", len(entriesData))

		// Özel kovaya (ana kova) yükle, görev işlendikten sonra temizle
		fileName, err := faqImportEntriesFileName(taskID, enqueuedAt)
		if err != nil {
			return "", fmt.Errorf("invalid task id for object name: %w", err)
		}
		entriesURL, err := s.fileSvc.SaveBytes(ctx, entriesData, tenantID, fileName, false)
		if err != nil {
			logger.Errorf(ctx, "Failed to upload FAQ entries to object storage: %v", err)
			return "", fmt.Errorf("failed to upload entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries uploaded to: %s", entriesURL)
		taskPayload.EntriesURL = entriesURL
		taskPayload.EntryCount = entryCount
	} else {
		// Veri miktarı küçük, doğrudan payload içinde depola
		taskPayload.Entries = payload.Entries
	}

	langfuse.InjectTracing(ctx, &taskPayload)
	payloadBytes, err := json.Marshal(taskPayload)
	if err != nil {
		logger.Errorf(ctx, "Failed to marshal FAQ import task payload: %v", err)
		return "", fmt.Errorf("failed to marshal task payload: %w", err)
	}

	// payload boyutunu yeniden kontrol et
	if len(payloadBytes) > payloadSizeThreshold && taskPayload.EntriesURL == "" {
		// payload çok büyük ancak henüz yüklenmedi, şimdi yükle
		entriesData, _ := json.Marshal(payload.Entries)
		fileName, nameErr := faqImportEntriesFileName(taskID, enqueuedAt)
		if nameErr != nil {
			return "", fmt.Errorf("invalid task id for object name: %w", nameErr)
		}
		entriesURL, err := s.fileSvc.SaveBytes(ctx, entriesData, tenantID, fileName, false)
		if err != nil {
			logger.Errorf(ctx, "Failed to upload FAQ entries to object storage: %v", err)
			return "", fmt.Errorf("failed to upload entries: %w", err)
		}

		logger.Infof(ctx, "FAQ entries uploaded to (size exceeded): %s", entriesURL)
		taskPayload.Entries = nil
		taskPayload.EntriesURL = entriesURL
		taskPayload.EntryCount = entryCount

		payloadBytes, _ = json.Marshal(taskPayload)
	}

	logger.Infof(ctx, "FAQ import task payload size: %d bytes", len(payloadBytes))

	maxRetry := 5
	if payload.DryRun {
		maxRetry = 3 // dry run yeniden deneme sayısını biraz daha az tut
	}

	// asynq için benzersiz görev kimliği olarak taskID:instanceID kullan
	asynqTaskID := fmt.Sprintf("%s:%s", taskID, instanceID)

	task := asynq.NewTask(
		types.TypeFAQImport,
		payloadBytes,
		asynq.TaskID(asynqTaskID),
		asynq.Queue(types.QueueMaintenance),
		asynq.MaxRetry(maxRetry),
		asynq.Timeout(2*time.Hour),
	)
	info, err := s.task.Enqueue(task)
	if err != nil {
		logger.Errorf(ctx, "Failed to enqueue FAQ import task: %v", err)
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}
	logger.Infof(ctx, "Enqueued FAQ import task: id=%s queue=%s task_id=%s dry_run=%v", info.ID, info.Queue, taskID, payload.DryRun)
	enqueueSucceeded = true

	if !payload.DryRun {
		recordKBActivity(ctx, s.audit, tenantID, kbID, types.AuditActionFAQImportStarted,
			"faq_entry", knowledgeID, types.AuditOutcomeAccepted,
			map[string]any{
				"task_id": taskID, "mode": payload.Mode, "total": len(payload.Entries),
				"trigger": kbActivityTrigger(ctx), "processing_status": "pending",
			})
	}

	return taskID, nil
}

func faqImportEntriesFileName(taskID string, enqueuedAt int64) (string, error) {
	return secutils.SafeFileName(fmt.Sprintf("faq_import_entries_%s_%d.json", taskID, enqueuedAt))
}

// generateFailedEntriesCSV başarısız kayıtların CSV dosyasını oluşturur ve yükler
func (s *knowledgeService) generateFailedEntriesCSV(ctx context.Context,
	tenantID uint64, taskID string, failedEntries []types.FAQFailedEntry,
) (string, error) {
	// CSV içeriğini oluştur
	var buf strings.Builder

	// Excel'in UTF-8'i doğru tanımasını desteklemek için BOM yaz
	buf.WriteString("\xEF\xBB\xBF")

	// Başlık satırını yaz
	buf.WriteString("error_reason,tag_name,standard_question,similar_questions,negative_questions,answers,reply_all,is_disabled\n")

	// Veri satırlarını yaz
	for _, entry := range failedEntries {
		// CSV kaçış işlemi: içerik virgül, tırnak veya satır sonu içeriyorsa tırnak içine alınmalı ve iç tırnaklar kaçırılmalıdır
		reason := csvEscape(entry.Reason)
		tagName := csvEscape(entry.TagName)
		standardQ := csvEscape(entry.StandardQuestion)
		similarQs := ""
		if len(entry.SimilarQuestions) > 0 {
			similarQs = csvEscape(strings.Join(entry.SimilarQuestions, "##"))
		}
		negativeQs := ""
		if len(entry.NegativeQuestions) > 0 {
			negativeQs = csvEscape(strings.Join(entry.NegativeQuestions, "##"))
		}
		answers := ""
		if len(entry.Answers) > 0 {
			answers = csvEscape(strings.Join(entry.Answers, "##"))
		}
		answerAll := "false"
		if entry.AnswerAll {
			answerAll = "true"
		}
		isDisabled := "false"
		if entry.IsDisabled {
			isDisabled = "true"
		}

		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
			reason, tagName, standardQ, similarQs, negativeQs, answers, answerAll, isDisabled))
	}

	// CSV dosyasını geçici depolamaya yükle (otomatik olarak süresi dolar)
	fileName, err := secutils.SafeFileName(fmt.Sprintf("faq_dryrun_failed_%s.csv", taskID))
	if err != nil {
		return "", fmt.Errorf("invalid task id for object name: %w", err)
	}
	filePath, err := s.fileSvc.SaveBytes(ctx, []byte(buf.String()), tenantID, fileName, true)
	if err != nil {
		return "", fmt.Errorf("failed to save CSV file: %w", err)
	}

	// İndirme URL'sini al
	fileURL, err := s.fileSvc.GetFileURL(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file URL: %w", err)
	}

	logger.Infof(ctx, "Generated failed entries CSV: %s, entries: %d", fileURL, len(failedEntries))
	return fileURL, nil
}

// csvEscape CSV alanlarını kaçırır
func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		// İç tırnakları iki tırnakla değiştir ve alanın tamamını tırnak içine al
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// saveFAQImportResultToDatabase FAQ içe aktarma sonucu istatistiklerini veritabanına kaydeder
func (s *knowledgeService) saveFAQImportResultToDatabase(ctx context.Context,
	payload *types.FAQImportPayload, progress *types.FAQImportProgress, originalTotalEntries int,
) error {
	// FAQ bilgi tabanı örneğini al
	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, payload.KnowledgeID)
	if err != nil {
		return fmt.Errorf("failed to get FAQ knowledge: %w", err)
	}

	// Atlanan kayıt sayısını hesapla (toplam - tamamen başarılı - kısmen başarısız - tamamen başarısız)
	skippedCount := originalTotalEntries - progress.SuccessCount - progress.PartialFailedCount - progress.FailedCount
	if skippedCount < 0 {
		skippedCount = 0
	}

	// İçe aktarma sonucu istatistiklerini oluştur
	importResult := &types.FAQImportResult{
		TotalEntries:       originalTotalEntries,
		SuccessCount:       progress.SuccessCount,
		FailedCount:        progress.FailedCount,
		PartialFailedCount: progress.PartialFailedCount,
		SkippedCount:       skippedCount,
		MergedCount:        progress.MergedCount,
		AddedCount:         progress.AddedCount,
		ImportMode:         payload.Mode,
		ImportedAt:         time.Now(),
		TaskID:             payload.TaskID,
		ProcessingTime:     time.Now().Unix() - progress.CreatedAt, // İşleme süresi (saniye)
		DisplayStatus:      "open",                                 // Yeni içe aktarılan sonuçlar varsayılan olarak gösterilir
	}

	// Başarısız/kısmen başarısız öğeler için CSV indirme URL'si varsa sonucu yaz (FailedEntries tamamen başarısız ve kısmen başarısız olanları içerir)
	if progress.FailedEntriesURL != "" {
		importResult.FailedEntriesURL = progress.FailedEntriesURL
	}

	// İçe aktarma sonucunu Knowledge metadata'sına ayarla
	if err := knowledge.SetLastFAQImportResult(importResult); err != nil {
		return fmt.Errorf("failed to set FAQ import result: %w", err)
	}

	// Veritabanını güncelle
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		return fmt.Errorf("failed to update knowledge with import result: %w", err)
	}

	logger.Infof(ctx, "Saved FAQ import result to database: knowledge_id=%s, task_id=%s, total=%d, success=%d, added=%d, merged=%d, failed=%d, partial_failed=%d, skipped=%d",
		payload.KnowledgeID, payload.TaskID, originalTotalEntries, progress.SuccessCount, progress.AddedCount, progress.MergedCount, progress.FailedCount, progress.PartialFailedCount, skippedCount)

	return nil
}

// buildFAQFailedEntry, FAQFailedEntry oluşturur
func buildFAQFailedEntry(idx int, reason string, entry *types.FAQEntryPayload) types.FAQFailedEntry {
	answerAll := false
	if entry.AnswerStrategy != nil && *entry.AnswerStrategy == types.AnswerStrategyAll {
		answerAll = true
	}
	isDisabled := false
	if entry.IsEnabled != nil && !*entry.IsEnabled {
		isDisabled = true
	}
	return types.FAQFailedEntry{
		Index:             idx,
		Reason:            reason,
		TagName:           entry.TagName,
		StandardQuestion:  strings.TrimSpace(entry.StandardQuestion),
		SimilarQuestions:  entry.SimilarQuestions,
		NegativeQuestions: entry.NegativeQuestions,
		Answers:           entry.Answers,
		AnswerAll:         answerAll,
		IsDisabled:        isDisabled,
	}
}

func buildFAQPartialFailedEntry(ctx context.Context, idx int, entry *types.FAQEntryPayload,
	removedSimilarQuestions, removedNegativeQuestions []string,
) types.FAQFailedEntry {
	answerAll := false
	if entry.AnswerStrategy != nil && *entry.AnswerStrategy == types.AnswerStrategyAll {
		answerAll = true
	}
	isDisabled := false
	if entry.IsEnabled != nil && !*entry.IsEnabled {
		isDisabled = true
	}

	// Başarısızlık nedeni açıklamasını oluştur: özet bilgi + ayrıntılı bilgi
	var summary []string
	if len(removedSimilarQuestions) > 0 {
		summary = append(summary, fmt.Sprintf(types.LocalizedText(ctx, "%d benzer soru kaldırıldı", "%d similar questions removed"), len(removedSimilarQuestions)))
	}
	if len(removedNegativeQuestions) > 0 {
		summary = append(summary, fmt.Sprintf(types.LocalizedText(ctx, "%d olumsuz örnek kaldırıldı", "%d negative examples removed"), len(removedNegativeQuestions)))
	}

	// Tam reason: özet | benzer soru ayrıntıları | karşı örnek ayrıntıları
	var reasonParts []string
	reasonParts = append(reasonParts, types.LocalizedText(ctx, "Kısmen başarılı: ", "Partially successful: ")+strings.Join(summary, ", "))
	if len(removedSimilarQuestions) > 0 {
		reasonParts = append(reasonParts, strings.Join(removedSimilarQuestions, "; "))
	}
	if len(removedNegativeQuestions) > 0 {
		reasonParts = append(reasonParts, strings.Join(removedNegativeQuestions, "; "))
	}

	return types.FAQFailedEntry{
		Index:                    idx,
		Reason:                   strings.Join(reasonParts, " | "),
		IsPartialFailure:         true,
		TagName:                  entry.TagName,
		StandardQuestion:         strings.TrimSpace(entry.StandardQuestion),
		SimilarQuestions:         entry.SimilarQuestions,
		NegativeQuestions:        entry.NegativeQuestions,
		Answers:                  entry.Answers,
		AnswerAll:                answerAll,
		IsDisabled:               isDisabled,
		RemovedSimilarQuestions:  removedSimilarQuestions,
		RemovedNegativeQuestions: removedNegativeQuestions,
	}
}

// executeFAQDryRunValidation, FAQ dry run doğrulamasını gerçekleştirir ve doğrulamayı geçen öğe dizinlerini döndürür
func (s *knowledgeService) executeFAQDryRunValidation(ctx context.Context,
	payload *types.FAQImportPayload, progress *types.FAQImportProgress,
) []int {
	entries := payload.Entries

	// Temel doğrulama ve yinelenen denetimini geçen öğe dizinlerini kaydetmek için kullanılır; ardından güvenlik denetimi yapılır
	validEntryIndices := make([]int, 0, len(entries))

	// Moda göre farklı doğrulama mantığı seç
	if payload.Mode == types.FAQBatchModeAppend {
		validEntryIndices = s.validateEntriesForAppendModeWithProgress(ctx, payload.TenantID, payload.KBID, entries, progress)
	} else {
		validEntryIndices = s.validateEntriesForReplaceModeWithProgress(ctx, entries, progress)
	}

	return validEntryIndices
}

// validateEntriesForAppendModeWithProgress, Append modundaki öğeleri doğrular (ilerleme güncellemeleriyle)
// Not: Doğrulama aşamasında Processed güncellenmez; yalnızca gerçek içe aktarma sırasında güncellenir
// validateEntriesForAppendModeWithProgress, Append modundaki öğeleri doğrular (ilerleme güncellemeleriyle)
// Not: Doğrulama aşamasında Processed güncellenmez; yalnızca gerçek içe aktarma sırasında güncellenir
//
// Dört aşamada doğrula:
// Birinci aşama (ön doğrulama):
//  1. Standart soru - yalnızca dosya içinde yinelenenleri kaldır; mevcut bir KB standart sorusu varsa birleştirme adayı olarak işaretle
//  2. Benzer soru - dosya + bilgi tabanıyla karşılaştır (birleştirme adayları kendi mevcut sorularını hariç tutar) → tek tek çakışan benzer soruları kaldır
//  3. Karşı örnek - yalnızca mevcut QA'nın kendi standart sorusu + benzer sorularıyla karşılaştır → tek tek çakışan karşı örnekleri kaldır
//
// İkinci aşama (son doğrulama, yalnızca birleştirme adayları):
//  4. Birleştirilmiş tam veri üzerinde karşı örnek doğrulamasını yeniden çalıştır → çakışma varsa tüm öğeyi birleştirme öncesi durumuna geri al
func (s *knowledgeService) validateEntriesForAppendModeWithProgress(ctx context.Context,
	tenantID uint64, kbID string, entries []types.FAQEntryPayload, progress *types.FAQImportProgress,
) []int {
	totalEntries := len(entries)

	// Bilgi tabanındaki mevcut tüm FAQ chunks'larının metadata'sını sorgula
	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		logger.Warnf(ctx, "Failed to list existing FAQ chunks for dry run: %v", err)
	}

	// Mevcut standart soru → chunk eşlemesini oluştur (birleştirme adaylarını belirlemek için)
	existingStdQToChunk := make(map[string]*types.Chunk)
	// Mevcut tüm sorular → ait oldukları chunkID eşlemesini oluştur (benzer soru çakışmalarını tespit etmek için)
	existingQuestionToChunkID := make(map[string]string)
	// Her chunk'ın sahip olduğu soru kümesini oluştur (birleştirme sırasında kendisini hariç tutmak için)
	existingChunkQuestions := make(map[string]map[string]bool)
	// chunkID → standart soru eşlemesini oluştur (çakışma başarısızlık nedenini göstermek için)
	existingChunkIDToStdQ := make(map[string]string)

	for _, chunk := range existingChunks {
		meta, err := chunk.FAQMetadata()
		if err != nil || meta == nil {
			continue
		}
		qs := make(map[string]bool)
		if meta.StandardQuestion != "" {
			existingStdQToChunk[meta.StandardQuestion] = chunk
			existingQuestionToChunkID[meta.StandardQuestion] = chunk.ID
			qs[meta.StandardQuestion] = true
		}
		for _, q := range meta.SimilarQuestions {
			if q != "" {
				existingQuestionToChunkID[q] = chunk.ID
				qs[q] = true
			}
		}
		existingChunkQuestions[chunk.ID] = qs
		existingChunkIDToStdQ[chunk.ID] = meta.StandardQuestion
	}

	// Birleştirme adayı takibi: entry index → hedefteki mevcut chunk
	mergeChunkMap := make(map[int]*types.Chunk)

	// ==================== İlk yineleme: temel biçim doğrulaması + dosya içi standart soru tekilleştirme + birleştirme adayı belirleme ====================
	batchStandardQuestions := make(map[string]int) // value, ilk görünümün indeksidir
	validIndicesAfterStdQ := make([]int, 0, totalEntries)

	for i, entry := range entries {
		if err := validateFAQEntryPayloadBasic(ctx, &entry); err != nil {
			progress.FailedCount++
			fe := buildFAQFailedEntry(i, err.Error(), &entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			continue
		}

		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Dosya içi standart soru tekilleştirme
		if firstIdx, exists := batchStandardQuestions[standardQ]; exists {
			progress.FailedCount++
			fe := buildFAQFailedEntry(i, fmt.Sprintf(types.LocalizedText(ctx, "Ana soru çakışması: bu gruptaki %d. ana sorunun kopyası", "Main question conflict: duplicates main question %d in this batch"), firstIdx+1), &entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			continue
		}

		// Birleştirme adayı olup olmadığını belirle
		if chunk, exists := existingStdQToChunk[standardQ]; exists {
			// Standart soru KB içinde zaten mevcut → birleştirme adayı olarak işaretle
			mergeChunkMap[i] = chunk
			logger.Infof(ctx, "FAQ entry %d: standard question '%s' exists in KB, marking as merge candidate (chunk_id=%s)", i, standardQ, chunk.ID)
		} else if conflictChunkID, hit := existingQuestionToChunkID[standardQ]; hit {
			// Standart soru mevcut hiçbir standart soruyla tekrarlanmıyor, ancak KB içindeki diğer kayıtların benzer sorularıyla çakışıyor
			// → Aşağı akışta calculateAppendOperations tarafından sessizce atılmasının istatistikleri bozmasını önlemek için ön doğrulama başarısız
			conflictStdQ := existingChunkIDToStdQ[conflictChunkID]
			progress.FailedCount++
			fe := buildFAQFailedEntry(i,
				fmt.Sprintf(types.LocalizedText(ctx, `Ana soru çakışması: "%s" ana sorusunun "%s" benzer sorusuyla aynı`, `Main question conflict: question "%[2]s" duplicates a similar question of "%[1]s"`), conflictStdQ, standardQ),
				&entry)
			fe.FailureType = "pre_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			logger.Infof(ctx,
				"FAQ entry %d: standard question '%s' conflicts with existing similar question of chunk_id=%s (std='%s')",
				i, standardQ, conflictChunkID, conflictStdQ)
			continue
		}

		batchStandardQuestions[standardQ] = i
		validIndicesAfterStdQ = append(validIndicesAfterStdQ, i)

		if (i+1)%100 == 0 {
			progress.Message = fmt.Sprintf(types.LocalizedText(ctx, "Ana sorular doğrulanıyor %d/%d...", "Validating main questions %d/%d..."), i+1, totalEntries)
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== İkinci yineleme: benzer soru çakışması tespiti ====================
	// Toplu işlemdeki tüm standart sorular ve benzer sorulardan oluşan kümeyi oluştur
	batchAllQuestions := make(map[string]int)
	for _, i := range validIndicesAfterStdQ {
		standardQ := strings.TrimSpace(entries[i].StandardQuestion)
		batchAllQuestions[standardQ] = i
		for _, q := range entries[i].SimilarQuestions {
			q = strings.TrimSpace(q)
			if q != "" {
				if _, exists := batchAllQuestions[q]; !exists {
					batchAllQuestions[q] = i
				}
			}
		}
	}

	removedSimilarQuestionsMap := make(map[int][]string)
	removedNegativeQuestionsMap := make(map[int][]string)

	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Birleştirme adayı: hariç tutmak için hedef chunk'ın kendi soru kümesini al
		var ownChunkQuestions map[string]bool
		if mergeChunk, isMerge := mergeChunkMap[i]; isMerge {
			ownChunkQuestions = existingChunkQuestions[mergeChunk.ID]
		}

		validSimilarQuestions := make([]string, 0, len(entry.SimilarQuestions))
		var removedSimilarQuestions []string
		for _, q := range entry.SimilarQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// Benzer soru, kendi standart sorusuyla aynı olamaz
			if q == standardQ {
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(types.LocalizedText(ctx, `Benzer soru çakışması: "%s" bu kaydın ana sorusuyla aynı`, `Similar question conflict: "%s" duplicates this entry's main question`), q))
				continue
			}
			// Benzer soru doğrulaması: bilgi tabanındaki mevcut standart sorular + benzer sorularla karşılaştır
			if _, exists := existingQuestionToChunkID[q]; exists {
				// Birleştirme adayı: soru birleştirme hedefinin kendi chunk'ına aitse izin verilir (tekilleştirilmiş birleştirme sırasında işlenir)
				if ownChunkQuestions != nil && ownChunkQuestions[q] {
					validSimilarQuestions = append(validSimilarQuestions, q)
					continue
				}
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(types.LocalizedText(ctx, `Benzer soru çakışması: "%s" bilgi tabanındaki bir soruyla aynı`, `Similar question conflict: "%s" duplicates a question in the knowledge base`), q))
				continue
			}
			// Benzer soru doğrulaması: toplu işlemdeki standart sorular + benzer sorularla karşılaştır
			if firstIdx, exists := batchAllQuestions[q]; exists && firstIdx != i {
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(types.LocalizedText(ctx, `Benzer soru çakışması: "%s" %d. satırdaki bir soruyla aynı`, `Similar question conflict: "%s" duplicates a question in row %d`), q, firstIdx+1))
				continue
			}
			validSimilarQuestions = append(validSimilarQuestions, q)
		}
		entries[i].SimilarQuestions = validSimilarQuestions

		if len(removedSimilarQuestions) > 0 {
			removedSimilarQuestionsMap[i] = removedSimilarQuestions
		}

		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf(types.LocalizedText(ctx, "Benzer sorular doğrulanıyor %d/%d...", "Validating similar questions %d/%d..."), idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Üçüncü yineleme: karşı örnek çakışması tespiti (ön doğrulama, yalnızca yeni kaydın kendi verilerini kontrol eder) ====================
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		currentQAQuestions := make(map[string]bool)
		currentQAQuestions[standardQ] = true
		for _, q := range entry.SimilarQuestions {
			currentQAQuestions[q] = true
		}

		validNegativeQuestions := make([]string, 0, len(entry.NegativeQuestions))
		var removedNegativeQuestions []string
		for _, q := range entry.NegativeQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			if currentQAQuestions[q] {
				removedNegativeQuestions = append(removedNegativeQuestions, fmt.Sprintf(types.LocalizedText(ctx, `Olumsuz örnek çakışması: "%s" bu kaydın sorularından biriyle aynı`, `Negative example conflict: "%s" duplicates a question in this entry`), q))
				continue
			}
			validNegativeQuestions = append(validNegativeQuestions, q)
		}
		entries[i].NegativeQuestions = validNegativeQuestions

		if len(removedNegativeQuestions) > 0 {
			removedNegativeQuestionsMap[i] = removedNegativeQuestions
		}

		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf(types.LocalizedText(ctx, "Olumsuz örnekler doğrulanıyor %d/%d...", "Validating negative examples %d/%d..."), idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Dördüncü yineleme: sonradan doğrulama (yalnızca birleştirme adayları) ====================
	// Birleştirilmiş tam veri üzerinde karşı örnek doğrulamasını yeniden çalıştır; çakışma varsa tüm kaydı birleştirme öncesi duruma geri al
	postValidationFailed := make(map[int]bool)
	mergeCount := 0
	for _, i := range validIndicesAfterStdQ {
		mergeChunk, isMerge := mergeChunkMap[i]
		if !isMerge {
			continue
		}

		existingMeta, err := mergeChunk.FAQMetadata()
		if err != nil || existingMeta == nil {
			logger.Warnf(ctx, "FAQ entry %d: failed to get merge target metadata, skipping post-validation", i)
			continue
		}

		entry := &entries[i]

		// Birleştirme sonrası tam veriyi hesapla
		mergedSimilar := unionStrings(existingMeta.SimilarQuestions, entry.SimilarQuestions)
		mergedNegative := unionStrings(existingMeta.NegativeQuestions, entry.NegativeQuestions)

		// Birleştirme sonrası çakışma kümesini oluştur (standart sorular + birleştirme sonrası tüm benzer sorular)
		mergedPositiveSet := make(map[string]bool)
		mergedPositiveSet[existingMeta.StandardQuestion] = true
		for _, q := range mergedSimilar {
			mergedPositiveSet[q] = true
		}

		// Birleştirme sonrası her karşı örneğin, birleştirme sonrası standart sorular / benzer sorularla çakışıp çakışmadığını kontrol et
		var conflictingNegatives []string
		for _, q := range mergedNegative {
			if mergedPositiveSet[q] {
				conflictingNegatives = append(conflictingNegatives, q)
			}
		}

		if len(conflictingNegatives) > 0 {
			// Sonradan doğrulama başarısız → tüm kaydı birleştirme öncesi duruma geri al
			postValidationFailed[i] = true
			delete(mergeChunkMap, i)
			progress.FailedCount++
			reason := fmt.Sprintf(types.LocalizedText(ctx, "Birleştirme sonrası doğrulama başarısız: olumsuz örnekler %s benzer sorularla çakışıyor", "Post-merge validation failed: negative examples %s conflict with similar questions"), strings.Join(conflictingNegatives, ", "))
			fe := buildFAQFailedEntry(i, reason, entry)
			fe.FailureType = "post_validation"
			progress.FailedEntries = append(progress.FailedEntries, fe)
			logger.Infof(ctx, "FAQ entry %d: post-validation failed, conflicting negatives after merge: %v", i, conflictingNegatives)
		} else {
			mergeCount++
		}
	}

	// Sonradan doğrulaması başarısız kayıtları geçerli indekslerden kaldır
	if len(postValidationFailed) > 0 {
		filtered := make([]int, 0, len(validIndicesAfterStdQ))
		for _, i := range validIndicesAfterStdQ {
			if !postValidationFailed[i] {
				filtered = append(filtered, i)
			}
		}
		validIndicesAfterStdQ = filtered
	}

	// Kısmi başarısızlık bilgilerini FailedEntries'e ekle
	for _, i := range validIndicesAfterStdQ {
		removedSimilar := removedSimilarQuestionsMap[i]
		removedNegative := removedNegativeQuestionsMap[i]
		if len(removedSimilar) > 0 || len(removedNegative) > 0 {
			pf := buildFAQPartialFailedEntry(ctx, i, &entries[i], removedSimilar, removedNegative)
			progress.FailedEntries = append(progress.FailedEntries, pf)
			progress.PartialFailedCount++
		}
	}

	// Birleştirilmiş kayıt indeksini progress'e kaydet (yürütme aşaması ve yeniden deneme için)
	mergeIndices := make([]int, 0, mergeCount)
	for _, i := range validIndicesAfterStdQ {
		if _, isMerge := mergeChunkMap[i]; isMerge {
			mergeIndices = append(mergeIndices, i)
		}
	}
	progress.MergeEntryIndices = mergeIndices

	logger.Infof(ctx, "Append mode validation completed: total=%d, valid=%d, merge_candidates=%d, failed=%d, partial_failed=%d",
		totalEntries, len(validIndicesAfterStdQ), mergeCount, progress.FailedCount, progress.PartialFailedCount)

	return validIndicesAfterStdQ
}

// validateEntriesForReplaceModeWithProgress Replace modundaki girişleri doğrular (ilerleme güncellemeleriyle)
// Not: Doğrulama aşamasında Processed güncellenmez, yalnızca gerçek içe aktarma sırasında güncellenir
// Üç yinelemede doğrulama yapılır; böylece önce filtrelenen verilerin sonrakilerde dikkate alınmaması sağlanır:
// 1. Standart soru - Tüm standart sorularla karşılaştır → Tüm QA başarısız olur «standart soru çakışması»
// 2. Benzer soru - Tüm standart sorular+benzer sorularla karşılaştır → Tek soru ifadesi başarısız olur «benzer soru çakışması» (yalnızca çakışan benzer soru kaldırılır)
// 3. Karşı örnek - Mevcut QA altındaki tüm standart sorular+benzer sorularla karşılaştır → Tek soru ifadesi başarısız olur «karşı örnek çakışması» (yalnızca çakışan karşı örnek kaldırılır)
func (s *knowledgeService) validateEntriesForReplaceModeWithProgress(ctx context.Context,
	entries []types.FAQEntryPayload, progress *types.FAQImportProgress,
) []int {
	totalEntries := len(entries)

	// ==================== İlk yineleme: temel biçim doğrulaması + standart soru çakışması tespiti ====================
	// Standart soru çakışması, tüm QA'nın başarısız olmasına neden olur
	batchStandardQuestions := make(map[string]int) // value, ilk görünümün dizinidir
	validIndicesAfterStdQ := make([]int, 0, totalEntries)

	for i, entry := range entries {
		// Girişlerin temel biçimini doğrula
		if err := validateFAQEntryPayloadBasic(ctx, &entry); err != nil {
			progress.FailedCount++
			progress.FailedEntries = append(progress.FailedEntries, buildFAQFailedEntry(i, err.Error(), &entry))
			continue
		}

		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Standart soru doğrulaması: Tüm standart sorularla karşılaştır → Tüm QA başarısız olur «standart soru çakışması»
		if firstIdx, exists := batchStandardQuestions[standardQ]; exists {
			progress.FailedCount++
			progress.FailedEntries = append(progress.FailedEntries, buildFAQFailedEntry(i, fmt.Sprintf(types.LocalizedText(ctx, "Ana soru çakışması: bu gruptaki %d. ana sorunun kopyası", "Main question conflict: duplicates main question %d in this batch"), firstIdx+1), &entry))
			continue
		}

		// Standart soruyu kaydet
		batchStandardQuestions[standardQ] = i
		validIndicesAfterStdQ = append(validIndicesAfterStdQ, i)

		// İlerleme mesajını düzenli olarak güncelle
		if (i+1)%100 == 0 {
			progress.Message = fmt.Sprintf(types.LocalizedText(ctx, "Ana sorular doğrulanıyor %d/%d...", "Validating main questions %d/%d..."), i+1, totalEntries)
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== İkinci yineleme: benzer soru çakışması tespiti ====================
	// Yalnızca ilk doğrulamayı geçen girişler işlenir; benzer soru çakışmalarında yalnızca çakışan benzer soru kaldırılır
	// Tüm standart sorular+benzer sorulardan oluşan kümeyi oluştur (yalnızca ilk doğrulamayı geçen girişleri içerir)
	batchAllQuestions := make(map[string]int) // value, ilk görünümün dizinidir
	for _, i := range validIndicesAfterStdQ {
		standardQ := strings.TrimSpace(entries[i].StandardQuestion)
		batchAllQuestions[standardQ] = i
		for _, q := range entries[i].SimilarQuestions {
			q = strings.TrimSpace(q)
			if q != "" {
				// Yalnızca ilk görünümün konumunu kaydet
				if _, exists := batchAllQuestions[q]; !exists {
					batchAllQuestions[q] = i
				}
			}
		}
	}

	// Her girişten kaldırılan benzer soruları ve karşı örnekleri toplamak için kullanılır
	removedSimilarQuestionsMap := make(map[int][]string)  // key, giriş dizinidir
	removedNegativeQuestionsMap := make(map[int][]string) // key, giriş dizinidir

	// İlk doğrulamayı geçen her giriş için çakışan benzer soruları filtrele
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		validSimilarQuestions := make([]string, 0, len(entry.SimilarQuestions))
		var removedSimilarQuestions []string
		for _, q := range entry.SimilarQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// Benzer soru doğrulaması: Tüm standart sorular+benzer sorularla karşılaştır → Tek soru ifadesi başarısız olur «benzer soru çakışması»
			// Bu benzer soru, başka girişlerin standart sorusu veya benzer sorusuyla çakışıyorsa (ve kendi standart sorusu değilse), kaldırılır
			if firstIdx, exists := batchAllQuestions[q]; exists && firstIdx != i {
				logger.Infof(ctx, "FAQ entry %d: similar question '%s' conflicts with entry %d, removing", i, q, firstIdx+1)
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(types.LocalizedText(ctx, `Benzer soru çakışması: "%s" %d. satırdaki bir soruyla aynı`, `Similar question conflict: "%s" duplicates a question in row %d`), q, firstIdx+1))
				continue
			}
			// Benzer soru, kendi standart sorusuyla aynı olamaz
			if q == standardQ {
				logger.Infof(ctx, "FAQ entry %d: similar question '%s' same as standard question, removing", i, q)
				removedSimilarQuestions = append(removedSimilarQuestions, fmt.Sprintf(types.LocalizedText(ctx, `Benzer soru çakışması: "%s" bu kaydın ana sorusuyla aynı`, `Similar question conflict: "%s" duplicates this entry's main question`), q))
				continue
			}
			validSimilarQuestions = append(validSimilarQuestions, q)
		}
		entries[i].SimilarQuestions = validSimilarQuestions

		// Kaldırılan benzer soruları kaydet
		if len(removedSimilarQuestions) > 0 {
			removedSimilarQuestionsMap[i] = removedSimilarQuestions
		}

		// İlerleme mesajını düzenli olarak güncelle
		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf(types.LocalizedText(ctx, "Benzer sorular doğrulanıyor %d/%d...", "Validating similar questions %d/%d..."), idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// ==================== Üçüncü yineleme: karşı örnek çakışması tespiti ====================
	// Yalnızca ilk iki kontrolden geçen öğeleri işle; karşı örnek çakışması yalnızca çakışan karşı örnekleri kaldırır
	for idx, i := range validIndicesAfterStdQ {
		entry := &entries[i]
		standardQ := strings.TrimSpace(entry.StandardQuestion)

		// Mevcut QA'nın tüm soru kümesini oluştur (standart soru + kontrolden geçen benzer sorular)
		currentQAQuestions := make(map[string]bool)
		currentQAQuestions[standardQ] = true
		for _, q := range entry.SimilarQuestions {
			currentQAQuestions[q] = true
		}

		validNegativeQuestions := make([]string, 0, len(entry.NegativeQuestions))
		var removedNegativeQuestions []string
		for _, q := range entry.NegativeQuestions {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			// Karşı örnek doğrulaması: mevcut QA altındaki tüm standart soruları + benzer soruları karşılaştır → tek bir soru ifadesi başarısız olur: «karşı örnek çakışması»
			if currentQAQuestions[q] {
				logger.Infof(ctx, "FAQ entry %d: negative question '%s' conflicts with current QA's questions, removing", i, q)
				removedNegativeQuestions = append(removedNegativeQuestions, fmt.Sprintf(types.LocalizedText(ctx, `Olumsuz örnek çakışması: "%s" bu kaydın sorularından biriyle aynı`, `Negative example conflict: "%s" duplicates a question in this entry`), q))
				continue
			}
			validNegativeQuestions = append(validNegativeQuestions, q)
		}
		entries[i].NegativeQuestions = validNegativeQuestions

		// Kaldırılan karşı örnekleri kaydet
		if len(removedNegativeQuestions) > 0 {
			removedNegativeQuestionsMap[i] = removedNegativeQuestions
		}

		// İlerleme mesajını düzenli olarak güncelle
		if (idx+1)%100 == 0 {
			progress.Message = fmt.Sprintf(types.LocalizedText(ctx, "Olumsuz örnekler doğrulanıyor %d/%d...", "Validating negative examples %d/%d..."), idx+1, len(validIndicesAfterStdQ))
			progress.UpdatedAt = time.Now().Unix()
			if err := s.saveFAQImportProgress(ctx, progress); err != nil {
				logger.Warnf(ctx, "Failed to update FAQ dry run progress: %v", err)
			}
		}
	}

	// Bazı başarısızlık bilgilerini FailedEntries'e ekle
	for _, i := range validIndicesAfterStdQ {
		removedSimilar := removedSimilarQuestionsMap[i]
		removedNegative := removedNegativeQuestionsMap[i]
		if len(removedSimilar) > 0 || len(removedNegative) > 0 {
			pf := buildFAQPartialFailedEntry(ctx, i, &entries[i], removedSimilar, removedNegative)
			progress.FailedEntries = append(progress.FailedEntries, pf)
			progress.PartialFailedCount++
		}
	}

	return validIndicesAfterStdQ
}

// unionStrings iki dize dilimini birleştirir ve yinelenenleri kaldırır (tam eşleşme)
func unionStrings(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// validateFAQEntryPayloadBasic, FAQ öğesinin temel biçimini doğrular
func validateFAQEntryPayloadBasic(ctx context.Context, entry *types.FAQEntryPayload) error {
	if entry == nil {
		return errors.New(types.LocalizedText(ctx, "Kayıt boş olamaz", "Entry cannot be empty"))
	}
	standardQ := strings.TrimSpace(entry.StandardQuestion)
	if standardQ == "" {
		return errors.New(types.LocalizedText(ctx, "Ana soru boş olamaz", "Main question cannot be empty"))
	}
	if len(entry.Answers) == 0 {
		return errors.New(types.LocalizedText(ctx, "Yanıt boş olamaz", "Answer cannot be empty"))
	}
	hasValidAnswer := false
	for _, a := range entry.Answers {
		if strings.TrimSpace(a) != "" {
			hasValidAnswer = true
			break
		}
	}
	if !hasValidAnswer {
		return errors.New(types.LocalizedText(ctx, "Yanıtların tümü boş olamaz", "Answers cannot all be empty"))
	}
	return nil
}

type faqMergeOperation struct {
	Entry         types.FAQEntryPayload
	ExistingChunk *types.Chunk
	OldMeta       *types.FAQChunkMetadata
	MergedMeta    *types.FAQChunkMetadata
	Detail        types.FAQMergeDetail
}

// calculateAppendOperations, Append modundaki işlemleri hesaplar (akıllı birleştirme desteklenir).
// entry'nin standart sorusu KB'de zaten varsa, bu bir birleştirme işlemi olarak değerlendirilir (benzer sorular / karşı örneklerin birleşimi,
// yanıt / strateji yeni öğe esas alınır); aksi hâlde yeni oluşturma olarak değerlendirilir. Değişiklik olmayan (hash aynı ve işlem biti
// değişmemiş) birleştirme hedefleri skip edilir; böylece geçersiz veritabanı yazımı / dizin yeniden oluşturma önlenir.
//
// Dahili master davranışı; açık kaynak sürümündeki yalnızca "yinelenen = başarısız" basitleştirilmiş mantığına karşılık gelir. FAQ içe aktarmanın
// yaygın kullanım biçimi "dışa aktarıp değiştirdikten sonra yeniden append etmektir"; geçmiş verileri kaybetmeden
// yeni benzer soruları doğru şekilde eklemek için bu birleştirme semantiği gereklidir.
func (s *knowledgeService) calculateAppendOperations(ctx context.Context,
	tenantID uint64, kbID string, entries []types.FAQEntryPayload,
) (newEntries []types.FAQEntryPayload, mergeOps []faqMergeOperation, skippedCount int, err error) {
	if len(entries) == 0 {
		return nil, nil, 0, nil
	}

	existingChunks, err := s.chunkRepo.ListAllFAQChunksWithMetadataByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to list existing FAQ chunks: %w", err)
	}

	existingStdQToChunk := make(map[string]*types.Chunk)
	existingQuestions := make(map[string]bool)
	for _, chunk := range existingChunks {
		meta, cErr := chunk.FAQMetadata()
		if cErr != nil || meta == nil {
			continue
		}
		if meta.StandardQuestion != "" {
			existingStdQToChunk[meta.StandardQuestion] = chunk
			existingQuestions[meta.StandardQuestion] = true
		}
		for _, q := range meta.SimilarQuestions {
			if q != "" {
				existingQuestions[q] = true
			}
		}
	}

	batchQuestions := make(map[string]bool)
	newEntries = make([]types.FAQEntryPayload, 0, len(entries))
	mergeOps = make([]faqMergeOperation, 0)

	for entryIdx, entry := range entries {
		meta, sErr := sanitizeFAQEntryPayload(ctx, &entry)
		if sErr != nil {
			skippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry: %v", sErr)
			continue
		}

		// Standart sorunun KB'de standart soru olarak bulunup bulunmadığını kontrol et → birleştirme adayı
		existingChunk, isMergeCandidate := existingStdQToChunk[meta.StandardQuestion]
		if isMergeCandidate {
			existingMeta, mErr := existingChunk.FAQMetadata()
			if mErr != nil || existingMeta == nil {
				isMergeCandidate = false
			} else {
				mergedSimilar := unionStrings(existingMeta.SimilarQuestions, meta.SimilarQuestions)
				mergedNegative := unionStrings(existingMeta.NegativeQuestions, meta.NegativeQuestions)

				mergedMeta := &types.FAQChunkMetadata{
					StandardQuestion:  existingMeta.StandardQuestion,
					SimilarQuestions:  mergedSimilar,
					NegativeQuestions: mergedNegative,
					Answers:           meta.Answers,
					AnswerStrategy:    meta.AnswerStrategy,
					Version:           existingMeta.Version + 1,
					Source:            existingMeta.Source,
				}

				newHash := types.CalculateFAQContentHash(mergedMeta)
				enabledChanged := entry.IsEnabled != nil && *entry.IsEnabled != existingChunk.IsEnabled
				recommendedChanged := entry.IsRecommended != nil &&
					*entry.IsRecommended != existingChunk.Flags.HasFlag(types.ChunkFlagRecommended)
				answerStrategyChanged := meta.AnswerStrategy != existingMeta.AnswerStrategy
				if existingChunk.ContentHash == newHash && !enabledChanged && !recommendedChanged && !answerStrategyChanged {
					skippedCount++
					logger.Infof(ctx, "Skipping merge for unchanged FAQ entry: %s", meta.StandardQuestion)
					continue
				}

				oldAnswerStr := strings.Join(existingMeta.Answers, "##")
				newAnswerStr := strings.Join(meta.Answers, "##")
				oldSimilarSet := make(map[string]bool, len(existingMeta.SimilarQuestions))
				for _, q := range existingMeta.SimilarQuestions {
					oldSimilarSet[q] = true
				}
				oldNegativeSet := make(map[string]bool, len(existingMeta.NegativeQuestions))
				for _, q := range existingMeta.NegativeQuestions {
					oldNegativeSet[q] = true
				}
				newSimilarCount := 0
				for _, q := range mergedSimilar {
					if !oldSimilarSet[q] {
						newSimilarCount++
					}
				}
				newNegativeCount := 0
				for _, q := range mergedNegative {
					if !oldNegativeSet[q] {
						newNegativeCount++
					}
				}

				mergeOps = append(mergeOps, faqMergeOperation{
					Entry:         entry,
					ExistingChunk: existingChunk,
					OldMeta:       existingMeta,
					MergedMeta:    mergedMeta,
					Detail: types.FAQMergeDetail{
						Index:            entryIdx,
						StandardQuestion: meta.StandardQuestion,
						AnswerChanged:    oldAnswerStr != newAnswerStr,
						NewSimilarCount:  newSimilarCount,
						NewNegativeCount: newNegativeCount,
					},
				})
				continue
			}
		}

		// Buraya gelinmesi, bunun ne bir birleştirme adayı ne de mevcut KB / geçerli toplu işlemin standart soruları / benzer sorularıyla çakıştığı anlamına gelir.
		// Normalde bu çakışmalar executeFAQDryRunValidation aşamasında engellenmelidir; burada
		// yalnızca savunmacı bir geri dönüş olarak bulunur; doğrulama katmanındaki kaçırılan denetimleri incelemeyi kolaylaştırmak için Warn düzeyinde işaretlenir.
		if existingQuestions[meta.StandardQuestion] || batchQuestions[meta.StandardQuestion] {
			skippedCount++
			logger.Warnf(ctx,
				"calculateAppendOperations: dropping FAQ entry with duplicate standard question %q "+
					"(should have been filtered at validation); entry_idx=%d",
				meta.StandardQuestion, entryIdx)
			continue
		}

		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}
		newEntries = append(newEntries, entry)
	}

	return newEntries, mergeOps, skippedCount, nil
}

// calculateReplaceOperations, Replace modunda silinmesi, oluşturulması ve güncellenmesi gereken öğeleri hesaplar
// Aynı toplu işlem içindeki standart soru veya benzer soru yinelenen öğeleri de filtrele
func (s *knowledgeService) calculateReplaceOperations(ctx context.Context,
	tenantID uint64, knowledgeID string, newEntries []types.FAQEntryPayload,
) ([]types.FAQEntryPayload, []*types.Chunk, int, error) {
	// tag'i çözümlemek için kbID'yi al
	var kbID string
	if len(newEntries) > 0 {
		// knowledgeID'den kbID al
		knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, knowledgeID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to get knowledge: %w", err)
		}
		if knowledge != nil {
			kbID = knowledge.KnowledgeBaseID
		}
	}

	// Tüm yeni kayıtların content hash'ini hesapla ve eşzamanlı olarak hash'ten kayda eşleme oluştur
	type entryWithHash struct {
		entry types.FAQEntryPayload
		hash  string
		meta  *types.FAQChunkMetadata
	}
	entriesWithHash := make([]entryWithHash, 0, len(newEntries))
	newHashSet := make(map[string]bool)
	// Aynı toplu işlem içindeki standart soruları ve benzer soruları yinelenenlerden arındırmak için
	batchQuestions := make(map[string]bool)
	batchSkippedCount := 0

	for _, entry := range newEntries {
		meta, err := sanitizeFAQEntryPayload(ctx, &entry)
		if err != nil {
			batchSkippedCount++
			logger.Warnf(ctx, "Skipping invalid FAQ entry in replace mode: %v", err)
			continue
		}

		// Standart sorunun aynı toplu işlem içinde yinelenip yinelenmediğini kontrol et
		if batchQuestions[meta.StandardQuestion] {
			batchSkippedCount++
			logger.Infof(ctx, "Skipping FAQ entry with duplicate standard question in batch: %s", meta.StandardQuestion)
			continue
		}

		// Benzer sorunun aynı toplu işlem içinde yinelenip yinelenmediğini kontrol et
		hasDuplicateSimilar := false
		for _, q := range meta.SimilarQuestions {
			if batchQuestions[q] {
				hasDuplicateSimilar = true
				logger.Infof(ctx, "Skipping FAQ entry with duplicate similar question in batch: %s (standard: %s)", q, meta.StandardQuestion)
				break
			}
		}
		if hasDuplicateSimilar {
			batchSkippedCount++
			continue
		}

		// Geçerli kaydın standart sorusunu ve benzer sorusunu toplu işlem kümesine ekle
		batchQuestions[meta.StandardQuestion] = true
		for _, q := range meta.SimilarQuestions {
			batchQuestions[q] = true
		}

		hash := types.CalculateFAQContentHash(meta)
		if hash != "" {
			entriesWithHash = append(entriesWithHash, entryWithHash{entry: entry, hash: hash, meta: meta})
			newHashSet[hash] = true
		}
	}

	// Mevcut tüm chunks'ları sorgula
	allExistingChunks, err := s.chunkRepo.ListAllFAQChunksByKnowledgeID(ctx, tenantID, knowledgeID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to list existing chunks: %w", err)
	}

	// Bellekte yeni kayıt hash'leriyle eşleşen chunks'ları filtrele ve map oluştur
	existingHashMap := make(map[string]*types.Chunk)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash != "" && newHashSet[chunk.ContentHash] {
			existingHashMap[chunk.ContentHash] = chunk
		}
	}

	// Silinmesi gereken chunks'ları hesapla (veritabanında olup yeni toplu işlemde olmayanlar veya hash'i eşleşmeyenler)
	chunksToDelete := make([]*types.Chunk, 0)
	for _, chunk := range allExistingChunks {
		if chunk.ContentHash == "" {
			// Hash yoksa silinmelidir (eski veri olabilir)
			chunksToDelete = append(chunksToDelete, chunk)
		} else if !newHashSet[chunk.ContentHash] {
			// Hash yeni kayıtlarda yok, silinmelidir
			chunksToDelete = append(chunksToDelete, chunk)
		}
	}

	// Döngü içinde veritabanını tek tek sorgulamayı önlemek için tag bilgilerini toplu olarak önceden yükle
	// Sorgulanması gereken tüm tag_id ve tag_name değerlerini topla
	tagSeqIDSet := make(map[int64]bool)
	tagNameSet := make(map[string]bool)
	for _, ewh := range entriesWithHash {
		if ewh.entry.TagID != 0 {
			tagSeqIDSet[ewh.entry.TagID] = true
		} else if ewh.entry.TagName != "" {
			tagNameSet[ewh.entry.TagName] = true
		} else {
			tagNameSet[types.UntaggedTagName] = true
		}
	}

	// seq_id ile tag'leri toplu olarak sorgula
	tagSeqIDToUUID := make(map[int64]string)
	if len(tagSeqIDSet) > 0 {
		seqIDs := make([]int64, 0, len(tagSeqIDSet))
		for seqID := range tagSeqIDSet {
			seqIDs = append(seqIDs, seqID)
		}
		tags, err := s.tagRepo.GetBySeqIDs(ctx, tenantID, seqIDs)
		if err != nil {
			logger.Warnf(ctx, "Failed to batch load tags by seq_ids: %v", err)
		} else {
			for _, tag := range tags {
				tagSeqIDToUUID[tag.SeqID] = tag.ID
			}
		}
	}

	// Ada göre tag'leri toplu olarak sorgula
	tagNameToUUID := make(map[string]string)
	if len(tagNameSet) > 0 && kbID != "" {
		for name := range tagNameSet {
			if tag, err := s.tagRepo.GetByName(ctx, tenantID, kbID, name); err == nil && tag != nil {
				tagNameToUUID[name] = tag.ID
			}
		}
	}

	logger.Infof(ctx, "Preloaded %d tags by seq_id, %d tags by name for %d entries",
		len(tagSeqIDToUUID), len(tagNameToUUID), len(entriesWithHash))

	// resolveTagIDFromCache, tag ID'sini önbellekten çözümler; önbellek isabeti yoksa veritabanı sorgusuna geri döner
	resolveTagIDFromCache := func(entry *types.FAQEntryPayload) (string, error) {
		if entry.TagID != 0 {
			if uuid, ok := tagSeqIDToUUID[entry.TagID]; ok {
				return uuid, nil
			}
			// Önbellek isabeti yok, veritabanı sorgusuna geri dön (oluşturmak gerekebilir)
			return s.resolveTagID(ctx, kbID, entry)
		}
		tagName := entry.TagName
		if tagName == "" {
			tagName = types.UntaggedTagName
		}
		if uuid, ok := tagNameToUUID[tagName]; ok {
			return uuid, nil
		}
		// Önbellek isabeti yok, veritabanı sorgusuna geri dön (oluşturmak gerekebilir)
		return s.resolveTagID(ctx, kbID, entry)
	}

	// Oluşturulması gereken kayıtları hesapla (tekrar hesaplamayı önlemek için önceden hesaplanmış hash'i kullan)
	entriesToProcess := make([]types.FAQEntryPayload, 0, len(entriesWithHash))
	skippedCount := batchSkippedCount

	for idx, ewh := range entriesWithHash {
		// Her 1000 kayıt işlendiğinde bir kez ilerleme günlüğü yazdır
		if idx > 0 && idx%1000 == 0 {
			logger.Infof(ctx, "calculateReplaceOperations progress: %d/%d entries processed", idx, len(entriesWithHash))
		}

		existingChunk := existingHashMap[ewh.hash]
		if existingChunk != nil {
			// Hash eşleşiyor, tag'in değişip değişmediğini kontrol et
			newTagID, err := resolveTagIDFromCache(&ewh.entry)
			if err != nil {
				logger.Warnf(ctx, "Failed to resolve tag for entry, treating as new: %v", err)
				entriesToProcess = append(entriesToProcess, ewh.entry)
				continue
			}

			enabledChanged := ewh.entry.IsEnabled != nil && *ewh.entry.IsEnabled != existingChunk.IsEnabled
			recommendedChanged := ewh.entry.IsRecommended != nil &&
				*ewh.entry.IsRecommended != existingChunk.Flags.HasFlag(types.ChunkFlagRecommended)
			answerStrategyChanged := false
			if existingMeta, metaErr := existingChunk.FAQMetadata(); metaErr == nil && existingMeta != nil {
				answerStrategyChanged = ewh.meta.AnswerStrategy != existingMeta.AnswerStrategy
			}

			if existingChunk.TagID != newTagID || enabledChanged || recommendedChanged || answerStrategyChanged {
				if existingChunk.TagID != newTagID {
					logger.Infof(ctx, "FAQ entry tag changed from %s to %s, will update", existingChunk.TagID, newTagID)
				}
				if enabledChanged || recommendedChanged || answerStrategyChanged {
					logger.Infof(ctx, "FAQ entry operational fields changed (enabled: %v->%v, recommended: %v->%v, answerStrategy changed: %v), will update",
						existingChunk.IsEnabled, ewh.entry.IsEnabled,
						existingChunk.Flags.HasFlag(types.ChunkFlagRecommended), ewh.entry.IsRecommended,
						answerStrategyChanged)
				}
				chunksToDelete = append(chunksToDelete, existingChunk)
				entriesToProcess = append(entriesToProcess, ewh.entry)
			} else {
				// Hash, tag ve operasyon durumu aynı, atla
				skippedCount++
			}
			continue
		}

		// Hash eşleşmiyor veya mevcut değil, oluşturulması gerekiyor
		entriesToProcess = append(entriesToProcess, ewh.entry)
	}

	return entriesToProcess, chunksToDelete, skippedCount, nil
}

// executeFAQImport, gerçek FAQ içe aktarma mantığını yürütür
func (s *knowledgeService) executeFAQImport(ctx context.Context, taskID string, kbID string,
	payload *types.FAQBatchUpsertPayload, tenantID uint64, processedCount int,
	progress *types.FAQImportProgress,
) (err error) {
	// Dizini temizlemek için bilgi tabanı ve embedding modeli bilgilerini kaydet
	var kb *types.KnowledgeBase
	var embeddingModel embedding.Embedder
	totalEntries := len(payload.Entries) + processedCount

	// Recovery mekanizması: herhangi bir hata veya panic oluşursa, oluşturulan tüm chunks ve indeks verilerini geri al
	defer func() {
		// panic'i yakala
		if r := recover(); r != nil {
			buf := make([]byte, 8192)
			n := runtime.Stack(buf, false)
			stack := string(buf[:n])
			logger.Errorf(ctx, "FAQ import task %s panicked: %v\n%s", taskID, r, stack)
			err = fmt.Errorf("panic during FAQ import: %v", r)
		}
	}()

	kb, ctx, err = s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}

	kb.EnsureDefaults()

	// Sonraki indeks temizliği için embedding modelini al
	embeddingModel, err = s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return fmt.Errorf("failed to get embedding model: %w", err)
	}
	faqKnowledge, err := s.ensureFAQKnowledge(ctx, tenantID, kb)
	if err != nil {
		return err
	}

	// İndeks modunu al
	indexMode := types.FAQIndexModeQuestionOnly
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	// Artımlı güncelleme mantığı: işlenmesi gereken öğeleri hesapla
	var entriesToProcess []types.FAQEntryPayload
	var chunksToDelete []*types.Chunk
	var skippedCount int

	if payload.Mode == types.FAQBatchModeReplace {
		// Replace modu: silinmesi, oluşturulması ve güncellenmesi gereken öğeleri hesapla
		entriesToProcess, chunksToDelete, skippedCount, err = s.calculateReplaceOperations(
			ctx,
			tenantID,
			faqKnowledge.ID,
			payload.Entries,
		)
		if err != nil {
			return fmt.Errorf("failed to calculate replace operations: %w", err)
		}

		// Silinmesi gereken chunks'ları sil (güncellenmesi gereken eski chunks'lar dahil)
		if len(chunksToDelete) > 0 {
			chunkIDsToDelete := make([]string, 0, len(chunksToDelete))
			for _, chunk := range chunksToDelete {
				chunkIDsToDelete = append(chunkIDsToDelete, chunk.ID)
			}
			if err := s.chunkRepo.DeleteChunks(ctx, tenantID, chunkIDsToDelete); err != nil {
				return fmt.Errorf("failed to delete chunks: %w", err)
			}
			// İndeksi sil
			if err := s.deleteFAQChunkVectors(ctx, kb, faqKnowledge, chunksToDelete); err != nil {
				return fmt.Errorf("failed to delete chunk vectors: %w", err)
			}
			logger.Infof(ctx, "FAQ import task %s: deleted %d chunks (including updates)", taskID, len(chunksToDelete))
		}
	} else {
		// Append modu (akıllı birleştirme): Mevcut standart soru öğeleri merge ops kullanır, diğerleri yeni olarak oluşturulur.
		var mergeOps []faqMergeOperation
		entriesToProcess, mergeOps, skippedCount, err = s.calculateAppendOperations(ctx, tenantID, kb.ID, payload.Entries)
		if err != nil {
			return fmt.Errorf("failed to calculate append operations: %w", err)
		}

		if len(mergeOps) > 0 {
			mergedCount, mergeErr := s.executeFAQMergeOperations(ctx, taskID, kb, faqKnowledge, embeddingModel, indexMode, mergeOps, progress)
			if mergeErr != nil {
				return fmt.Errorf("failed to execute merge operations: %w", mergeErr)
			}
			logger.Infof(ctx, "FAQ import task %s: merged %d entries", taskID, mergedCount)
			progress.MergedCount = mergedCount
			for _, op := range mergeOps {
				progress.MergeDetails = append(progress.MergeDetails, op.Detail)
			}
		}
	}
	logger.Infof(
		ctx,
		"FAQ import task %s: total entries: %d, new to create: %d, skipped: %d, merged: %d",
		taskID,
		len(payload.Entries),
		len(entriesToProcess),
		skippedCount,
		progress.MergedCount,
	)

	// İşlenmesi gereken öğe yoksa doğrudan dön
	if len(entriesToProcess) == 0 {
		logger.Infof(ctx, "FAQ import task %s: no new entries to create", taskID)
		return nil
	}

	// Oluşturulması gereken öğeleri toplu olarak işle
	remainingEntries := len(entriesToProcess)
	totalStartTime := time.Now()
	actualProcessed := skippedCount + processedCount + progress.MergedCount

	logger.Infof(
		ctx,
		"FAQ import task %s: starting batch processing, remaining entries: %d, total entries: %d, batch size: %d",
		taskID,
		remainingEntries,
		totalEntries,
		faqImportBatchSize,
	)

	for i := 0; i < remainingEntries; i += faqImportBatchSize {
		batchStartTime := time.Now()
		end := i + faqImportBatchSize
		if end > remainingEntries {
			end = remainingEntries
		}

		batch := entriesToProcess[i:end]
		logger.Infof(ctx, "FAQ import task %s: processing batch %d-%d (%d entries)", taskID, i+1, end, len(batch))

		// chunks oluştur
		buildStartTime := time.Now()
		chunks := make([]*types.Chunk, 0, len(batch))
		chunkIds := make([]string, 0, len(batch))
		for idx, entry := range batch {
			meta, err := sanitizeFAQEntryPayload(ctx, &entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to sanitize entry at index %d: %w", i+idx, err)
			}

			// TagID'yi ayrıştır
			tagID, err := s.resolveTagID(ctx, kbID, &entry)
			if err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"entry":   entry,
					"task_id": taskID,
				})
				return fmt.Errorf("failed to resolve tag for entry at index %d: %w", i+idx, err)
			}

			isEnabled := true
			if entry.IsEnabled != nil {
				isEnabled = *entry.IsEnabled
			}
			// ChunkIndex hesaplaması: startChunkIndex + (i+idx) + initialProcessed
			chunk := &types.Chunk{
				ID:              uuid.New().String(),
				TenantID:        tenantID,
				KnowledgeID:     faqKnowledge.ID,
				KnowledgeBaseID: kb.ID,
				Content:         buildFAQChunkContent(meta, indexMode),
				// ChunkIndex:      0,
				IsEnabled: isEnabled,
				ChunkType: types.ChunkTypeFAQ,
				TagID:     tagID,                        // Ayrıştırılmış TagID'yi kullan
				Status:    int(types.ChunkStatusStored), // store but not indexed
			}
			// ID belirtilmişse (veri taşıma için), SeqID'yi ayarla
			if entry.ID != nil && *entry.ID > 0 {
				chunk.SeqID = *entry.ID
			}
			if err := chunk.SetFAQMetadata(meta); err != nil {
				return fmt.Errorf("failed to set FAQ metadata: %w", err)
			}
			chunks = append(chunks, chunk)
			chunkIds = append(chunkIds, chunk.ID)
		}
		buildDuration := time.Since(buildStartTime)
		logger.Debugf(ctx, "FAQ import task %s: batch %d-%d built %d chunks in %v, chunk IDs: %v",
			taskID, i+1, end, len(chunks), buildDuration, chunkIds)
		// chunks oluştur
		createStartTime := time.Now()
		if err := s.chunkService.CreateChunks(ctx, chunks); err != nil {
			return fmt.Errorf("failed to create chunks: %w", err)
		}
		createDuration := time.Since(createStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d created %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			createDuration,
		)

		// chunks'ları indeksle
		indexStartTime := time.Now()
		// Not: İndeksleme başarısız olursa, defer içindeki recovery mekanizması oluşturulan chunks'ları ve indeks verilerini otomatik olarak geri alır
		if err := s.indexFAQChunks(ctx, kb, faqKnowledge, chunks, embeddingModel, true, false); err != nil {
			return fmt.Errorf("failed to index chunks: %w", err)
		}
		indexDuration := time.Since(indexStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d indexed %d chunks in %v",
			taskID,
			i+1,
			end,
			len(chunks),
			indexDuration,
		)

		// chunks'ların Status değerini indekslendi olarak güncelle: tüm satırlara aynı değer yazılır, tek bir UPDATE ... WHERE id IN yeterlidir,
		// content gibi alanları tekrar göndermeye gerek yoktur.
		for _, chunk := range chunks {
			chunk.Status = int(types.ChunkStatusIndexed) // indexed
		}
		if err := s.chunkRepo.UpdateChunkFieldsByIDs(ctx, tenantID, chunkIds, map[string]interface{}{
			"status": int(types.ChunkStatusIndexed),
		}); err != nil {
			return fmt.Errorf("failed to update chunks status: %w", err)
		}

		// Başarılı öğe bilgilerini topla (her öğe için tek tek sorgulamak yerine tag bilgileri toplu olarak bir kez alınır)
		tagsByID := s.loadFAQTagsForChunks(ctx, tenantID, chunks)
		for idx, chunk := range chunks {
			entryIdx := i + idx + processedCount // Özgün öğe indeksi
			meta, _ := chunk.FAQMetadata()
			standardQ := ""
			if meta != nil {
				standardQ = meta.StandardQuestion
			}
			tagID, tagName := faqTagInfo(tagsByID, chunk.TagID)
			progress.SuccessEntries = append(progress.SuccessEntries, types.FAQSuccessEntry{
				Index:            entryIdx,
				SeqID:            chunk.SeqID,
				TagID:            tagID,
				TagName:          tagName,
				StandardQuestion: standardQ,
			})
		}

		actualProcessed += len(batch)
		// Görev ilerlemesini güncelle
		progress := int(float64(actualProcessed) / float64(totalEntries) * 100)
		if err := s.updateFAQImportProgressStatus(ctx, taskID, "", 0, types.FAQImportStatusProcessing, progress, totalEntries, actualProcessed, fmt.Sprintf(types.LocalizedText(ctx, "İşlenen kayıt %d/%d", "Processing entry %d/%d"), actualProcessed, totalEntries), ""); err != nil {
			logger.Errorf(ctx, "Failed to update task progress: %v", err)
		}

		batchDuration := time.Since(batchStartTime)
		logger.Infof(
			ctx,
			"FAQ import task %s: batch %d-%d completed in %v (build: %v, create: %v, index: %v), total progress: %d/%d (%d%%)",
			taskID,
			i+1,
			end,
			batchDuration,
			buildDuration,
			createDuration,
			indexDuration,
			actualProcessed,
			totalEntries,
			progress,
		)
	}

	totalDuration := time.Since(totalStartTime)
	var avgPerEntry time.Duration
	if actualProcessed > 0 {
		avgPerEntry = totalDuration / time.Duration(actualProcessed)
	}
	logger.Infof(
		ctx,
		"FAQ import task %s: all batches completed, processed: %d entries (skipped: %d) in %v, avg: %v per entry",
		taskID,
		actualProcessed,
		skippedCount,
		totalDuration,
		avgPerEntry,
	)

	return nil
}

// updateFAQImportProgressStatus updates the FAQ import progress in Redis
func (s *knowledgeService) updateFAQImportProgressStatus(
	ctx context.Context,
	taskID string,
	instanceID string,
	enqueuedAt int64,
	status types.FAQImportTaskStatus,
	progress, total, processed int,
	message, errorMsg string,
) error {
	// Get existing progress from Redis
	existingProgress, err := s.GetFAQImportProgress(ctx, taskID)
	if err != nil {
		// If not found, create a new progress entry
		existingProgress = &types.FAQImportProgress{
			TaskID:    taskID,
			CreatedAt: time.Now().Unix(),
		}
	}

	// Update progress fields
	existingProgress.Status = status
	existingProgress.Progress = progress
	existingProgress.Total = total
	existingProgress.Processed = processed
	if message != "" {
		existingProgress.Message = message
	}
	existingProgress.Error = errorMsg
	if status == types.FAQImportStatusCompleted {
		existingProgress.Error = ""
	}

	// Görev tamamlandığında veya başarısız olduğunda running key'i temizle
	if status == types.FAQImportStatusCompleted || status == types.FAQImportStatusFailed {
		if existingProgress.KBID != "" {
			if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, existingProgress.KBID, taskID, instanceID, enqueuedAt); clearErr != nil {
				logger.Errorf(ctx, "Failed to clear running FAQ import task ID: %v", clearErr)
			}
		}
	}

	return s.saveFAQImportProgress(ctx, existingProgress)
}

// cleanupFAQEntriesFileOnFinalFailure, görev nihai olarak başarısız olduğunda nesne depolamadaki entries dosyasını temizler
// Temizlik yalnızca retryCount >= maxRetry olduğunda yapılır; aksi halde dosya yeniden denemelerde hâlâ gereklidir
func (s *knowledgeService) cleanupFAQEntriesFileOnFinalFailure(ctx context.Context, entriesURL string, retryCount, maxRetry int) {
	if entriesURL == "" || retryCount < maxRetry {
		return
	}
	if err := s.fileSvc.DeleteFile(ctx, entriesURL); err != nil {
		logger.Warnf(ctx, "Failed to delete FAQ entries file from object storage on final failure: %v", err)
	} else {
		logger.Infof(ctx, "Deleted FAQ entries file from object storage on final failure: %s", entriesURL)
	}
}

// runningFAQImportInfo stores the task ID and enqueued timestamp for uniquely identifying a task instance
type runningFAQImportInfo struct {
	TaskID     string `json:"task_id"`
	EnqueuedAt int64  `json:"enqueued_at"`
	InstanceID string `json:"instance_id,omitempty"`
}

// getRunningFAQImportInfo checks if there's a running FAQ import task for the given KB
// Returns the task info if found, nil otherwise
func (s *knowledgeService) getRunningFAQImportInfo(ctx context.Context, kbID string) (*runningFAQImportInfo, error) {
	if s.redisClient == nil {
		if v, ok := s.memFAQRunningImport.Load(kbID); ok {
			return v.(*runningFAQImportInfo), nil
		}
		return nil, nil
	}
	key := getFAQImportRunningKey(kbID)
	data, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get running FAQ import task: %w", err)
	}

	// Try to parse as JSON first (new format)
	var info runningFAQImportInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		// Fallback: old format was just taskID string
		return &runningFAQImportInfo{TaskID: data, EnqueuedAt: 0}, nil
	}
	return &info, nil
}

// getRunningFAQImportTaskID checks if there's a running FAQ import task for the given KB
// Returns the task ID if found, empty string otherwise (for backward compatibility)
func (s *knowledgeService) getRunningFAQImportTaskID(ctx context.Context, kbID string) (string, error) {
	info, err := s.getRunningFAQImportInfo(ctx, kbID)
	if err != nil {
		return "", err
	}
	if info == nil {
		return "", nil
	}
	return info.TaskID, nil
}

// setRunningFAQImportInfo sets the running task info for a KB
func (s *knowledgeService) setRunningFAQImportInfo(ctx context.Context, kbID string, info *runningFAQImportInfo) error {
	if s.redisClient == nil {
		s.memFAQRunningImport.Store(kbID, info)
		return nil
	}
	key := getFAQImportRunningKey(kbID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal running info: %w", err)
	}
	return s.redisClient.Set(ctx, key, data, faqImportProgressTTL).Err()
}

// clearRunningFAQImportTaskID clears the running task ID for a KB
func (s *knowledgeService) clearRunningFAQImportTaskID(ctx context.Context, kbID string) error {
	if s.redisClient == nil {
		s.memFAQRunningImport.Delete(kbID)
		return nil
	}
	key := getFAQImportRunningKey(kbID)
	return s.redisClient.Del(ctx, key).Err()
}

func (s *knowledgeService) clearRunningFAQImportInfoIfMatches(ctx context.Context, kbID, taskID, instanceID string, enqueuedAt int64) error {
	if s.redisClient == nil {
		if v, ok := s.memFAQRunningImport.Load(kbID); ok {
			info, _ := v.(*runningFAQImportInfo)
			if runningFAQImportInfoMatches(info, taskID, instanceID, enqueuedAt) {
				s.memFAQRunningImport.Delete(kbID)
			}
		}
		return nil
	}

	info, err := s.getRunningFAQImportInfo(ctx, kbID)
	if err != nil {
		return err
	}
	if !runningFAQImportInfoMatches(info, taskID, instanceID, enqueuedAt) {
		return nil
	}

	key := getFAQImportRunningKey(kbID)
	return s.redisClient.Del(ctx, key).Err()
}

func runningFAQImportInfoMatches(info *runningFAQImportInfo, taskID, instanceID string, enqueuedAt int64) bool {
	if info == nil || info.TaskID != taskID {
		return false
	}
	if info.InstanceID != "" && instanceID != "" {
		return info.InstanceID == instanceID
	}
	return enqueuedAt == 0 || info.EnqueuedAt == 0 || info.EnqueuedAt == enqueuedAt
}

// incrementalIndexFAQEntry, FAQ girdilerinin dizinini artımlı olarak günceller
// Yalnızca içeriği değişen bölümler için embedding hesaplaması ve dizin güncellemesi yapar, değişmeyen bölümleri atlar
func (s *knowledgeService) incrementalIndexFAQEntry(
	ctx context.Context,
	kb *types.KnowledgeBase,
	knowledge *types.Knowledge,
	chunk *types.Chunk,
	embeddingModel embedding.Embedder,
	oldStandardQuestion string,
	oldSimilarQuestions []string,
	oldAnswers []string,
	newMeta *types.FAQChunkMetadata,
) error {
	indexStartTime := time.Now()
	logger.Debugf(ctx, "incrementalIndexFAQEntry: starting for chunk=%s, oldSimilarQuestions=%d, newSimilarQuestions=%d",
		chunk.ID, len(oldSimilarQuestions), len(newMeta.SimilarQuestions))

	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, types.MustTenantIDFromContext(ctx), kb.VectorStoreID)
	if err != nil {
		return err
	}

	indexMode := types.FAQIndexModeQuestionAnswer
	if kb.FAQConfig != nil && kb.FAQConfig.IndexMode != "" {
		indexMode = kb.FAQConfig.IndexMode
	}

	// buildFAQIndexInfoList davranışıyla tutarlılığı sağlamak için eski ve yeni verileri normalize eder
	// Eski verileri normalize et
	oldStandardQuestion = types.NormalizeQuestion(oldStandardQuestion)
	normalizedOldSimilarQuestions := make([]string, 0, len(oldSimilarQuestions))
	for _, q := range oldSimilarQuestions {
		if nq := types.NormalizeQuestion(q); nq != "" {
			normalizedOldSimilarQuestions = append(normalizedOldSimilarQuestions, nq)
		}
	}
	oldSimilarQuestions = normalizedOldSimilarQuestions
	oldAnswers = types.SanitizeStrings(oldAnswers)
	// Yeni verileri normalize et
	normalizedNewMeta := newMeta.Normalize()

	// Dizin içeriğini oluştur
	buildContent := func(question string, answers []string) string {
		if indexMode == types.FAQIndexModeQuestionAnswer && len(answers) > 0 {
			var builder strings.Builder
			builder.WriteString(question)
			for _, ans := range answers {
				builder.WriteString("\n")
				builder.WriteString(ans)
			}
			return builder.String()
		}
		return question
	}

	// Yanıtın değişip değişmediğini kontrol et (yalnızca QuestionAnswer modunda dizini etkiler)
	answersChanged := indexMode == types.FAQIndexModeQuestionAnswer && !slices.Equal(oldAnswers, normalizedNewMeta.Answers)
	logger.Debugf(ctx, "incrementalIndexFAQEntry: answersChanged=%v (indexMode=%s), oldAnswers=%d, newAnswers=%d",
		answersChanged, indexMode, len(oldAnswers), len(normalizedNewMeta.Answers))

	// Güncellenmesi gereken dizin öğelerini topla
	var indexInfoToUpdate []*types.IndexInfo

	// 1. Standart sorunun güncellenmesi gerekip gerekmediğini kontrol et
	oldStdContent := buildContent(oldStandardQuestion, oldAnswers)
	newStdContent := buildContent(normalizedNewMeta.StandardQuestion, normalizedNewMeta.Answers)
	stdQuestionChanged := oldStdContent != newStdContent
	if stdQuestionChanged {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: standard question changed, sourceID=%s", chunk.ID)
		indexInfoToUpdate = append(indexInfoToUpdate, &types.IndexInfo{
			Content:         newStdContent,
			SourceID:        chunk.ID,
			SourceType:      types.ChunkSourceType,
			ChunkID:         chunk.ID,
			KnowledgeID:     chunk.KnowledgeID,
			KnowledgeBaseID: chunk.KnowledgeBaseID,
			KnowledgeType:   types.KnowledgeTypeFAQ,
			TagID:           chunk.TagID,
			IsEnabled:       chunk.IsEnabled,
			IsRecommended:   chunk.Flags.HasFlag(types.ChunkFlagRecommended),
		})
	}

	// 2. İçerik karmasına göre benzer soruların ekleme, silme ve güncelleme işlemlerini yap
	// Eski soru kümesini oluştur (soru -> mevcut olup olmadığı)
	oldQuestionsSet := make(map[string]struct{}, len(oldSimilarQuestions))
	for _, q := range oldSimilarQuestions {
		oldQuestionsSet[q] = struct{}{}
	}

	// Yeni soru kümesini oluştur
	newQuestionsSet := make(map[string]struct{}, len(normalizedNewMeta.SimilarQuestions))
	for _, q := range normalizedNewMeta.SimilarQuestions {
		newQuestionsSet[q] = struct{}{}
	}

	// Silinmesi gereken soruları bul (eski kümede olup yeni kümede olmayanlar)
	var sourceIDsToDelete []string
	var deletedQuestions []string
	for oldQ := range oldQuestionsSet {
		if _, exists := newQuestionsSet[oldQ]; !exists {
			sourceID := fmt.Sprintf("%s-%s", chunk.ID, hashQuestion(oldQ))
			sourceIDsToDelete = append(sourceIDsToDelete, sourceID)
			deletedQuestions = append(deletedQuestions, oldQ)
		}
	}

	// Eklenmesi veya güncellenmesi gereken soruları bul
	var addedQuestions, updatedQuestions []string
	for newQ := range newQuestionsSet {
		_, existedBefore := oldQuestionsSet[newQ]
		// Güncelleme koşulları:
		// 1. Yeni soru (daha önce yoktu)
		// 2. Yanıt değişti (embedding yeniden hesaplanmalı)
		if !existedBefore || answersChanged {
			sourceID := fmt.Sprintf("%s-%s", chunk.ID, hashQuestion(newQ))
			indexInfoToUpdate = append(indexInfoToUpdate, &types.IndexInfo{
				Content:         buildContent(newQ, normalizedNewMeta.Answers),
				SourceID:        sourceID,
				SourceType:      types.ChunkSourceType,
				ChunkID:         chunk.ID,
				KnowledgeID:     chunk.KnowledgeID,
				KnowledgeBaseID: chunk.KnowledgeBaseID,
				KnowledgeType:   types.KnowledgeTypeFAQ,
				TagID:           chunk.TagID,
				IsEnabled:       chunk.IsEnabled,
				IsRecommended:   chunk.Flags.HasFlag(types.ChunkFlagRecommended),
			})
			if !existedBefore {
				addedQuestions = append(addedQuestions, newQ)
			} else {
				updatedQuestions = append(updatedQuestions, newQ)
			}
		}
	}

	// Ayrıntılı değişiklik günlüklerini çıktıla
	if len(deletedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: deleted similar questions: %v", deletedQuestions)
	}
	if len(addedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: added similar questions: %v", addedQuestions)
	}
	if len(updatedQuestions) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: updated similar questions (answers changed): %v", updatedQuestions)
	}

	// 3. Artık mevcut olmayan benzer soru dizinlerini ve yeniden oluşturulacak eski dizinleri sil.
	// Aynı SourceID için BatchIndex üzerine yazmaya güvenilemez: ES v8, Qdrant ve benzeri motorlar her yazmada
	// yeni belge ID'leri oluşturur; önce silinmezse içeriği güncel olmayan yinelenen girdiler kalır.
	for _, info := range indexInfoToUpdate {
		sourceIDsToDelete = append(sourceIDsToDelete, info.SourceID)
	}
	if len(sourceIDsToDelete) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: deleting %d obsolete sourceIDs: %v", len(sourceIDsToDelete), sourceIDsToDelete)
		if delErr := retrieveEngine.DeleteBySourceIDList(ctx, sourceIDsToDelete, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); delErr != nil {
			logger.Warnf(ctx, "incrementalIndexFAQEntry: failed to delete obsolete source IDs: %v", delErr)
		}
	}

	// 4. Güncellenmesi gereken içeriği toplu olarak dizinle
	newCount := len(normalizedNewMeta.SimilarQuestions)
	if len(indexInfoToUpdate) > 0 {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: updating %d index entries (skipped %d unchanged)",
			len(indexInfoToUpdate), 1+newCount-len(indexInfoToUpdate))
		if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfoToUpdate); err != nil {
			return err
		}
	} else {
		logger.Debugf(ctx, "incrementalIndexFAQEntry: all %d entries unchanged, skipping index update", 1+newCount)
	}

	// 5. knowledge kaydını güncelle
	now := time.Now()
	knowledge.UpdatedAt = now
	knowledge.ProcessedAt = &now
	if err := s.repo.UpdateKnowledge(ctx, knowledge); err != nil {
		return err
	}

	totalDuration := time.Since(indexStartTime)
	logger.Debugf(ctx, "incrementalIndexFAQEntry: completed in %v, updated %d/%d entries",
		totalDuration, len(indexInfoToUpdate), 1+newCount)

	return nil
}

func (s *knowledgeService) indexFAQChunks(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge,
	chunks []*types.Chunk, embeddingModel embedding.Embedder,
	adjustStorage bool, needDelete bool,
) error {
	if len(chunks) == 0 {
		return nil
	}
	indexStartTime := time.Now()
	logger.Debugf(ctx, "indexFAQChunks: starting to index %d chunks", len(chunks))

	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
	if err != nil {
		return err
	}

	// Dizin bilgilerini oluştur
	buildIndexInfoStartTime := time.Now()
	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}
	buildIndexInfoDuration := time.Since(buildIndexInfoStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: built %d index info entries for %d chunks in %v",
		len(indexInfo),
		len(chunks),
		buildIndexInfoDuration,
	)

	var size int64
	if adjustStorage {
		estimateStartTime := time.Now()
		size = retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
		estimateDuration := time.Since(estimateStartTime)
		logger.Debugf(ctx, "indexFAQChunks: estimated storage size %d bytes in %v", size, estimateDuration)
		if tenantInfo.StorageQuota > 0 && tenantInfo.StorageUsed+size > tenantInfo.StorageQuota {
			return types.NewStorageQuotaExceededError()
		}
	}

	// Eski vektörleri sil
	var deleteDuration time.Duration
	if needDelete {
		deleteStartTime := time.Now()
		if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
			logger.Warnf(ctx, "Delete FAQ vectors failed: %v", err)
		}
		deleteDuration = time.Since(deleteStartTime)
		if deleteDuration > 100*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: deleted old vectors for %d chunks in %v", len(chunkIDs), deleteDuration)
		}
	}

	// Toplu dizinleme (burası performans darboğazı olabilir)
	batchIndexStartTime := time.Now()
	if err := retrieveEngine.BatchIndex(ctx, embeddingModel, indexInfo); err != nil {
		return err
	}
	batchIndexDuration := time.Since(batchIndexStartTime)
	var avgPerEntry time.Duration
	if len(indexInfo) > 0 {
		avgPerEntry = batchIndexDuration / time.Duration(len(indexInfo))
	}
	logger.Debugf(ctx, "indexFAQChunks: batch indexed %d index info entries in %v (avg: %v per entry)",
		len(indexInfo), batchIndexDuration, avgPerEntry)

	if adjustStorage && size > 0 {
		adjustStartTime := time.Now()
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, size); err == nil {
			tenantInfo.StorageUsed += size
		}
		knowledge.StorageSize += size
		adjustDuration := time.Since(adjustStartTime)
		if adjustDuration > 50*time.Millisecond {
			logger.Debugf(ctx, "indexFAQChunks: adjusted storage in %v", adjustDuration)
		}
	}

	updateStartTime := time.Now()
	now := time.Now()
	knowledge.UpdatedAt = now
	knowledge.ProcessedAt = &now
	err = s.repo.UpdateKnowledge(ctx, knowledge)
	updateDuration := time.Since(updateStartTime)
	if updateDuration > 50*time.Millisecond {
		logger.Debugf(ctx, "indexFAQChunks: updated knowledge in %v", updateDuration)
	}

	totalDuration := time.Since(indexStartTime)
	logger.Debugf(
		ctx,
		"indexFAQChunks: completed indexing %d chunks in %v (build: %v, delete: %v, batchIndex: %v, update: %v)",
		len(chunks),
		totalDuration,
		buildIndexInfoDuration,
		deleteDuration,
		batchIndexDuration,
		updateDuration,
	)

	return err
}

func (s *knowledgeService) deleteFAQChunkVectors(ctx context.Context,
	kb *types.KnowledgeBase, knowledge *types.Knowledge, chunks []*types.Chunk,
) error {
	if len(chunks) == 0 {
		return nil
	}
	embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
	if err != nil {
		return err
	}
	tenantInfo := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
		ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
	if err != nil {
		return err
	}

	indexInfo := make([]*types.IndexInfo, 0)
	chunkIDs := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		infoList, err := s.buildFAQIndexInfoList(ctx, kb, chunk)
		if err != nil {
			return err
		}
		indexInfo = append(indexInfo, infoList...)
		chunkIDs = append(chunkIDs, chunk.ID)
	}

	size := retrieveEngine.EstimateStorageSize(ctx, embeddingModel, indexInfo)
	if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
		return err
	}
	if size > 0 {
		if err := s.tenantRepo.AdjustStorageUsed(ctx, tenantInfo.ID, -size); err == nil {
			tenantInfo.StorageUsed -= size
			if tenantInfo.StorageUsed < 0 {
				tenantInfo.StorageUsed = 0
			}
		}
		if knowledge.StorageSize >= size {
			knowledge.StorageSize -= size
		} else {
			knowledge.StorageSize = 0
		}
	}
	knowledge.UpdatedAt = time.Now()
	return s.repo.UpdateKnowledge(ctx, knowledge)
}

func faqImportCompletedOutcome(successCount, failedCount, skippedCount int) types.AuditOutcome {
	if successCount > 0 && (failedCount > 0 || skippedCount > 0) {
		return types.AuditOutcomePartial
	}
	if successCount > 0 {
		return types.AuditOutcomeSuccess
	}
	if failedCount > 0 && skippedCount > 0 {
		return types.AuditOutcomePartial
	}
	if failedCount > 0 || skippedCount > 0 {
		return types.AuditOutcomeFailed
	}
	return types.AuditOutcomeSuccess
}

func faqImportActivityDetails(payload *types.FAQImportPayload, progress *types.FAQImportProgress, totalEntries int) map[string]any {
	details := map[string]any{"mode": payload.Mode}
	if progress == nil {
		return details
	}
	total := totalEntries
	if total <= 0 {
		total = progress.Total
	}
	if total > 0 {
		details["total"] = total
	}
	details["count"] = progress.SuccessCount
	if progress.FailedCount > 0 {
		details["failed"] = progress.FailedCount
	}
	skipped := progress.SkippedCount
	if skipped <= 0 && total > 0 {
		skipped = total - progress.SuccessCount - progress.FailedCount
		if skipped < 0 {
			skipped = 0
		}
	}
	if skipped > 0 {
		details["skipped"] = skipped
	}
	return details
}

func (s *knowledgeService) recordFAQImportKBActivity(
	ctx context.Context,
	payload *types.FAQImportPayload,
	progress *types.FAQImportProgress,
	totalEntries int,
	action types.AuditAction,
	outcome types.AuditOutcome,
) {
	if s == nil || payload == nil || payload.DryRun || payload.KBID == "" {
		return
	}
	recordKBActivity(ctx, s.audit, payload.TenantID, payload.KBID, action,
		"faq_entry", payload.KnowledgeID, outcome, faqImportActivityDetails(payload, progress, totalEntries))
}

// ProcessFAQImport handles Asynq FAQ import tasks (including dry run mode)
func (s *knowledgeService) ProcessFAQImport(ctx context.Context, t *asynq.Task) error {
	var payload types.FAQImportPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		logger.Errorf(ctx, "failed to unmarshal FAQ import task payload: %v", err)
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}
	ctx = payload.Initiator.Apply(ctx)
	ctx = context.WithValue(ctx, types.LanguageContextKey, types.ResolveLanguage(ctx, payload.Language))
	ctx = withKBActivityTask(ctx, payload.TaskID, kbActivityTrigger(ctx))

	ctx = logger.WithRequestID(ctx, uuid.New().String())
	ctx = logger.WithField(ctx, "faq_import", payload.TaskID)
	ctx = types.WithExecutionTenant(ctx, payload.TenantID)
	kb, err := s.validateFAQKnowledgeBase(ctx, payload.KBID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeBaseNotFound) {
			return fmt.Errorf("%w: FAQ task KB no longer exists", asynq.SkipRetry)
		}
		return err
	}
	ctx, err = access.WithKBTaskWrite(ctx, kb, payload.TenantID)
	if err != nil {
		return fmt.Errorf("%w: FAQ task KB does not belong to its tenant", asynq.SkipRetry)
	}
	knowledge, err := s.repo.GetKnowledgeByID(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		if errors.Is(err, repository.ErrKnowledgeNotFound) {
			return fmt.Errorf("%w: FAQ task document no longer exists", asynq.SkipRetry)
		}
		return err
	}
	if knowledge == nil || knowledge.TenantID != payload.TenantID || knowledge.KnowledgeBaseID != payload.KBID ||
		knowledge.Type != types.KnowledgeTypeFAQ {
		return fmt.Errorf("%w: FAQ task document does not belong to its KB", asynq.SkipRetry)
	}

	// Son yeniden deneme olup olmadığını belirlemek için görev yeniden deneme bilgilerini al
	retryCount, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	isLastRetry := retryCount >= maxRetry

	tenantInfo, err := s.tenantRepo.GetTenantByID(ctx, payload.TenantID)
	if err != nil {
		logger.Errorf(ctx, "failed to get tenant: %v", err)
		return nil
	}
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenantInfo)

	// entries nesne depolamada saklanıyorsa önce indir
	if payload.EntriesURL != "" && len(payload.Entries) == 0 {
		logger.Infof(ctx, "Downloading FAQ entries from object storage: %s", payload.EntriesURL)
		reader, err := s.fileSvc.GetFile(ctx, payload.EntriesURL)
		if err != nil {
			logger.Errorf(ctx, "Failed to download FAQ entries from object storage: %v", err)
			return fmt.Errorf("failed to download entries: %w", err)
		}
		defer reader.Close()

		entriesData, err := io.ReadAll(reader)
		if err != nil {
			logger.Errorf(ctx, "Failed to read FAQ entries data: %v", err)
			return fmt.Errorf("failed to read entries data: %w", err)
		}

		var entries []types.FAQEntryPayload
		if err := json.Unmarshal(entriesData, &entries); err != nil {
			logger.Errorf(ctx, "Failed to unmarshal FAQ entries: %v", err)
			return fmt.Errorf("failed to unmarshal entries: %w", err)
		}

		payload.Entries = entries
		logger.Infof(ctx, "Downloaded %d FAQ entries from object storage", len(entries))
	}

	logger.Infof(ctx, "Processing FAQ import task: task_id=%s, kb_id=%s, total_entries=%d, dry_run=%v, retry=%d/%d",
		payload.TaskID, payload.KBID, len(payload.Entries), payload.DryRun, retryCount, maxRetry)
	if err := s.validateFAQImportTags(ctx, kb, payload.Entries); err != nil {
		return err
	}

	// Orijinal toplam sayıyı kaydet
	originalTotalEntries := len(payload.Entries)

	// İlerlemeyi başlat
	// Mevcut doğrulama sonucu olup olmadığını kontrol et (yeniden denemede doğrulamayı atlamak için)
	// Not: Yeni progress kaydedilmeden önce sorgulanmalıdır, aksi halde üzerine yazılır
	existingProgress, _ := s.GetFAQImportProgress(ctx, payload.TaskID)

	progress := &types.FAQImportProgress{
		TaskID:         payload.TaskID,
		KBID:           payload.KBID,
		KnowledgeID:    payload.KnowledgeID,
		Status:         types.FAQImportStatusProcessing,
		Progress:       0,
		Total:          originalTotalEntries,
		Processed:      0,
		SuccessCount:   0,
		FailedCount:    0,
		FailedEntries:  make([]types.FAQFailedEntry, 0),
		SuccessEntries: make([]types.FAQSuccessEntry, 0),
		Message:        types.LocalizedText(ctx, "Kayıtlar doğrulanıyor...", "Validating entries..."),
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
		DryRun:         payload.DryRun,
	}
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to save initial FAQ import progress: %v", err)
	}

	var validEntryIndices []int
	if existingProgress != nil && len(existingProgress.ValidEntryIndices) > 0 {
		// Yeniden denemede önceki doğrulama sonucunu doğrudan kullan
		validEntryIndices = existingProgress.ValidEntryIndices
		progress.FailedCount = existingProgress.FailedCount
		progress.FailedEntries = existingProgress.FailedEntries
		logger.Infof(ctx, "Reusing previous validation result: valid=%d, failed=%d",
			len(validEntryIndices), progress.FailedCount)
	} else {
		// İlk adım: Doğrulamayı gerçekleştir (dry run veya import modu fark etmeksizin doğrulama gereklidir)
		validEntryIndices = s.executeFAQDryRunValidation(ctx, &payload, progress)
		// Yeniden denemede doğrulamayı atlamak için doğrulamayı geçen dizinleri kaydet
		progress.ValidEntryIndices = validEntryIndices
		if err := s.saveFAQImportProgress(ctx, progress); err != nil {
			logger.Warnf(ctx, "Failed to save validation result: %v", err)
		}
		logger.Infof(ctx, "FAQ validation completed: total=%d, valid=%d, failed=%d",
			originalTotalEntries, len(validEntryIndices), progress.FailedCount)
	}

	// Dry run modu: Doğrulama tamamlandıktan sonra sonucu doğrudan döndür
	if payload.DryRun {
		return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
	}

	// Import modu: İçe aktarılması gereken geçerli kayıtlar olup olmadığını kontrol et
	if len(validEntryIndices) == 0 {
		// Geçerli kayıt yok, doğrudan tamamla
		return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
	}

	// Geçerli kayıtları çıkar
	validEntries := make([]types.FAQEntryPayload, 0, len(validEntryIndices))
	for _, idx := range validEntryIndices {
		validEntries = append(validEntries, payload.Entries[idx])
	}

	// İlerleme mesajını güncelle
	progress.Message = fmt.Sprintf(types.LocalizedText(ctx, "Doğrulama tamamlandı; %d geçerli kayıt içe aktarılıyor...", "Validation complete; importing %d valid entries..."), len(validEntries))
	progress.UpdatedAt = time.Now().Unix()
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to update FAQ import progress: %v", err)
	}

	// Görev durumunu kontrol et - idempotentlik işlemi (daha önce alınan existingProgress'i yeniden kullan)
	var processedCount int
	if existingProgress != nil {
		if existingProgress.Status == types.FAQImportStatusCompleted {
			logger.Infof(ctx, "FAQ import already completed, skipping: %s", payload.TaskID)
			if clearErr := s.clearRunningFAQImportInfoIfMatches(ctx, payload.KBID, payload.TaskID, payload.InstanceID, payload.EnqueuedAt); clearErr != nil {
				logger.Warnf(ctx, "Failed to clear running FAQ import info for completed task: %v", clearErr)
			}
			return nil // İdempotent: Tamamlanmış görevleri doğrudan döndür
		}
		// İşlenmiş sayıyı al (Not: Bu, validEntries'e göre olan indekstir)
		processedCount = existingProgress.Processed - progress.FailedCount // İşlenmiş sayı - doğrulama başarısızlığı sayısı = içe aktarılmış geçerli kayıt sayısı
		if processedCount < 0 {
			processedCount = 0
		}
		logger.Infof(ctx, "Resuming FAQ import from progress: %d/%d (valid entries)", processedCount, len(validEntries))
	}

	// İdempotentlik işlemi: Kısmen işlenmiş olabilecek chunks ve dizin verilerini temizle
	chunksDeleted, err := s.chunkRepo.DeleteUnindexedChunks(ctx, payload.TenantID, payload.KnowledgeID)
	if err != nil {
		logger.Errorf(ctx, "Failed to delete unindexed chunks: %v", err)
		// Son yeniden denemeyse durumu başarısız olarak güncelle
		if isLastRetry {
			if updateErr := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusFailed, 0, originalTotalEntries, 0, types.LocalizedText(ctx, "Dizinlenmemiş veriler temizlenemedi", "Failed to clean up unindexed data"), err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
			s.recordFAQImportKBActivity(ctx, &payload, progress, originalTotalEntries, types.AuditActionFAQImportFailed, types.AuditOutcomeFailed)
		}
		s.cleanupFAQEntriesFileOnFinalFailure(ctx, payload.EntriesURL, retryCount, maxRetry)
		return fmt.Errorf("failed to delete unindexed chunks: %w", err)
	}
	if len(chunksDeleted) > 0 {
		logger.Infof(ctx, "Deleted unindexed chunks: %d", len(chunksDeleted))

		// Dizin verilerini sil
		embeddingModel, err := s.modelService.GetEmbeddingModel(ctx, kb.EmbeddingModelID)
		if err == nil {
			retrieveEngine, err := retriever.CreateRetrieveEngineForKB(
				ctx, s.retrieveEngine, s.ownership, tenantInfo.ID, kb.VectorStoreID)
			if err == nil {
				chunkIDs := make([]string, 0, len(chunksDeleted))
				for _, chunk := range chunksDeleted {
					chunkIDs = append(chunkIDs, chunk.ID)
				}
				if err := retrieveEngine.DeleteByChunkIDList(ctx, chunkIDs, embeddingModel.GetDimensions(), types.KnowledgeTypeFAQ); err != nil {
					logger.Warnf(ctx, "Failed to delete index data for chunks (may not exist): %v", err)
				} else {
					logger.Infof(ctx, "Successfully deleted index data for %d chunks", len(chunksDeleted))
				}
			}
		}
	}

	// Geçerli kayıtların bir kısmı zaten işlendiyse bu konumdan devam et
	entriesToImport := validEntries
	importMode := payload.Mode
	if processedCount > 0 && processedCount < len(validEntries) {
		entriesToImport = validEntries[processedCount:]
		// Yeniden deneme senaryosunda, daha önce verilerin bir kısmı işlendiyse Append moduna geçmek gerekir
		// Çünkü Replace modundaki silme işlemi ilk çalıştırmada zaten gerçekleştirildi
		// Replace modunu kullanmaya devam ederseniz, calculateReplaceOperations daha önce başarıyla içe aktarılan verileri silinecek olarak işaretler
		// Veri kaybına yol açar
		if payload.Mode == types.FAQBatchModeReplace {
			importMode = types.FAQBatchModeAppend
			logger.Infof(ctx, "Switching to Append mode for retry, original mode was Replace")
		}
		logger.Infof(ctx, "Continuing FAQ import from entry %d, remaining: %d entries", processedCount, len(entriesToImport))
	}

	// FAQBatchUpsertPayload oluştur (doğrulamayı geçen geçerli girdileri kullanarak)
	faqPayload := &types.FAQBatchUpsertPayload{
		Entries: entriesToImport,
		Mode:    importMode,
	}

	// FAQ içe aktarmayı yürüt (ilerleme hesaplaması için işlenmiş ofseti ilet)
	if err := s.executeFAQImport(ctx, payload.TaskID, payload.KBID, faqPayload, payload.TenantID, progress.FailedCount+processedCount, progress); err != nil {
		logger.Errorf(ctx, "FAQ import task failed: %s, error: %v", payload.TaskID, err)
		// Son yeniden denemeyse, durumu başarısız olarak güncelle
		if isLastRetry {
			if updateErr := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusFailed, 0, originalTotalEntries, len(validEntries), types.LocalizedText(ctx, "İçe aktarma başarısız", "Import failed"), err.Error()); updateErr != nil {
				logger.Errorf(ctx, "Failed to update task status to failed: %v", updateErr)
			}
			s.recordFAQImportKBActivity(ctx, &payload, progress, originalTotalEntries, types.AuditActionFAQImportFailed, types.AuditOutcomeFailed)
		}
		s.cleanupFAQEntriesFileOnFinalFailure(ctx, payload.EntriesURL, retryCount, maxRetry)
		return fmt.Errorf("FAQ import failed: %w", err)
	}

	// Görev başarıyla tamamlandı
	logger.Infof(ctx, "FAQ import task completed: %s, imported: %d, failed: %d",
		payload.TaskID, len(progress.SuccessEntries), progress.FailedCount)

	// Nihai işlemeyi tamamla (başarısız girdilerin CSV'sini oluşturma vb.)
	return s.finalizeFAQValidation(ctx, &payload, progress, originalTotalEntries)
}

// finalizeFAQValidation, FAQ doğrulama/içe aktarma görevini tamamlar ve başarısız girdilerin CSV'sini oluşturur (varsa)
func (s *knowledgeService) finalizeFAQValidation(ctx context.Context, payload *types.FAQImportPayload,
	progress *types.FAQImportProgress, originalTotalEntries int,
) error {
	// Nesne depolamadaki entries dosyasını temizle (varsa)
	if payload.EntriesURL != "" {
		if err := s.fileSvc.DeleteFile(ctx, payload.EntriesURL); err != nil {
			logger.Warnf(ctx, "Failed to delete FAQ entries file from object storage: %v", err)
		} else {
			logger.Infof(ctx, "Deleted FAQ entries file from object storage: %s", payload.EntriesURL)
		}
	}
	progress.UpdatedAt = time.Now().Unix()

	// Başarısız girdiler varsa CSV dosyası oluştur
	if len(progress.FailedEntries) > 0 {
		csvURL, err := s.generateFailedEntriesCSV(ctx, payload.TenantID, payload.TaskID, progress.FailedEntries)
		if err != nil {
			logger.Warnf(ctx, "Failed to generate failed entries CSV: %v", err)
		} else {
			progress.FailedEntriesURL = csvURL
			progress.FailedEntries = nil // Satır içi verileri temizle, URL kullan
			progress.Message += types.LocalizedText(ctx, " (başarısız kayıtlar CSV olarak dışa aktarıldı)", " (failed entries exported as CSV)")
		}
	}

	// Nihai istatistikler saveFAQImportResultToDatabase öncesinde hesaplanmalıdır.
	progress.Status = types.FAQImportStatusCompleted
	progress.Progress = 100
	progress.Processed = originalTotalEntries

	if len(progress.ValidEntryIndices) > 0 {
		progress.SuccessCount = len(progress.ValidEntryIndices) - progress.PartialFailedCount
	} else if len(progress.SuccessEntries) > 0 {
		progress.SuccessCount = len(progress.SuccessEntries) - progress.PartialFailedCount
	} else {
		progress.SuccessCount = originalTotalEntries - progress.FailedCount - progress.PartialFailedCount
	}
	if progress.SuccessCount < 0 {
		progress.SuccessCount = 0
	}

	if progress.AddedCount == 0 && progress.MergedCount > 0 {
		progress.AddedCount = progress.SuccessCount - progress.MergedCount
		if progress.AddedCount < 0 {
			progress.AddedCount = 0
		}
	} else if progress.AddedCount == 0 {
		progress.AddedCount = progress.SuccessCount
	}

	skippedCount := originalTotalEntries - progress.SuccessCount - progress.PartialFailedCount - progress.FailedCount
	if skippedCount < 0 {
		skippedCount = 0
	}
	progress.SkippedCount = skippedCount

	if payload.DryRun {
		progress.Message = s.buildFAQImportResultMessage(ctx, types.LocalizedText(ctx, "Doğrulama tamamlandı", "Validation complete"), progress)
	} else {
		progress.Message = s.buildFAQImportResultMessage(ctx, types.LocalizedText(ctx, "İçe aktarma tamamlandı", "Import complete"), progress)
	}

	// dry run modu değilse, içe aktarma sonuç istatistiklerini veritabanına kaydet
	if !payload.DryRun {
		if err := s.saveFAQImportResultToDatabase(ctx, payload, progress, originalTotalEntries); err != nil {
			logger.Warnf(ctx, "Failed to save FAQ import result to database: %v", err)
		}

		// Yalnızca replace modu kullanılmayan Tag'leri temizler
		// append modu, kullanıcının önceden oluşturduğu boş etiketleri silmemelidir
		if payload.Mode == types.FAQBatchModeReplace {
			deletedTags, err := s.tagRepo.DeleteUnusedTags(ctx, payload.TenantID, payload.KBID)
			if err != nil {
				logger.Warnf(ctx, "FAQ import task %s: failed to cleanup unused tags: %v", payload.TaskID, err)
			} else if deletedTags > 0 {
				logger.Infof(ctx, "FAQ import task %s: cleaned up %d unused tags after replace import", payload.TaskID, deletedTags)
			}
		}
	}

	// running key'in doğru şekilde temizlendiğinden emin olmak için updateFAQImportProgressStatus kullan
	// Ancak önce diğer alanları kaydetmek gerekir, çünkü updateFAQImportProgressStatus tüm alanları kaydetmez
	if err := s.saveFAQImportProgress(ctx, progress); err != nil {
		logger.Warnf(ctx, "Failed to save final FAQ import progress: %v", err)
	}

	// Ardından running key'i temizlemek için durum güncellemesini çağır
	if err := s.updateFAQImportProgressStatus(ctx, payload.TaskID, payload.InstanceID, payload.EnqueuedAt, types.FAQImportStatusCompleted,
		100, originalTotalEntries, originalTotalEntries, progress.Message, ""); err != nil {
		logger.Warnf(ctx, "Failed to update final FAQ import status: %v", err)
	}

	logger.Infof(ctx, "FAQ task completed: %s, dry_run=%v, success: %d, added: %d, merged: %d, failed: %d, partial_failed: %d",
		payload.TaskID, payload.DryRun, progress.SuccessCount, progress.AddedCount, progress.MergedCount, progress.FailedCount, progress.PartialFailedCount)

	if !payload.DryRun {
		outcome := faqImportCompletedOutcome(progress.SuccessCount, progress.FailedCount, progress.SkippedCount)
		s.recordFAQImportKBActivity(ctx, payload, progress, originalTotalEntries,
			types.AuditActionFAQImportCompleted, outcome)
	}

	return nil
}

// executeFAQMergeOperations, append-mode birleştirme işlemlerini toplu olarak yürütür: mevcut chunk'ları günceller
// metadata / content / dizinlerini günceller. DB gidiş gelişlerini azaltmak için ListChunksByID ile toplu yükleme + SaveChunks işlemsel toplu kaydetme kullanır
// Herhangi bir toplu işlem başarısız olursa hemen döner; tüm içe aktarmanın geri alınıp alınmayacağına executeFAQImport
// içindeki defer recovery karar verir.
//
// Gerekirse burada chunks'ları toplu birleştirmek yerine tek tek fan-out dizinleme (EFPutDocument) yap
// Yeniden oluştur: dizinin temel katmanı SourceID'ye göre üzerine yazar; birleştirilmiş nihai içeriği doğrudan put et
// yeterlidir。
func (s *knowledgeService) executeFAQMergeOperations(
	ctx context.Context,
	taskID string,
	kb *types.KnowledgeBase,
	faqKnowledge *types.Knowledge,
	embeddingModel embedding.Embedder,
	indexMode types.FAQIndexMode,
	mergeOps []faqMergeOperation,
	progress *types.FAQImportProgress,
) (int, error) {
	if len(mergeOps) == 0 {
		return 0, nil
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	mergedCount := 0

	for batchStart := 0; batchStart < len(mergeOps); batchStart += faqImportBatchSize {
		batchEnd := batchStart + faqImportBatchSize
		if batchEnd > len(mergeOps) {
			batchEnd = len(mergeOps)
		}
		batch := mergeOps[batchStart:batchEnd]

		// 1. Tam chunk'ları toplu yükle（calculateAppendOperations içinde yalnızca bazı alanlar yüklendi，
		//    status/is_enabled/flags/seq_id vb. eksik; doğrudan güncelleme bu alanların sıfır değerlerle üzerine yazılmasına neden olur）
		chunkIDs := make([]string, len(batch))
		for i, op := range batch {
			chunkIDs[i] = op.ExistingChunk.ID
		}
		fullChunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, chunkIDs)
		if err != nil {
			logger.Errorf(ctx, "FAQ import task %s: failed to batch reload chunks for merge: %v", taskID, err)
			return mergedCount, fmt.Errorf("failed to batch reload chunks for merge: %w", err)
		}
		chunkMap := make(map[string]*types.Chunk, len(fullChunks))
		for _, c := range fullChunks {
			chunkMap[c.ID] = c
		}

		// 2. Birleştirilmiş verileri tek tek uygula
		mergedChunks := make([]*types.Chunk, 0, len(batch))
		for _, op := range batch {
			fullChunk, ok := chunkMap[op.ExistingChunk.ID]
			if !ok {
				logger.Errorf(ctx, "FAQ import task %s: chunk %s not found during batch reload", taskID, op.ExistingChunk.ID)
				return mergedCount, fmt.Errorf("chunk %s not found during batch reload", op.ExistingChunk.ID)
			}

			if err := fullChunk.SetFAQMetadata(op.MergedMeta); err != nil {
				logger.Errorf(ctx, "FAQ import task %s: failed to set merged metadata for chunk %s: %v", taskID, fullChunk.ID, err)
				return mergedCount, fmt.Errorf("failed to set merged FAQ metadata: %w", err)
			}

			fullChunk.Content = buildFAQChunkContent(op.MergedMeta, indexMode)
			fullChunk.ContentHash = types.CalculateFAQContentHash(op.MergedMeta)
			fullChunk.UpdatedAt = time.Now()

			// Operasyon durumunu yeni değerle üzerine yaz
			if op.Entry.IsEnabled != nil {
				fullChunk.IsEnabled = *op.Entry.IsEnabled
			}
			if op.Entry.IsRecommended != nil {
				if *op.Entry.IsRecommended {
					fullChunk.Flags = fullChunk.Flags.SetFlag(types.ChunkFlagRecommended)
				} else {
					fullChunk.Flags = fullChunk.Flags.ClearFlag(types.ChunkFlagRecommended)
				}
			}

			mergedChunks = append(mergedChunks, fullChunk)
		}

		// 3. İşlem içinde toplu kaydet（GORM Save tüm alanları günceller; metadata/content_hash kalıcılığını sağlar）
		if err := s.chunkRepo.SaveChunks(ctx, mergedChunks); err != nil {
			logger.Errorf(ctx, "FAQ import task %s: failed to batch save merged chunks: %v", taskID, err)
			return mergedCount, fmt.Errorf("failed to batch save merged chunks: %w", err)
		}

		// 4. Dizini yeniden oluştur。Önce eski dizini sil: ES v8, Qdrant gibi motorlar aynı SourceID'nin üzerine yazmaz。
		if err := s.indexFAQChunks(ctx, kb, faqKnowledge, mergedChunks, embeddingModel, false, true); err != nil {
			return mergedCount, fmt.Errorf("failed to re-index merged chunks: %w", err)
		}

		// 5. Başarılı kayıtların bilgilerini topla
		tagsByID := s.loadFAQTagsForChunks(ctx, tenantID, mergedChunks)
		for i, op := range batch {
			chunk := mergedChunks[i]
			meta := op.MergedMeta
			tagID, tagName := faqTagInfo(tagsByID, chunk.TagID)
			progress.SuccessEntries = append(progress.SuccessEntries, types.FAQSuccessEntry{
				Index:            op.Detail.Index,
				SeqID:            chunk.SeqID,
				TagID:            tagID,
				TagName:          tagName,
				StandardQuestion: meta.StandardQuestion,
			})
		}

		mergedCount += len(batch)

		logger.Infof(ctx, "FAQ import task %s: merged batch %d-%d (%d chunks)", taskID, batchStart+1, batchEnd, len(mergedChunks))
	}

	return mergedCount, nil
}

// loadFAQTagsForChunks resolves every distinct tag referenced by chunks with a
// single query. Lookup failures are logged and yield an empty map so the
// import result degrades to "no tag info" instead of aborting the batch.
func (s *knowledgeService) loadFAQTagsForChunks(
	ctx context.Context, tenantID uint64, chunks []*types.Chunk,
) map[string]*types.KnowledgeTag {
	tagsByID := make(map[string]*types.KnowledgeTag)
	seen := make(map[string]struct{})
	ids := make([]string, 0)
	for _, chunk := range chunks {
		if chunk == nil || chunk.TagID == "" {
			continue
		}
		if _, ok := seen[chunk.TagID]; ok {
			continue
		}
		seen[chunk.TagID] = struct{}{}
		ids = append(ids, chunk.TagID)
	}
	if len(ids) == 0 {
		return tagsByID
	}
	tags, err := s.tagRepo.GetByIDs(ctx, tenantID, ids)
	if err != nil {
		logger.Warnf(ctx, "Failed to load FAQ tags for import result: %v", err)
		return tagsByID
	}
	for _, tag := range tags {
		if tag != nil {
			tagsByID[tag.ID] = tag
		}
	}
	return tagsByID
}

// faqTagInfo returns the external (seq_id, name) pair for tagID, or zero values
// when the chunk has no tag or the tag could not be loaded.
func faqTagInfo(tagsByID map[string]*types.KnowledgeTag, tagID string) (int64, string) {
	if tagID == "" {
		return 0, ""
	}
	if tag, ok := tagsByID[tagID]; ok && tag != nil {
		return tag.SeqID, tag.Name
	}
	return 0, ""
}

// buildFAQImportResultMessage, FAQ içe aktarma / doğrulamanın nihai sonucuna ilişkin insan tarafından okunabilir mesajı oluşturur。
// Ön yüz bunu doğrudan toast / görev listesinde gösterir, bu nedenle basit ve net tut:
//   - Varsayılan biçim："İçe aktarma tamamlandı / N kayıt yüklendi / X kayıt başarılı [/ Y kayıt başarısız] [/ Z kayıt kısmen başarısız]"
//   - MergedCount > 0 olduğunda bölünmüş biçimi kullan："/ X kayıt eklendi / Y kayıt birleştirilerek güncellendi"，
//     böylece kullanıcı append modunda yalnızca toplam başarılı kayıt sayısını değil, kaç geçmiş FAQ'nun birleştirildiğini de görür。
//
// Dahili master özgün uygulaması; HEAD sürümünden önce yoktu, tüm tamamlanma mesajları yalnızca "N/M kaydı işleniyor" idi。
func (s *knowledgeService) buildFAQImportResultMessage(ctx context.Context, prefix string, progress *types.FAQImportProgress) string {
	parts := []string{prefix}
	parts = append(parts, fmt.Sprintf(types.LocalizedText(ctx, "Yüklenen: %d", "Uploaded: %d"), progress.Total))

	if progress.MergedCount > 0 {
		parts = append(parts, fmt.Sprintf(types.LocalizedText(ctx, "Eklenen: %d", "Added: %d"), progress.AddedCount))
		parts = append(parts, fmt.Sprintf(types.LocalizedText(ctx, "Birleştirilip güncellenen: %d", "Merged and updated: %d"), progress.MergedCount))
	} else {
		parts = append(parts, fmt.Sprintf(types.LocalizedText(ctx, "Başarılı: %d", "Succeeded: %d"), progress.SuccessCount))
	}

	if progress.FailedCount > 0 {
		parts = append(parts, fmt.Sprintf(types.LocalizedText(ctx, "Başarısız: %d", "Failed: %d"), progress.FailedCount))
	}
	if progress.PartialFailedCount > 0 {
		parts = append(parts, fmt.Sprintf(types.LocalizedText(ctx, "Kısmen başarısız: %d", "Partially failed: %d"), progress.PartialFailedCount))
	}

	return strings.Join(parts, " / ")
}

const (
	faqImportProgressKeyPrefix = "faq_import_progress:"
	faqImportRunningKeyPrefix  = "faq_import_running:"
	faqImportProgressTTL       = 3 * time.Hour
)

// getFAQImportProgressKey returns the Redis key for storing FAQ import progress
func getFAQImportProgressKey(taskID string) string {
	return faqImportProgressKeyPrefix + taskID
}

// getFAQImportRunningKey returns the Redis key for storing running task ID by KB ID
func getFAQImportRunningKey(kbID string) string {
	return faqImportRunningKeyPrefix + kbID
}

// saveFAQImportProgress saves the FAQ import progress to Redis
func (s *knowledgeService) saveFAQImportProgress(ctx context.Context, progress *types.FAQImportProgress) error {
	if s.redisClient == nil {
		progress.UpdatedAt = time.Now().Unix()
		s.memFAQProgress.Store(progress.TaskID, progress)
		return nil
	}
	key := getFAQImportProgressKey(progress.TaskID)
	progress.UpdatedAt = time.Now().Unix()
	data, err := json.Marshal(progress)
	if err != nil {
		return fmt.Errorf("failed to marshal FAQ import progress: %w", err)
	}
	return s.redisClient.Set(ctx, key, data, faqImportProgressTTL).Err()
}

// GetFAQImportProgress retrieves the progress of an FAQ import task
func (s *knowledgeService) GetFAQImportProgress(ctx context.Context, taskID string) (*types.FAQImportProgress, error) {
	if s.redisClient == nil {
		if v, ok := s.memFAQProgress.Load(taskID); ok {
			return v.(*types.FAQImportProgress), nil
		}
		return nil, werrors.NewNotFoundError("FAQ import task not found")
	}
	key := getFAQImportProgressKey(taskID)
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, werrors.NewNotFoundError("FAQ import task not found")
		}
		return nil, fmt.Errorf("failed to get FAQ import progress from Redis: %w", err)
	}

	var progress types.FAQImportProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FAQ import progress: %w", err)
	}

	// If task is completed, enrich with persisted result fields from database
	if progress.Status == types.FAQImportStatusCompleted && progress.KnowledgeID != "" {
		tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
		knowledge, err := s.repo.GetKnowledgeByID(ctx, tenantID, progress.KnowledgeID)
		if err == nil && knowledge != nil {
			if result, err := knowledge.GetLastFAQImportResult(); err == nil && result != nil {
				progress.SuccessCount = result.SuccessCount
				progress.FailedCount = result.FailedCount
				progress.PartialFailedCount = result.PartialFailedCount
				progress.SkippedCount = result.SkippedCount
				progress.MergedCount = result.MergedCount
				progress.AddedCount = result.AddedCount
				progress.ImportMode = result.ImportMode
				progress.ImportedAt = result.ImportedAt
				progress.DisplayStatus = result.DisplayStatus
				progress.ProcessingTime = result.ProcessingTime
				if result.FailedEntriesURL != "" {
					progress.FailedEntriesURL = result.FailedEntriesURL
				}
			}
		}
	}

	return &progress, nil
}

// UpdateLastFAQImportResultDisplayStatus updates the display status of FAQ import result
func (s *knowledgeService) UpdateLastFAQImportResultDisplayStatus(ctx context.Context, kbID string, displayStatus string) error {
	// displayStatus parametresini doğrula
	if displayStatus != "open" && displayStatus != "close" {
		return werrors.NewBadRequestError("invalid display status, must be 'open' or 'close'")
	}

	kb, ctx, err := s.writableFAQKnowledgeBase(ctx, kbID)
	if err != nil {
		return err
	}
	tenantID := kb.TenantID

	// FAQ türündeki knowledge'ı bul
	knowledgeList, err := s.repo.ListKnowledgeByKnowledgeBaseID(ctx, tenantID, kbID)
	if err != nil {
		return fmt.Errorf("failed to list knowledge: %w", err)
	}

	// FAQ türündeki knowledge'ı bul
	var faqKnowledge *types.Knowledge
	for _, k := range knowledgeList {
		if k.Type == types.KnowledgeTypeFAQ {
			faqKnowledge = k
			break
		}
	}

	if faqKnowledge == nil {
		return werrors.NewNotFoundError("FAQ knowledge not found in this knowledge base")
	}

	// Mevcut içe aktarma sonucunu ayrıştır
	result, err := faqKnowledge.GetLastFAQImportResult()
	if err != nil {
		return fmt.Errorf("failed to parse FAQ import result: %w", err)
	}

	if result == nil {
		return werrors.NewNotFoundError("no FAQ import result found")
	}

	// Görüntüleme durumunu güncelle
	result.DisplayStatus = displayStatus

	// Güncellenmiş sonucu kaydet
	if err := faqKnowledge.SetLastFAQImportResult(result); err != nil {
		return fmt.Errorf("failed to set FAQ import result: %w", err)
	}

	// Veritabanını güncelle
	if err := s.repo.UpdateKnowledge(ctx, faqKnowledge); err != nil {
		return fmt.Errorf("failed to update knowledge: %w", err)
	}

	return nil
}
