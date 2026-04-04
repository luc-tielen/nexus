package server

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// toolReq builds a CallToolRequest with the given string arguments.
func toolReq(args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

// textContent extracts the text from the first content item.
func textContent(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// --- get_time ---

func TestHandleGetTime(t *testing.T) {
	res, err := handleGetTime(mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content")
	}
}
