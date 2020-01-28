package kvstore

import "github.com/dgraph-io/badger"

// Get the value for a given key from the ABCI database.
func get(db *badger.DB, key []byte) (value []byte, err error) {
	err = db.View(func(txn *badger.Txn) error {
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
