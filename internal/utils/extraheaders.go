package utils

import (
	"net/http"
	"strings"
)

// reservedHeaderKeys, kullanıcı tarafından tanımlanan başlıkların üzerine yazmasına izin verilmeyen önemli istek başlıklarını listeler.
// Bu başlıklar, her provider için imza, yetkilendirme veya SSE akış denetimi tarafından kullanılır; üzerlerine yazılması çağrının doğrudan başarısız olmasına neden olabilir.
var reservedHeaderKeys = map[string]struct{}{
	"authorization":     {},
	"api-key":           {},
	"x-api-key":         {},
	"x-goog-api-key":    {},
	"content-type":      {},
	"content-length":    {},
	"accept-encoding":   {},
	"host":              {},
	"connection":        {},
	"transfer-encoding": {},
}

// IsReservedHeader, bir header key'in ayrılmış bir header olup olmadığını belirler; ayrılmış header'ların kullanıcı tanımlı başlıklar tarafından üzerine yazılmasına izin verilmez.
func IsReservedHeader(key string) bool {
	_, ok := reservedHeaderKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

// ApplyCustomHeaders, kullanıcının tanımladığı header'ları http.Request içine yazar.
// Kimlik doğrulama/imzayı bozmamak için ayrılmış header'lar (Authorization、api-key、Content-Type vb.) atlanır.
// Diğer header'lar aynı adlı girdilerin üzerine doğrudan yazılır; kullanıcıların varsayılan değerleri (örneğin Accept) değiştirmesine izin verilir.
func ApplyCustomHeaders(req *http.Request, headers map[string]string) {
	if req == nil || len(headers) == 0 {
		return
	}
	for k, v := range headers {
		name := strings.TrimSpace(k)
		if name == "" {
			continue
		}
		if IsReservedHeader(name) {
			continue
		}
		req.Header.Set(name, v)
	}
}
