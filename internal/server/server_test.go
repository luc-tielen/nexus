package server

import (
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
