package views

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
	"github.com/yjinheon/lazyflow/pkg/airflow/models"
)

type DagInfoView struct {
	*tview.TextView
}

func NewDagInfoView() *DagInfoView {
	v := &DagInfoView{
		TextView: tview.NewTextView(),
	}
	v.SetBorder(true).SetTitle(" DAG Info ")
	v.SetDynamicColors(true).SetScrollable(true)
	v.SetText("[gray]Select a DAG to view details")
	return v
}

func (v *DagInfoView) Update(dag models.DAG) {
	tags := make([]string, len(dag.Tags))
	for i, t := range dag.Tags {
		tags[i] = t.Name
	}
	tagStr := "(none)"
	if len(tags) > 0 {
		tagStr = strings.Join(tags, ", ")
	}

	owners := "(none)"
	if len(dag.Owners) > 0 {
		owners = strings.Join(dag.Owners, ", ")
	}

	state := "Active"
	stateColor := "green"
	if dag.IsPaused {
		state = "Paused"
		stateColor = "yellow"
	}

	text := fmt.Sprintf(
		"[yellow]DAG ID:[-]    %s\n"+
			"[yellow]State:[-]     [%s]%s[-]\n"+
			"[yellow]Owner:[-]     %s\n"+
			"[yellow]Schedule:[-]  %s\n"+
			"[yellow]Tags:[-]      %s\n"+
			"[yellow]File:[-]      %s\n"+
			"\n[yellow]Description:[-]\n%s",
		dag.DagId,
		stateColor, state,
		owners,
		dag.Schedule(),
		tagStr,
		dag.Fileloc,
		derefStr(dag.Description),
	)
	v.SetText(text)
}

func (v *DagInfoView) Root() *tview.TextView {
	return v.TextView
}
