package webhookreadiness

import (
	"fmt"
	"go.uber.org/zap"
)

// StartReadinessServer to check webhook readiness
func StartReadinessServer(log *zap.SugaredLogger) {
	log.Info(fmt.Println("put something here"))

}
