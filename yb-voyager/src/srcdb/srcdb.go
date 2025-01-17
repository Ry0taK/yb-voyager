package srcdb

import (
	"context"
	"fmt"

	"github.com/yugabyte/yb-voyager/yb-voyager/src/utils"
	"github.com/yugabyte/yb-voyager/yb-voyager/src/utils/sqlname"
)

type SourceDB interface {
	Connect() error
	GetTableRowCount(tableName string) int64
	GetTableApproxRowCount(tableProgressMetadata *utils.TableProgressMetadata) int64
	CheckRequiredToolsAreInstalled()
	GetVersion() string
	GetAllTableNames() []*sqlname.SourceName
	GetAllPartitionNames(tableName string) []string
	ExportSchema(exportDir string)
	ExportData(ctx context.Context, exportDir string, tableList []*sqlname.SourceName, quitChan chan bool, exportDataStart chan bool, exportSuccessChan chan bool)
	ExportDataPostProcessing(exportDir string, tablesProgressMetadata map[string]*utils.TableProgressMetadata)
	GetCharset() (string, error)
	FilterUnsupportedTables(tableList []*sqlname.SourceName) []*sqlname.SourceName
}

func newSourceDB(source *Source) SourceDB {
	switch source.DBType {
	case "postgresql":
		return newPostgreSQL(source)
	case "mysql":
		return newMySQL(source)
	case "oracle":
		return newOracle(source)
	default:
		panic(fmt.Sprintf("unknown source database type %q", source.DBType))
	}
}
