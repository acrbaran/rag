# QA veri kümesi örnekleme aracı

OpenAI GPT modelleriyle yanıt üreten kapsamlı bir QA veri kümesi örnekleme aracı. Büyük ölçekli veri kümelerinden (MS MARCO gibi) yüksek kaliteli soru-cevap veri kümeleri oluşturmanıza yardımcı olur.

## Özellikler

- **Akıllı örnekleme**: Büyük veri kümelerinden sorguları, belgeleri ve ilgililik yargılarını akıllıca örnekler
- **Yanıt üretimi**: OpenAI GPT modelleriyle otomatik olarak yüksek kaliteli yanıtlar üretir
- **Kaldığı yerden devam**: Kesintiden sonra üretime son konumdan devam eder
- **İlerleme takibi**: Gerçek zamanlı ilerleme güncellemeleri ve istatistikler
- **Sonuç görselleştirme**: Soru-cevap çiftlerini tam bağlamıyla okunması kolay biçimde gösterir

## Kurulum kılavuzu

### Sistem gereksinimleri

- Python 3.7+
- OpenAI API anahtarı

### Bağımlılıkları kurma

```bash
pip install pandas pyarrow openai
```

### Ortam değişkenlerini ayarlama

```bash
export OPENAI_API_KEY="openai-api-anahtarınız"
# İsteğe bağlı: özel OpenAI uç noktası kullanma
export OPENAI_BASE_URL="https://api.openai.com/v1"
```

### Veri kümesini hazırlama

Biçim gereksinimlerini karşılayan herhangi bir QA veri kümesini kullanabilir ya da önceden işlenmiş örnekleri indirebilirsiniz:

**HuggingFace/ModelScope örneklerini kullanma**
Popüler QA veri kümelerinden önceden işlenmiş örnekler sunuyoruz:
- MarkrAI/eli5_sample_autorag
- MarkrAI/msmarco_sample_autorag
- MarkrAI/triviaqa_sample_autorag
- gnekt/hotpotqa_small_sample_autorag

**Kendi veri kümenizi kullanma**
Veri kümenizin şu dosyaları içerdiğinden emin olun:
- `queries.parquet` (sütunlar: id, text)
- `corpus.parquet` (sütunlar: id, text)
- `qrels.parquet` (sütunlar: qid, pid)

## Hızlı başlangıç

### 1. Büyük veri kümesinden örnekleme

Önce tam veri kümesinden sorguların, belgelerin ve ilgililik yargılarının bir alt kümesini örnekleyin:

```bash
python dataset/qa_dataset.py sample \
  --queries ~/dataset/mmarco-queries.parquet \
  --corpus ~/dataset/mmarco-corpus.parquet \
  --qrels ~/dataset/mmarco-qrels.parquet \
  --nq 100 \
  --output_dir ./dataset/samples
```

### 2. Yanıt üretme

Örneklenen sorular için OpenAI GPT modelleriyle yanıt üretin:

```bash
python dataset/qa_dataset.py generate \
  --input_dir ./dataset/samples \
  --output_dir ./dataset/samples
```

### 3. Sonuçları görüntüleme

Üretilen soru-cevap çiftlerini bağlamlarıyla birlikte gösterin:

```bash
python dataset/qa_dataset.py show \
  --input_dir ./dataset/samples \
  -n 5
```

## Ayrıntılı kullanım

### Örnekleme komutu

Tam veri kümesinden temsili bir örnek oluşturur.

```bash
python dataset/qa_dataset.py sample [seçenekler]
```

**Zorunlu parametreler:**
- `--queries`: Sorgu parquet dosyasının yolu (sütunlar: `id`, `text`)
- `--corpus`: Derlem parquet dosyasının yolu (sütunlar: `id`, `text`)
- `--qrels`: İlgililik yargıları parquet dosyasının yolu (sütunlar: `qid`, `pid`)

**İsteğe bağlı parametreler:**
- `--nq`: Örneklenecek sorgu sayısı (varsayılan: 1000)
- `--output_dir`: Örneklenen verilerin çıktı dizini (varsayılan: ./save)

**Örnek:**
```bash
python dataset/qa_dataset.py sample \
  --queries data/queries.parquet \
  --corpus data/corpus.parquet \
  --qrels data/qrels.parquet \
  --nq 500 \
  --output_dir ./my_sample
```

### Üretim komutu

Örneklenen sorular için OpenAI API ile yanıt üretir.

```bash
python dataset/qa_dataset.py generate [seçenekler]
```

**Zorunlu parametreler:**
- `--input_dir`: Örneklenen verileri içeren dizin (queries.parquet, corpus.parquet, qrels.parquet)

**İsteğe bağlı parametreler:**
- `--output_dir`: Üretilen yanıtların çıktı dizini (varsayılan: ./save)

**Özellikler:**
- **Kaldığı yerden devam**: Kesintiden sonra otomatik olarak son konumdan devam eder
- **Hata işleme**: API çağrısı başarısız olursa otomatik olarak 3 kez yeniden dener
- **İlerleme kaydı**: Her başarılı yanıt üretiminden sonra ilerlemeyi kaydeder

