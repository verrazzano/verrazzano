package istio

import "go.uber.org/zap"

func Upgrade(log *zap.SugaredLogger) error {
	log.Info("-------Reached istio upgrade function!!!!----------------")
	return nil
}
