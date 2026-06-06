package helpers

import (
	"net/http/httptest"
	"testing"
	"uas/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	id := GetRequestId(req)
	assert.Equal(t, "", id)

	req = SetRequestId(req, "test-id-123")
	id = GetRequestId(req)
	assert.Equal(t, "test-id-123", id)
}

func TestUserId(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	id := GetUserId(req)
	assert.Equal(t, "", id)

	req = SetUserId(req, "user-abc")
	id = GetUserId(req)
	assert.Equal(t, "user-abc", id)
}

func TestDepartmentId(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	id := GetDepartmentId(req)
	assert.Equal(t, "", id)

	req = SetDepartmentId(req, "dept-xyz")
	id = GetDepartmentId(req)
	assert.Equal(t, "dept-xyz", id)
}

func TestRole(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	role := GetRole(req)
	assert.Equal(t, models.Role(""), role)

	req = SetRole(req, models.Admin)
	role = GetRole(req)
	assert.Equal(t, models.Admin, role)

	req = SetRole(req, models.PlatformAdmin)
	role = GetRole(req)
	assert.Equal(t, models.PlatformAdmin, role)
}

func TestAllContextValues(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	req = SetRequestId(req, "req-1")
	req = SetUserId(req, "user-1")
	req = SetDepartmentId(req, "dept-1")
	req = SetRole(req, models.TenantAdmin)

	assert.Equal(t, "req-1", GetRequestId(req))
	assert.Equal(t, "user-1", GetUserId(req))
	assert.Equal(t, "dept-1", GetDepartmentId(req))
	assert.Equal(t, models.TenantAdmin, GetRole(req))
}
