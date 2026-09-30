package api

import (
	"bufio"
	"io"
	"strings"
)

// SSEEvent bir Server-Sent Events olayını belirtir
type SSEEvent struct {
	Data []byte
	Done bool
}

// SSEReader, SSE akışını okumak için kullanılır
type SSEReader struct {
	scanner *bufio.Scanner
}

// NewSSEReader bir SSE okuyucusu oluşturur
func NewSSEReader(reader io.Reader) *SSEReader {
	scanner := bufio.NewScanner(reader)
	// Uzun satırları işlemek için daha büyük bir arabellek ayarlanır (düşünce zinciri içeriği çok uzun olabilir)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	return &SSEReader{scanner: scanner}
}

// ReadEvent sonraki SSE olayını okur
func (r *SSEReader) ReadEvent() (*SSEEvent, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()

		// Boş satır, atla
		if line == "" {
			continue
		}

		// data satırını ayrıştırır. SSE standardı yalnızca "data:" önekini gerektirir; iki noktanın ardından gelen tek boşluk isteğe bağlıdır,
		// bu nedenle her iki yazım da kabul edilmelidir.
		if !strings.HasPrefix(line, "data:") {
			// Diğer satırlar (ör. event:, id: vb.) atlanır
			continue
		}
		payload := strings.TrimPrefix(line[len("data:"):], " ")

		// Bitiş işareti olup olmadığını kontrol eder. Bazı ağ geçitleri bunu "data:[DONE]" olarak yazar veya sonda boşluk bırakır,
		// Burada karar, baştaki ve sondaki boşluklar kaldırıldıktan sonraki içeriğe göre tutarlı biçimde verilir; böylece bekçi değer JSON olarak ayrıştırılmaz.
		if strings.TrimSpace(payload) == "[DONE]" {
			return &SSEEvent{Done: true}, nil
		}

		return &SSEEvent{Data: []byte(payload)}, nil

		// Diğer satırlar (ör. event:, id: vb.) atlanır
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}
