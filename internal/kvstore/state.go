package kvstore

import (
	"encoding/json"

	tmdb "github.com/tendermint/tm-db"
)

// The key to store app state under.
const stateKey string = "~kvstore.internal.state.v1"

// State is the current state of the kvstore application.
type State struct {
	// Height is the last processed block height.
	Height int64
	// Hash is the latest calculated hash.
	Hash []byte
}

// Reads the current application state from a badger database.
func readState(db tmdb.DB) *State {
	value, err := db.Get([]byte(stateKey))
	if err != nil {
		panic(err)
	}
	s := &State{}
	if value == nil {
		return s
	}
	if err := json.Unmarshal(value, s); err != nil {
		panic(err)
	}
	return s
}

// Writes the current application state to a badger transaction.
func writeState(txn tmdb.Batch, s *State) error {
	stateJSON, err := json.Marshal(s)
	if err != nil {
		return err
	}
	txn.Set([]byte(stateKey), stateJSON)
	return nil
}