**Örnek:**
```bash
python dataset/qa_dataset.py generate \
  --input_dir ./my_sample \
  --output_dir ./my_sample
```

### Gösterim komutu

Üretilen soru-cevap çiftlerini tam bağlamlarıyla gösterir.

```bash
python dataset/qa_dataset.py show [seçenekler]
```

**Zorunlu parametreler:**
- `--input_dir`: QA verilerini içeren dizin (queries.parquet, corpus.parquet, qrels.parquet, qas.parquet, answers.parquet)

**İsteğe bağlı parametreler:**
- `-n`: Gösterilecek sonuç sayısı (varsayılan: 5)

**Örnek:**
```bash
python dataset/qa_dataset.py show \
  --input_dir ./my_sample \
  -n 3
```

## Girdi veri biçimi

### Sorgu dosyası (queries.parquet)
| Sütun adı | Tür | Açıklama |
|------|------|------|
| id | string | Benzersiz sorgu tanımlayıcısı |
| text | string | Asıl soru metni |

### Derlem dosyası (corpus.parquet)
| Sütun adı | Tür | Açıklama |
|------|------|------|
| id | string | Benzersiz paragraf/belge tanımlayıcısı |
| text | string | Paragraf/belge içeriği |

### İlgililik yargıları dosyası (qrels.parquet)
| Sütun adı | Tür | Açıklama |
|------|------|------|
| qid | string | Sorgu ID'si (queries.id ile eşleşir) |
| pid | string | Paragraf ID'si (corpus.id ile eşleşir) |

## Çıktı dosyaları

Tüm komutlar çalıştırıldıktan sonra çıktı dizini şunları içerir:

### Örneklenen veriler
- `queries.parquet`: Örneklenen sorgu alt kümesi
- `corpus.parquet`: Örneklenen belge alt kümesi
- `qrels.parquet`: Örneklenen ilgililik yargıları

### Üretilen yanıtlar
- `answers.parquet`: Üretilen yanıtlar (benzersiz ID'lerle)
- `qas.parquet`: Soru-cevap eşlemesi (qid → aid)

## İleri kullanım

### Özel OpenAI yapılandırması

Farklı OpenAI modelleri veya uç noktaları kullanabilirsiniz:

```bash
# GPT-4 Turbo kullanma
export OPENAI_API_KEY="anahtarınız"
python dataset/qa_dataset.py generate --input_dir ./samples

# Azure OpenAI kullanma
export OPENAI_API_KEY="azure-anahtarı"
export OPENAI_BASE_URL="https://kaynaginiz.openai.azure.com/openai/deployments/gpt-4"
python dataset/qa_dataset.py generate --input_dir ./samples
```

### Büyük veri kümelerinden örnekleme

Çok büyük veri kümeleri için toplu örnekleme önerilir:

```bash
# Birinci toplu iş
python dataset/qa_dataset.py sample --nq 1000 --output_dir ./batch1
python dataset/qa_dataset.py generate --input_dir ./batch1

# İkinci toplu iş
python dataset/qa_dataset.py sample --nq 1000 --output_dir ./batch2
python dataset/qa_dataset.py generate --input_dir ./batch2
```

## Sorun giderme

### Sık karşılaşılan sorunlar

**1. OpenAI API hataları**
- API anahtarının doğru ayarlandığından emin olun: `echo $OPENAI_API_KEY`
- API kotasını ve faturalandırma durumunu kontrol edin
- OpenAI ile ağ bağlantısını doğrulayın

**2. Büyük veri kümelerinde bellek sorunları**
- Daha küçük örnek için `--nq` parametresini azaltın
- pandas işlemleri için yeterli RAM olduğundan emin olun
- Daha küçük parquet dosyaları kullanmayı düşünün

**3. Dosya bulunamadı hataları**
- Tüm girdi dosyası yollarının doğru olduğunu doğrulayın
- parquet dosyalarının sütun adlarının doğru olduğundan emin olun
- Dosya izinlerini kontrol edin

### Hata ayıklama modu

print ifadeleri ekleyerek veya Python hata ayıklayıcısını kullanarak ayrıntılı çıktıyı etkinleştirin:

```bash
python -m pdb dataset/qa_dataset.py sample --queries ...
```

## Örnek iş akışı

```bash
# 1. Ortamı ayarlama
export OPENAI_API_KEY="sk-..."

# 2. MS MARCO'dan 200 sorgu örnekleme
python dataset/qa_dataset.py sample \
  --queries ~/mmarco/queries.parquet \
  --corpus ~/mmarco/corpus.parquet \
  --qrels ~/mmarco/qrels.parquet \
  --nq 200 \
  --output_dir ./marco_sample

# 3. Yanıt üretme (API hız sınırına göre biraz zaman alabilir)
python dataset/qa_dataset.py generate \
  --input_dir ./marco_sample \
  --output_dir ./marco_sample

# 4. Sonuçları görüntüleme
python dataset/qa_dataset.py show \
  --input_dir ./marco_sample \
  -n 10
```

## Katkıda bulunma

Sorun bildirimleri ve özellik önerileri memnuniyetle karşılanır!

## Lisans

MIT lisansı - araştırma ve projelerde serbestçe kullanılabilir.