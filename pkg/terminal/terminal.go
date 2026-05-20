// terminal 终端模块
package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
)

// Terminal 终端接口
type Terminal interface {
	// 读取
	ReadLine() (string, error)
	ReadPassword(prompt string) (string, error)
	ReadConfirmation(prompt string) (bool, error)

	// 输出
	Print(a ...any) error
	Println(a ...any) error
	Printf(format string, a ...any) error
	PrintStyled(style Style, a ...any) error

	// 样式
	SetStyle(style Style) Terminal
	ResetStyle() Terminal

	// 光标控制
	Clear() error
	ClearLine() error
	MoveCursor(x, y int) error

	// 尺寸
	Size() (width, height int, err error)
}

// Style 样式
type Style struct {
	Color     Color
	Bold      bool
	Dim       bool
	Underline bool
	Reverse   bool
}

// Color 颜色
type Color string

const (
	ColorDefault Color = ""
	ColorBlack   Color = "black"
	ColorRed     Color = "red"
	ColorGreen   Color = "green"
	ColorYellow  Color = "yellow"
	ColorBlue    Color = "blue"
	ColorMagenta Color = "magenta"
	ColorCyan    Color = "cyan"
	ColorWhite   Color = "white"
)

// defaultTerminal 默认终端实现
type defaultTerminal struct {
	reader *bufio.Reader
	style  Style
}

// NewDefault 创建默认终端
func NewDefault() Terminal {
	return &defaultTerminal{
		reader: bufio.NewReader(os.Stdin),
	}
}

func (t *defaultTerminal) ReadLine() (string, error) {
	fmt.Print("> ")
	input, err := t.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func (t *defaultTerminal) ReadPassword(promptText string) (string, error) {
	p := promptui.Prompt{
		Label: promptText,
		Mask:  '*',
	}
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}

func (t *defaultTerminal) ReadConfirmation(promptText string) (bool, error) {
	p := promptui.Prompt{
		Label:     promptText + " (y/N)",
		IsConfirm: true,
	}
	_, err := p.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (t *defaultTerminal) Print(a ...any) error {
	_, err := fmt.Print(a...)
	return err
}

func (t *defaultTerminal) Println(a ...any) error {
	return t.Print(fmt.Sprintln(a...))
}

func (t *defaultTerminal) Printf(format string, a ...any) error {
	return t.Print(fmt.Sprintf(format, a...))
}

func (t *defaultTerminal) PrintStyled(style Style, a ...any) error {
	// 简化实现
	return t.Print(a...)
}

func (t *defaultTerminal) SetStyle(style Style) Terminal {
	t.style = style
	return t
}

func (t *defaultTerminal) ResetStyle() Terminal {
	t.style = Style{}
	return t
}

func (t *defaultTerminal) Clear() error {
	fmt.Print("\033[2J")
	return nil
}

func (t *defaultTerminal) ClearLine() error {
	fmt.Print("\033[2K")
	return nil
}

func (t *defaultTerminal) MoveCursor(x, y int) error {
	fmt.Printf("\033[%d;%dH", y, x)
	return nil
}

func (t *defaultTerminal) Size() (width, height int, err error) {
	// 简化实现
	return 80, 24, nil
}
