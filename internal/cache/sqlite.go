package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS backfills (
  dag_id TEXT NOT NULL,
  id INTEGER NOT NULL,
  state TEXT NOT NULL,
  from_date TEXT,
  to_date TEXT,
  completed_runs INTEGER NOT NULL DEFAULT 0,
  failed_runs INTEGER NOT NULL DEFAULT 0,
  running_runs INTEGER NOT NULL DEFAULT 0,
  total_runs INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  raw_json TEXT NOT NULL,
  PRIMARY KEY (dag_id, id)
);

CREATE TABLE IF NOT EXISTS dag_runs (
  dag_id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  state TEXT NOT NULL,
  logical_date TEXT,
  run_after TEXT,
  start_date TEXT,
  end_date TEXT,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  run_type TEXT,
  note TEXT,
  updated_at TEXT NOT NULL,
  raw_json TEXT NOT NULL,
  PRIMARY KEY (dag_id, run_id)
);

CREATE TABLE IF NOT EXISTS task_instances (
  dag_id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  task_id TEXT NOT NULL,
  state TEXT NOT NULL,
  start_date TEXT,
  end_date TEXT,
  queued_at TEXT,
  duration_ms INTEGER NOT NULL DEFAULT 0,
  queue_ms INTEGER NOT NULL DEFAULT 0,
  try_number INTEGER NOT NULL DEFAULT 0,
  operator TEXT,
  pool TEXT,
  queue TEXT,
  hostname TEXT,
  updated_at TEXT NOT NULL,
  raw_json TEXT NOT NULL,
  PRIMARY KEY (dag_id, run_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_dag_runs_dag_time
  ON dag_runs (dag_id, run_after DESC);

CREATE INDEX IF NOT EXISTS idx_dag_runs_time
  ON dag_runs (run_after DESC);

CREATE INDEX IF NOT EXISTS idx_task_instances_dag_time
  ON task_instances (dag_id, start_date DESC);

CREATE INDEX IF NOT EXISTS idx_task_instances_run
  ON task_instances (dag_id, run_id);

INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (1, datetime('now'));
`

type writeKind int

const (
	writeBackfills writeKind = iota
	writeDAGRuns
	writeTaskInstances
)

type writeOp struct {
	kind      writeKind
	dagId     string
	runId     string
	backfills []models.Backfill
	dagRuns   []models.DAGRun
	tasks     []models.TaskInstance
}

type sqliteCache struct {
	db          *sql.DB
	writes      chan writeOp
	done        chan struct{}
	retention   time.Duration
	lastCleanup time.Time

	closeMu sync.Mutex
	closed  bool
}

func NewSQLite(path string, opts Options) (Cache, error) {
	if opts.Retention <= 0 {
		opts.Retention = 30 * 24 * time.Hour
	}
	if opts.WriteBuffer <= 0 {
		opts.WriteBuffer = 256
	}
	expanded, err := expandPath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(expanded), 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	db, err := sql.Open("sqlite", expanded)
	if err != nil {
		return nil, fmt.Errorf("open sqlite cache: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	c := &sqliteCache{
		db:          db,
		writes:      make(chan writeOp, opts.WriteBuffer),
		done:        make(chan struct{}),
		retention:   opts.Retention,
		lastCleanup: time.Now(),
	}
	if err := c.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	go c.writeLoop()
	return c, nil
}

func (c *sqliteCache) init() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
	}
	for _, stmt := range pragmas {
		if _, err := c.db.Exec(stmt); err != nil {
			return fmt.Errorf("sqlite pragma %q: %w", stmt, err)
		}
	}
	if _, err := c.db.Exec(sqliteSchema); err != nil {
		return fmt.Errorf("sqlite schema: %w", err)
	}
	return c.cleanup(context.Background(), time.Now())
}

func (c *sqliteCache) GetBackfills(dagId string) ([]models.Backfill, bool) {
	rows, err := c.db.Query(`
SELECT id, dag_id, from_date, to_date, completed_runs, failed_runs, running_runs, total_runs, raw_json
FROM backfills
WHERE dag_id = ?
ORDER BY id DESC`, dagId)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	out := make([]models.Backfill, 0)
	for rows.Next() {
		var (
			b                                 models.Backfill
			raw                               string
			fromDate, toDate                  sql.NullString
			completed, failed, running, total int
		)
		if err := rows.Scan(&b.ID, &b.DagId, &fromDate, &toDate, &completed, &failed, &running, &total, &raw); err != nil {
			return nil, false
		}
		_ = json.Unmarshal([]byte(raw), &b)
		if b.DagId == "" {
			b.DagId = dagId
		}
		if b.FromDate.IsZero() {
			b.FromDate = parseTime(fromDate)
		}
		if b.ToDate.IsZero() {
			b.ToDate = parseTime(toDate)
		}
		b.CompletedRuns = completed
		b.FailedRuns = failed
		b.RunningRuns = running
		b.TotalRuns = total
		out = append(out, b)
	}
	if len(out) == 0 || rows.Err() != nil {
		return nil, false
	}
	return out, true
}

func (c *sqliteCache) PutBackfills(dagId string, bfs []models.Backfill) {
	dup := make([]models.Backfill, len(bfs))
	copy(dup, bfs)
	c.enqueue(writeOp{kind: writeBackfills, dagId: dagId, backfills: dup})
}

func (c *sqliteCache) PutDAGRuns(dagId string, runs []models.DAGRun) {
	dup := make([]models.DAGRun, len(runs))
	copy(dup, runs)
	c.enqueue(writeOp{kind: writeDAGRuns, dagId: dagId, dagRuns: dup})
}

func (c *sqliteCache) GetDAGRunsHistory(dagId string, since time.Time, limit int) ([]models.DAGRun, bool) {
	args := []any{dagId, formatTime(since)}
	q := `
SELECT dag_id, run_id, state, logical_date, run_after, start_date, end_date, run_type, note
FROM dag_runs
WHERE dag_id = ? AND run_after >= ?
ORDER BY run_after DESC`
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	return c.queryDAGRuns(q, args...)
}

func (c *sqliteCache) GetAllDAGRunsHistory(since time.Time, limit int) ([]models.DAGRun, bool) {
	args := []any{formatTime(since)}
	q := `
SELECT dag_id, run_id, state, logical_date, run_after, start_date, end_date, run_type, note
FROM dag_runs
WHERE run_after >= ?
ORDER BY run_after DESC`
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	return c.queryDAGRuns(q, args...)
}

func (c *sqliteCache) PutTaskInstances(dagId, runId string, tasks []models.TaskInstance) {
	dup := make([]models.TaskInstance, len(tasks))
	copy(dup, tasks)
	c.enqueue(writeOp{kind: writeTaskInstances, dagId: dagId, runId: runId, tasks: dup})
}

func (c *sqliteCache) GetTaskInstancesHistory(dagId string, since time.Time, limit int) ([]models.TaskInstance, bool) {
	args := []any{dagId, formatTime(since)}
	q := `
SELECT dag_id, run_id, task_id, state, start_date, end_date, queued_at, duration_ms, try_number, operator, pool, queue, hostname
FROM task_instances
WHERE dag_id = ? AND COALESCE(start_date, updated_at) >= ?
ORDER BY COALESCE(start_date, updated_at) DESC`
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := c.db.Query(q, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	out := make([]models.TaskInstance, 0)
	for rows.Next() {
		var (
			ti                              models.TaskInstance
			start, end, queued              sql.NullString
			durationMS                      int64
			operator, pool, queue, hostname sql.NullString
		)
		if err := rows.Scan(&ti.DagId, &ti.RunId, &ti.TaskId, &ti.State, &start, &end, &queued, &durationMS, &ti.TryNumber, &operator, &pool, &queue, &hostname); err != nil {
			return nil, false
		}
		ti.StartDate = parseTimePtr(start)
		ti.EndDate = parseTimePtr(end)
		ti.QueuedDttm = parseTimePtr(queued)
		ti.Duration = time.Duration(durationMS * int64(time.Millisecond)).Seconds()
		ti.Operator = operator.String
		ti.Pool = pool.String
		ti.Queue = queue.String
		ti.Hostname = hostname.String
		out = append(out, ti)
	}
	if len(out) == 0 || rows.Err() != nil {
		return nil, false
	}
	return out, true
}

func (c *sqliteCache) GetDagDashboardRows(since time.Time, limit int) ([]DagDashboardRow, bool) {
	args := []any{formatTime(since), formatTime(since), formatTime(since)}
	q := `
WITH latest AS (
  SELECT dag_id, state, run_after,
         ROW_NUMBER() OVER (PARTITION BY dag_id ORDER BY run_after DESC) AS rn
  FROM dag_runs
  WHERE run_after >= ?
),
run_agg AS (
  SELECT dag_id,
         COUNT(*) AS runs,
         SUM(CASE WHEN state = 'success' THEN 1 ELSE 0 END) AS success,
         SUM(CASE WHEN state = 'failed' THEN 1 ELSE 0 END) AS failed,
         SUM(CASE WHEN state = 'running' THEN 1 ELSE 0 END) AS running,
         SUM(CASE WHEN state = 'queued' THEN 1 ELSE 0 END) AS queued,
         AVG(NULLIF(duration_ms, 0)) AS avg_duration_ms,
         MAX(duration_ms) AS max_duration_ms
  FROM dag_runs
  WHERE run_after >= ?
  GROUP BY dag_id
),
task_agg AS (
  SELECT dag_id,
         AVG(NULLIF(queue_ms, 0)) AS avg_queue_ms,
         SUM(CASE WHEN state = 'failed' THEN 1 ELSE 0 END) AS failed_tasks,
         SUM(CASE WHEN try_number > 1 THEN 1 ELSE 0 END) AS retried_tasks
  FROM task_instances
  WHERE COALESCE(start_date, updated_at) >= ?
  GROUP BY dag_id
)
SELECT r.dag_id, r.runs, r.success, r.failed, r.running, r.queued,
       COALESCE(l.state, ''), COALESCE(l.run_after, ''),
       COALESCE(r.avg_duration_ms, 0), COALESCE(r.max_duration_ms, 0),
       COALESCE(t.avg_queue_ms, 0), COALESCE(t.failed_tasks, 0), COALESCE(t.retried_tasks, 0)
FROM run_agg r
LEFT JOIN latest l ON l.dag_id = r.dag_id AND l.rn = 1
LEFT JOIN task_agg t ON t.dag_id = r.dag_id
ORDER BY r.failed DESC, r.dag_id ASC`
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := c.db.Query(q, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	out := make([]DagDashboardRow, 0)
	for rows.Next() {
		var (
			row                                      DagDashboardRow
			lastRunAfter                             string
			avgDurationMS, maxDurationMS, avgQueueMS float64
		)
		if err := rows.Scan(&row.DagId, &row.Runs, &row.Success, &row.Failed, &row.Running, &row.Queued, &row.LastState, &lastRunAfter, &avgDurationMS, &maxDurationMS, &avgQueueMS, &row.FailedTasks, &row.RetriedTasks); err != nil {
			return nil, false
		}
		row.LastRunAfter = parseTime(sql.NullString{String: lastRunAfter, Valid: lastRunAfter != ""})
		row.AvgDuration = time.Duration(avgDurationMS) * time.Millisecond
		row.MaxDuration = time.Duration(maxDurationMS) * time.Millisecond
		row.AvgQueueTime = time.Duration(avgQueueMS) * time.Millisecond
		out = append(out, row)
	}
	if len(out) == 0 || rows.Err() != nil {
		return nil, false
	}
	return out, true
}

func (c *sqliteCache) Close() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closed = true
	close(c.writes)
	c.closeMu.Unlock()

	<-c.done
	return c.db.Close()
}

func (c *sqliteCache) enqueue(op writeOp) {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return
	}
	defer c.closeMu.Unlock()

	select {
	case c.writes <- op:
	default:
		select {
		case <-c.writes:
		default:
		}
		c.writes <- op
	}
}

func (c *sqliteCache) writeLoop() {
	defer close(c.done)
	for op := range c.writes {
		_ = c.applyWrite(context.Background(), op)
		now := time.Now()
		if now.Sub(c.lastCleanup) >= 24*time.Hour {
			_ = c.cleanup(context.Background(), now)
			c.lastCleanup = now
		}
	}
}

func (c *sqliteCache) applyWrite(ctx context.Context, op writeOp) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var applyErr error
	switch op.kind {
	case writeBackfills:
		applyErr = insertBackfills(ctx, tx, op.dagId, op.backfills)
	case writeDAGRuns:
		applyErr = insertDAGRuns(ctx, tx, op.dagId, op.dagRuns)
	case writeTaskInstances:
		applyErr = insertTaskInstances(ctx, tx, op.dagId, op.runId, op.tasks)
	default:
		applyErr = errors.New("unknown cache write op")
	}
	if applyErr != nil {
		_ = tx.Rollback()
		return applyErr
	}
	return tx.Commit()
}

func insertBackfills(ctx context.Context, tx *sql.Tx, dagId string, bfs []models.Backfill) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO backfills (
  dag_id, id, state, from_date, to_date, completed_runs, failed_runs, running_runs, total_runs, updated_at, raw_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(dag_id, id) DO UPDATE SET
  state=excluded.state, from_date=excluded.from_date, to_date=excluded.to_date,
  completed_runs=excluded.completed_runs, failed_runs=excluded.failed_runs,
  running_runs=excluded.running_runs, total_runs=excluded.total_runs,
  updated_at=excluded.updated_at, raw_json=excluded.raw_json`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, bf := range bfs {
		id := dagId
		if bf.DagId != "" {
			id = bf.DagId
		}
		raw, _ := json.Marshal(bf)
		if _, err := stmt.ExecContext(ctx, id, bf.ID, bf.State(), formatTime(bf.FromDate), formatTime(bf.ToDate), bf.CompletedRuns, bf.FailedRuns, bf.RunningRuns, bf.TotalRuns, formatTime(time.Now()), string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func insertDAGRuns(ctx context.Context, tx *sql.Tx, dagId string, runs []models.DAGRun) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO dag_runs (
  dag_id, run_id, state, logical_date, run_after, start_date, end_date, duration_ms, run_type, note, updated_at, raw_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(dag_id, run_id) DO UPDATE SET
  state=excluded.state, logical_date=excluded.logical_date, run_after=excluded.run_after,
  start_date=excluded.start_date, end_date=excluded.end_date, duration_ms=excluded.duration_ms,
  run_type=excluded.run_type, note=excluded.note, updated_at=excluded.updated_at, raw_json=excluded.raw_json`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, run := range runs {
		id := dagId
		if run.DagId != "" {
			id = run.DagId
		}
		ts := runTime(run)
		raw, _ := json.Marshal(run)
		if _, err := stmt.ExecContext(ctx, id, run.RunId, run.State, formatTime(run.LogicalDate), formatTime(ts), formatTime(run.StartDate), formatTime(run.EndDate), run.Duration().Milliseconds(), run.RunType, run.Note, formatTime(time.Now()), string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func insertTaskInstances(ctx context.Context, tx *sql.Tx, dagId, runId string, tasks []models.TaskInstance) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO task_instances (
  dag_id, run_id, task_id, state, start_date, end_date, queued_at, duration_ms, queue_ms,
  try_number, operator, pool, queue, hostname, updated_at, raw_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(dag_id, run_id, task_id) DO UPDATE SET
  state=excluded.state, start_date=excluded.start_date, end_date=excluded.end_date,
  queued_at=excluded.queued_at, duration_ms=excluded.duration_ms, queue_ms=excluded.queue_ms,
  try_number=excluded.try_number, operator=excluded.operator, pool=excluded.pool,
  queue=excluded.queue, hostname=excluded.hostname, updated_at=excluded.updated_at,
  raw_json=excluded.raw_json`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, ti := range tasks {
		did := dagId
		if ti.DagId != "" {
			did = ti.DagId
		}
		rid := runId
		if ti.RunId != "" {
			rid = ti.RunId
		}
		raw, _ := json.Marshal(ti)
		if _, err := stmt.ExecContext(ctx, did, rid, ti.TaskId, ti.State, formatTimePtr(ti.StartDate), formatTimePtr(ti.EndDate), formatTimePtr(ti.QueuedDttm), taskDuration(ti).Milliseconds(), queueDuration(ti).Milliseconds(), ti.TryNumber, ti.Operator, ti.Pool, ti.Queue, ti.Hostname, formatTime(time.Now()), string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (c *sqliteCache) queryDAGRuns(q string, args ...any) ([]models.DAGRun, bool) {
	rows, err := c.db.Query(q, args...)
	if err != nil {
		return nil, false
	}
	defer rows.Close()
	out := make([]models.DAGRun, 0)
	for rows.Next() {
		var (
			r                                         models.DAGRun
			logicalDate, runAfter, startDate, endDate sql.NullString
		)
		if err := rows.Scan(&r.DagId, &r.RunId, &r.State, &logicalDate, &runAfter, &startDate, &endDate, &r.RunType, &r.Note); err != nil {
			return nil, false
		}
		r.LogicalDate = parseTime(logicalDate)
		r.RunAfter = parseTime(runAfter)
		r.StartDate = parseTime(startDate)
		r.EndDate = parseTime(endDate)
		out = append(out, r)
	}
	if len(out) == 0 || rows.Err() != nil {
		return nil, false
	}
	return out, true
}

func (c *sqliteCache) cleanup(ctx context.Context, now time.Time) error {
	cutoff := formatTime(now.Add(-c.retention))
	for _, stmt := range []string{
		"DELETE FROM dag_runs WHERE run_after < ?",
		"DELETE FROM task_instances WHERE COALESCE(start_date, updated_at) < ?",
		"DELETE FROM backfills WHERE updated_at < ?",
	} {
		if _, err := c.db.ExecContext(ctx, stmt, cutoff); err != nil {
			return err
		}
	}
	return nil
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty sqlite cache path")
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if len(path) > 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func parseTime(v sql.NullString) time.Time {
	if !v.Valid || v.String == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, v.String)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseTimePtr(v sql.NullString) *time.Time {
	t := parseTime(v)
	if t.IsZero() {
		return nil
	}
	return &t
}

func taskDuration(ti models.TaskInstance) time.Duration {
	if ti.Duration > 0 {
		return time.Duration(ti.Duration * float64(time.Second))
	}
	if ti.StartDate != nil && ti.EndDate != nil && ti.EndDate.After(*ti.StartDate) {
		return ti.EndDate.Sub(*ti.StartDate)
	}
	return 0
}
