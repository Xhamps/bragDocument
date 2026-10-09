package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// The PRD-0004 §7 table, row by row: owner, editor, viewer.
func TestCanMatchesPRD(t *testing.T) {
	rows := []struct {
		perm                  Permission
		owner, editor, viewer bool
	}{
		{PermRead, true, true, true},
		{PermWriteLogs, true, true, false},
		{PermManage, true, false, false},
		{PermDelete, true, false, false},
		{PermShare, true, false, false},
		{PermTransfer, true, false, false},
	}
	for _, r := range rows {
		require.Equal(t, r.owner, Can(RoleOwner, r.perm), "owner %s", r.perm)
		require.Equal(t, r.editor, Can(RoleEditor, r.perm), "editor %s", r.perm)
		require.Equal(t, r.viewer, Can(RoleViewer, r.perm), "viewer %s", r.perm)
		require.False(t, Can("", r.perm), "no role %s", r.perm)
	}
}

func TestAccessErrorExplainsAndIsForbidden(t *testing.T) {
	var err error = &AccessError{Role: RoleViewer, Perm: PermWriteLogs}
	require.True(t, errors.Is(err, ErrForbidden))
	require.Equal(t, "you are viewer on this document and cannot create, edit, or delete logs", err.Error())
}

func TestGrantable(t *testing.T) {
	require.True(t, RoleEditor.Grantable())
	require.True(t, RoleViewer.Grantable())
	require.False(t, RoleOwner.Grantable())
	require.False(t, Role("admin").Grantable())
}
