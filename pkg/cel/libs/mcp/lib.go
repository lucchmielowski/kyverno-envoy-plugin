package mcp

import (
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/ext"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	MCPType        = types.NewOpaqueType("mcp.MCP")
	MCPRequestType = types.NewObjectType("mcp.MCPRequest")
	MCPToolType    = types.NewObjectType("mcp.MCPTool")
)

type MCP struct{}

type MCPRequest struct {
	request *mcp.Request
}

type MCPTool struct {
	Name      string
	Arguments map[string]any
}

type lib struct {
	mcp MCP
}

func Lib() cel.EnvOption {
	return cel.Lib(&lib{})
}

func (*lib) LibraryName() string {
	return "kyverno.mcp"
}

func (l *lib) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Variable("mcp", MCPType),
		// Use native types to register custom structs with the ext native type provider
		ext.NativeTypes(
			reflect.TypeFor[MCP](),
			reflect.TypeFor[MCPRequest](),
			reflect.TypeFor[MCPTool](),
			ext.ParseStructTags(true),
		),
		// extend environment with function overloads
		l.extendEnv,
	}
}

func (l *lib) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{
		cel.Globals(
			map[string]any{
				"mcp": l.mcp,
			},
		),
	}
}

func (*lib) extendEnv(env *cel.Env) (*cel.Env, error) {
	impl := impl{
		Adapter: env.CELTypeAdapter(),
	}

	libraryDecls := map[string][]cel.FunctionOpt{
		"mcp.Parse": {
			// Member function on MCP type: mcp.Parse(string) -> MCPRequest
			cel.Overload("mcp_parse_string",
				[]*cel.Type{types.StringType},
				MCPRequestType,
				cel.UnaryBinding(impl.mcp_parse_string),
			),
			// Member function on MCP type: mcp.Parse(bytes) -> MCPRequest
			cel.Overload("mcp_parse_bytes",
				[]*cel.Type{cel.BytesType},
				MCPRequestType,
				cel.UnaryBinding(impl.mcp_parse_bytes),
			),
		},
		"IsToolCall": {
			// Member function on MCPRequest type: request.IsToolCall() -> bool
			cel.MemberOverload("mcp_request_is_tool_call",
				[]*cel.Type{types.NewObjectType("mcp.MCPRequest")},
				cel.BoolType,
				cel.UnaryBinding(impl.request_is_tool_call),
			),
		},
		"IsToolList": {
			// Member function on MCPRequest type: request.IsToolList() -> bool
			cel.MemberOverload("mcp_request_is_tool_list",
				[]*cel.Type{types.NewObjectType("mcp.MCPRequest")},
				cel.BoolType,
				cel.UnaryBinding(impl.request_is_tool_list),
			),
		},
		"Tool": {
			// Member function on MCPRequest type: request.Tool() -> MCPTool
			cel.MemberOverload("mcp_request_tool",
				[]*cel.Type{types.NewObjectType("mcp.MCPRequest")},
				MCPToolType,
				cel.UnaryBinding(impl.request_tool),
			),
		},
		"Name": {
			// Member function on MCPTool type: tool.Name() -> string
			cel.MemberOverload("mcp_tool_name",
				[]*cel.Type{types.NewObjectType("mcp.MCPTool")},
				cel.StringType,
				cel.UnaryBinding(impl.get_tool_name),
			),
		},
		"Arguments": {
			// Member function on MCPTool type: tool.Arguments() -> map
			cel.MemberOverload("mcp_tool_get_arguments",
				[]*cel.Type{types.NewObjectType("mcp.MCPTool")},
				types.NewMapType(cel.StringType, cel.DynType),
				cel.UnaryBinding(impl.get_tool_arguments),
			),
		},
		"HasArgument": {
			// Member function on MCPTool type: tool.HasArgument(name) -> bool
			cel.MemberOverload("mcp_tool_has_argument",
				[]*cel.Type{types.NewObjectType("mcp.MCPTool"), cel.StringType},
				cel.BoolType,
				cel.BinaryBinding(impl.has_tool_argument),
			),
		},
		"GetArgument": {
			// Member function on MCPTool type: tool.GetArgument(name) -> any
			cel.MemberOverload("mcp_tool_argument",
				[]*cel.Type{types.NewObjectType("mcp.MCPTool"), cel.StringType},
				cel.DynType,
				cel.BinaryBinding(impl.get_tool_argument),
			),
		},
	}

	options := []cel.EnvOption{}
	for name, overloads := range libraryDecls {
		options = append(options, cel.Function(name, overloads...))
	}

	return env.Extend(options...)
}
