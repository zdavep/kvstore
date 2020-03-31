package kvstore

import "github.com/tendermint/tendermint/libs/log"

// SetLogger makes App a LoggerReceiver.
func (app *App) SetLogger(logger log.Logger) {
	app.log = logger.With("module", "kvstore")
}
