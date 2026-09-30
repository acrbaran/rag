package searchutil

import (
	"sort"
	"strings"

	"github.com/acrbaran/rag/internal/types"
)

// JoinChunkContent joins two current chunk bodies without relying on parser
// offsets. Exact containment is collapsed, a real suffix/prefix overlap is
// removed, and otherwise both bodies are retained with separator between
// them. The conservative fallback intentionally prefers small duplication
// over silently dropping edited content.
func JoinChunkContent(acc, next, separator string) string {
	if acc == "" {
		return next
	}
	if next == "" {
		return acc
	}
	if ContainsChunkContent(acc, next) {
		return acc
	}
	if ContainsChunkContent(next, acc) {
		return next
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)
	maxOverlap := minInt(len(accRunes), len(nextRunes))
	// Editable chunks may be much larger than parser-produced chunks. Bound
	// suffix matching so an adversarial 200 KB edit cannot turn retrieval into
	// quadratic work. Parser overlap windows are normally far below this cap;
	// larger unmatched overlap is safely retained as duplication.
	if maxOverlap > defaultSearchSpan {
		maxOverlap = defaultSearchSpan
	}
	for overlap := maxOverlap; overlap >= minOverlapRunes; overlap-- {
		if runeSlicesEqual(accRunes[len(accRunes)-overlap:], nextRunes[:overlap]) {
			return acc + string(nextRunes[overlap:])
		}
	}
	return acc + separator + next
}

// ContainsChunkContent reports whether the complete current body is safely
// represented by another body. Very short substrings are not treated as
// containment because common words and punctuation would create false drops.
func ContainsChunkContent(container, contained string) bool {
	if container == "" || contained == "" {
		return false
	}
	if container == contained {
		return true
	}
	return len([]rune(contained)) >= minOverlapRunes && strings.Contains(container, contained)
}

func runeSlicesEqual(left, right []rune) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// Burada chunk içeriklerinin «örtüşmeli birleştirme» ortak mantığı uygulanır; belge yeniden oluşturma (reconstructContent),
// bilgi grafiği içerik birleştirme (graph mergeChunkContents) gibi yollar tarafından yeniden kullanılır. Sohbet arama zinciri
// kullanıcının Chunk düzenlemesine izin verir; orijinal konum koordinatlarına bağımlı kalmamak için yukarıdaki JoinChunkContent kullanılır.
//
// Geçmişte her yerde örtüşme «konuma göre» formülle kırpılıyordu (offset = len(content) - (EndAt -
// lastEndAt) gibi); bu, len([]rune(Content)) == EndAt-StartAt olduğunu varsayar. Ancak iki tür
// veri bu değişmezi bozar ve birleştirmede kayma, karakter kaybı veya tekrar oluşmasına neden olur:
//  1. Ebeveyn-çocuk parçalayıcı bölünen tablolara «başlık satırı ekler»; eklenen başlık sıfır genişliklidir
//     (start == end), konum koordinatları bunu ifade edemez ve content EndAt-StartAt değerinden uzundur;
//  2. content içinde HTML varlıkları (ör. &#34; / &gt;) kalabilir; karakter sayısı orijinal aralıktan uzundur.
//
// Bu yüzden burada örtüşme «metne göre» eşleştirilir: sonraki parçanın başındaki pencerede birleşmiş metnin sonekinin ilk
// geçtiği konum aranır ve o konumdan sonrası eklenir. Konum bilgisi (StartAt/EndAt) yalnızca arama penceresini tahmin etmek için
// boyutunu tahmin etmek için kullanılır, artık kırpma için kullanılmaz.

const (
	// minOverlapRunes, eşleşmeye katılan en kısa sonek uzunluğudur. Çok kısa olursa (ör. tablo ayırıcı satırı |---|)
	// yanlış eşleşmeye yatkındır; bu nedenle yok sayılır.
	minOverlapRunes = 12
	// defaultSearchSpan arama penceresinin alt sınırıdır; konum bilgisi eksik/0 olsa bile
	// belirli bir aralıktaki gerçek örtüşmenin algılanmasını sağlar.
	defaultSearchSpan = 400
)

// AppendWithOverlap, next'i acc'nin sonuna ekler ve ikisi arasındaki örtüşmeyi kaldırır.
//
// positionOverlap, StartAt/EndAt'tan tahmin edilen örtüşme miktarıdır (lastEnd - curStart); yalnızca
// arama penceresi boyutunu belirlemek için kullanılır; gerçek örtüşme metin eşleştirmesiyle bulunur ve eklenen tablo başlıkları ile HTML varlık uzunluğu farklarıyla uyumludur.
// Metinsel örtüşme bulunamazsa olduğu gibi birleştirilir (kırpılmaz); içeriği bozmaktansa korumak tercih edilir.
//
// positionOverlap <= 0 olduğunda iki parça konum olarak tam bitişik veya ayrıktır, kaldırılacak örtüşme yoktur;
// bu durumda metin eşleştirmeye girilirse headSlack alt sınırı 320 nedeniyle next başındaki pencerede yanlışlıkla
// acc sonekindeki gerçek içerik tekrarı (ör. aynı cümlenin belgede birden çok kez geçmesi) yakalanır ve next'in başı
// tamamının eklenmiş tablo başlığı sanılıp silinmesine ve geri döndürülemez içerik kaybına yol açar. Doğrudan birleştirilir; eklenmiş tablo başlığı
// tekrarı, çağıran tarafın sonraki işlemesine bırakılır.
func AppendWithOverlap(acc, next string, positionOverlap int) string {
	if acc == "" {
		return next
	}
	if next == "" {
		return acc
	}
	if positionOverlap <= 0 {
		return acc + next
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)

	span := positionOverlap

	maxK := minInt(len(accRunes), len(nextRunes))
	if cap := maxInt(span*3, defaultSearchSpan); maxK > cap {
		maxK = cap
	}
	// Örtüşen içerikten önce en fazla kaç ön ekin atlanmasına izin verilir (yani sonradan yazılan başlıklar vb. birleştirilmiş metinler).
	headSlack := maxInt(span*2, 320)

	for k := maxK; k >= minOverlapRunes; k-- {
		needle := accRunes[len(accRunes)-k:]
		if pos := indexRunes(nextRunes, needle, headSlack); pos >= 0 {
			return acc + string(nextRunes[pos+k:])
		}
	}
	return acc + next
}

