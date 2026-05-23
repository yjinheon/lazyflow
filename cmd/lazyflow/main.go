package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/internal/api"
	"github.com/yjinheon/lazyflow/internal/app"
	"github.com/yjinheon/lazyflow/internal/cache"
	"github.com/yjinheon/lazyflow/internal/debugutil"
	"github.com/yjinheon/lazyflow/internal/state"
	ui "github.com/yjinheon/lazyflow/internal/ui"
	"github.com/yjinheon/lazyflow/internal/ui/layout"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/internal/ui/views"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func main() {
	// Debug log to file with microsecond resolution so we can correlate
	// freezes with the last log line emitted before the UI stopped responding.
	logFile, _ := os.Create("lazyflow.log")
	if logFile != nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	debugutil.Setup()
	debugutil.StartRuntimeSampler(2 * time.Second)
	debugutil.Tag("FZ-INIT", "lazyflow starting pid=%d", os.Getpid())

	// Created up-front so the watchdog (started below) can ping it.
	tviewApp := tview.NewApplication()
	debugutil.StartWatchdog("tview-main", 2*time.Second, 3*time.Second, func() <-chan struct{} {
		done := make(chan struct{}, 1)
		// tview.Application.QueueUpdate blocks the caller until the queued
		// func runs on main. Spawn the enqueue in a goroutine so a wedged
		// main loop does not also wedge the watchdog itself.
		go tviewApp.QueueUpdate(func() {
			select {
			case done <- struct{}{}:
			default:
			}
		})
		return done
	})

	dispatcher := app.NewDispatcher(256)
	go dispatcher.Start(context.Background(), tviewApp)

	bfCache := cache.NewMemory(30 * time.Second)
	defer bfCache.Close()

	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	theme.ApplyTheme(theme.DefaultDarkTheme)
	mainLayout := layout.NewMainLayout(tviewApp)
	store := state.NewStore()

	client := api.NewClient(api.ClientConfig{
		BaseURL:  cfg.Airflow.BaseURL,
		Username: cfg.Airflow.Auth.Username,
		Password: cfg.Airflow.Auth.Password,
		Token:    cfg.Airflow.Auth.Token,
		AuthType: cfg.Airflow.Auth.Type,
		Timeout:  app.ParseDuration(cfg.Airflow.Timeout, 30*time.Second),
	})

	poller := app.NewPoller(context.Background())
	defer poller.Stop()

	// ---------- Event wiring ----------

	// DAGs updated → refresh DAG list + header count
	store.Subscribe(state.EventDAGsUpdated, func(_ any) {
		dispatcher.Post(func() {
			dags := store.GetDAGs()
			mainLayout.DagList().Update(dags)
			mainLayout.Header().SetInfo(cfg.Airflow.BaseURL, true, len(dags))
		})
	})

	// Health updated → refresh cluster panel + monitor
	store.Subscribe(state.EventHealthUpdated, func(_ any) {
		dispatcher.Post(func() {
			h := store.GetHealth()
			mainLayout.ClusterInfo().Update(h)
			mainLayout.Monitor().Update(h)
		})
	})

	// Selection → update status bar
	// NOTE: These subscribers are called synchronously from tview's main goroutine
	// (via SetSelectedFunc → store.Select*). dispatcher.Post is non-blocking so it
	// is safe to call directly from the tview main goroutine — no deadlock risk.
	store.Subscribe(state.EventDAGSelected, func(_ any) {
		dispatcher.Post(func() {
			mainLayout.StatusBar().SetInfo(store.SelectedDAG(), "", "")
		})
	})
	store.Subscribe(state.EventRunSelected, func(_ any) {
		dispatcher.Post(func() {
			mainLayout.StatusBar().SetInfo(store.SelectedDAG(), store.SelectedRun(), "")
		})
	})
	store.Subscribe(state.EventTaskSelected, func(_ any) {
		dispatcher.Post(func() {
			mainLayout.StatusBar().SetInfo(store.SelectedDAG(), store.SelectedRun(), store.SelectedTask())
		})
	})

	// DAG runs updated → refresh runs view
	store.Subscribe(state.EventDAGRunsUpdated, func(_ any) {
		dispatcher.Post(func() {
			runs := store.GetDAGRuns(store.SelectedDAG())
			mainLayout.Runs().Update(runs)
		})
	})

	// Task instances updated → refresh tasks view (table or gantt)
	// and recompute critical path.
	store.Subscribe(state.EventTaskInstancesUpdated, func(_ any) {
		dispatcher.Post(func() {
			tis := store.GetTaskInstances(store.SelectedDAG(), store.SelectedRun())
			if store.GanttMode() {
				mainLayout.Tasks().UpdateGantt(store.SelectedRun(), tis, store.GetCriticalPath())
			} else {
				mainLayout.Tasks().Update(tis)
			}
		})
		// Critical-path recompute off the UI goroutine.
		go func() {
			tasks := store.GetTasks(store.SelectedDAG())
			tis := store.GetTaskInstances(store.SelectedDAG(), store.SelectedRun())
			cp := views.ComputeCriticalPath(tasks, tis, time.Now())
			store.SetCriticalPath(cp) // notifies only if changed
		}()
	})

	// Lineage updated → refresh lineage view
	store.Subscribe(state.EventLineageUpdated, func(_ any) {
		dispatcher.Post(func() {
			tasks := store.GetTasks(store.SelectedDAG())
			mainLayout.Lineage().SetTasks(store.SelectedDAG(), tasks)
		})
	})

	// Critical-path changed → if gantt active, re-render
	store.Subscribe(state.EventCriticalPathChanged, func(_ any) {
		dispatcher.Post(func() {
			if !store.GanttMode() {
				return
			}
			tis := store.GetTaskInstances(store.SelectedDAG(), store.SelectedRun())
			mainLayout.Tasks().UpdateGantt(store.SelectedRun(), tis, store.GetCriticalPath())
		})
	})

	// Gantt mode toggled → switch page; if entering gantt, render now.
	store.Subscribe(state.EventGanttModeChanged, func(_ any) {
		dispatcher.Post(func() {
			on := store.GanttMode()
			mainLayout.Tasks().SetGanttMode(on)
			if on {
				tis := store.GetTaskInstances(store.SelectedDAG(), store.SelectedRun())
				mainLayout.Tasks().UpdateGantt(store.SelectedRun(), tis, store.GetCriticalPath())
			}
		})
	})

	// Backfills list updated → refresh list view
	store.Subscribe(state.EventBackfillsUpdated, func(_ any) {
		dispatcher.Post(func() {
			bfs := store.GetBackfills(store.SelectedDAG())
			mainLayout.Backfills().UpdateList(bfs)
		})
	})

	// Backfill selected → refresh detail pane
	store.Subscribe(state.EventBackfillSelected, func(_ any) {
		dispatcher.Post(func() {
			id := store.SelectedBackfill()
			bfs := store.GetBackfills(store.SelectedDAG())
			var bf *models.Backfill
			for i := range bfs {
				if bfs[i].ID == id {
					bf = &bfs[i]
					break
				}
			}
			mainLayout.Backfills().UpdateDetail(bf)
		})
	})

	// DAG selected → info panel, fetch runs + lineage + code
	mainLayout.DagList().SetOnSelected(func(dagId string) {
		debugutil.Tag("FZ-evt", "DagList.OnSelected START dagId=%s", dagId)
		defer debugutil.Tag("FZ-evt", "DagList.OnSelected END dagId=%s", dagId)
		store.SelectDAG(dagId)

		for _, d := range store.GetDAGs() {
			if d.DagId == dagId {
				mainLayout.DagInfo().Update(d)
				break
			}
		}

		// Fetch runs (immediate)
		go func() {
			ctx := context.Background()
			runs, err := client.GetDAGRuns(ctx, dagId, &api.ListOptions{Limit: 50, OrderBy: "-start_date"})
			if err != nil {
				log.Printf("[ERROR] GetDAGRuns: %v", err)
				return
			}
			log.Printf("[DATA] DAGRuns fetched: %d runs for %s", len(runs.DAGRuns), dagId)
			store.SetDAGRuns(dagId, runs.DAGRuns)
		}()

		// Fetch lineage
		go func() {
			ctx := context.Background()
			tasks, err := client.GetTasks(ctx, dagId)
			if err != nil {
				return
			}
			store.SetTasks(dagId, tasks.Tasks)
		}()

		// Fetch DAG source code
		go func() {
			ctx := context.Background()
			code, err := client.GetDAGSource(ctx, dagId)
			if err != nil {
				dispatcher.Post(func() { mainLayout.Code().SetError(err.Error()) })
				return
			}
			dispatcher.Post(func() { mainLayout.Code().SetContent(code) })
		}()
	})

	// Run selected → fetch task instances + drill down to Tasks tab
	mainLayout.Runs().SetOnSelected(func(runId string) {
		debugutil.Tag("FZ-evt", "Runs.OnSelected START runId=%s", runId)
		defer debugutil.Tag("FZ-evt", "Runs.OnSelected END runId=%s", runId)
		store.SelectRun(runId)
		dagId := store.SelectedDAG()

		// Drill down: switch to Tasks tab
		mainLayout.SwitchTab("tasks")
		store.SetActiveTab("tasks")
		tviewApp.SetFocus(mainLayout.ActiveTabPrimitive())

		go func() {
			ctx := context.Background()
			ti, err := client.GetTaskInstances(ctx, dagId, runId, &api.ListOptions{Limit: 100})
			if err != nil {
				log.Printf("[ERROR] GetTaskInstances: %v", err)
				return
			}
			log.Printf("[DATA] TaskInstances fetched: %d tasks for %s/%s", len(ti.TaskInstances), dagId, runId)
			store.SetTaskInstances(dagId, runId, ti.TaskInstances)
		}()
	})

	// Task selected → fetch logs + drill down to Logs tab
	mainLayout.Tasks().SetOnSelected(func(taskId string) {
		debugutil.Tag("FZ-evt", "Tasks.OnSelected START taskId=%s", taskId)
		defer debugutil.Tag("FZ-evt", "Tasks.OnSelected END taskId=%s", taskId)
		store.SelectTask(taskId)
		dagId := store.SelectedDAG()
		runId := store.SelectedRun()

		mainLayout.SwitchTab("logs")
		store.SetActiveTab("logs")
		tviewApp.SetFocus(mainLayout.ActiveTabPrimitive())

		go func() {
			ctx := context.Background()
			logs, err := client.GetTaskLogs(ctx, dagId, runId, taskId, 1)
			if err != nil {
				log.Printf("[ERROR] GetTaskLogs: %v", err)
				dispatcher.Post(func() { mainLayout.Logs().SetError(err.Error()) })
				return
			}
			log.Printf("[DATA] TaskLogs fetched: %d chars", len(logs))
			dispatcher.Post(func() { mainLayout.Logs().SetContent(logs) })
		}()
	})

	// Backfills view selection callback
	mainLayout.Backfills().SetOnSelected(func(id int) {
		store.SelectBackfill(id)
	})

	// ---------- Keybindings ----------

	kb := ui.NewKeyBindings(tviewApp, mainLayout, store)

	kb.SetOnTrigger(func(dagId string) {
		mainLayout.ShowTriggerModal(dagId, func(params layout.TriggerParams) {
			go func() {
				ctx := context.Background()
				body := map[string]any{
					"logical_date": params.LogicalDate,
				}
				if params.Conf != "" && params.Conf != "{}" {
					var conf map[string]any
					if err := json.Unmarshal([]byte(params.Conf), &conf); err == nil {
						body["conf"] = conf
					}
				}
				_, err := client.TriggerDAGRun(ctx, dagId, body)
				dispatcher.Post(func() {
					if err != nil {
						mainLayout.StatusBar().SetError(fmt.Sprintf("Trigger failed: %v", err))
					} else {
						mainLayout.StatusBar().SetStatus(fmt.Sprintf("[green]DAG %s triggered[-]", dagId))
					}
				})
			}()
		})
	})

	kb.SetOnPause(func(dagId string) {
		var dag models.DAG
		for _, d := range store.GetDAGs() {
			if d.DagId == dagId {
				dag = d
				break
			}
		}
		action := "Pause"
		if dag.IsPaused {
			action = "Unpause"
		}
		mainLayout.ShowConfirmModal(
			fmt.Sprintf(" %s DAG ", action),
			fmt.Sprintf("%s DAG [yellow]%s[-]?", action, dagId),
			func() {
				go func() {
					ctx := context.Background()
					var err error
					if dag.IsPaused {
						err = client.UnpauseDAG(ctx, dagId)
					} else {
						err = client.PauseDAG(ctx, dagId)
					}
					dispatcher.Post(func() {
						if err != nil {
							mainLayout.StatusBar().SetError(fmt.Sprintf("%s failed: %v", action, err))
						} else {
							mainLayout.StatusBar().SetStatus(fmt.Sprintf("[green]DAG %s %sd[-]", dagId, strings.ToLower(action)))
						}
					})
				}()
			},
		)
	})

	kb.SetOnBackfill(func(dagId string) {
		mainLayout.ShowBackfillModal(dagId, func(params layout.BackfillParams) {
			go func() {
				ctx := context.Background()
				body := map[string]any{
					"dag_id":    dagId,
					"from_date": params.FromDate,
					"to_date":   params.ToDate,
				}
				if params.MaxActiveRuns != "" {
					var n int
					if _, err := fmt.Sscanf(params.MaxActiveRuns, "%d", &n); err == nil && n > 0 {
						body["max_active_runs"] = n
					}
				}
				if params.DagRunConf != "" && params.DagRunConf != "{}" {
					var conf map[string]any
					if err := json.Unmarshal([]byte(params.DagRunConf), &conf); err == nil {
						body["dag_run_conf"] = conf
					}
				}
				_, err := client.CreateBackfill(ctx, body)
				dispatcher.Post(func() {
					if err != nil {
						mainLayout.StatusBar().SetError(fmt.Sprintf("Backfill failed: %v", err))
					} else {
						mainLayout.StatusBar().SetStatus(fmt.Sprintf("[green]Backfill created for %s[-]", dagId))
					}
				})
			}()
		})
	})

	kb.SetOnBackfillCancel(func(id int) {
		mainLayout.ShowBackfillCancelModal(id, func() {
			go func() {
				if err := client.CancelBackfill(context.Background(), id); err != nil {
					dispatcher.Post(func() {
						mainLayout.StatusBar().SetError("cancel: " + err.Error())
					})
					return
				}
				// Optimistic refresh.
				if col, err := client.ListBackfills(context.Background(), store.SelectedDAG(), nil); err == nil {
					store.SetBackfills(store.SelectedDAG(), col.Backfills)
				}
			}()
		})
	})

	kb.SetOnBackfillPause(func(id int) {
		go func() {
			if err := client.PauseBackfill(context.Background(), id); err != nil {
				dispatcher.Post(func() {
					mainLayout.StatusBar().SetError("pause: " + err.Error())
				})
			}
		}()
	})

	kb.SetOnBackfillUnpause(func(id int) {
		go func() {
			if err := client.UnpauseBackfill(context.Background(), id); err != nil {
				dispatcher.Post(func() {
					mainLayout.StatusBar().SetError("unpause: " + err.Error())
				})
			}
		}()
	})

	kb.Install()

	// ---------- Polling ----------

	dagInterval := app.ParseDuration(cfg.UI.RefreshIntervals.DAGs, 5*time.Second)
	healthInterval := app.ParseDuration(cfg.UI.RefreshIntervals.Health, 10*time.Second)
	runsInterval := app.ParseDuration(cfg.UI.RefreshIntervals.Runs, 3*time.Second)
	tasksInterval := app.ParseDuration(cfg.UI.RefreshIntervals.Tasks, 2*time.Second)

	// Fixed: DAGs
	poller.Fixed(dagInterval, true, func(ctx context.Context) {
		dags, err := client.GetDAGs(ctx, &api.ListOptions{Limit: 100})
		if err != nil {
			return
		}
		store.SetDAGs(dags.DAGs)
	})

	// Fixed: Health
	poller.Fixed(healthInterval, true, func(ctx context.Context) {
		h, err := client.GetHealth(ctx)
		if err != nil {
			return
		}
		store.SetHealth(h)
	})

	// Dynamic: Runs (restart on DAG selection)
	store.Subscribe(state.EventDAGSelected, func(_ any) {
		dagId := store.SelectedDAG()
		poller.Restart("runs", runsInterval, func(ctx context.Context) {
			runs, err := client.GetDAGRuns(ctx, dagId, &api.ListOptions{Limit: 50, OrderBy: "-start_date"})
			if err != nil {
				return
			}
			store.SetDAGRuns(dagId, runs.DAGRuns)
		})
	})

	// Dynamic: TaskInstances (restart on Run selection)
	store.Subscribe(state.EventRunSelected, func(_ any) {
		runId := store.SelectedRun()
		dagId := store.SelectedDAG()
		poller.Restart("tasks", tasksInterval, func(ctx context.Context) {
			ti, err := client.GetTaskInstances(ctx, dagId, runId, &api.ListOptions{Limit: 100})
			if err != nil {
				return
			}
			store.SetTaskInstances(dagId, runId, ti.TaskInstances)
		})
	})

	// Backfills poller — runs only when the backfills tab is active AND a DAG is selected.
	backfillsInterval := 5 * time.Second
	startBackfillsPoll := func() {
		dagId := store.SelectedDAG()
		if dagId == "" {
			return
		}
		poller.Restart("backfills", backfillsInterval, func(ctx context.Context) {
			col, err := client.ListBackfills(ctx, dagId, nil)
			if err != nil {
				debugutil.Tag("FZ-bf", "ListBackfills err=%v", err)
				return
			}
			// Compute progress per backfill from DAG runs in date range.
			runs, _ := client.GetDAGRuns(ctx, dagId, nil)
			if runs != nil {
				for i := range col.Backfills {
					bf := &col.Backfills[i]
					countBackfillRuns(bf, runs.DAGRuns)
				}
			}
			bfCache.PutBackfills(dagId, col.Backfills)
			store.SetBackfills(dagId, col.Backfills)
		})
	}
	store.Subscribe(state.EventTabChanged, func(_ any) {
		if store.ActiveTab() == "backfills" {
			startBackfillsPoll()
		} else {
			poller.StopSub("backfills")
		}
	})

	// One-shot fetches for Connections, Variables, Config (loaded once at startup)
	go func() {
		ctx := context.Background()

		conns, err := client.GetConnections(ctx, &api.ListOptions{Limit: 100})
		if err == nil {
			dispatcher.Post(func() { mainLayout.Connections().Update(conns.Connections) })
		}

		vars, err := client.GetVariables(ctx, &api.ListOptions{Limit: 100})
		if err == nil {
			dispatcher.Post(func() { mainLayout.Variables().Update(vars.Variables) })
		}

		afCfg, err := client.GetConfig(ctx)
		if err == nil {
			dispatcher.Post(func() { mainLayout.Config().Update(afCfg) })
		}
	}()

	// Show connection status in header
	mainLayout.Header().SetConnection(cfg.Airflow.BaseURL, true)

	if err := tviewApp.SetRoot(mainLayout.Root(), true).EnableMouse(true).Run(); err != nil {
		log.Fatalf("error running application: %v", err)
	}
}

// countBackfillRuns populates the derived fields on bf based on DAG runs
// whose logical_date falls within [bf.FromDate, bf.ToDate] and whose
// run_type is "backfill". The field names on DAGRun may differ from Airflow's
// raw API — use what's defined in models.DAGRun.
func countBackfillRuns(bf *models.Backfill, runs []models.DAGRun) {
	bf.TotalRuns = 0
	bf.CompletedRuns = 0
	bf.FailedRuns = 0
	bf.RunningRuns = 0
	for _, r := range runs {
		if !inDateRange(r, bf) {
			continue
		}
		bf.TotalRuns++
		switch r.State {
		case "success":
			bf.CompletedRuns++
		case "failed":
			bf.FailedRuns++
		case "running", "queued":
			bf.RunningRuns++
		}
	}
}

func inDateRange(r models.DAGRun, bf *models.Backfill) bool {
	// Only count backfill-type runs.
	if r.RunType != "backfill" {
		return false
	}
	// LogicalDate is time.Time (not a pointer); skip zero values.
	if r.LogicalDate.IsZero() {
		return false
	}
	return !r.LogicalDate.Before(bf.FromDate) && !r.LogicalDate.After(bf.ToDate)
}
