package service

import (
	"context"

	apperrors "github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/types"
)

func (s *knowledgeTagService) requireTagWrite(
	ctx context.Context,
	tag *types.KnowledgeTag,
) (*types.KnowledgeBase, context.Context, error) {
	if tag == nil {
		return nil, ctx, apperrors.NewNotFoundError(types.LocalizedText(ctx, "Etiket bulunamadı", "Tag not found"))
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, tag.KnowledgeBaseID)
	if err != nil {
		return nil, ctx, err
	}
	if kb == nil || kb.ID != tag.KnowledgeBaseID || kb.TenantID != tag.TenantID {
		return nil, ctx, apperrors.NewForbiddenError(types.LocalizedText(ctx, "Etiket geçerli bilgi tabanına ait değil", "Tag does not belong to the current knowledge base"))
	}
	ctx, err = requireKBWrite(ctx, kb)
	return kb, ctx, err
}

func (s *knowledgeTagService) validateTagDeleteExclusions(
	ctx context.Context,
	kb *types.KnowledgeBase,
	ids []string,
) error {
	if len(ids) == 0 {
		return nil
	}
	if kb.Type != types.KnowledgeBaseTypeFAQ {
		return apperrors.NewBadRequestError(types.LocalizedText(ctx, "Yalnızca SSS kaydı silme işleminde hariç tutulacak kayıtlar belirtilebilir", "Excluded entries are supported only when deleting FAQ entries"))
	}
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	chunks, err := s.chunkRepo.ListChunksByID(ctx, kb.TenantID, ids)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		if chunk == nil || !wanted[chunk.ID] {
			continue
		}
		if chunk.TenantID != kb.TenantID || chunk.KnowledgeBaseID != kb.ID || chunk.ChunkType != types.ChunkTypeFAQ {
			return apperrors.NewForbiddenError(types.LocalizedText(ctx, "Hariç tutulan kayıt geçerli bilgi tabanına ait değil", "Excluded entry does not belong to the current knowledge base"))
		}
		delete(wanted, chunk.ID)
	}
	if len(wanted) != 0 {
		return apperrors.NewNotFoundError(types.LocalizedText(ctx, "Hariç tutulan kayıt bulunamadı", "Excluded entry not found"))
	}
	return nil
}
