package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	rw "github.com/mattn/go-runewidth"
	"sync"
	"unicode/utf8"
)

const (
	PROMPT_TEXT                = ":"
	SEARCH_PROMPT_TEXT         = "/"
	REVERSE_SEARCH_PROMPT_TEXT = "?"
	FILTER_PROMPT_TEXT         = "query: "
)

type PromptType int

const (
	PT_NONE PromptType = iota
	PT_COMMAND
	PT_SEARCH
	PT_FILTER
)

type PropertyValue struct {
	Property string
	Value    string
}

type StatusBarView struct {
	rootView      RootView
	repoData      RepoData
	channels      *Channels
	config        ConfigSetter
	active        bool
	promptType    PromptType
	pendingStatus string
	lock          sync.Mutex
}

func NewStatusBarView(rootView RootView, repoData RepoData, channels *Channels, config ConfigSetter) *StatusBarView {
	return &StatusBarView{
		rootView: rootView,
		repoData: repoData,
		channels: channels,
		config:   config,
	}
}

func (statusBarView *StatusBarView) Initialise() (err error) {
	return
}

func (statusBarView *StatusBarView) HandleKeyPress(keystring string) (err error) {
	return
}

func (statusBarView *StatusBarView) HandleAction(action Action) (err error) {
	switch action.ActionType {
	case ACTION_PROMPT:
		statusBarView.showCommandPrompt()
	case ACTION_SEARCH_PROMPT:
		statusBarView.showSearchPrompt(SEARCH_PROMPT_TEXT, ACTION_SEARCH)
	case ACTION_REVERSE_SEARCH_PROMPT:
		statusBarView.showSearchPrompt(REVERSE_SEARCH_PROMPT_TEXT, ACTION_REVERSE_SEARCH)
	case ACTION_FILTER_PROMPT:
		statusBarView.showFilterPrompt()
	case ACTION_SHOW_STATUS:
		statusBarView.lock.Lock()
		defer statusBarView.lock.Unlock()

		if len(action.Args) > 0 {
			status, ok := action.Args[0].(string)
			if ok {
				statusBarView.pendingStatus = status
				log.Infof("Received status: %v", status)
				statusBarView.channels.UpdateDisplay()
				return
			}
		}

		err = fmt.Errorf("Expected status argument but received: %v", action.Args)
	}

	return
}

func (statusBarView *StatusBarView) showCommandPrompt() {
	statusBarView.promptType = PT_COMMAND
	input := Prompt(PROMPT_TEXT)
	errors := statusBarView.config.Evaluate(input)
	statusBarView.channels.ReportErrors(errors)
	statusBarView.promptType = PT_NONE
}

func (statusBarView *StatusBarView) showSearchPrompt(prompt string, actionType ActionType) {
	statusBarView.promptType = PT_SEARCH
	input := Prompt(prompt)

	if input == "" {
		statusBarView.channels.DoAction(Action{
			ActionType: ACTION_CLEAR_SEARCH,
		})
	} else {
		statusBarView.channels.DoAction(Action{
			ActionType: actionType,
			Args:       []interface{}{input},
		})
	}

	statusBarView.promptType = PT_NONE
}

func (statusBarView *StatusBarView) showFilterPrompt() {
	statusBarView.promptType = PT_FILTER
	input := Prompt(FILTER_PROMPT_TEXT)

	if input != "" {
		statusBarView.channels.DoAction(Action{
			ActionType: ACTION_ADD_FILTER,
			Args:       []interface{}{input},
		})
	}

	statusBarView.promptType = PT_NONE
}

func (statusBarView *StatusBarView) OnActiveChange(active bool) {
	statusBarView.lock.Lock()
	defer statusBarView.lock.Unlock()

	log.Debugf("StatusBarView active: %v", active)
	statusBarView.active = active

	return
}

func (statusBarView *StatusBarView) ViewId() ViewId {
	return VIEW_STATUS_BAR
}

func (statusBarView *StatusBarView) Render(win RenderWindow) (err error) {
	statusBarView.lock.Lock()
	defer statusBarView.lock.Unlock()

	lineBuilder, err := win.LineBuilder(0, 1)
	if err != nil {
		return
	}

	if statusBarView.active {
		promptText, promptInput, promptPoint := PromptState()
		lineBuilder.Append("%v%v", promptText, promptInput)
		bytes := 0
		characters := len(promptText)

		for _, char := range promptInput {
			bytes += utf8.RuneLen(char)

			if bytes > promptPoint {
				break
			}

			if rw.RuneWidth(char) > 0 {
				characters++
			}
		}

		win.SetCursor(0, uint(characters))
	} else {
		lineBuilder.Append(" %v", statusBarView.pendingStatus)
		win.ApplyStyle(CMP_STATUSBARVIEW_NORMAL)
	}

	return
}

func (statusBarView *StatusBarView) RenderStatusBar(lineBuilder *LineBuilder) (err error) {
	return
}

func (statusBarView *StatusBarView) RenderHelpBar(lineBuilder *LineBuilder) (err error) {
	message := ""

	switch statusBarView.promptType {
	case PT_COMMAND:
		message = "Enter a command"
	case PT_SEARCH:
		message = "Enter a regex pattern"
	case PT_FILTER:
		message = "Enter a filter query"
	}

	if message != "" {
		lineBuilder.AppendWithStyle(CMP_HELPBARVIEW_SPECIAL, message)
	}

	return
}

func RenderStatusProperties(lineBuilder *LineBuilder, propertyValues []PropertyValue) {
	for _, propValue := range propertyValues {
		lineBuilder.Append("%v: %v     ", propValue.Property, propValue.Value)
	}
}