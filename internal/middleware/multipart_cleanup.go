package middleware

import "github.com/gin-gonic/gin"

// MultipartFormCleanup istek tamamlandıktan sonra multipart ayrıştırmasının sistem geçici dizinine yazdığı dosyaları siler.
// Girdi: Gin istek bağlamı c; çıktı: yok. Multipart formu ayrıştırılmamış veya yalnızca bellek kullanan isteklerde silme işlemi gerçekleştirilmez.
// Bu middleware, başarılı, başarısız ve panic kurtarması sonrası yükleme isteklerini kapsamak için tüm rotalardan önce kaydedilmelidir.
func MultipartFormCleanup() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			// Go, ParseMultipartForm diske yazdıktan sonra MultipartForm'u ayarlar.
			// RemoveAll yalnızca otomatik oluşturduğu multipart geçici dosyalarını siler; kalıcılaştırılmış iş dosyalarını etkilemez.
			if c.Request != nil && c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()
		c.Next()
	}
}
