package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func GetCertificateKey(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

func TransformToken(clusterToken string) string {
	hash := sha256.New()
	hash.Write([]byte(clusterToken))
	hashString := hex.EncodeToString(hash.Sum(nil))
	return fmt.Sprintf("%s.%s", hashString[len(hashString)-6:], hashString[:16])
}

func GetCertSansRevision(certsan []string) string {
	h := sha256.New()
	v, _ := json.Marshal(certsan)
	h.Write(v)
	return fmt.Sprintf("%x", h.Sum(nil))
}
