package mashery

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/vault/sdk/logical"
	"math"
	"time"
)

const (
	storedRoleKeyPathSuffix        = "/key"
	storedRolePrivateKeyPathSuffix = "/pk"
	storedRoleUsageKeyPathSuffix   = "/usage"
)

func roleKeysPath[T any](reqCtx *RequestHandlerContext[T]) string {
	return reqCtx.storagePath + storedRoleKeyPathSuffix
}

func roleUsagePath[T any](reqCtx *RequestHandlerContext[T]) string {
	return reqCtx.storagePath + storedRoleUsageKeyPathSuffix
}

func rolePrivateKeyPath[T any](reqCtx *RequestHandlerContext[T]) string {
	return reqCtx.storagePath + storedRolePrivateKeyPathSuffix
}

func readRoleDo[T RoleContext](ctx context.Context, reqCtx *RequestHandlerContext[T], requireRole bool) (*logical.Response, error) {
	sr := reqCtx.plugin.InitialRole(reqCtx.data)

	if roleFound, err := reqCtx.ReadPath(ctx, roleKeysPath(reqCtx), &sr.Keys); err != nil {
		return nil, err
	} else if !roleFound && requireRole {
		return logical.ErrorResponse("role is not found"), nil
	}

	if usageFound, err := reqCtx.ReadPath(ctx, roleUsagePath(reqCtx), &sr.Usage); err != nil {
		return nil, err
	} else if !usageFound && requireRole {
		return logical.ErrorResponse("role usage is not found"), nil
	}

	reqCtx.heap.CarryRole(&sr)

	return nil, nil
}

func readRole[T RoleContext](mustBePresent bool) func(ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	return func(ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
		return readRoleDo(ctx, reqCtx, mustBePresent)
	}
}

func updateRoleKeysFromRequest(_ context.Context, reqCtx *RequestHandlerContext[RoleContext]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()
	if role == nil {
		return nil, errors.New("updateRoleKeysFromRequest requires a non-nil role to operate")
	}

	data := reqCtx.data
	retVal := &role.Keys

	if areaIdRaw, ok := data.GetOk(roleAreaIdField); ok {
		retVal.AreaId = areaIdRaw.(string)
	}
	if areaNidRaw, ok := data.GetOk(roleAreaNidField); ok {
		retVal.AreaNid = areaNidRaw.(int)
	}
	if apiKeyRaw, ok := data.GetOk(roleApiKeField); ok {
		retVal.ApiKey = apiKeyRaw.(string)
	}
	if keySecretRaw, ok := data.GetOk(roleSecretField); ok {
		retVal.KeySecret = keySecretRaw.(string)
	}
	if usernameRaw, ok := data.GetOk(roleUsernameField); ok {
		retVal.Username = usernameRaw.(string)
	}
	if passwordRaw, ok := data.GetOk(rolePasswordField); ok {
		retVal.Password = passwordRaw.(string)
	}

	if secretQpsRaw, ok := data.GetOk(roleQpsField); ok {
		retVal.MaxQPS = secretQpsRaw.(int)
	}

	return nil, nil
}

func blockOperationOnImportedRole[T RoleContext](_ context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	if reqCtx.heap.GetRole().Keys.Imported {
		return logical.ErrorResponse("operation is not permitted on an imported role"), nil
	} else {
		return nil, nil
	}
}

func blockOperationOnForceProxyRole[T RoleContext](_ context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	if reqCtx.heap.GetRole().Keys.ForceProxyMode {
		return logical.ErrorResponse("operation is not permitted as this role requires proxy mode"), nil
	} else {
		return nil, nil
	}
}

