package views

import (
	"fmt"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type ClusterInfoView struct {
	*tview.TextView
}

func NewClusterInfoView() *ClusterInfoView {
	v := &ClusterInfoView{
		TextView: tview.NewTextView(),
	}
	v.SetBorder(true).SetTitle(" Cluster ")
	v.SetDynamicColors(true)
	v.SetText("[gray]Waiting for health check...")
	return v
}

func (v *ClusterInfoView) Update(health *models.HealthInfo) {
	if health == nil {
		v.SetText("[red]Health check failed")
		return
	}

	text := fmt.Sprintf(
		"[yellow]Scheduler:[-]    %s\n"+
			"[yellow]Metadatabase:[-] %s\n"+
			"[yellow]Triggerer:[-]    %s\n"+
			"[yellow]DAG Proc:[-]     %s",
		formatHealth(health.Scheduler),
		formatHealth(health.Metadatabase),
		formatHealth(health.Triggerer),
		formatHealth(health.DagProcessor),
	)
	v.SetText(text)
}

func formatHealth(s *models.HealthStatus) string {
	if s == nil {
		return "[gray]N/A[-]"
	}
	if s.Status == "healthy" {
		return "[green]healthy[-]"
	}
	return fmt.Sprintf("[red]%s[-]", s.Status)
}

func (v *ClusterInfoView) Root() *tview.TextView {
	return v.TextView
}
