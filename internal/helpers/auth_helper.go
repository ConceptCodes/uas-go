package helpers

import "encoding/base64"

func GenerateToken(tenantId string, tenantSecret string) string {
	return base64.StdEncoding.EncodeToString([]byte(tenantId + ":" + tenantSecret))
}
