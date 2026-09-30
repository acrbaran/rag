package service

import (
	"strings"
	"testing"

	"github.com/acrbaran/rag/internal/types"
	"github.com/stretchr/testify/require"
)

func TestUpdateMyProfileValidatesAndPersists(t *testing.T) {
	repo := &switchTenantUserRepo{users: map[string]types.User{
		"alice": {ID: "alice", FirstName: "Old", LastName: "Name", Phone: "+905550000000"},
	}}
	svc := &userService{userRepo: repo}

	user, err := svc.UpdateMyProfile(t.Context(), "alice", "  Baran ", " Acar ", " +905412992300 ")
	require.NoError(t, err)
	require.Equal(t, "Baran", user.FirstName)
	require.Equal(t, "Acar", user.LastName)
	require.Equal(t, "+905412992300", user.Phone)
	require.Equal(t, "Baran", repo.users["alice"].FirstName)

	invalid := []struct{ first, last, phone string }{
		{"", "Acar", "+905412992300"},
		{"Baran", "  ", "+905412992300"},
		{"Baran", "Acar", ""},
		{"Baran", "Acar", "05412992300"},
		{strings.Repeat("a", 101), "Acar", "+905412992300"},
	}
	for _, tc := range invalid {
		_, err := svc.UpdateMyProfile(t.Context(), "alice", tc.first, tc.last, tc.phone)
		require.Error(t, err)
	}
	require.Equal(t, "Baran", repo.users["alice"].FirstName, "rejected updates must not persist")
	require.Equal(t, 1, repo.updateCalls)
}
