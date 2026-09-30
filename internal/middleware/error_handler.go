package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/acrbaran/rag/internal/errors"
)

// ErrorHandler, uygulama hatalarını işleyen bir middleware'dir
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// İsteği işle
		c.Next()

		// Hata olup olmadığını kontrol et
		if len(c.Errors) > 0 {
			// Son hatayı al
			err := c.Errors.Last().Err

			// Uygulama hatası olup olmadığını kontrol et
			if appErr, ok := errors.IsAppError(err); ok {
				// Uygulama hatasını döndür
				c.JSON(appErr.HTTPCode, gin.H{
					"success": false,
					"error": gin.H{
						"code":    appErr.Code,
						"message": appErr.Message,
						"details": appErr.Details,
					},
				})
				return
			}

			// Diğer türdeki hataları işle
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    errors.ErrInternalServer,
					"message": "Internal server error",
				},
			})
		}
	}
}
