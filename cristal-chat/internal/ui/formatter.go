package ui

import (
	"fmt"
	"github.com/fatih/color"
)

type Formatter struct {
	colorEnabled bool

	// Cores
	primary   *color.Color
	secondary *color.Color
	error_    *color.Color
	success   *color.Color
	muted     *color.Color
}

func NewFormatter(colorEnabled bool) *Formatter {
	return &Formatter{
		colorEnabled: colorEnabled,
		primary:      color.New(color.FgCyan, color.Bold),
		secondary:    color.New(color.FgYellow),
		error_:       color.New(color.FgRed, color.Bold),
		success:      color.New(color.FgGreen),
		muted:        color.New(color.FgHiBlack),
	}
}

func (f *Formatter) Logo() string {
	if f.colorEnabled {
		return f.primary.Sprint("🔮 Cristal Chat")
	}
	return "Cristal Chat"
}

func (f *Formatter) Prompt() string {
	if f.colorEnabled {
		return f.primary.Sprint("🔮 > ")
	}
	return "> "
}

func (f *Formatter) PrintError(err error) {
	if f.colorEnabled {
		fmt.Println(f.error_.Sprintf("❌ Erro: %v", err))
	} else {
		fmt.Printf("Erro: %v\n", err)
	}
}

func (f *Formatter) PrintSearching() {
	if f.colorEnabled {
		fmt.Println(f.muted.Sprint("🔍 Pesquisando..."))
	} else {
		fmt.Println("Pesquisando...")
	}
}

// Será expandido em M2.2
