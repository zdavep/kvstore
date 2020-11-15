package kvstore

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/golang/protobuf/proto" // nolint
	txfmt "github.com/zdavep/kvstore-txfmt"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/version"
	tmdb "github.com/tendermint/tm-db"
)

// App version
var appVersion uint64 = 1

// App is the kvstore base application.
type App struct {
	abci.BaseApplication
	log    log.Logger // Logging
	db     tmdb.DB    // Tendermint key-value database
	batch  tmdb.Batch // Current block transaction
	height int64      // Current block height
	state  *State     // Current state
}

// NewApp creates a new kvstore base application.
func NewApp(db tmdb.DB) abci.Application {
	return &App{db: db, state: readState(db)}
}

// Info returns the last processed block height and version info.
func (app *App) Info(req abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{
		Version:          version.ABCIVersion,
		AppVersion:       appVersion,
		LastBlockHeight:  app.state.Height,
		LastBlockAppHash: app.state.Hash,
	}
}

// CheckTx determines whether a transaction can be committed.
func (app *App) CheckTx(req abci.RequestCheckTx) abci.ResponseCheckTx {
	// Decode transaction
	entries := txfmt.Entries{}
	if err := proto.Unmarshal(req.Tx, &entries); err != nil {
		app.log.Error("failed to unmarshal tx", "err", err)
		return abci.ResponseCheckTx{
			Code: codeInvalidFormat,
			Log:  logForCode(codeInvalidFormat),
		}
	}
	// Validate
	if code := app.isValid(entries); code != codeSuccess {
		return abci.ResponseCheckTx{Code: code, Log: logForCode(code)}
	}
	return abci.ResponseCheckTx{Log: logForCode(codeSuccess)}
}

// BeginBlock starts a new database transaction.
func (app *App) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	app.batch = app.db.NewBatch()
	app.height = req.Header.Height
	return abci.ResponseBeginBlock{}
}

// DeliverTx sets a key-value pair in the current transaction.
func (app *App) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	// Decode transaction
	entries := txfmt.Entries{}
	if err := proto.Unmarshal(req.Tx, &entries); err != nil {
		app.log.Error("failed to unmarshal tx", "err", err)
		return abci.ResponseDeliverTx{Code: codeInvalidFormat, Log: logForCode(codeInvalidFormat)}
	}
	// Validate
	if code := app.isValid(entries); code != codeSuccess {
		return abci.ResponseDeliverTx{Code: code, Log: logForCode(code)}
	}
	// Set hash for updates
	hsh := sha256.New()
	if _, err := hsh.Write(app.state.Hash); err != nil {
		app.log.Error("unable to initialize state hash udpate", "err", err)
		return abci.ResponseDeliverTx{Code: codeHashErr, Log: logForCode(codeHashErr)}
	}
	// Store entries
	for _, e := range entries.Entries {
		app.batch.Set(e.Key, e.Value)
		if _, err := hsh.Write(e.Value); err != nil {
			app.log.Error("unable to update app state hash", "err", err)
			return abci.ResponseDeliverTx{Code: codeHashErr, Log: logForCode(codeHashErr)}
		}
	}
	// Update state
	app.state.Height = app.height
	sum := hsh.Sum(nil)
	app.state.Hash = make([]byte, len(sum))
	copy(app.state.Hash, sum)
	return abci.ResponseDeliverTx{Code: codeSuccess, Log: logForCode(codeSuccess)}
}

// Commit writes the current batch to the database.
func (app *App) Commit() abci.ResponseCommit {
	defer app.batch.Close()
	if err := writeState(app.batch, app.state); err != nil {
		app.log.Error("error saving app state", "err", err)
		panic(err)
	}
	if err := app.batch.Write(); err != nil {
		app.log.Error("error during batch write", "err", err)
		panic(err)
	}
	return abci.ResponseCommit{Data: app.state.Hash}
}

// Query fetches the value for a key from the database.
func (app *App) Query(req abci.RequestQuery) (res abci.ResponseQuery) {
	res.Key = req.Data
	var err error
	if res.Value, err = app.db.Get(req.Data); err != nil {
		app.log.Error("query error", "err", err)
		res.Log = "exists:false"
		return
	}
	res.Log = fmt.Sprintf("exists:%v", res.Value != nil)
	return
}

// Determine whether a value can committed.
func (app *App) isValid(entries txfmt.Entries) uint32 {
	// Check whether the key-value pair already exists.
	for _, e := range entries.Entries {
		existing, err := app.db.Get(e.Key)
		if err != nil {
			app.log.Error("db get error", "err", err)
			return codeDatabaseErr
		}
		if existing != nil && bytes.Equal(e.Value, existing) {
			app.log.Error("isValid", "msg", "value already exists")
			return codeDupValueErr
		}
	}
	return codeSuccess
}
