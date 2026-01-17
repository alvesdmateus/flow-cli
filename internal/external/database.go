package external

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Note: To use this package, import the appropriate database driver in your main package:
// For SQLite:    import _ "github.com/mattn/go-sqlite3"
// For PostgreSQL: import _ "github.com/lib/pq"
// For MySQL:      import _ "github.com/go-sql-driver/mysql"

// DatabaseType represents a database type.
type DatabaseType string

const (
	DatabaseSQLite   DatabaseType = "sqlite"
	DatabasePostgres DatabaseType = "postgres"
	DatabaseMySQL    DatabaseType = "mysql"
)

// DatabaseClient provides database operations.
type DatabaseClient struct {
	db       *sql.DB
	dbType   DatabaseType
	readOnly bool
	timeout  time.Duration
}

// DatabaseConfig configures the database client.
type DatabaseConfig struct {
	Type            DatabaseType
	ConnectionString string
	ReadOnly        bool
	Timeout         time.Duration
	MaxConnections  int
}

// QueryResult represents the result of a query.
type QueryResult struct {
	Columns      []string        `json:"columns"`
	Rows         [][]interface{} `json:"rows"`
	RowsAffected int64           `json:"rows_affected,omitempty"`
	LastInsertID int64           `json:"last_insert_id,omitempty"`
}

// TableInfo contains information about a table.
type TableInfo struct {
	Name       string       `json:"name"`
	Schema     string       `json:"schema,omitempty"`
	Columns    []ColumnInfo `json:"columns"`
	PrimaryKey []string     `json:"primary_key,omitempty"`
	RowCount   int64        `json:"row_count,omitempty"`
}

// ColumnInfo contains information about a column.
type ColumnInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	Default    string `json:"default,omitempty"`
	PrimaryKey bool   `json:"primary_key"`
}

