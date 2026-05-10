package admin

import (
	"strconv"
	"strings"
	"time"
)

var accountSuccessRateBatchCache = newSnapshotCache(30 * time.Second)
var dashboardSuccessRateTrendCache = newSnapshotCache(30 * time.Second)

func buildAccountSuccessRateBatchCacheKey(accountIDs []int64) string {
	if len(accountIDs) == 0 {
		return "accounts_success_rate_empty"
	}
	var b strings.Builder
	b.Grow(len(accountIDs) * 6)
	_, _ = b.WriteString("accounts_success_rate:")
	for i, id := range accountIDs {
		if i > 0 {
			_ = b.WriteByte(',')
		}
		_, _ = b.WriteString(strconv.FormatInt(id, 10))
	}
	return b.String()
}
