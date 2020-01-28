package kvstore

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	txfmt "github.com/zdavep/kvstore-txfmt"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/version"
)

// App version
var appVersion uint64 = 1

// App is the kvstore base application.
type App struct {
	abci.BaseApplication
	log    log.Logger  // Logging
	db     *badger.DB  // Key-value database
	batch  *badger.Txn // Block transaction
	height int64       // Current block height
	state  *State      // Current state
}

// NewApp creates a new kvstore base application.
func NewApp(db *badger.DB) abci.Application {
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
	entries := &txfmt.Entries{}
	if err := proto.Unmarshal(req.Tx, entries); err != nil {
		app.log.Error("failed to unmarshal tx", "err", err)
		return abci.ResponseCheckTx{Code: codeInvalidFormat, Log: "error"}
	}
	// Validate
	if code := app.isValid(entries); code != codeSuccess {
		return abci.ResponseCheckTx{Code: code, Log: "error"}
	}
	return abci.ResponseCheckTx{Log: "success"}
}

// BeginBlock starts a new database transaction.
func (app *App) BeginBlock(req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	app.batch = app.db.NewTransaction(true)
	app.height = req.Header.Height
	return abci.ResponseBeginBlock{}
}

// DeliverTx sets a key-value pair in the current transaction.
func (app *App) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	// Decode transaction
	entries := &txfmt.Entries{}
	if err := proto.Unmarshal(req.Tx, entries); err != nil {
		app.log.Error("failed to unmarshal tx", "err", err)
		return abci.ResponseDeliverTx{Code: codeInvalidFormat, Log: "error"}
	}
	// Validate
	if code := app.isValid(entries); code != codeSuccess {
		return abci.ResponseDeliverTx{Code: code, Log: "error"}
	}
	// Set hash for updates
	hsh := sha256.New()
	if _, err := hsh.Write(app.state.Hash); err != nil {
		app.log.Error("unable to initialize state hash udpate", "err", err)
		return abci.ResponseDeliverTx{Code: codeHashErr, Log: "error"}
	}
	// Store entries
	for _, e := range entries.Entries {
		if err := app.batch.Set(e.Key, e.Value); err != nil {
			app.log.Error("unable to set value", "err", err)
			return abci.ResponseDeliverTx{Code: codeDatabaseErr, Log: "error"}
		}
		if _, err := hsh.Write(e.Value); err != nil {
			app.log.Error("unable to update app state hash", "err", err)
			return abci.ResponseDeliverTx{Code: codeHashErr, Log: "error"}
		}
	}
	// Update state
	app.state.Height = app.height
	copy(app.state.Hash, hsh.Sum(nil))
	return abci.ResponseDeliverTx{Log: "success"}
}

// Commit writes the current batch to the database.
func (app *App) Commit() abci.ResponseCommit {
	defer app.batch.Discard()
	if err := writeState(app.batch, app.state); err != nil {
		app.log.Error("error saving app state", "err", err)
		panic(err)
	}
	if err := app.batch.Commit(); err != nil {
		app.log.Error("error during transaction commit", "err", err)
		panic(err)
	}
	return abci.ResponseCommit{}
}

// Query fetches the value for a key from the database.
func (app *App) Query(req abci.RequestQuery) (res abci.ResponseQuery) {
	res.Key = req.Data
	var err error
	if res.Value, err = get(app.db, req.Data); err != nil {
		app.log.Error("query error", "err", err)
		res.Log = "exists:false"
		return
	}
	res.Log = fmt.Sprintf("exists:%v", res.Value != nil)
	return
}

// Determine whether a value can committed.
func (app *App) isValid(entries *txfmt.Entries) uint32 {
	// Check whether the key-value pair already exists.
	for _, e := range entries.Entries {
		existing, err := get(app.db, e.Key)
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