// IndexInfo contains information about an index.
type IndexInfo struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// NewDatabaseClient creates a new database client.
func NewDatabaseClient(config DatabaseConfig) (*DatabaseClient, error) {
	var driverName string
	switch config.Type {
	case DatabaseSQLite:
		driverName = "sqlite3"
	case DatabasePostgres:
		driverName = "postgres"
	case DatabaseMySQL:
		driverName = "mysql"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	db, err := sql.Open(driverName, config.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	if config.MaxConnections > 0 {
		db.SetMaxOpenConns(config.MaxConnections)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &DatabaseClient{
		db:       db,
		dbType:   config.Type,
		readOnly: config.ReadOnly,
		timeout:  timeout,
	}, nil
}

// Close closes the database connection.
func (dc *DatabaseClient) Close() error {
	return dc.db.Close()
}

// Query executes a SELECT query and returns results.
func (dc *DatabaseClient) Query(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	// Validate query is read-only if in read-only mode
	if dc.readOnly && !isReadOnlyQuery(query) {
		return nil, fmt.Errorf("only SELECT queries are allowed in read-only mode")
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	rows, err := dc.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan results
	result := &QueryResult{
		Columns: columns,
		Rows:    make([][]interface{}, 0),
	}

	for rows.Next() {
		// Create slice for row values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert values to JSON-safe types
		rowData := make([]interface{}, len(columns))
		for i, v := range values {
			rowData[i] = convertToJSONSafe(v)
		}
		result.Rows = append(result.Rows, rowData)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// Execute executes a non-SELECT query (INSERT, UPDATE, DELETE).
func (dc *DatabaseClient) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	if dc.readOnly {
		return nil, fmt.Errorf("write operations not allowed in read-only mode")
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	result, err := dc.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("execute failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return &QueryResult{
		RowsAffected: rowsAffected,
		LastInsertID: lastInsertID,
	}, nil
}

// ListTables returns a list of tables in the database.
func (dc *DatabaseClient) ListTables(ctx context.Context) ([]string, error) {
	var query string
	switch dc.dbType {
	case DatabaseSQLite:
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	case DatabasePostgres:
		query = "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename"
	case DatabaseMySQL:
		query = "SHOW TABLES"
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dc.dbType)
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	rows, err := dc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// DescribeTable returns information about a table.
func (dc *DatabaseClient) DescribeTable(ctx context.Context, tableName string) (*TableInfo, error) {
	info := &TableInfo{
		Name:    tableName,
		Columns: make([]ColumnInfo, 0),
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	var err error
	switch dc.dbType {
	case DatabaseSQLite:
		err = dc.describeSQLiteTable(ctx, tableName, info)
	case DatabasePostgres:
		err = dc.describePostgresTable(ctx, tableName, info)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dc.dbType)
	}

	if err != nil {
		return nil, err
	}

	// Get row count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	row := dc.db.QueryRowContext(ctx, countQuery)
	row.Scan(&info.RowCount)

	return info, nil
}

func (dc *DatabaseClient) describeSQLiteTable(ctx context.Context, tableName string, info *TableInfo) error {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := dc.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}

		col := ColumnInfo{
			Name:       name,
			Type:       colType,
			Nullable:   notNull == 0,
			PrimaryKey: pk > 0,
		}
		if dfltValue.Valid {
			col.Default = dfltValue.String
		}

		info.Columns = append(info.Columns, col)
		if pk > 0 {
			info.PrimaryKey = append(info.PrimaryKey, name)
		}
	}

	return nil
}

func (dc *DatabaseClient) describePostgresTable(ctx context.Context, tableName string, info *TableInfo) error {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT kcu.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
			WHERE tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY'
		) pk ON c.column_name = pk.column_name
		WHERE c.table_name = $1
		ORDER BY c.ordinal_position
	`

	rows, err := dc.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return fmt.Errorf("failed to describe table: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, colType, nullable string
		var defaultVal sql.NullString
		var isPrimary bool

		if err := rows.Scan(&name, &colType, &nullable, &defaultVal, &isPrimary); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}

		col := ColumnInfo{
			Name:       name,
			Type:       colType,
			Nullable:   nullable == "YES",
			PrimaryKey: isPrimary,
		}
		if defaultVal.Valid {
			col.Default = defaultVal.String
		}

		info.Columns = append(info.Columns, col)
		if isPrimary {
			info.PrimaryKey = append(info.PrimaryKey, name)
		}
	}

	return nil
}

// ListIndexes returns indexes for a table.
func (dc *DatabaseClient) ListIndexes(ctx context.Context, tableName string) ([]IndexInfo, error) {
	var query string
	switch dc.dbType {
	case DatabaseSQLite:
		query = fmt.Sprintf("PRAGMA index_list(%s)", tableName)
	case DatabasePostgres:
		query = `
			SELECT indexname, indexdef
			FROM pg_indexes
			WHERE tablename = $1
		`
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dc.dbType)
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	var indexes []IndexInfo

	if dc.dbType == DatabaseSQLite {
		rows, err := dc.db.QueryContext(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to list indexes: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var seq int
			var name string
			var unique int
			var origin, partial string

			if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
				continue
			}

			idx := IndexInfo{
				Name:   name,
				Table:  tableName,
				Unique: unique == 1,
			}

			// Get columns for this index
			colQuery := fmt.Sprintf("PRAGMA index_info(%s)", name)
			colRows, err := dc.db.QueryContext(ctx, colQuery)
			if err == nil {
				for colRows.Next() {
					var seqno, cid int
					var colName string
					if colRows.Scan(&seqno, &cid, &colName) == nil {
						idx.Columns = append(idx.Columns, colName)
					}
				}
				colRows.Close()
			}

			indexes = append(indexes, idx)
		}
	} else if dc.dbType == DatabasePostgres {
		rows, err := dc.db.QueryContext(ctx, query, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to list indexes: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var name, indexDef string
			if err := rows.Scan(&name, &indexDef); err != nil {
				continue
			}

			idx := IndexInfo{
				Name:   name,
				Table:  tableName,
				Unique: strings.Contains(indexDef, "UNIQUE"),
			}
			indexes = append(indexes, idx)
		}
	}

	return indexes, nil
}

// Transaction executes multiple queries in a transaction.
func (dc *DatabaseClient) Transaction(ctx context.Context, queries []string) error {
	if dc.readOnly {
		return fmt.Errorf("transactions not allowed in read-only mode")
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	tx, err := dc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			tx.Rollback()
			return fmt.Errorf("transaction failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Explain returns the query execution plan.
func (dc *DatabaseClient) Explain(ctx context.Context, query string) (string, error) {
	var explainQuery string
	switch dc.dbType {
	case DatabaseSQLite:
		explainQuery = "EXPLAIN QUERY PLAN " + query
	case DatabasePostgres:
		explainQuery = "EXPLAIN ANALYZE " + query
	default:
		return "", fmt.Errorf("unsupported database type: %s", dc.dbType)
	}

	ctx, cancel := context.WithTimeout(ctx, dc.timeout)
	defer cancel()

	rows, err := dc.db.QueryContext(ctx, explainQuery)
	if err != nil {
		return "", fmt.Errorf("explain failed: %w", err)
	}
	defer rows.Close()

	var lines []string
	cols, _ := rows.Columns()
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		var parts []string
		for _, v := range values {
			parts = append(parts, fmt.Sprintf("%v", v))
		}
		lines = append(lines, strings.Join(parts, " | "))
	}

	return strings.Join(lines, "\n"), nil
}

// ToJSON converts query result to JSON.
func (qr *QueryResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(qr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToMap converts each row to a map.
func (qr *QueryResult) ToMaps() []map[string]interface{} {
	var results []map[string]interface{}
	for _, row := range qr.Rows {
		rowMap := make(map[string]interface{})
		for i, col := range qr.Columns {
			if i < len(row) {
				rowMap[col] = row[i]
			}
		}
		results = append(results, rowMap)
	}
	return results
}

// Helper functions

func isReadOnlyQuery(query string) bool {
	query = strings.TrimSpace(strings.ToUpper(query))
	readOnlyPrefixes := []string{"SELECT", "SHOW", "DESCRIBE", "EXPLAIN", "PRAGMA"}
	for _, prefix := range readOnlyPrefixes {
		if strings.HasPrefix(query, prefix) {
			return true
		}
	}
	return false
}

func convertToJSONSafe(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return val
	}
}
