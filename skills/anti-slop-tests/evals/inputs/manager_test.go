package users

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	repo := newFakeRepo()
	m := NewManager(repo)
	require.NotNil(t, m)
	require.Equal(t, repo, m.repo)
}

func TestManager_GetUser(t *testing.T) {
	repo := newFakeRepo()
	repo.EXPECT().GetUser(gomock.Any(), "u1").Return(&User{
		ID:    "u1",
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	}, nil)
	m := NewManager(repo)

	got, err := m.GetUser(context.Background(), "u1")

	require.NoError(t, err)
	require.Equal(t, "u1", got.ID)
	require.Equal(t, "Alice", got.Name)
	require.Equal(t, "alice@example.com", got.Email)
	require.Equal(t, 30, got.Age)
	require.True(t, got.Active)
}

func TestManager_Save(t *testing.T) {
	repo := newFakeRepo()
	m := NewManager(repo)

	m.Save(context.Background(), &User{ID: "u1"})

	repo.AssertExpectations(t)
}

func TestManager_Delete(t *testing.T) {
	m := NewManager(newFakeRepo())
	m.Delete(context.Background(), "u1")
}

func TestManager_Validate(t *testing.T) {
	m := NewManager(newFakeRepo())

	_, err := m.Validate(context.Background(), &User{})

	require.EqualError(t, err, "validate user: name is required")
}

func TestManager_Sync(t *testing.T) {
	m := NewManager(newFakeRepo())
	go m.Sync(context.Background())
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 1, m.syncCount)
}

func TestManager_Normalize(t *testing.T) {
	m := NewManager(newFakeRepo())
	got := m.Normalize(&User{Name: " alice "})
	require.True(t, reflect.DeepEqual(&User{Name: "alice"}, got))
}

func TestManager_Rate(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "small", in: 10, want: 100},
		{name: "medium", in: 20, want: 100},
		{name: "large", in: 30, want: 100},
		{name: "huge", in: 40, want: 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, Rate(tc.in))
		})
	}
}

func TestManager_Legacy(t *testing.T) {
	t.Skip("broken since the storage migration")
	require.True(t, false)
}
