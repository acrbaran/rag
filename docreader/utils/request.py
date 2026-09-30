import contextlib
import logging
import time
import uuid
from contextvars import ContextVar
from logging import LogRecord
from typing import Optional

# Günlüğü yapılandır
logger = logging.getLogger(__name__)

# Bağlam değişkenlerini tanımla
request_id_var = ContextVar("request_id", default=None)
_request_start_time_ctx = ContextVar("request_start_time", default=None)


def set_request_id(request_id: str) -> None:
    """Geçerli bağlamın istek kimliğini ayarla"""
    request_id_var.set(request_id)


def get_request_id() -> Optional[str]:
    """Geçerli bağlamın istek kimliğini al"""
    return request_id_var.get()


class MillisecondFormatter(logging.Formatter):
    """Özel günlük biçimlendirici; mikrosaniye (6 basamak) yerine yalnızca milisaniye düzeyindeki zaman damgasını (3 basamak) gösterir"""

    def formatTime(self, record, datefmt=None):
        """Mikrosaniyeleri milisaniye olarak biçimlendirmek için formatTime metodunu yeniden yaz"""
        # Önce standart biçimlendirilmiş zamanı al
        result = super().formatTime(record, datefmt)

        # Eğer `.%f` içeren bir biçim kullanılıyorsa, mikrosaniyeleri (6 basamak) milisaniyelere (3 basamak) kes
        if datefmt and ".%f" in datefmt:
            # Biçimlendirilmiş zaman dizgesinin sonunda 6 basamaklı mikrosaniye değeri olmalıdır
            parts = result.split(".")
            if len(parts) > 1 and len(parts[1]) >= 6:
                # Milisaniye olarak yalnızca ilk 3 basamağı koru
                millis = parts[1][:3]
                result = f"{parts[0]}.{millis}"

        return result


def init_logging_request_id():
    """
    Initialize logging to include request ID in log messages.
    Add the custom filter to all existing handlers
    """
    logger.info("Initializing request ID logging")
    root_logger = logging.getLogger()

    # Tüm işleyicilere özel filtre ekle
    for handler in root_logger.handlers:
        # İstek ID filtresi ekle
        handler.addFilter(RequestIdFilter())

        # İstek ID'sini içerecek şekilde biçimlendiriciyi güncelle, biçimi daha kompakt ve düzenli hâle getir
        formatter = logging.Formatter(
            fmt="%(asctime)s.%(msecs)03d [%(request_id)s] %(levelname)-5s %(name)-20s | %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
        handler.setFormatter(formatter)

    logger.info(
        f"Updated {len(root_logger.handlers)} handlers with request ID formatting"
    )

    # İşleyici yoksa standart çıktı işleyicisi ekle
    if not root_logger.handlers:
        handler = logging.StreamHandler()
        formatter = logging.Formatter(
            fmt="%(asctime)s.%(msecs)03d [%(request_id)s] %(levelname)-5s %(name)-20s | %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
        handler.setFormatter(formatter)
        handler.addFilter(RequestIdFilter())
        root_logger.addHandler(handler)
        logger.info("Added new StreamHandler with request ID formatting")


class RequestIdFilter(logging.Filter):
    """Filter that adds request ID to log messages"""

    def filter(self, record: LogRecord) -> bool:
        request_id = request_id_var.get()
        if request_id is not None:
            # Günlük kaydına kısa biçimde istek ID'si özelliği ekle
            if len(request_id) > 8:
                # Düzenli görünmesini sağlamak için ID'nin ilk 8 karakterini al
                short_id = request_id[:8]
                if "-" in request_id:
                    # Örneğin `test-req-1-XXX` gibi biçimi korumaya çalış
                    parts = request_id.split("-")
                    if len(parts) >= 3:
                        # Biçim `xxx-xxx-n-randompart` ise
                        short_id = f"{parts[0]}-{parts[1]}-{parts[2]}"
                record.request_id = short_id
            else:
                record.request_id = request_id

            # Yürütme süresi özelliği ekle
            start_time = _request_start_time_ctx.get()
            if start_time is not None:
                elapsed_ms = int((time.time() - start_time) * 1000)
                record.elapsed_ms = elapsed_ms
                # Yürütme süresini mesaja ekle
                if not hasattr(record, "message_with_elapsed"):
                    record.message_with_elapsed = True
                    record.msg = f"{record.msg} (elapsed: {elapsed_ms}ms)"
        else:
            # İstek ID'si yoksa yer tutucu kullan
            record.request_id = "no-req-id"

        return True


@contextlib.contextmanager
def request_id_context(request_id: str = None):
    """Context manager that sets a request ID for the current context

    Args:
        request_id: Kullanılacak istek kimliği; None ise otomatik oluşturulur

    Example:
        with request_id_context("req-123"):
            # Bu kod bloğundaki tüm günlükler `req-123` istek ID'sini içerecek
            logging.info("Processing request")
    """
    # Generate or use provided request ID
    req_id = request_id or str(uuid.uuid4())

    # Set start time and request ID
    start_time = time.time()
    req_token = request_id_var.set(req_id)
    time_token = _request_start_time_ctx.set(start_time)

    logger.info(f"Starting new request with ID: {req_id}")

    try:
        yield request_id_var.get()
    finally:
        # Log completion and reset context vars
        elapsed_ms = int((time.time() - start_time) * 1000)
        logger.info(f"Request {req_id} completed in {elapsed_ms}ms")
        request_id_var.reset(req_token)
        _request_start_time_ctx.reset(time_token)
