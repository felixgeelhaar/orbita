package mcp

import (
	"testing"

	"github.com/felixgeelhaar/mcp-go"
	"github.com/felixgeelhaar/mcp-go/testutil"
	"github.com/felixgeelhaar/orbita/adapter/cli"
	"github.com/stretchr/testify/require"
)

func TestRegisterCLITools_ListTools(t *testing.T) {
	srv := mcp.NewServer(mcp.ServerInfo{
		Name:    "test",
		Version: "1.0.0",
		Capabilities: mcp.Capabilities{
			Tools: true,
		},
	})

	app := &cli.App{}
	require.NoError(t, RegisterCLITools(srv, ToolDependencies{App: app}))

	tc := testutil.NewTestClient(t, srv)
	defer tc.Close()

	tools, err := tc.ListTools()
	require.NoError(t, err)

	found := false
	for _, tool := range tools {
		if tool["name"] == "cli.health" {
			found = true
			break
		}
	}
	require.True(t, found, "cli.health tool should be registered")
}
