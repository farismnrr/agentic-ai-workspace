package domain

import "errors"

// ToolSubmitPlan explicitly asks the person to review a saved plan.
var ToolSubmitPlan = workTool("submit_plan", "Submit plan", "Send the saved plan to the person for approval.")

// ToolFinishResearch checks whether a sourced research round is saved.
var ToolFinishResearch = workTool("finish_research", "Finish research",
	"Check whether saved research is ready for planning.")

// ToolAskQuestion asks the project owner a persistent question before planning continues.
var ToolAskQuestion = workTool("ask_question", "Ask the person", "Save one to four questions for the project owner.")

// ErrBadQuestion is returned when an MCP question form has no usable question.
var ErrBadQuestion = errors.New("provide one to four questions with a question for each")
