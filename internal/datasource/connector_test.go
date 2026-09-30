package datasource

import "testing"

func TestAvailableConnectorsExcludeRemovedSources(t *testing.T) {
	for _, meta := range ListAvailableConnectors() {
		switch meta.Type {
		case "feishu", "lark", "feishu_drive", "lark_drive", "yuque", "dingtalk", "ima":
			t.Fatalf("removed connector %s is still available", meta.Type)
		}
	}
}
