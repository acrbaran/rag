-- Bilgi tabanları için sabitleme (üste tutturma) desteği ekle
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS is_pinned BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS pinned_at TIMESTAMP WITH TIME ZONE NULL;
