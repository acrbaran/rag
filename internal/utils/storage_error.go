package utils

import "strings"

// SanitizeStorageConnectivityError converts a raw storage connectivity error
// into a safe, user-facing message. It deliberately avoids echoing the raw
// driver/network error so responses never leak internal hostnames, IPs, ports
// or TLS/certificate details. Callers that need the full error must log it
// server-side instead of returning it to the client.
func SanitizeStorageConnectivityError(err error, language string) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	var turkish, english string
	switch {
	case strings.Contains(msg, "Endpoint url cannot have fully qualified paths"):
		turkish, english = "Endpoint biçimi geçersiz: http:// veya https:// önekini kaldırıp yalnızca alan adı veya IP adresi ile portu girin (ör. minio.example.com:9000)", "Invalid endpoint format: remove the http:// or https:// prefix and enter only the domain or IP address and port (e.g. minio.example.com:9000)"
	case strings.Contains(msg, "no such host"):
		turkish, english = "DNS çözümlemesi başarısız; adresi kontrol edin", "DNS lookup failed; check the address"
	case strings.Contains(msg, "connection refused"):
		turkish, english = "Bağlantı reddedildi; hizmetin çalıştığını ve portu kontrol edin", "Connection refused; check that the service is running and the port is correct"
	case strings.Contains(msg, "no route to host"):
		turkish, english = "Hedef adrese ulaşılamıyor; ağ yapılandırmasını kontrol edin", "No route to the destination; check the network configuration"
	case strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "context deadline"):
		turkish, english = "Bağlantı zaman aşımına uğradı; ağı veya hizmet durumunu kontrol edin", "Connection timed out; check the network or service status"
	case strings.Contains(msg, "403") || strings.Contains(msg, "AccessDenied") || strings.Contains(msg, "access denied"):
		turkish, english = "Kimlik doğrulama başarısız; erişim bilgilerini kontrol edin", "Authentication failed; check the access credentials"
	case strings.Contains(msg, "certificate") || strings.Contains(msg, "tls") || strings.Contains(msg, "x509"):
		turkish, english = "TLS/SSL sertifika hatası; SSL yapılandırmasını kontrol edin", "TLS/SSL certificate error; check the SSL configuration"
	case strings.Contains(msg, "404") || strings.Contains(msg, "NoSuchBucket"):
		turkish, english = "Bucket bulunamadı; adını ve Region değerini kontrol edin", "Bucket not found; check its name and Region"
	default:
		turkish, english = "Bağlantı başarısız; yapılandırma parametrelerini kontrol edin", "Connection failed; check the configuration parameters"
	}
	if language == "en-US" {
		return english
	}
	return turkish
}
