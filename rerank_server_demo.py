import gc
import torch
import uvicorn
from fastapi import FastAPI
from pydantic import BaseModel, Field
from transformers import AutoModelForSequenceClassification, AutoTokenizer
from typing import List

# CUDA hata ayıklamayı etkinleştir
# import os
# os.environ['CUDA_LAUNCH_BLOCKING']='1'

# --- 1. API için istek ve yanıt veri yapılarını tanımla ---

# İstek gövdesi yapısı değişmeden kalır
class RerankRequest(BaseModel):
    query: str
    documents: List[str]

# --- Değişiklik başlangıcı: test için yanıt yapısını tanımla, alan adı "score" olsun ---

# DocumentInfo yapısı değişmeden kalır
class DocumentInfo(BaseModel):
    text: str

# Önceki GoRankResult öğesini TestRankResult olarak değiştir
# Temel değişiklik: "relevance_score" alanını "score" olarak yeniden adlandır
class TestRankResult(BaseModel):
    index: int
    document: DocumentInfo
    score: float  # <--- 【Temel değişiklik noktası】Alan adı relevance_score yerine score olarak değiştirildi

# Nihai yanıt gövdesi yapısı; "results" listesi TestRankResult içerir
class TestFinalResponse(BaseModel):
    results: List[TestRankResult]

# --- Değişiklik sonu ---


# --- 2. Modeli yükle (hizmet başlatılırken bir kez çalıştırılır) ---
print("Model yükleniyor, lütfen bekleyin...")
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
print(f"Kullanılan cihaz: {device}")
try:
    # Buradaki yolun doğru olduğundan emin olun
    model_path = '/data1/home/lwx/work/Download/rerank_model_weight'
    tokenizer = AutoTokenizer.from_pretrained(model_path)
    model = AutoModelForSequenceClassification.from_pretrained(model_path)
    model.to(device)
    model.eval()
    print("Model başarıyla yüklendi!")
except Exception as e:
    print(f"Model yüklenemedi: {e}")
    # Test ortamında model yüklenemezse, geçersiz bir hizmetin çalışmasını önlemek için çıkmayı düşünebilirsiniz
    exit()

# --- 3. FastAPI uygulamasını oluştur ---
app = FastAPI(
    title="Reranker API (Test Version)",
    description="Go istemcisi uyumluluğunu test etmek için 'score' alanı döndüren bir API hizmeti",
    version="1.0.2"
)

# --- 4. API uç noktalarını tanımla ---
# --- Değişiklik başlangıcı: response_model öğesini yeni test yanıt yapısına yönlendir ---
@app.post("/rerank", response_model=TestFinalResponse) # <--- 【Temel değişiklik noktası】response_model, TestFinalResponse olarak değiştirildi
def rerank_endpoint(request: RerankRequest):
    # --- Değişiklik sonu ---

    pairs = [[request.query, doc] for doc in request.documents]

    with torch.no_grad():
        inputs = outputs = logits = None

        try:
            inputs = tokenizer(pairs, padding=True, truncation=True, return_tensors='pt', max_length=1024).to(device)
            outputs = model(**inputs, return_dict=True)
            logits = outputs.logits.view(-1, ).float()
            scores = torch.sigmoid(logits)
        finally:
            # GPU kaynak kullanımını serbest bırak
            del inputs, outputs, logits
            gc.collect()

            if torch.cuda.is_available():
                torch.cuda.empty_cache()
            elif hasattr(torch, "mps") and torch.mps.is_available():
                torch.mps.empty_cache()


    # --- Değişiklik başlangıcı: sonuçları test yapısına göre oluştur ---
    results = []
    for i, (text, score_val) in enumerate(zip(request.documents, scores)):
        
        # 1. İç içe geçmiş document nesnesini oluştur
        doc_info = DocumentInfo(text=text)
        
        # 2. TestRankResult nesnesini oluştur
        #    Alan adlarına dikkat edin: index, document, score
        test_result = TestRankResult(
            index=i,
            document=doc_info,
            score=score_val.item()  # <--- 【Temel değişiklik noktası】"score" alanına ata
        )
        results.append(test_result)

    # 3. Sırala (key de buna uygun olarak score şeklinde değiştirilmelidir)
    sorted_results = sorted(results, key=lambda x: x.score, reverse=True)
    # --- Değişiklik sonu ---
    
    # Bir sözlük döndürür; FastAPI bunu response_model (TestFinalResponse) temelinde doğrular ve serileştirir
    # Nihai oluşturulan JSON şu şekilde olacaktır: {"results": [{"index": ..., "document": ..., "score": ...}]}
    return {"results": sorted_results}

@app.get("/")
def read_root():
    return {"status": "Reranker API (Test Version) is running"}

# --- 5. Hizmeti başlat ---
if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
