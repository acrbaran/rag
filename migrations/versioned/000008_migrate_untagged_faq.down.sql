-- Rollback: This migration cannot be fully rolled back as we don't know which entries
-- were originally untagged. We can only clear the tag_id for entries that reference
-- bir "Sınıflandırılmamış" etiketi, ancak bu kasıtlı olarak etiketlenmiş girdileri etkileyebilir.

-- WARNING: This rollback is destructive and should only be used if absolutely necessary.
-- "Sınıflandırılmamış" etiketlerine başvuran tüm parçalar, bilgiler ve gömüler için tag_id değerini boş dizeye ayarlayacaktır.

DO $$
DECLARE
    kb_record RECORD;
    untagged_tag_id VARCHAR(36);
    updated_chunks INT;
    updated_knowledges INT;
BEGIN
    RAISE NOTICE '[Migration 000008 Rollback] WARNING: This rollback will clear tag_id for all entries referencing "未分类" tags';
    
    -- Tüm "Sınıflandırılmamış" etiketlerini bul
    FOR kb_record IN 
        SELECT id, tenant_id, knowledge_base_id 
        FROM knowledge_tags
        WHERE name = '未分类'
    LOOP
        untagged_tag_id := kb_record.id;
        
        -- Clear tag_id for chunks referencing this tag (both faq and document types)
        UPDATE chunks 
        SET tag_id = '', updated_at = NOW()
        WHERE tag_id = untagged_tag_id
        AND chunk_type IN ('faq', 'document');
        
        GET DIAGNOSTICS updated_chunks = ROW_COUNT;
        RAISE NOTICE '[Migration 000008 Rollback] Cleared tag_id for % chunks referencing tag %', 
            updated_chunks, untagged_tag_id;

        -- Clear tag_id for knowledges referencing this tag
        UPDATE knowledges 
        SET tag_id = '', updated_at = NOW()
        WHERE tag_id = untagged_tag_id;
        
        GET DIAGNOSTICS updated_knowledges = ROW_COUNT;
        RAISE NOTICE '[Migration 000008 Rollback] Cleared tag_id for % knowledges referencing tag %', 
            updated_knowledges, untagged_tag_id;

        -- Clear tag_id in embeddings if column exists
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'embeddings' AND column_name = 'tag_id'
        ) THEN
            UPDATE embeddings 
            SET tag_id = ''
            WHERE tag_id = untagged_tag_id;
        END IF;

        -- "Sınıflandırılmamış" etiketini sil
        DELETE FROM knowledge_tags WHERE id = untagged_tag_id;
        RAISE NOTICE '[Migration 000008 Rollback] Deleted "未分类" tag: %', untagged_tag_id;
    END LOOP;

    RAISE NOTICE '[Migration 000008 Rollback] Completed rollback';
END $$;
