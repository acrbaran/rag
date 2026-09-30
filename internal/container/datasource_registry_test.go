package container

import (
	"testing"

	"github.com/acrbaran/rag/internal/datasource"
	"github.com/acrbaran/rag/internal/types"
)

func TestConnectorRegistryOnlySupportsRemainingSources(t *testing.T) {
	registry, err := initConnectorRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{types.ConnectorTypeNotion, types.ConnectorTypeConfluence, types.ConnectorTypeRSS, types.ConnectorTypeGitLab} {
		if _, err := registry.Get(name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, name := range []string{"feishu", "lark", "feishu_drive", "lark_drive", "yuque", "dingtalk", "ima"} {
		if _, err := registry.Get(name); err != datasource.ErrConnectorNotFound {
			t.Errorf("%s remains registered: %v", name, err)
		}
	}
}
