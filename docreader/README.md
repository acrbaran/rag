# DocReader Service

DocReader, Rethra projesinde belge ayrıştırma ve işlemeden sorumlu gRPC servisidir. Birçok belge biçimini okuma, OCR tanıma ve çok kipli işleme gibi özellikleri destekler.

## Docker Compose ortam değişkeni yapılandırması

`docker-compose.yml` dosyasında docreader servisi için şu ortam değişkenleri yapılandırılmıştır:

```yaml
docreader:
  image: rethra-docreader:${RETHRA_VERSION:-latest}
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://localhost:${MINIO_PORT:-9000}
    - MINERU_ENDPOINT=${MINERU_ENDPOINT:-}
    - MAX_FILE_SIZE_MB=${MAX_FILE_SIZE_MB:-}
```

### Ortam değişkenlerinin açıklaması

#### 1. MINIO_ENDPOINT

- **Açıklama**: MinIO servisinin dahili erişim adresi (konteynerler arası iletişim)
- **Varsayılan**: `minio:9000`
- **Kullanım**: DocReader servisi bu adresle MinIO nesne depolama servisine bağlanır; belge işleme sırasında dosyaları okumak ve depolamak için kullanılır
- **Yapılandırma örneği**:
  ```yaml
  - MINIO_ENDPOINT=minio:9000  # Docker ağı içindeki adres
  ```

#### 2. MINIO_PUBLIC_ENDPOINT

- **Açıklama**: MinIO servisinin genel erişim adresi (dış erişim)
- **Varsayılan**: `http://localhost:9000`
- **Kullanım**: Dışarıdan erişilebilen dosya URL'leri üretmek için kullanılır; örneğin belge ayrıştırıldıktan sonra görsel bağlantıları döndürülürken
- **Önemli not**: 
  - Başka cihazlardan ya da konteynerlerden erişilmesi gerekiyorsa `localhost` gerçek IP adresiyle değiştirilmelidir
  - Bağlantı noktasını özelleştirmek için `.env` dosyasında `MINIO_PORT` yapılandırılabilir
- **Yapılandırma örneği**:
  ```bash
  # .env dosyası
  MINIO_PORT=9000
  ```
  Ya da doğrudan docker-compose.yml içinde değiştirin:
  ```yaml
  - MINIO_PUBLIC_ENDPOINT=http://192.168.1.100:9000  # gerçek IP kullanın
  ```

#### 3. MINERU_ENDPOINT

- **Açıklama**: MinerU servisinin erişim adresi (isteğe bağlı)
- **Varsayılan**: Boş (MinerU kullanılmaz)
- **Kullanım**: MinerU, daha karmaşık belge yapılarını tanıyıp işleyebilen gelişmiş bir belge ayrıştırma servisidir. Bu değişken yapılandırıldığında DocReader belge ayrıştırma için MinerU'yu çağırabilir
- **Yapılandırma örneği**:
  ```bash
  # .env dosyası
  MINERU_ENDPOINT=http://mineru-service:8080
  ```

#### 4. MAX_FILE_SIZE_MB

- **Açıklama**: Yüklenmesine izin verilen en büyük dosya boyutu (birim: MB)
- **Varsayılan**: `50` MB
- **Kullanım**: gRPC servisinin kabul ettiği dosya boyutunu sınırlar; çok büyük dosyaların servisi çökertmesini ya da performans sorunlarına yol açmasını önler
- **Yapılandırma örneği**:
  ```bash
  # .env dosyası
  MAX_FILE_SIZE_MB=100  # en fazla 100MB dosyaya izin ver
  ```

## Yapılandırılabilen diğer ortam değişkenleri

docker-compose.yml içinde yapılandırılmış değişkenlere ek olarak DocReader şu ortam değişkenlerini de destekler (gerektiğinde eklenebilir):

### gRPC yapılandırması

- `DOCREADER_GRPC_MAX_WORKERS`: gRPC servisinin en fazla iş parçacığı sayısı (varsayılan: 4)
- `DOCREADER_GRPC_PORT`: gRPC servisinin dinlediği bağlantı noktası (varsayılan: 50051)

### Ayrıştırıcı kaynak denetimi

- `DOCREADER_MARKITDOWN_MAX_WORKERS`: MarkItDown ayrıştırmasının en fazla eşzamanlılık sayısı (varsayılan: 1; 0 yapılırsa sınırlama kapanır)
- `DOCREADER_PDF_RENDER_MAX_WORKERS`: Taranmış PDF'lerin görsele dönüştürülmesinde en fazla eşzamanlılık sayısı (varsayılan: 1; 0 yapılırsa sınırlama kapanır)
- `DOCREADER_PDF_RENDER_DPI`: Taranmış PDF dönüştürme DPI değeri (varsayılan: 200)
- `DOCREADER_PDF_JPEG_QUALITY`: Taranmış PDF çıktısının JPEG kalitesi (varsayılan: 85; değer otomatik olarak 1-95 aralığında tutulur)

### OCR / VLM

DocReader artık kendi içinde OCR ve VLM arka ucu barındırmaz. Taranmış PDF'ler JPEG görsellere dönüştürülür ve OCR/VLM servislerini Go App tarafı çağırır; ilgili yapılandırma için ana proje belgelerine bakın.

### Depolama yapılandırması

DocReader birden çok depolama arka ucunu destekler:

#### MinIO/S3 depolama (önerilen)

