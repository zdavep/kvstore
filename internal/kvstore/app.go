package kvstore

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	txfmt "github.com/zdavep/kvstore-txfmt"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/version"
)

// Last processed block height key
var lastBlockHeightKey []byte = []byte("~kvstore.internal.v1.last_block_height")

// App version
var appVersion uint64 = 1

// App is the kvstore base application.
type App struct {
	abci.BaseApplication
	log    log.Logger  // Logging
	db     *badger.DB  // Key-value database
	batch  *badger.Txn // Block transaction
	height int64       // Current block height
}

// NewApp creates a new kvstore base application.
func NewApp(db *badger.DB) abci.Application {
	return &App{db: db}
}

// Info returns the last processed block height and version info.
func (app *App) Info(req abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{
		Version:         version.ABCIVersion,
		AppVersion:      appVersion,
		LastBlockHeight: app.lastProcessedHeight(),
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
	// Store
	for _, e := range entries.Entries {
		if err := app.batch.Set(e.Key, e.Value); err != nil {
			app.log.Error("unable to set value", "err", err)
			return abci.ResponseDeliverTx{Code: codeDatabaseErr, Log: "error"}
		}
	}
	app.snapshotHeight()
	return abci.ResponseDeliverTx{Log: "success"}
}

// Commit writes the current batch to the database.
func (app *App) Commit() abci.ResponseCommit {
	if err := app.batch.Commit(); err != nil {
		app.log.Error("error during transaction commit", "err", err)
	}
	return abci.ResponseCommit{}
}

// Query fetches the value for a key from the database.
func (app *App) Query(req abci.RequestQuery) (res abci.ResponseQuery) {
	res.Key = req.Data
	var err error
	if res.Value, err = app.get(req.Data); err != nil {
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
		existing, err := app.get(e.Key)
		if err != nil {
			app.log.Error("db get error", "err", err)
			return codeDatabaseErr
		}
		if existing != nil && bytes.Equal(e.Value, existing) {
			app.log.Error("isValid", "msg", "value already exists")
			return codeDupValue
		}
	}
	return codeSuccess
}

// Get the value for a given key from the ABCI database.
func (app *App) get(key []byte) (value []byte, err error) {
	err = app.db.View(func(txn *badger.Txn) error {
		item, e := txn.Get(key)
		if e == badger.ErrKeyNotFound {
			return nil
		}
		if e != nil {
			return e
		}
		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)
			return nil
		})
	})
	return
}

// Retreive the last processed block height from the ABCI database.
func (app *App) lastProcessedHeight() (i int64) {
	if val, err := app.get(lastBlockHeightKey); err == nil {
		if i, err = binary.ReadVarint(bytes.NewReader(val)); err != nil {
			app.log.Error("failed to get last processed block height", "err", err)
		}
	}
	return
}

// Save the current block height to the ABCI database.
func (app *App) snapshotHeight() {
	buf := make([]byte, 8)
	binary.PutVarint(buf, app.height)
	if err := app.batch.Set(lastBlockHeightKey, buf); err != nil {
		app.log.Error("failed to snapshot block height", "err", err)
	}
}
