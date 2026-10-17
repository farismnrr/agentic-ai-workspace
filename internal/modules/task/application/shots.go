package application

import (
	"context"
	"net/http"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Shots keeps the screenshots of a task: the PNGs the model looks at and the person sees in the thread.
type Shots interface {
	Save(ctx context.Context, taskID string, png []byte) (id string, err error)
	// Load returns domain.ErrNotFound for a screenshot the task does not have.
	Load(ctx context.Context, taskID, id string) ([]byte, error)
	// Remove deletes every screenshot of a task.
	Remove(ctx context.Context, taskID string) error
}

// keepImages is how many of the latest screenshots the model gets to see again each turn. Older ones stay in the
// transcript as their text report: every image is sent again with every request, and costs as much each time.
const keepImages = 2

// withImages is the conversation with the bytes of the latest screenshots filled in, without touching the stored
// transcript.
func (r *Runner) withImages(ctx context.Context, task domain.Task, messages []domain.Message) []domain.Message {
	if r.deps.Shots == nil {
		return messages
	}

	out := make([]domain.Message, len(messages))
	copy(out, messages)

	left := keepImages

	for i := len(out) - 1; i >= 0 && left > 0; i-- {
		blocks := append([]domain.Block(nil), out[i].Blocks...)
		left = r.fillImages(ctx, task.ID, blocks, left)
		out[i].Blocks = blocks
	}

	return out
}

// fillImages loads the screenshots of the blocks, last first, while `left` allows; it returns what is left.
func (r *Runner) fillImages(ctx context.Context, taskID string, blocks []domain.Block, left int) int {
	for j := len(blocks) - 1; j >= 0 && left > 0; j-- {
		if blocks[j].Kind != domain.BlockToolResult || blocks[j].Image == "" {
			continue
		}

		if data, err := r.deps.Shots.Load(ctx, taskID, blocks[j].Image); err == nil {
			blocks[j].Data = data
			left--
		}
	}

	return left
}

func hasImages(messages []domain.Message) bool {
	for _, msg := range messages {
		for _, block := range msg.Blocks {
			if len(block.Data) > 0 {
				return true
			}
		}
	}

	return false
}

func withoutImages(messages []domain.Message) []domain.Message {
	out := make([]domain.Message, len(messages))

	for i, msg := range messages {
		blocks := append([]domain.Block(nil), msg.Blocks...)
		for j := range blocks {
			blocks[j].Data = nil
		}

		out[i] = domain.Message{Role: msg.Role, Blocks: blocks}
	}

	return out
}

// askSeeing asks the model with the screenshots. A model that cannot read images answers with a request error;
// the call is then repeated once without them and the task goes on with the text reports only.
func (r *Runner) askSeeing(ctx context.Context, client Model, request Request, state *runState) (Reply, error) {
	reply, err := r.ask(ctx, client, request)
	if err == nil || !hasImages(request.Messages) || !rejectedRequest(err) {
		return reply, err
	}

	state.noImages = true
	request.Messages = withoutImages(request.Messages)

	return r.ask(ctx, client, request)
}

func rejectedRequest(err error) bool {
	modelErr, ok := domain.AsModelError(err)
	if !ok || modelErr.Retryable {
		return false
	}

	switch modelErr.Status {
	case 0, http.StatusBadRequest, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType,
		http.StatusUnprocessableEntity:
		return true
	}

	return false
}

// saveShot keeps the screenshot a tool produced and answers the call with it. It returns the saved name, or
// nothing when the result has no screenshot, so the call is answered as plain text.
func (r *Runner) saveShot(ctx context.Context, task domain.Task, call Call, result ToolResult) string {
	if len(result.Image) == 0 || result.IsError || r.deps.Shots == nil {
		return ""
	}

	id, err := r.deps.Shots.Save(ctx, task.ID, result.Image)
	if err != nil {
		return ""
	}

	r.resultImage(ctx, task, call.ID, result.Output, id)

	return id
}
