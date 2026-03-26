/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package security

import (
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/security"
)

func GetTenantInfo(o interface{}) (tenantID string, ownerID string) {
	if accessor, ok := o.(orm.SystemFieldAccessor); ok {
		return accessor.GetSystemString(orm.TenantIDKey), accessor.GetSystemString(orm.OwnerIDKey)
	}
	return "", ""
}

func SetTenantInfo(o interface{}, tenantID string, ownerID string) {
	v, ok := o.(orm.SystemFieldAccessor)
	if !ok {
		panic("object does not implement SystemAccessor")
	}
	v.SetSystemValue(orm.TenantIDKey, tenantID)
	v.SetSystemValue(orm.OwnerIDKey, ownerID)
}

func SetUserSessionWithTenantInfo(sessionInfo *security.UserSessionInfo, tenantID string, userID string) {
	sessionInfo.Set(orm.TenantIDKey, tenantID)
	sessionInfo.Set(orm.OwnerIDKey, userID)
}

func GetTenantInfoFromUserSession(o *security.UserSessionInfo) (tenantID string, ownerID string) {
	tenantID, _ = o.GetString(orm.TenantIDKey)
	ownerID, _ = o.GetString(orm.OwnerIDKey)
	return
}
