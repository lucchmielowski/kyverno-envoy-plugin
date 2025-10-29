package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/kyverno/kyverno/pkg/cel/utils"
	"github.com/mark3labs/mcp-go/mcp"
)

type impl struct {
	types.Adapter
}

func (c *impl) mcp_parse_string(content ref.Val) ref.Val {
	// Convert content to string (from DynType)
	str, err := utils.ConvertToNative[string](content)
	if err != nil {
		return types.WrapErr(err)
	}

	// Parse JSON string into mcp.Request
	var req mcp.Request
	if err := json.Unmarshal([]byte(str), &req); err != nil {
		return types.WrapErr(fmt.Errorf("failed to parse MCP request: %w", err))
	}

	mcpReq := &MCPRequest{
		request: &req,
	}
	return c.NativeToValue(mcpReq)
}

func (c *impl) mcp_parse_bytes(content ref.Val) ref.Val {
	// Convert content to []byte
	bytes, err := utils.ConvertToNative[[]byte](content)
	if err != nil {
		return types.WrapErr(err)
	}

	// Parse JSON bytes into mcp.Request
	var req mcp.Request
	if err := json.Unmarshal(bytes, &req); err != nil {
		return types.WrapErr(fmt.Errorf("failed to parse MCP request: %w", err))
	}

	mcpReq := &MCPRequest{
		request: &req,
	}
	return c.NativeToValue(mcpReq)
}

func (c *impl) request_is_tool_call(val ref.Val) ref.Val {
	mcpReq, err := utils.ConvertToNative[*MCPRequest](val)
	if err != nil {
		return types.WrapErr(err)
	}

	fmt.Println(mcpReq)

	return types.Bool(mcpReq.request.Method == string(mcp.MethodToolsCall))
}

func (c *impl) request_is_tool_list(val ref.Val) ref.Val {
	mcpReq, err := utils.ConvertToNative[*MCPRequest](val)
	if err != nil {
		return types.WrapErr(err)
	}

	return types.Bool(mcpReq.request.Method == string(mcp.MethodToolsList))
}

func (c *impl) request_tool(val ref.Val) ref.Val {
	mcpReq, err := utils.ConvertToNative[*MCPRequest](val)
	if err != nil {
		return types.WrapErr(err)
	}

	// Check if the request is a CallToolRequest
	if mcpReq.request.Method != string(mcp.MethodToolsCall) {
		return types.WrapErr(fmt.Errorf("request is not a tool call"))
	}

	// Need to re-marshal and unmarshal to get the full params structure
	// because mcp.Request.Params is RequestParams, not CallToolParams
	requestBytes, err := json.Marshal(mcpReq.request)
	if err != nil {
		return types.WrapErr(fmt.Errorf("failed to marshal request: %w", err))
	}

	// Parse as CallToolRequest to get the full params
	var callToolReq mcp.CallToolRequest
	if err := json.Unmarshal(requestBytes, &callToolReq); err != nil {
		return types.WrapErr(fmt.Errorf("failed to parse call tool request: %w", err))
	}

	tool := &MCPTool{
		Name:      callToolReq.Params.Name,
		Arguments: callToolReq.GetArguments(),
	}
	return c.NativeToValue(tool)
}

func (c *impl) get_tool_name(val ref.Val) ref.Val {
	tool, err := utils.ConvertToNative[*MCPTool](val)
	if err != nil {
		return types.WrapErr(err)
	}

	return types.String(tool.Name)
}

func (c *impl) get_tool_arguments(val ref.Val) ref.Val {
	tool, err := utils.ConvertToNative[*MCPTool](val)
	if err != nil {
		return types.WrapErr(err)
	}

	return c.NativeToValue(tool.Arguments)
}

func (c *impl) has_tool_argument(toolVal ref.Val, argName ref.Val) ref.Val {
	tool, err := utils.ConvertToNative[*MCPTool](toolVal)
	if err != nil {
		return types.WrapErr(err)
	}

	argStr, ok := argName.(types.String)
	if !ok {
		return types.WrapErr(fmt.Errorf("argument name must be a string"))
	}

	_, exists := tool.Arguments[string(argStr)]
	return types.Bool(exists)
}

func (c *impl) get_tool_argument(toolVal ref.Val, argName ref.Val) ref.Val {
	tool, err := utils.ConvertToNative[*MCPTool](toolVal)
	if err != nil {
		return types.WrapErr(err)
	}

	argStr, ok := argName.(types.String)
	if !ok {
		return types.WrapErr(fmt.Errorf("argument name must be a string"))
	}

	argValue, exists := tool.Arguments[string(argStr)]
	if !exists {
		return nil
	}

	return c.NativeToValue(argValue)
}
