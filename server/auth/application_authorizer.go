package auth

import (
	"context"
	"sync"
	"time"

	"github.com/palantir/stacktrace"
	"github.com/patrickmn/go-cache"
	uuid "github.com/satori/go.uuid"
	"github.com/tnyim/jungletv/utils/event"
)

// ThirdPartyAuthorizer is responsible for authorizing external systems to act on a user's behalf
type ThirdPartyAuthorizer struct {
	jwtManager *JWTManager
	processes  *cache.Cache[string, *ThirdPartyAuthorizationProcess]
}

// NewThirdPartyAuthorizer returns a new initialized ThirdPartyAuthorizer
func NewThirdPartyAuthorizer(jwtManager *JWTManager) *ThirdPartyAuthorizer {
	a := &ThirdPartyAuthorizer{
		jwtManager: jwtManager,
		processes:  cache.New[string, *ThirdPartyAuthorizationProcess](5*time.Minute, 30*time.Second),
	}
	a.processes.OnEvicted(a.onEvicted)
	return a
}

// ThirdPartyAuthorizationProcess is the process for authorizing a third party to act on a user's behalf
type ThirdPartyAuthorizationProcess struct {
	mu              sync.Mutex
	authorizer      *ThirdPartyAuthorizer
	ID              string                                           // generated by the server
	ApplicationName string                                           // provided by the third party on a per-request basis
	PermissionLevel PermissionLevel                                  // provided by the third party on a per-request basis
	Reason          string                                           // provided by the third party on a per-request basis, shown to the user making it clear the third-party is being quoted
	Complete        bool                                             // set to true once the user consents or dissents
	UserConsented   event.Event[ThirdPartyAuthenticationCredentials] // fired when the user approves the authorization request
	UserDissented   event.NoArgEvent                                 // fired when the user rejects the authorization request or when it expires
}

// ThirdPartyAuthenticationCredentials are the credentials provided to a third party when the authorization process is approved by the user
type ThirdPartyAuthenticationCredentials struct {
	AuthToken string
	Expiry    time.Time
}

func (authorizer *ThirdPartyAuthorizer) onEvicted(id string, process *ThirdPartyAuthorizationProcess) {
	process.mu.Lock()
	defer process.mu.Unlock()

	if !process.Complete {
		process.UserDissented.Notify(false)
	}
}

func (authorizer *ThirdPartyAuthorizer) BeginProcess(applicationName string, permissionLevel PermissionLevel, reason string) *ThirdPartyAuthorizationProcess {
	process := &ThirdPartyAuthorizationProcess{
		authorizer:      authorizer,
		ID:              uuid.NewV4().String(),
		ApplicationName: applicationName,
		PermissionLevel: permissionLevel,
		Reason:          reason,
		UserConsented:   event.New[ThirdPartyAuthenticationCredentials](),
		UserDissented:   event.NewNoArg(),
	}
	authorizer.processes.SetDefault(process.ID, process)
	return process
}

func (authorizer *ThirdPartyAuthorizer) GetProcess(id string) (*ThirdPartyAuthorizationProcess, bool) {
	return authorizer.processes.Get(id)
}

func (process *ThirdPartyAuthorizationProcess) Consent(ctx context.Context, user User) error {
	defer process.authorizer.processes.Delete(process.ID)
	process.mu.Lock()
	defer process.mu.Unlock()

	if !UserPermissionLevelIsAtLeast(user, process.PermissionLevel) {
		return stacktrace.NewError("cannot authorize third-party for higher permission level than that of the user")
	}

	username := user.Address()[:14]
	if user.Nickname() != nil && *user.Nickname() != "" {
		username = *user.Nickname()
	}
	token, expiry, err := process.authorizer.jwtManager.Generate(ctx, user.Address(), process.PermissionLevel, username)
	if err != nil {
		return stacktrace.Propagate(err, "")
	}

	process.Complete = true
	process.UserConsented.Notify(ThirdPartyAuthenticationCredentials{
		AuthToken: token,
		Expiry:    expiry,
	}, false)

	return nil
}

func (process *ThirdPartyAuthorizationProcess) Dissent() {
	defer process.authorizer.processes.Delete(process.ID)
	process.mu.Lock()
	defer process.mu.Unlock()

	process.Complete = true
	process.UserDissented.Notify(false)
}
