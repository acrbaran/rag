package repository

import (
	"context"
	"testing"

	"github.com/acrbaran/rag/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEmbedChannelCreatePreservesFalseOptions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.EmbedChannel{}))

	channel := &types.EmbedChannel{
		TenantID: 1, AgentID: types.BuiltinQuickAnswerID,
		Name: "disabled", PublishToken: "em_test", AllowedOrigins: types.JSON(`[]`),
	}
	require.NoError(t, NewEmbedChannelRepository(db).Create(context.Background(), channel))
	stored, err := NewEmbedChannelRepository(db).GetByID(context.Background(), channel.ID)
	require.NoError(t, err)
	require.False(t, stored.Enabled)
	require.False(t, stored.ShowSuggestedQuestions)
}
