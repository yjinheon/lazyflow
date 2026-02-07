package layout

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TriggerParams struct {
	LogicalDate string
	Conf        string
}

type BackfillParams struct {
	FromDate      string
	ToDate        string
	MaxActiveRuns string
	DagRunConf    string
}

func (m *MainLayout) ShowTriggerModal(dagId string, onSubmit func(TriggerParams)) {
	if dagId == "" {
		return
	}

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Trigger DAG: %s ", dagId)).
		SetBorderColor(tcell.ColorBlue)

	now := time.Now().UTC().Format(time.RFC3339)
	form.AddInputField("Logical Date", now, 40, nil, nil)
	form.AddTextArea("Conf (JSON)", "{}", 40, 4, 0, nil)

	form.AddButton("Trigger", func() {
		logicalDate := form.GetFormItemByLabel("Logical Date").(*tview.InputField).GetText()
		conf := form.GetFormItemByLabel("Conf (JSON)").(*tview.TextArea).GetText()
		m.dismissModal()
		onSubmit(TriggerParams{LogicalDate: logicalDate, Conf: conf})
	})
	form.AddButton("Cancel", func() {
		m.dismissModal()
	})

	form.SetCancelFunc(func() {
		m.dismissModal()
	})

	m.showModal(form, 60, 14)
}

func (m *MainLayout) ShowBackfillModal(dagId string, onSubmit func(BackfillParams)) {
	if dagId == "" {
		return
	}

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Backfill DAG: %s ", dagId)).
		SetBorderColor(tcell.ColorBlue)

	yesterday := time.Now().AddDate(0, 0, -1).UTC().Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)

	form.AddInputField("From Date", yesterday, 40, nil, nil)
	form.AddInputField("To Date", now, 40, nil, nil)
	form.AddInputField("Max Active Runs", "10", 10, nil, nil)
	form.AddTextArea("Conf (JSON)", "{}", 40, 4, 0, nil)

	form.AddButton("Backfill", func() {
		fromDate := form.GetFormItemByLabel("From Date").(*tview.InputField).GetText()
		toDate := form.GetFormItemByLabel("To Date").(*tview.InputField).GetText()
		maxRuns := form.GetFormItemByLabel("Max Active Runs").(*tview.InputField).GetText()
		conf := form.GetFormItemByLabel("Conf (JSON)").(*tview.TextArea).GetText()
		m.dismissModal()
		onSubmit(BackfillParams{
			FromDate:      fromDate,
			ToDate:        toDate,
			MaxActiveRuns: maxRuns,
			DagRunConf:    conf,
		})
	})
	form.AddButton("Cancel", func() {
		m.dismissModal()
	})

	form.SetCancelFunc(func() {
		m.dismissModal()
	})

	m.showModal(form, 60, 18)
}

func (m *MainLayout) ShowConfirmModal(title, message string, onConfirm func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Confirm", "Cancel"}).
		SetDoneFunc(func(_ int, label string) {
			m.dismissModal()
			if label == "Confirm" {
				onConfirm()
			}
		})
	modal.SetTitle(title).SetBorder(true).SetBorderColor(tcell.ColorYellow)

	overlay := tview.NewPages().
		AddPage("main", m.root, true, true).
		AddPage("modal", modal, true, true)
	m.app.SetRoot(overlay, true)
	m.app.SetFocus(modal)
}

func (m *MainLayout) ShowNotification(message string) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			m.dismissModal()
		})

	overlay := tview.NewPages().
		AddPage("main", m.root, true, true).
		AddPage("modal", modal, true, true)
	m.app.SetRoot(overlay, true)
	m.app.SetFocus(modal)
}

func (m *MainLayout) showModal(content tview.Primitive, width, height int) {
	centered := centerPrimitive(content, width, height)
	overlay := tview.NewPages().
		AddPage("main", m.root, true, true).
		AddPage("modal", centered, true, true)
	m.app.SetRoot(overlay, true)
	m.app.SetFocus(content)
}

func (m *MainLayout) dismissModal() {
	m.app.SetRoot(m.root, true)
	m.app.SetFocus(m.dagList)
}
