package handler

import (
	"context"
	"os"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteBuiltin(t *testing.T) {
	dbPath := "test_builtin.db"
	defer os.Remove(dbPath)

	mgr := NewSQLiteManager()
	defer mgr.Close()

	engine := template.New(false)
	d := &Dispatcher{
		sqliteManager:  mgr,
		templateEngine: engine,
		verbose:        true,
		Log:            func(f string, a ...interface{}) { t.Logf(f, a...) },
	}

	builtin := &SQLiteBuiltin{}
	tmplCtx := template.NewContext()

	// 1. Create table
	action := &Action{
		DB: dbPath,
		SQL: "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)",
	}
	err := builtin.Execute(context.Background(), d, action, tmplCtx)
	require.NoError(t, err)

	// 2. Insert data
	action.SQL = "INSERT INTO test (name) VALUES ('alice'), ('bob')"
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 2, tmplCtx.RowCount)

	// 3. Select data
	action.SQL = "SELECT * FROM test ORDER BY name"
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 2, tmplCtx.RowCount)
	require.Len(t, tmplCtx.Rows, 2)
	assert.Equal(t, "alice", tmplCtx.Rows[0]["name"])
	assert.Equal(t, "bob", tmplCtx.Rows[1]["name"])

	// 4. Update data
	action.SQL = "UPDATE test SET name = 'charlie' WHERE name = 'alice'"
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 1, tmplCtx.RowCount)

	// 5. Verify update
	action.SQL = "SELECT name FROM test WHERE name = 'charlie'"
	err = builtin.Execute(context.Background(), d, action, tmplCtx)
	require.NoError(t, err)
	assert.Equal(t, 1, tmplCtx.RowCount)
	assert.Equal(t, "charlie", tmplCtx.Rows[0]["name"])
}

func TestSQLiteManager_SharedConnection(t *testing.T) {
	dbPath := "test_shared.db"
	defer os.Remove(dbPath)

	mgr := NewSQLiteManager().(*SQLiteManagerImpl)
	defer mgr.Close()

	db1, err := mgr.getDB(dbPath)
	require.NoError(t, err)

	db2, err := mgr.getDB(dbPath)
	require.NoError(t, err)

	assert.True(t, db1 == db2, "Connections to the same path should be shared")
}
