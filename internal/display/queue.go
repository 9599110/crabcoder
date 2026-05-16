package display

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/crabcoder/crabcoder/internal/event"
)

// ANSI color codes for task identification
const (
	colorReset   = "\033[0m"
	colorCyan    = "\033[36m"
	colorMagenta = "\033[35m"
	colorYellow  = "\033[33m"
	colorGreen   = "\033[32m"
	colorBlue    = "\033[34m"
	colorRed     = "\033[31m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
)

var taskColors = []string{colorCyan, colorMagenta, colorYellow, colorGreen, colorBlue, colorRed}

// DisplayItem represents something to render on the CLI.
// Parallel tasks push items to the queue; a single consumer goroutine
// processes them one-by-one (serialising all terminal output).
type DisplayItem struct {
	Kind       string // "output", "status", "approval"
	TaskID     string
	TaskDesc   string
	Message    string
	Status     string // for "status": "started", "completed", "failed"
	Risk       string
	ResponseCh chan bool // for "approval" items: CLI sends user's answer back
}

// DisplayQueue serialises parallel task output into ordered terminal rendering.
// Each task gets a unique color so the user can visually track which task
// produced which line. Approval items block the queue until the user responds.
type DisplayQueue struct {
	items    chan DisplayItem
	colorIdx int
	colorMap map[string]string
	colorMu  sync.Mutex
	reader   *bufio.Reader
}

// NewDisplayQueue creates a new display queue.
func NewDisplayQueue() *DisplayQueue {
	return &DisplayQueue{
		items:    make(chan DisplayItem, 256),
		colorMap: make(map[string]string),
		reader:   bufio.NewReader(os.Stdin),
	}
}

func (dq *DisplayQueue) getTaskColor(taskID string) string {
	dq.colorMu.Lock()
	defer dq.colorMu.Unlock()
	if c, ok := dq.colorMap[taskID]; ok {
		return c
	}
	c := taskColors[dq.colorIdx%len(taskColors)]
	dq.colorIdx++
	dq.colorMap[taskID] = c
	return c
}

// Start runs the display consumer loop (blocks until the queue is closed).
func (dq *DisplayQueue) Start() {
	for item := range dq.items {
		dq.render(item)
	}
}

// Push adds an item to the display queue.
func (dq *DisplayQueue) Push(item DisplayItem) {
	dq.items <- item
}

// Done closes the queue, causing Start to return after draining remaining items.
func (dq *DisplayQueue) Done() {
	close(dq.items)
}

// SubscribeFromBus wires the display queue to consume engine events and
// convert them into display items. Call this before Start.
func (dq *DisplayQueue) SubscribeFromBus(bus *event.Bus) {
	types := []event.EventType{
		event.TaskStarted,
		event.TaskCompleted,
		event.TaskFailed,
		event.TaskOutput,
		event.ApprovalRequired,
	}
	for _, et := range types {
		ch := bus.Subscribe(et)
		go func(sub <-chan event.Event) {
			for e := range sub {
				dq.convertEvent(e)
			}
		}(ch)
	}
}

func (dq *DisplayQueue) convertEvent(e event.Event) {
	switch e.Type {
	case event.TaskStarted:
		taskID, _ := e.Data["task_id"].(string)
		desc, _ := e.Data["description"].(string)
		dq.Push(DisplayItem{Kind: "status", TaskID: taskID, TaskDesc: desc, Status: "started"})

	case event.TaskCompleted:
		taskID, _ := e.Data["task_id"].(string)
		output, _ := e.Data["output"].(string)
		dq.Push(DisplayItem{Kind: "status", TaskID: taskID, Status: "completed", Message: output})

	case event.TaskFailed:
		taskID, _ := e.Data["task_id"].(string)
		errMsg, _ := e.Data["error"].(string)
		dq.Push(DisplayItem{Kind: "status", TaskID: taskID, Status: "failed", Message: errMsg})

	case event.TaskOutput:
		taskID, _ := e.Data["task_id"].(string)
		msg, _ := e.Data["message"].(string)
		dq.Push(DisplayItem{Kind: "output", TaskID: taskID, Message: msg})

	case event.ApprovalRequired:
		taskID, _ := e.Data["task_id"].(string)
		desc, _ := e.Data["description"].(string)
		risk, _ := e.Data["risk"].(string)
		respCh, _ := e.Data["response_ch"].(chan bool)
		dq.Push(DisplayItem{Kind: "approval", TaskID: taskID, TaskDesc: desc, Risk: risk, ResponseCh: respCh})
	}
}

func (dq *DisplayQueue) render(item DisplayItem) {
	color := dq.getTaskColor(item.TaskID)

	switch item.Kind {
	case "output":
		fmt.Printf("%s[%s]%s %s\n", color, item.TaskID, colorReset, item.Message)

	case "status":
		switch item.Status {
		case "started":
			fmt.Printf("%s[%s]%s %s▶ 开始执行: %s%s\n",
				color, item.TaskID, colorReset, colorBold, item.TaskDesc, colorReset)
		case "completed":
			fmt.Printf("%s[%s]%s %s✓ 完成%s\n",
				color, item.TaskID, colorReset, colorGreen, colorReset)
		case "failed":
			fmt.Printf("%s[%s]%s %s✗ 失败: %s%s\n",
				color, item.TaskID, colorReset, colorRed, item.Message, colorReset)
		}

	case "approval":
		fmt.Printf("\n%s%s⚠ 需要确认 [%s]%s\n", colorBold, color, item.TaskID, colorReset)
		fmt.Printf("  任务: %s\n", item.TaskDesc)
		fmt.Printf("  风险: %s\n", item.Risk)
		fmt.Printf("  %s允许执行吗？[y/N]: %s", colorBold, colorReset)

		input, _ := dq.reader.ReadString('\n')
		approved := strings.TrimSpace(strings.ToLower(input)) == "y"

		if item.ResponseCh != nil {
			item.ResponseCh <- approved
		}

		if approved {
			fmt.Printf("%s  ✓ 已批准%s\n\n", colorGreen, colorReset)
		} else {
			fmt.Printf("%s  ✗ 已拒绝%s\n\n", colorRed, colorReset)
		}
	}
}
