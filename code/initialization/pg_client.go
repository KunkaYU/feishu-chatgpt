package initialization

import (
	"context"

	pg "github.com/jackc/pgx/v5"
)

var pgClient *pg.Conn

func LoadPGClient(config Config) {
	pgClient, _ = pg.Connect(context.Background(), config.DBURL)
}

func GetPGClient() *pg.Conn {
	return pgClient
}
