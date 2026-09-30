// Package runtime, uygulama çalışma zamanı için bir bağımlılık enjeksiyonu kapsayıcısı sağlar
// Bu paket, bağımlılık enjeksiyonunu yönetmek için uber'ın dig kütüphanesini kullanır
package runtime

import (
	"go.uber.org/dig"
)

// container, uygulamanın genel bağımlılık enjeksiyonu kapsayıcısıdır
// Tüm hizmetler ve bileşenler bunun aracılığıyla kaydedilir ve çözülür
var container *dig.Container

// init bağımlılık enjeksiyonu kapsayıcısını başlatır
// Program başlatıldığında otomatik olarak çağrılır
func init() {
	container = dig.New()
}

// GetContainer genel bağımlılık enjeksiyonu kapsayıcısının başvurusunu döndürür
// Diğer paketlerin hizmet kaydetmesi veya alması için
func GetContainer() *dig.Container {
	return container
}