func blockUsageExceedingLimits[T RoleContext](_ context.Context,
	reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {

	role := reqCtx.heap.GetRole()
	if role.Usage.Expired() {
		return logical.ErrorResponse("this role has expired (granted until %s)", role.Usage.ExpiryTimeString()), nil
	} else if role.Usage.Depleted() {
		return logical.ErrorResponse("this role has depleted its usage quota"), nil
	}

	return nil, nil
}

func decreaseRemainingUsageQuota[T RoleContext](ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()

	if role.Usage.HasUsageQuota() {
		role.Usage.ReduceRemainingQuota()
		if err := reqCtx.WritePath(ctx, roleUsagePath(reqCtx), &role.Usage); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func blockRoleIncapableOf[T RoleContext](apiLevel int) func(ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	return func(_ context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
		return doBlockRoleIncapableOf(reqCtx.heap.GetRole(), apiLevel)
	}
}

func blockNonExportableRole(_ context.Context,
	reqCtx *RequestHandlerContext[RoleExportContext]) (*logical.Response, error) {

	if !reqCtx.heap.GetRole().Keys.Exportable {
		return logical.ErrorResponse("this role is not exportable"), nil
	} else {
		return nil, nil
	}
}

func doBlockRoleIncapableOf(role *StoredRole, apiLevel int) (*logical.Response, error) {
	capable := false
	switch apiLevel {
	case 2:
		capable = role.Keys.IsV2Capable()
	case 3:
		capable = role.Keys.IsV3Capable()
	default:
		return logical.ErrorResponse("unsupported api version: %d", apiLevel), nil
	}

	if !capable {
		return logical.ErrorResponse("role is not capable of api version %d", apiLevel), nil
	} else {
		return nil, nil
	}
}

func allowOnlyV2CapableRole[T RoleContext](_ context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	if !reqCtx.heap.GetRole().Keys.IsV2Capable() {
		return logical.ErrorResponse("this role is not V2 capable"), nil
	} else {
		return nil, nil
	}
}

func allowOnlyV3CapableRole[T RoleContext](_ context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	if !reqCtx.heap.GetRole().Keys.IsV3Capable() {
		return logical.ErrorResponse("this role is not V3 capable"), nil
	} else {
		return nil, nil
	}
}

func saveRoleKeys[T RoleContext](ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	err := reqCtx.WritePath(ctx, roleKeysPath(reqCtx), &reqCtx.heap.GetRole().Keys)
	return nil, err
}

func setInitialRoleUsage[T RoleContext](_ context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	role := reqCtx.heap.GetRole()

	role.Usage.UnboundedUsage()
	role.Usage.ResetToken()
	return nil, nil
}

func forgetUsedToken(_ context.Context, reqCtx *RequestHandlerContext[RoleContext]) (*logical.Response, error) {
	reqCtx.heap.GetRole().Usage.ResetToken()
	return nil, nil
}

func saveRoleUsage[T RoleContext](ctx context.Context, reqCtx *RequestHandlerContext[T]) (*logical.Response, error) {
	err := reqCtx.WritePath(ctx, roleUsagePath(reqCtx), reqCtx.heap.GetRole().Usage)
	return nil, err
}

func renderRole(_ context.Context, reqCtx *RequestHandlerContext[RoleContext]) (*logical.Response, error) {
	term := "∞"
	termRemaining := "∞"
	usageRemaining := "∞"

	role := reqCtx.heap.GetRole()

	if role.Usage.ExplicitTerm > 0 {
		termTime := time.Unix(role.Usage.ExplicitTerm, 0)

		term = termTime.Format(time.RFC822)
		dur := termTime.Sub(time.Now())
		if dur > 0 {
			termRemaining = dur.String()
		} else {
			termRemaining = "---EXPIRED---"
		}
	}

	if role.Usage.ExplicitNumUses > 0 {
		if role.Usage.RemainingNumUses <= 0 {
			usageRemaining = "---DEPLETED---"
		} else {
			used := role.Usage.ExplicitNumUses - role.Usage.RemainingNumUses
			usePct := int(math.Round(float64(100) * float64(used) / float64(role.Usage.ExplicitNumUses)))

			usageRemaining = fmt.Sprintf("%d times (%d%% used)", role.Usage.RemainingNumUses, usePct)
		}
	}

	v3Token := "---NOT-SET---"
	v3TokenLife := "n/a"

	if len(role.Usage.V3Token) > 0 {
		v3Token = "---ACQUIRED---"

		if role.Usage.V3TokenExpired() {
			v3Token = "---EXPIRED---"
			now := time.Now()
			expTime := time.Unix(role.Usage.V3TokenExpiry, 0)
			diff := now.Sub(expTime)

			v3TokenLife = fmt.Sprintf("expired on %s (%s ago)", expTime, diff)
		} else {
			if role.Usage.V3TokenNeedsRenew() {
				v3Token = "---NEEDS-RENEW---"
			}

			life := time.Unix(role.Usage.V3TokenExpiry, 0).Sub(time.Now())
			v3TokenLife = life.String()
		}
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"v2_capable":        role.Keys.IsV2Capable(),
			"v3_capable":        role.Keys.IsV3Capable(),
			"qps":               role.Keys.MaxQPS,
			"term":              term,
			"term_remaining":    termRemaining,
			"use_remaining":     usageRemaining,
			"exportable":        role.Keys.Exportable,
			"forced_proxy_mode": role.Keys.ForceProxyMode,
			"v3_token":          v3Token,
			"v3_token_life":     v3TokenLife,
		},
	}

	return resp, nil
}
