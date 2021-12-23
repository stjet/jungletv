package server

import (
	"context"
	"time"

	"github.com/palantir/stacktrace"
	"github.com/patrickmn/go-cache"
	"github.com/tnyim/jungletv/server/auth"
)

// UserCache caches public user information
type UserCache interface {
	GetOrFetchUser(ctx context.Context, address string) (User, error)
	CacheUser(ctx context.Context, user User) error
}

// MemoryUserCache is a memory-based nickname cache
type MemoryUserCache struct {
	c *cache.Cache
}

// NewMemoryUserCache returns a new MemoryNicknameCache
func NewMemoryUserCache() *MemoryUserCache {
	return &MemoryUserCache{
		c: cache.New(1*time.Hour, 11*time.Minute),
	}
}

// GetOrFetchUser loads user info from cache, falling back to the database if necessary
func (c *MemoryUserCache) GetOrFetchUser(ctxCtx context.Context, address string) (User, error) {
	i, present := c.c.Get(address)
	if !present {
		ctx, err := BeginTransaction(ctxCtx)
		if err != nil {
			return nil, stacktrace.Propagate(err, "")
		}
		defer ctx.Commit() // read-only tx

		var userRecord struct {
			Nickname        *string
			PermissionLevel string `db:"permission_level"`
		}
		err = ctx.Tx().GetContext(ctx, &userRecord, `SELECT nickname, permission_level FROM chat_user WHERE address = $1`, address)
		if err != nil {
			// no nickname for this user
			return nil, nil
		}

		user := NewAddressOnlyUserWithPermissionLevel(address, auth.PermissionLevel(userRecord.PermissionLevel))
		user.SetNickname(userRecord.Nickname)

		c.c.SetDefault(address, user)
		return user, nil
	}
	return i.(User), nil
}

// CacheUser saves user information in cache
func (c *MemoryUserCache) CacheUser(_ context.Context, user User) error {
	if user == nil || user.IsUnknown() {
		return stacktrace.NewError("attempt to cache invalid user")
	}
	c.c.SetDefault(user.Address(), user)
	return nil
}
