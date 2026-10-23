package application

type stop int

const (
	stopWaiting stop = iota + 1
	stopFinished
	stopFailed
)

type outcome struct {
	text string
	hint string
	// document is a markdown document a teammate sends with its report, such as the PRD.
	document string
	stop     stop
}

func failed(text, hint string) outcome {
	return outcome{stop: stopFailed, text: text, hint: hint}
}