- `STORAGE_TYPE`: `minio` olarak ayarlayın
- `MINIO_ACCESS_KEY_ID`: MinIO erişim anahtarı ID'si (varsayılan: minioadmin)
- `MINIO_SECRET_ACCESS_KEY`: MinIO gizli erişim anahtarı (varsayılan: minioadmin)
- `MINIO_BUCKET_NAME`: MinIO bucket adı (varsayılan: Rethra)
- `MINIO_PATH_PREFIX`: Dosya yolu öneki
- `MINIO_USE_SSL`: SSL kullanılıp kullanılmayacağı (varsayılan: false)

#### Tencent Cloud COS depolama

- `STORAGE_TYPE`: `cos` olarak ayarlayın
- `COS_SECRET_ID`: COS erişim anahtarı ID'si
- `COS_SECRET_KEY`: COS gizli erişim anahtarı
- `COS_REGION`: COS bölgesi
- `COS_BUCKET_NAME`: COS bucket adı
- `COS_APP_ID`: COS uygulama ID'si
- `COS_PATH_PREFIX`: Dosya yolu öneki
- `COS_ENABLE_OLD_DOMAIN`: Eski alan adının kullanılıp kullanılmayacağı (varsayılan: true)

#### Alibaba Cloud OSS depolama

- `STORAGE_TYPE`: `oss` olarak ayarlayın
- `OSS_ACCESS_KEY_ID`: OSS erişim anahtarı ID'si
- `OSS_ACCESS_KEY_SECRET`: OSS gizli erişim anahtarı
- `OSS_ENDPOINT`: OSS uç noktası (örneğin `oss-cn-hangzhou.aliyuncs.com`)
- `OSS_BUCKET_NAME`: OSS bucket adı
- `OSS_REGION`: OSS bölgesi (örneğin `cn-hangzhou`)
- `OSS_PATH_PREFIX`: Dosya yolu öneki

### Proxy yapılandırması

Dış servislere proxy üzerinden erişilmesi gerekiyorsa:

- `EXTERNAL_HTTP_PROXY`: HTTP proxy adresi
- `EXTERNAL_HTTPS_PROXY`: HTTPS proxy adresi

### Görsel işleme yapılandırması

Taranmış PDF'ler JPEG görsellere dönüştürülür ve OCR için Go App tarafına verilir. Birden çok büyük PDF içe aktarılırken kaynak kullanımı çok yükselirse
önce `DOCREADER_PDF_RENDER_MAX_WORKERS` ya da `DOCREADER_MARKITDOWN_MAX_WORKERS` değerini düşürün.

## Yapılandırma örnekleri

### Temel yapılandırma (MinIO ile)

```yaml
docreader:
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://localhost:9000
    - MAX_FILE_SIZE_MB=50
```

### Gelişmiş yapılandırma (MinerU etkin)

```yaml
docreader:
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://192.168.1.100:9000
    - MINERU_ENDPOINT=http://mineru:8080
    - MAX_FILE_SIZE_MB=100
```

### Tencent Cloud COS kullanımı

```yaml
docreader:
  environment:
    - STORAGE_TYPE=cos
    - COS_SECRET_ID=your_secret_id
    - COS_SECRET_KEY=your_secret_key
    - COS_REGION=ap-guangzhou
    - COS_BUCKET_NAME=your-bucket
    - COS_APP_ID=your_app_id
    - MAX_FILE_SIZE_MB=50
```

### Alibaba Cloud OSS kullanımı

```yaml
docreader:
  environment:
    - STORAGE_TYPE=oss
    - OSS_ACCESS_KEY_ID=your_access_key_id
    - OSS_ACCESS_KEY_SECRET=your_access_key_secret
    - OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
    - OSS_BUCKET_NAME=your-bucket
    - OSS_REGION=cn-hangzhou
    - MAX_FILE_SIZE_MB=50
```

## Sık sorulan sorular

### 1. DocReader servisi başlamıyor mu?

Konteyner günlüklerinde eksik bağımlılık ya da izinle ilgili hata olup olmadığına bakın; gerekirse `MINIO_ENDPOINT` / depolama ile ilgili ortam değişkenlerinin doğru yapılandırıldığını doğrulayın.

### 2. Görseller görüntülenmiyor mu?

`MINIO_PUBLIC_ENDPOINT` yapılandırmasını kontrol edin:
- Tarayıcıdan erişilebilen bir adres kullanıldığından emin olun
- Başka bir cihazdan erişiliyorsa `localhost` yerine gerçek IP adresini kullanın

### 3. Dosya yükleme başarısız mı?

`MAX_FILE_SIZE_MB` yapılandırmasını kontrol edin ve sınırın yeterince büyük olduğundan emin olun. Ayrıca ön yüz ve arka uç servislerinin dosya boyutu sınırlarının aynı olması gerekir.

## Servis sağlık kontrolü

DocReader servisi için sağlık kontrolü yapılandırılmıştır:

```yaml
healthcheck:
  test: ["CMD", "grpc_health_probe", "-addr=localhost:50051"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 60s
```

Servis durumunu şu komutla kontrol edebilirsiniz:

```bash
docker ps | grep docreader
docker logs Rethra-docreader
```

## Daha fazla bilgi

- Servis bağlantı noktası: 50051 (gRPC)
- Konteyner adı: Rethra-docreader
- Ağ: Rethra-network
- Yeniden başlatma ilkesi: unless-stopped
