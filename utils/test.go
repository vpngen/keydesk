package utils

import (
	"fmt"
	"github.com/vpngen/keydesk/keydesk/storage"
	"log"
	"os"
	"testing"
	"time"
)

type TestMainMiddleware func(m *testing.M) int

func BrigadeTestMiddleware(db *storage.BrigadeStorage, mw TestMainMiddleware) TestMainMiddleware {
	return func(m *testing.M) int {
		tmpdir := fmt.Sprintf("test-%d", time.Now().Unix())
		if err := os.Mkdir(tmpdir, 0755); err != nil {
			log.Fatal("failed to create tmpdir:", err)
		}

		*db = storage.BrigadeStorage{
			BrigadeID:       NewBrigadeID(),
			BrigadeFilename: fmt.Sprintf("%s/brigade.json", tmpdir),
			BrigadeSpinlock: fmt.Sprintf("%s/brigade.lock", tmpdir),
		}

		if err := db.CreateBrigade(
			&storage.BrigadeConfig{BrigadeID: db.BrigadeID},
			&storage.BrigadeWgConfig{},
			&storage.BrigadeOvcConfig{},
			&storage.BrigadeIPSecConfig{},
			&storage.BrigadeOutlineConfig{},
		); err != nil {
			log.Fatal("failed to create brigade:", err)
		}

		code := mw(m)

		if err := db.DestroyBrigade(); err != nil {
			log.Println("failed to destroy brigade:", err)
		}

		if err := os.RemoveAll(tmpdir); err != nil {
			log.Fatal("failed to remove tmpdir:", err)
		}

		return code
	}
}
