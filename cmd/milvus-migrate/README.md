# Milvus çok dilli BM25 taşıması

Eski Rethra sürümlerinde Milvus Collection yalnızca bir varsayılan metin çözümleyiciye sahiptir; Çince içerik geçerli BM25 anahtar kelimeleri üretmeyebilir. `milvus-migrate` eski Collection'ı korur ve mevcut yoğun vektörleri yeni çok dilli Collection'a kopyalar; Milvus her satırın `language` alanına göre BM25 seyrek vektörlerini yeniden üretir.

Yoğun vektörlerin metriği (IP / COSINE / L2) varsayılan olarak kaynak Collection'ın embedding indeksinden okunur. Kaynak Collection'dan farklı bir değere değiştirmeyin; aksi halde aynı vektörler yanlış uzaklığa göre sıralanır.

## Kullanım

Önce Milvus'un çalıştığından emin olun, ardından proje kök dizininde çalıştırın:

```bash
go run ./cmd/milvus-migrate --source rethra_embeddings --target rethra_embeddings_multilingual
```

Geçerli shell `MILVUS_ADDRESS` ve `MILVUS_COLLECTION` değişkenlerini zaten içe aktarmışsa ilgili parametreler atlanabilir. Değişkenleri yalnızca `.env.local` dosyasına yazmak onları `go run` sürecine otomatik aktarmaz; emin değilseniz parametreleri açıkça verin:

```bash
go run ./cmd/milvus-migrate \
  --address 127.0.0.1:19530 \
  --source rethra_embeddings \
  --target rethra_embeddings_multilingual
```

Metriği doğrulamak için kaynak Collection ile aynı `--metric-type` (ve `MILVUS_METRIC_TYPE` ortam değişkeni) verilebilir. Verilen değer kaynak indeksle aynı olmalıdır, aksi halde taşıma başarısız olur.

Taşıma tamamlandıktan sonra Rethra'nın `MILVUS_COLLECTION` değerini hedef önek olarak değiştirip servisi yeniden başlatın; **mevcut `MILVUS_METRIC_TYPE` değerini değiştirmeyin**:

```dotenv
MILVUS_COLLECTION=rethra_embeddings_multilingual
```

Arama listesi `{önek}_{boyut}` biçimine tam eşleşir; bu yüzden `rethra_embeddings` yanlışlıkla `rethra_embeddings_multilingual_*` içinde arama yapmaz. Önek değiştirildikten sonra yeni yazmalar ve vektör/anahtar kelime aramaları yeni Collection'ı kullanır.

Taşıma programı eski Collection'ı silmez. Çince ve İngilizce BM25 geri çağırmanın çalıştığını doğruladıktan sonra eski Collection'ı Milvus yönetim aracıyla silin; silmeden önce yedek alın.

Taşıma varsayılan olarak her partide 64 satır okur ve yalnızca hedef Collection'ı yeniden oluşturmak için gereken alanları okur; eski Collection'da üretilmiş BM25 seyrek vektörlerini okumaz. Tek bir metin parçası çok uzunsa parti boyutunu açıkça düşürebilirsiniz, örneğin `--batch-size 32` ekleyerek.

## Windows PowerShell notları

Rethra'nın `internal/utils` paketi SQL ayrıştırmak için `pg_query_go` kullanır; bu bağımlılık CGO gerektirir. `go run` doğrudan çalıştırıldığında geçerli oturumda `CGO_ENABLED=0` ise `undefined: pg_query.Parse` veya `undefined: pg_query.Deparse` hatası görülür.

Proje MSYS2 GCC ile birlikte gelir; geçerli PowerShell oturumunda CGO'yu geçici olarak etkinleştirip taşımayı çalıştırabilirsiniz. Aşağıdaki ayarlar yalnızca geçerli pencereyi etkiler, sistem düzeyindeki Go yapılandırmasını değiştirmez:

```powershell
# Geçerli dizinin go.mod içeren proje kökü olduğunu doğrula
if (!(Test-Path -LiteralPath '.\go.mod')) { throw 'Lütfen önce Rethra proje kök dizinine geçin' }

# Go'nun C derleyicisini bulamamasını önlemek için projeyle gelen GCC'yi kullan
$compilerBin = Join-Path (Get-Location) '.local-tools\msys64\ucrt64\bin'
$env:CGO_ENABLED = '1'
$env:CC = Join-Path $compilerBin 'gcc.exe'
$env:CXX = Join-Path $compilerBin 'g++.exe'
$env:PATH = "$compilerBin;$env:PATH"

# Taşımayı çalıştır; metrik kaynak Collection'dan alınır, eski Collection silinmez
go run ./cmd/milvus-migrate --address 127.0.0.1:19530 --source rethra_embeddings --target rethra_embeddings_multilingual
```

Proje dizininde `.local-tools\msys64\ucrt64\bin` yoksa önce kullanılabilir bir GCC kurun ve `$env:CC`, `$env:CXX` değerlerini ilgili `gcc.exe`, `g++.exe` mutlak yollarına değiştirin.
