package afero_tg

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"sort"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"

	"github.com/ffenix113/teleporter/config"
)

var cnf = config.Load("/home/eugene/GoProjects/teleporter/config.yml")

func TestTelegram_Complex(t *testing.T) {
	tg := prepareTg(t)
	now := tg.Now()

	require.NoError(t, tg.MkdirAll("/test/one/three", 0777))
	require.NoError(t, tg.MkdirAll("/test/one/five", 0777))

	dirInfo, err := tg.Stat("/")
	require.NoError(t, err)
	require.Equal(t, DBFileInfo{
		ID:           dirInfo.(DBFileInfo).ID,
		Path:         "/",
		ModeField:    0777 | fs.ModeDir,
		ModTimeField: now,
		IsDirField:   true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, dirInfo)

	dirInfo, err = tg.Stat("/test")
	require.NoError(t, err)
	require.Equal(t, DBFileInfo{
		ID:           dirInfo.(DBFileInfo).ID,
		Path:         "/",
		NameField:    "test",
		ModeField:    0777 | fs.ModeDir,
		ModTimeField: now,
		IsDirField:   true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, dirInfo)

	dir, err := os.Open(".")
	require.NoError(t, err)
	t.Log(dir.Name())
	dir.Close()

	dirInfo, err = tg.Stat("/test/one/three")
	require.NoError(t, err)
	require.Equal(t, "/test/one/three", dirInfo.Name())

	filesInfo, err := tg.Open("/test/one")
	require.NoError(t, err)

	dirNames, err := filesInfo.Readdirnames(-1)
	sort.Strings(dirNames)

	require.Equal(t, []string{"/test/one/five", "/test/one/three"}, dirNames)
}

func TestTelegram_Remove(t *testing.T) {
	tg := prepareTg(t)

	require.NoError(t, tg.MkdirAll("/test/one/three", 0777))
	require.NoError(t, tg.MkdirAll("/test/one/five", 0777))

	require.EqualError(t, tg.Remove("/test"), "cannot remove non-empty dir: /test")
}

func TestTelegram_Rename(t *testing.T) {
	tg := prepareTg(t)

	require.NoError(t, tg.MkdirAll("/test/one/three", 0777))
	require.NoError(t, tg.MkdirAll("/test/one/five", 0777))

	require.NoError(t, tg.Rename("/test/one/three", "/test/one/two"))

	filesInfo, err := tg.Open("/test/one")
	require.NoError(t, err)

	dirNames, err := filesInfo.Readdirnames(-1)
	require.NoError(t, err)
	sort.Strings(dirNames)

	require.Equal(t, []string{"/test/one/five", "/test/one/two"}, dirNames)
}

func prepareTg(t testing.TB) *Telegram {
	nowMonotonic := time.Now()
	now, err := time.Parse(time.RFC3339Nano, nowMonotonic.Truncate(time.Microsecond).Format(time.RFC3339Nano))
	require.NoError(t, err)

	tg := &Telegram{db: createDB(t, cnf), Now: func() time.Time {
		return now
	},
		logger: zap.NewNop(),
	}

	cleanDB(t, tg.db)

	return tg
}

func createDB(t testing.TB, cnf config.Config) *bun.DB {
	dbConn, err := sql.Open("pgx", cnf.DB.DSN)
	if err != nil {
		require.NoError(t, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	if err := dbConn.PingContext(ctx); err != nil {
		require.NoError(t, err)
	}
	cancel()

	return bun.NewDB(dbConn, pgdialect.New())
}

func cleanDB(t testing.TB, db *bun.DB) {
	_, err := db.NewDelete().Model(&DBFileInfo{}).Where("1=1").Exec(context.Background())
	require.NoError(t, err)
}
