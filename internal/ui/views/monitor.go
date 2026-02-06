package views

import (
	"fmt"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type MonitorView struct {
	*tview.TextView
}

func NewMonitorView() *MonitorView {
	v := &MonitorView{
		TextView: tview.NewTextView(),
	}
	v.SetBorder(true).SetTitle(" System Monitor ")
	v.SetDynamicColors(true).SetScrollable(true)
	v.SetText("[gray]Waiting for health data...")
	return v
}

func (v *MonitorView) Update(health *models.HealthInfo) {
	if health == nil {
		v.SetText("[gray]No health data available")
		return
	}

	text := "[yellow::b]System Status[-::-]\n\n"
	text += v.formatComponent("Scheduler", health.Scheduler, health.Scheduler.LatestSchedulerHeartbeat)
	text += v.formatComponent("Metadatabase", health.Metadatabase, "")
	text += v.formatComponent("Triggerer", health.Triggerer, health.Triggerer.LatestTriggererHeartbeat)
	text += v.formatComponent("DAG Processor", health.DagProcessor, health.DagProcessor.LatestDagProcessorHeartbeat)

	v.SetText(text)
}

func (v *MonitorView) formatComponent(name string, s *models.HealthStatus, heartbeat string) string {
	if s == nil {
		return fmt.Sprintf("  [white]%-16s [gray]unknown\n", name)
	}

	color := "green"
	symbol := "[+]"
	if s.Status != "healthy" {
		color = "red"
		symbol = "[!]"
	}

	line := fmt.Sprintf("  [%s]%s %-16s[-] [white]%s[-]", color, symbol, name, s.Status)
	if heartbeat != "" {
		line += fmt.Sprintf("  [gray](heartbeat: %s)[-]", heartbeat)
	}
	return line + "\n"
}

func (v *MonitorView) Root() *tview.TextView {
	return v.TextView
}
