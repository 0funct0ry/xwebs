package handler

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
	"github.com/0funct0ry/xwebs/internal/template"
)

func init() {
	MustRegister(&SQLiteBuiltin{})
}

// SQLiteManagerImpl implements shared connection management for SQLite databases.
type SQLiteManagerImpl struct {
	mu  sync.Mutex
	dbs map[string]*sql.DB
	initialized map[string]bool
}

// NewSQLiteManager creates a new SQLite connection manager.
func NewSQLiteManager() SQLiteManager {
	return &SQLiteManagerImpl{
		dbs:         make(map[string]*sql.DB),
		initialized: make(map[string]bool),
	}
}

func (m *SQLiteManagerImpl) getDB(dbPath string) (*sql.DB, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if db, ok := m.dbs[dbPath]; ok {
		return db, nil
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "/" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("creating directory for database: %w", err)
			}
		}
	}

	// modernc.org/sqlite uses "sqlite" as driver name
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	m.dbs[dbPath] = db
	return db, nil
}

// Execute runs the SQL statement and returns results (for queries) and row count.
func (m *SQLiteManagerImpl) Execute(ctx context.Context, dbPath, query, initSQL string) ([]map[string]interface{}, int, error) {
	db, err := m.getDB(dbPath)
	if err != nil {
		return nil, 0, err
	}

	if initSQL != "" {
		m.mu.Lock()
		if !m.initialized[dbPath] {
			_, err := db.ExecContext(ctx, initSQL)
			if err != nil {
				m.mu.Unlock()
				return nil, 0, fmt.Errorf("initializing database: %w", err)
			}
			m.initialized[dbPath] = true
		}
		m.mu.Unlock()
	}

	q := strings.TrimSpace(query)
	upperQ := strings.ToUpper(q)
	
	// Heuristic to decide between Query and Exec
	isQuery := strings.HasPrefix(upperQ, "SELECT") || 
               strings.HasPrefix(upperQ, "WITH") || 
               strings.HasPrefix(upperQ, "PRAGMA") || 
               strings.Contains(upperQ, "RETURNING")

	if isQuery {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return nil, 0, err
		}

		var results []map[string]interface{}
		for rows.Next() {
			row := make(map[string]interface{})
			columnPointers := make([]interface{}, len(cols))
			for i := range columnPointers {
				var v interface{}
				columnPointers[i] = &v
			}

			if err := rows.Scan(columnPointers...); err != nil {
				return nil, 0, err
			}

			for i, colName := range cols {
				val := *(columnPointers[i].(*interface{}))
				// Handle byte slices (often returned by sqlite for strings)
				if b, ok := val.([]byte); ok {
					row[colName] = string(b)
				} else {
					row[colName] = val
				}
			}
			results = append(results, row)
		}
		return results, len(results), nil
	} else {
		res, err := db.ExecContext(ctx, query)
		if err != nil {
			return nil, 0, err
		}
		affected, _ := res.RowsAffected()
		return nil, int(affected), nil
	}
}

// Close closes all managed database connections.
func (m *SQLiteManagerImpl) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for path, db := range m.dbs {
		_ = db.Close()
		delete(m.dbs, path)
		delete(m.initialized, path)
	}
	return nil
}

// SQLiteBuiltin implements the 'sqlite' builtin action.
type SQLiteBuiltin struct{}

func (b *SQLiteBuiltin) Name() string { return "sqlite" }
func (b *SQLiteBuiltin) Description() string {
	return "Execute a parameterized SQL statement against an embedded SQLite database."
}
func (b *SQLiteBuiltin) Scope() BuiltinScope { return Shared }

func (b *SQLiteBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Execute a parameterized SQL statement against an embedded SQLite database.",
		Fields: []BuiltinField{
			{Name: "db", Type: "string", Required: true, Description: "Path to SQLite database file (supports templates)."},
			{Name: "sql", Type: "string", Required: true, Description: "SQL statement to execute (supports templates)."},
			{Name: "init", Type: "string", Required: false, Description: "Optional SQL to run once on first connection (e.g. DDL)."},
		},
		TemplateVars: map[string]string{
			".Rows":     "Slice of maps containing query results",
			".RowCount": "Number of rows affected or returned",
		},
		YAMLReplExample: "builtin: sqlite\ndb: 'messages.db'\ninit: 'CREATE TABLE IF NOT EXISTS msgs (data TEXT)'\nsql: 'INSERT INTO msgs (data) VALUES ({{.Message | quote}})'",
		REPLAddExample:  ":handler add -m '*' --builtin sqlite --db 'data.db' --sql 'SELECT count(*) as total FROM users' -R 'Total users: {{index .Rows 0 \"total\"}}'",
	}
}

func (b *SQLiteBuiltin) Validate(a Action) error {
	if a.DB == "" {
		return fmt.Errorf("builtin sqlite: missing 'db'")
	}
	if a.SQL == "" {
		return fmt.Errorf("builtin sqlite: missing 'sql'")
	}
	return nil
}

func (b *SQLiteBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.sqliteManager == nil {
		return fmt.Errorf("sqlite manager not available")
	}

	dbPath, err := d.templateEngine.Execute("sqlite-db", a.DB, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering db path template: %w", err)
	}
	
	// Resolve relative path
	if !filepath.IsAbs(dbPath) && a.BaseDir != "" {
		dbPath = filepath.Join(a.BaseDir, dbPath)
	}

	sqlQuery, err := d.templateEngine.Execute("sqlite-sql", a.SQL, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering sql template: %w", err)
	}

	initSQL := a.Init
	if initSQL != "" {
		rendered, err := d.templateEngine.Execute("sqlite-init", initSQL, tmplCtx)
		if err == nil {
			initSQL = rendered
		}
	}

	rows, count, err := d.sqliteManager.Execute(ctx, dbPath, sqlQuery, initSQL)
	if err != nil {
		return fmt.Errorf("executing sql: %w", err)
	}

	tmplCtx.Rows = rows
	tmplCtx.RowCount = count

	if d.verbose {
		d.log("  [builtin:sqlite] executed on %s: %d rows affected/returned\n", a.DB, count)
	}

	return nil
}
