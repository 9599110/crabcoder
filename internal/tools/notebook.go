package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type notebook struct {
	Cells []nbCell `json:"cells"`
}

type nbCell struct {
	ID       string   `json:"id"`
	CellType string   `json:"cell_type"`
	Source   []string `json:"source"`
}

type NotebookEditExecutor struct{}

func (e *NotebookEditExecutor) Execute(ctx context.Context, args map[string]any) (*model.TaskResult, error) {
	path, _ := args["notebook_path"].(string)
	newSource, _ := args["new_source"].(string)
	cellType, _ := args["cell_type"].(string)
	editMode, _ := args["edit_mode"].(string)
	cellID, _ := args["cell_id"].(string)

	if path == "" {
		return &model.TaskResult{Success: false, Error: "notebook_path is required"}, nil
	}
	if !strings.HasSuffix(path, ".ipynb") {
		return &model.TaskResult{Success: false, Error: "notebook_path must end with .ipynb"}, nil
	}
	if cellType == "" {
		cellType = "code"
	}
	if editMode == "" {
		editMode = "replace"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	var nb notebook
	if err := json.Unmarshal(data, &nb); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	switch editMode {
	case "replace":
		if cellID == "" {
			if len(nb.Cells) == 0 {
				nb.Cells = append(nb.Cells, nbCell{CellType: cellType, Source: []string{newSource}})
			} else {
				nb.Cells[len(nb.Cells)-1].Source = []string{newSource}
			}
		} else {
			found := false
			for i := range nb.Cells {
				if nb.Cells[i].ID == cellID {
					nb.Cells[i].Source = []string{newSource}
					if cellType != "" {
						nb.Cells[i].CellType = cellType
					}
					found = true
					break
				}
			}
			if !found {
				return &model.TaskResult{Success: false, Error: "cell not found: " + cellID}, nil
			}
		}

	case "insert":
		newCell := nbCell{CellType: cellType, Source: []string{newSource}}
		if cellID == "" {
			nb.Cells = append(nb.Cells, newCell)
		} else {
			found := false
			for i := range nb.Cells {
				if nb.Cells[i].ID == cellID {
					nb.Cells = append(nb.Cells[:i+1], append([]nbCell{newCell}, nb.Cells[i+1:]...)...)
					found = true
					break
				}
			}
			if !found {
				nb.Cells = append(nb.Cells, newCell)
			}
		}

	case "delete":
		if cellID == "" {
			return &model.TaskResult{Success: false, Error: "cell_id is required for delete mode"}, nil
		}
		found := false
		for i := range nb.Cells {
			if nb.Cells[i].ID == cellID {
				nb.Cells = append(nb.Cells[:i], nb.Cells[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return &model.TaskResult{Success: false, Error: "cell not found: " + cellID}, nil
		}

	default:
		return &model.TaskResult{Success: false, Error: "unknown edit_mode: " + editMode}, nil
	}

	out, err := json.MarshalIndent(nb, "", "  ")
	if err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return &model.TaskResult{Success: false, Error: err.Error()}, nil
	}

	return &model.TaskResult{
		Success: true,
		Output:  fmt.Sprintf("Notebook edited: %s (mode=%s)", path, editMode),
	}, nil
}

func (e *NotebookEditExecutor) Validate(args map[string]any) error {
	p, _ := args["notebook_path"].(string)
	if p == "" {
		return fmt.Errorf("notebook_path is required")
	}
	return nil
}

func (e *NotebookEditExecutor) GetDefinition() model.ToolDefinition {
	return model.ToolDefinition{
		Name:        "notebook_edit",
		Description: "Edit a Jupyter notebook (.ipynb file) by replacing, inserting, or deleting cells.",
		Parameters: model.ParameterSchema{
			Type: "object",
			Properties: map[string]model.ParameterProperty{
				"notebook_path": {Type: "string", Description: "The path to the notebook file (required)."},
				"cell_id":       {Type: "string", Description: "The ID of the cell to edit."},
				"new_source":    {Type: "string", Description: "The new source for the cell."},
				"cell_type":     {Type: "string", Description: "code or markdown."},
				"edit_mode":     {Type: "string", Description: "replace, insert, or delete (default: replace)."},
			},
			Required: []string{"notebook_path", "new_source"},
		},
	}
}

func (e *NotebookEditExecutor) GetRiskLevel() RiskLevel { return RiskMedium }
