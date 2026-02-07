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
	"github.com/yjinheon/lazyflow/internal/state"
	ui "github.com/yjinheon/lazyflow/internal/ui"
	"github.com/yjinheon/lazyflow/internal/ui/layout"
	"github.com/yjinheon/lazyflow/internal/ui/theme"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

func main() {
	// Debug log to file
	logFile, _ := os.Create("lazyflow.log")
	if logFile != nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	tviewApp := tview.NewApplication()
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
		tviewApp.QueueUpdateDraw(func() {
			dags := store.GetDAGs()
			mainLayout.DagList().Update(dags)
			mainLayout.Header().SetInfo(cfg.Airflow.BaseURL, true, len(dags))
		})
	})

	// Health updated → refresh cluster panel + monitor
	store.Subscribe(state.EventHealthUpdated, func(_ any) {
		tviewApp.QueueUpdateDraw(func() {
			h := store.GetHealth()
			mainLayout.ClusterInfo().Update(h)
			mainLayout.Monitor().Update(h)
		})
	})

	// Selection → update status bar
	// NOTE: These subscribers are called synchronously from tview's main goroutine
	// (via SetSelectedFunc → store.Select*). Using QueueUpdateDraw here would
	// deadlock because QueueUpdate blocks until processed, but the main goroutine
	// is the one that processes the queue. Use "go" to avoid blocking.
	store.Subscribe(state.EventDAGSelected, func(data any) {
		dagId := data.(string)
		go tviewApp.QueueUpdateDraw(func() {
			mainLayout.StatusBar().SetInfo(dagId, "", "")
		})
	})
	store.Subscribe(state.EventRunSelected, func(data any) {
		runId := data.(string)
		go tviewApp.QueueUpdateDraw(func() {
			mainLayout.StatusBar().SetInfo(store.SelectedDAG(), runId, "")
		})
	})
	store.Subscribe(state.EventTaskSelected, func(data any) {
		taskId := data.(string)
		go tviewApp.QueueUpdateDraw(func() {
			mainLayout.StatusBar().SetInfo(store.SelectedDAG(), store.SelectedRun(), taskId)
		})
	})

	// DAG selected → info panel, fetch runs + lineage + code
	mainLayout.DagList().SetOnSelected(func(dagId string) {
		log.Printf("[EVENT] DAG selected: %s", dagId)
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
			tviewApp.QueueUpdateDraw(func() { mainLayout.Runs().Update(runs.DAGRuns) })
		}()

		// Fetch lineage
		go func() {
			ctx := context.Background()
			tasks, err := client.GetTasks(ctx, dagId)
			if err != nil {
				return
			}
			tviewApp.QueueUpdateDraw(func() { mainLayout.Lineage().SetTasks(dagId, tasks.Tasks) })
		}()

		// Fetch DAG source code
		go func() {
			ctx := context.Background()
			code, err := client.GetDAGSource(ctx, dagId)
			if err != nil {
				tviewApp.QueueUpdateDraw(func() { mainLayout.Code().SetError(err.Error()) })
				return
			}
			tviewApp.QueueUpdateDraw(func() { mainLayout.Code().SetContent(code) })
		}()
	})

	// Run selected → fetch task instances + drill down to Tasks tab
	mainLayout.Runs().SetOnSelected(func(runId string) {
		log.Printf("[EVENT] Run selected: %s", runId)
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
			tviewApp.QueueUpdateDraw(func() { mainLayout.Tasks().Update(ti.TaskInstances) })
		}()
	})

	// Task selected → fetch logs + drill down to Logs tab
	mainLayout.Tasks().SetOnSelected(func(taskId string) {
		log.Printf("[EVENT] Task selected: %s", taskId)
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
				tviewApp.QueueUpdateDraw(func() { mainLayout.Logs().SetError(err.Error()) })
				return
			}
			log.Printf("[DATA] TaskLogs fetched: %d chars", len(logs))
			tviewApp.QueueUpdateDraw(func() { mainLayout.Logs().SetContent(logs) })
		}()
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
				tviewApp.QueueUpdateDraw(func() {
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
					tviewApp.QueueUpdateDraw(func() {
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
				tviewApp.QueueUpdateDraw(func() {
					if err != nil {
						mainLayout.StatusBar().SetError(fmt.Sprintf("Backfill failed: %v", err))
					} else {
						mainLayout.StatusBar().SetStatus(fmt.Sprintf("[green]Backfill created for %s[-]", dagId))
					}
				})
			}()
		})
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
	store.Subscribe(state.EventDAGSelected, func(data any) {
		dagId := data.(string)
		poller.Restart("runs", runsInterval, func(ctx context.Context) {
			runs, err := client.GetDAGRuns(ctx, dagId, &api.ListOptions{Limit: 50, OrderBy: "-start_date"})
			if err != nil {
				return
			}
			store.SetDAGRuns(dagId, runs.DAGRuns)
			tviewApp.QueueUpdateDraw(func() { mainLayout.Runs().Update(runs.DAGRuns) })
		})
	})

	// Dynamic: TaskInstances (restart on Run selection)
	store.Subscribe(state.EventRunSelected, func(data any) {
		runId := data.(string)
		dagId := store.SelectedDAG()
		poller.Restart("tasks", tasksInterval, func(ctx context.Context) {
			ti, err := client.GetTaskInstances(ctx, dagId, runId, &api.ListOptions{Limit: 100})
			if err != nil {
				return
			}
			store.SetTaskInstances(dagId, runId, ti.TaskInstances)
			tviewApp.QueueUpdateDraw(func() { mainLayout.Tasks().Update(ti.TaskInstances) })
		})
	})

	// One-shot fetches for Connections, Variables, Config (loaded once at startup)
	go func() {
		ctx := context.Background()

		conns, err := client.GetConnections(ctx, &api.ListOptions{Limit: 100})
		if err == nil {
			tviewApp.QueueUpdateDraw(func() { mainLayout.Connections().Update(conns.Connections) })
		}

		vars, err := client.GetVariables(ctx, &api.ListOptions{Limit: 100})
		if err == nil {
			tviewApp.QueueUpdateDraw(func() { mainLayout.Variables().Update(vars.Variables) })
		}

		afCfg, err := client.GetConfig(ctx)
		if err == nil {
			tviewApp.QueueUpdateDraw(func() { mainLayout.Config().Update(afCfg) })
		}
	}()

	// Show connection status in header
	mainLayout.Header().SetConnection(cfg.Airflow.BaseURL, true)

	if err := tviewApp.SetRoot(mainLayout.Root(), true).EnableMouse(true).Run(); err != nil {
		log.Fatalf("error running application: %v", err)
	}
}
