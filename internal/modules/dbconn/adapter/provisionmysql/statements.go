package provisionmysql

import (
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

func statements(name, password string) ([]string, error) {
	quoted, err := quoteIdentifier(name)
	if err != nil {
		return nil, err
	}
	account := quoteAccount(name)
	secret := quoteLiteral(password)
	grantName := strings.ReplaceAll(quoted, "_", `\_`)
	return []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoted),
		fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED BY %s", account, secret),
		fmt.Sprintf("ALTER USER %s IDENTIFIED BY %s", account, secret),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO %s", grantName, account),
	}, nil
}

func quoteIdentifier(value string) (string, error) {
	if !domain.ValidDatabaseIdentifier(value) {
		return "", domain.ErrInvalidDatabaseIdentifier
	}
	return "`" + strings.ReplaceAll(value, "`", "``") + "`", nil
}

func quoteAccount(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'@'%'"
}

func quoteLiteral(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "'", "''")
	return "'" + value + "'"
}
