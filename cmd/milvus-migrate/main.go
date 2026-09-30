// `milvus-migrate`, eski Milvus Collection'ı Çince ve İngilizce BM25 destekleyen yeni bir Collection'a kopyalar.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	milvusRepo "github.com/acrbaran/rag/internal/application/repository/retriever/milvus"
	client "github.com/milvus-io/milvus/client/v2/milvusclient"
)

func main() {
	sourceDefault := strings.TrimSpace(os.Getenv("MILVUS_COLLECTION"))
	if sourceDefault == "" {
		sourceDefault = "rethra_embeddings"
	}
	targetDefault := sourceDefault + "_multilingual"

	source := flag.String("source", sourceDefault, "Eski Collection öneki, örneğin rethra_embeddings")
	target := flag.String("target", targetDefault, "Yeni Collection öneki, örneğin rethra_embeddings_multilingual")
	address := flag.String("address", envOr("MILVUS_ADDRESS", "localhost:19530"), "Milvus adresi")
	username := flag.String("username", os.Getenv("MILVUS_USERNAME"), "Milvus kullanıcı adı")
	password := flag.String("password", os.Getenv("MILVUS_PASSWORD"), "Milvus parolası")
	database := flag.String("database", os.Getenv("MILVUS_DB_NAME"), "Milvus veritabanı adı")
	metric := flag.String(
		"metric-type",
		strings.TrimSpace(os.Getenv("MILVUS_METRIC_TYPE")),
		"Yoğun vektör mesafesi: IP, COSINE veya L2; belirtilmezse kaynak Collection kullanılır",
	)
	batchSize := flag.Int("batch-size", 64, "Her taşıma grubundaki satır sayısı; metin uzunsa daha da azaltılabilir")
	flag.Parse()

	metricType, err := milvusRepo.ParseMetricType(*metric)
	if err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(*source) == strings.TrimSpace(*target) {
		log.Fatal("--source ve --target farklı olmalıdır; taşıma yalnızca yeni Collection'a kopyalar, eski Collection'ın üzerine yazmaz")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	connectCtx, cancelConnect := context.WithTimeout(ctx, 30*time.Second)
	milvusClient, err := client.New(connectCtx, &client.ClientConfig{
		Address:  *address,
		Username: *username,
		Password: *password,
		DBName:   *database,
	})
	cancelConnect()
	if err != nil {
		log.Fatalf("Milvus bağlantısı başarısız: %v", err)
	}
	defer func() {
		closeCtx, cancelClose := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelClose()
		if err := milvusClient.Close(closeCtx); err != nil {
			log.Printf("Milvus bağlantısı kapatılamadı: %v", err)
		}
	}()

	summary, err := milvusRepo.MigrateLegacyCollections(ctx, milvusClient, milvusRepo.MultilingualMigrationOptions{
		SourceCollectionBaseName: *source,
		TargetCollectionBaseName: *target,
		MetricType:               metricType,
		BatchSize:                *batchSize,
	})
	if err != nil {
		log.Fatalf(
			"Taşıma başarısız (%d Collection kontrol edildi, %d Collection ve %d satır kopyalandı): %v",
			summary.ExaminedCollections,
			summary.MigratedCollections,
			summary.MigratedRows,
			err,
		)
	}
	fmt.Printf(
		"Taşıma tamamlandı: %d Collection kontrol edildi, %d Collection ve %d satır kopyalandı. Eski Collection silinmeden korundu.\n",
		summary.ExaminedCollections,
		summary.MigratedCollections,
		summary.MigratedRows,
	)
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
