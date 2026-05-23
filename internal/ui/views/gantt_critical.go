package views

import (
	"sort"
	"time"

	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

// ComputeCriticalPath returns the set of task IDs on the longest-duration path
// from any source to any sink. O(V + E). Deterministic tie-break: on equal
// upstream candidates, the smaller task_id wins.
func ComputeCriticalPath(tasks []models.Task, tis []models.TaskInstance, now time.Time) map[string]bool {
	if len(tasks) == 0 {
		return map[string]bool{}
	}

	durByID := make(map[string]time.Duration, len(tis))
	for _, ti := range tis {
		durByID[ti.TaskId] = effectiveDuration(ti, now)
	}

	tByID := make(map[string]models.Task, len(tasks))
	upstream := make(map[string][]string, len(tasks))
	for _, t := range tasks {
		tByID[t.TaskId] = t
		sorted := append([]string(nil), t.UpstreamTaskIds...)
		sort.Strings(sorted)
		upstream[t.TaskId] = sorted
	}

	// Topological order (Kahn's algorithm variant) — process nodes whose upstreams
	// are all visited, sorted by id for determinism.
	visited := make(map[string]bool, len(tasks))
	order := make([]string, 0, len(tasks))
	for len(order) < len(tasks) {
		ids := make([]string, 0)
		for id := range tByID {
			if visited[id] {
				continue
			}
			ready := true
			for _, u := range upstream[id] {
				if !visited[u] {
					ready = false
					break
				}
			}
			if ready {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			// cycle (should never happen for Airflow DAGs)
			break
		}
		sort.Strings(ids)
		for _, id := range ids {
			visited[id] = true
			order = append(order, id)
		}
	}

	// DP: dp[v] = duration[v] + max over upstream of dp[u].
	// parent[v] = chosen upstream (smaller id on tie).
	dp := make(map[string]time.Duration, len(tasks))
	parent := make(map[string]string, len(tasks))
	for _, id := range order {
		best := time.Duration(-1)
		bestUp := ""
		for _, u := range upstream[id] {
			if dp[u] > best || (dp[u] == best && (bestUp == "" || u < bestUp)) {
				best = dp[u]
				bestUp = u
			}
		}
		dp[id] = durByID[id]
		if best > 0 {
			dp[id] += best
		}
		if bestUp != "" {
			parent[id] = bestUp
		}
	}

	// Find sink with max dp; tie-break smaller id.
	var sink string
	var sinkDP time.Duration
	ids := make([]string, 0, len(dp))
	for id := range dp {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if dp[id] > sinkDP || (dp[id] == sinkDP && (sink == "" || id < sink)) {
			sinkDP = dp[id]
			sink = id
		}
	}

	// Backtrace.
	result := map[string]bool{}
	cur := sink
	for cur != "" {
		result[cur] = true
		next, ok := parent[cur]
		if !ok {
			break
		}
		cur = next
	}
	return result
}