// AppendWithExactOverlap, çağıran konum koordinatlarının güvenilir olduğunu doğruladığında, koordinatların verdiği kesin örtüşme miktarını
// acc ile next'i birleştirir: acc'nin son overlap karakteri ile next'in ilk overlap karakterinin karakter karakter
// eşit olduğunu doğrular; eşitse tam olarak kırpar, overlap 0 ise doğrudan birleştirir.
//
// AppendWithOverlap'tan farkı «tahmin etmemesidir»: o, eklenen başlık satırları, HTML varlıkları gibi uzunluk
// sapmalarına uyum sağlamak için pencerede en uzun sonek eşleşmesini arar; yinelenen periyodik metinler (tablolar, günlükler) yanlışlıkla
// örtüşme sayılıp gerçek içerik kesilebilir. Koordinatlar güvenilir olduğunda örtüşme miktarı bilinir; arama gerekmez.
//
// Doğrulama başarısız olursa ok=false döner; AppendWithOverlap'e geri dönülüp dönülmeyeceğine çağıran karar verir.
func AppendWithExactOverlap(acc, next string, overlap int) (string, bool) {
	if acc == "" {
		return next, true
	}
	if next == "" {
		return acc, true
	}
	if overlap < 0 {
		return "", false
	}
	if overlap == 0 {
		return acc + next, true
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)
	if overlap > len(accRunes) || overlap > len(nextRunes) {
		return "", false
	}
	if !runeSlicesEqual(accRunes[len(accRunes)-overlap:], nextRunes[:overlap]) {
		return "", false
	}
	return acc + string(nextRunes[overlap:]), true
}

// MergeTextChunks, StartAt'e göre (eşitlikte ChunkIndex'e göre) sıraladıktan sonra, AppendWithOverlap kullanarak
// birden çok chunk içeriğini tam metin olarak yeniden oluşturur. gapSep, konumları bitişik olmayan (aralarında boşluk bulunan) iki parça arasındaki
// ayırıcıdır (örneğin "\n"); boş bir dize verilirse doğrudan birleştirilir.
//
// Çağıran önce tür filtrelemesini yapmaktan sorumludur (örneğin yalnızca metin chunk'larını tutmak); bu işlev ChunkType'ı dikkate almaz.
func MergeTextChunks(chunks []*types.Chunk, gapSep string) string {
	if len(chunks) == 0 {
		return ""
	}

	sorted := make([]*types.Chunk, len(chunks))
	copy(sorted, chunks)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].StartAt == sorted[j].StartAt {
			return sorted[i].ChunkIndex < sorted[j].ChunkIndex
		}
		return sorted[i].StartAt < sorted[j].StartAt
	})

	merged := ""
	mergedEnd := -1
	for _, c := range sorted {
		if c == nil || c.Content == "" {
			continue
		}
		if merged == "" {
			merged = c.Content
			if c.EndAt > 0 {
				mergedEnd = c.EndAt
			}
			continue
		}

		// Boşluk / konum bilgisi eksikliği (EndAt==0): bağımsız paragraflar olarak birleştirilir, örtüşme kırpması yapılmaz.
		if c.StartAt > mergedEnd || c.EndAt == 0 {
			if gapSep != "" {
				merged += gapSep
			}
			merged += c.Content
			if c.EndAt > 0 {
				mergedEnd = c.EndAt
			}
			continue
		}

		// Kısmi örtüşme veya uç uca eklenme: metin eşleşmesine göre örtüşme kaldırıldıktan sonra birleştirilir.
		if c.EndAt > mergedEnd {
			merged = AppendWithOverlap(merged, c.Content, mergedEnd-c.StartAt)
			mergedEnd = c.EndAt
		}
		// Aksi halde önceki parça tarafından tamamen kapsanmıştır, atlanır.
	}

	return merged
}

// indexRunes, haystack içinde needle'ın ilk geçtiği rune dizinini arar ve başlangıç konumu
// maxStart'ı aşmaz. Bulunamazsa -1 döner.
func indexRunes(haystack, needle []rune, maxStart int) int {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return -1
	}
	limit := len(haystack) - len(needle)
	if maxStart < limit {
		limit = maxStart
	}
	for i := 0; i <= limit; i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
