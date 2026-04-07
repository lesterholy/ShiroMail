package database

import "testing"

func TestParseCreateIndexRecognizesFulltextIndex(t *testing.T) {
	indexName, tableName, ok := parseCreateIndex(
		"CREATE FULLTEXT INDEX ft_messages_search ON messages (from_addr, subject, text_preview)",
	)
	if !ok {
		t.Fatal("expected FULLTEXT index statement to be recognized")
	}
	if indexName != "ft_messages_search" {
		t.Fatalf("expected index name ft_messages_search, got %q", indexName)
	}
	if tableName != "messages" {
		t.Fatalf("expected table name messages, got %q", tableName)
	}
}
