package mashery

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdatingUsage(t *testing.T) {
	reqCtx := &RequestHandlerContext[RoleContext]{
		heap: &RoleContainer{
			role: &StoredRole{
				Usage: StoredRoleUsage{
					V3Token: "oldToken",
				},
			},
		},
	}

	role := reqCtx.heap.GetRole()
	func(ctx *RequestHandlerContext[RoleContext]) {
		role.Usage.V3Token = "updated"
	}(reqCtx)

	assert.Equal(t, "updated", role.Usage.V3Token)
}
