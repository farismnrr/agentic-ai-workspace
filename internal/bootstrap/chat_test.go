package bootstrap

import (
	"encoding/json"
	"net/http"
	"testing"
)

func chatReply(text string) string {
	body, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"content": text}}},
		"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 5},
	})
	return string(body)
}

func TestChatConversationAndEscalation(t *testing.T) {
	t.Parallel()

	reply := chatReply("Hello! I can help you with this project.")
	a := newAgentRig(t, reply)

	// 1. Get chat
	res := do(t, a.env.app, "GET", "/api/projects/"+a.project+"/chat", "", jsonCookie(a.cookie))
	if res.Status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Status, res.Body)
	}

	// 2. Post a chat message
	chatPath := "/api/projects/" + a.project + "/chat"
	chatRes := do(t, a.env.app, "POST", chatPath, `{"prompt":"Hi Awam"}`, jsonCookie(a.cookie))
	if chatRes.Status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", chatRes.Status, chatRes.Body)
	}
	mustContain(t, chatRes.Body, "Hi Awam", "Hello! I can help you with this project.")

	// 3. Escalate to task
	escPath := "/api/projects/" + a.project + "/chat/escalate"
	escRes := do(t, a.env.app, "POST", escPath, `{"prompt":"Build login screen"}`, jsonCookie(a.cookie))
	if escRes.Status != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", escRes.Status, escRes.Body)
	}
	taskID, _ := escRes.JSON(t)["taskId"].(string)
	if taskID == "" {
		t.Fatalf("expected taskId in response: %s", escRes.Body)
	}

	// Verify task exists
	taskRes := do(t, a.env.app, "GET", "/api/tasks/"+taskID, "", jsonCookie(a.cookie))
	if taskRes.Status != http.StatusOK {
		t.Fatalf("expected task to exist, got %d", taskRes.Status)
	}
	mustContain(t, taskRes.Body, "Build login screen")

	// 4. Clear chat
	delRes := do(t, a.env.app, "DELETE", "/api/projects/"+a.project+"/chat", "", jsonCookie(a.cookie))
	if delRes.Status != http.StatusOK {
		t.Fatalf("expected 200, got %d", delRes.Status)
	}
}
