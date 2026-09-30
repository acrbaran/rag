// Çalışma zamanı yapılandırması (yerel geliştirme varsayılanları, Docker ortamında entrypoint betiği tarafından geçersiz kılınır)
window.__RUNTIME_CONFIG__ = {
  MAX_FILE_SIZE_MB: 50,
  MAX_SKILL_BUNDLE_SIZE_MB: 256,
  // Optional: serve embed on a dedicated origin, e.g. 'https://embed.example.com'
  EMBED_BASE_URL: '',
  // Optional: default UI locale for first-time visitors (en-US | tr-TR)
  DEFAULT_LOCALE: '',
};
