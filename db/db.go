package db

import (
	"context"
	"github.com/rish1988/ipfs-files/log"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	db  *leveldb.DB
	err error
)

const DefaultDBDir  = "~/.ipfs-ipfs/db"

func DB(dbPath string, context context.Context) (*leveldb.DB, error) {
	if db == nil {
		log.Infof("-- Starting LevelDB --")
		if len(dbPath) == 0 {
			dbPath = DefaultDBDir
		}

		if db, err = leveldb.OpenFile(dbPath, nil); err != nil {
			log.Error(err)
			return nil, err
		} else {
			go func() {
				select {
				case <-context.Done():
					if err = db.Close(); err != nil {
						log.Error(err)
					}
				}
			}()
			return db, nil
		}
	} else {
		return db, nil
	}
}
