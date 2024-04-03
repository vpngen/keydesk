package storage

import (
	"fmt"
	"github.com/vpngen/keydesk/utils"
	"log"
	"os"
	"testing"
	"time"
)

func BrigadeTestMiddleware(db *BrigadeStorage, mw utils.TestMainMiddleware) utils.TestMainMiddleware {
	return func(m *testing.M) int {
		tmpdir := fmt.Sprintf("test-%d", time.Now().Unix())
		if err := os.Mkdir(tmpdir, 0755); err != nil {
			log.Fatal("failed to create tmpdir:", err)
		}

		*db = BrigadeStorage{
			BrigadeID:       utils.NewBrigadeID(),
			BrigadeFilename: fmt.Sprintf("%s/brigade.json", tmpdir),
			BrigadeSpinlock: fmt.Sprintf("%s/brigade.lock", tmpdir),
		}

		if err := db.CreateBrigade(
			&BrigadeConfig{BrigadeID: db.BrigadeID},
			&BrigadeWgConfig{},
			&BrigadeOvcConfig{},
			&BrigadeIPSecConfig{},
			&BrigadeOutlineConfig{},
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
