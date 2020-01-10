package kvstore

import "github.com/tendermint/tendermint/libs/log"

// LoggerReceiver takes a tendermint logger.
type LoggerReceiver interface {
	SetLogger(logger log.Logger)
}

// SetLogger makes App a LoggerReceiver.
func (app *App) SetLogger(logger log.Logger) {
	app.log = logger.With("module", "kvstore")
}
