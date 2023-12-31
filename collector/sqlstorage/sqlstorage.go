package sqlstorage

import (
	"encoding/json"
	"github.com/abel-yang/crawler/collector"
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/sqldb"
	"go.uber.org/zap"
)

type SqlStore struct {
	dataDocker  []*collector.DataCell //分批输出结果缓存
	columnNames []sqldb.Field         // 标题字段
	db          sqldb.DBer
	Table       map[string]struct{}
	options
}

func New(opts ...Option) (*SqlStore, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	s := &SqlStore{}
	s.options = options
	s.Table = make(map[string]struct{})
	var err error
	s.db, err = sqldb.New(sqldb.WithSqlUrl(s.sqlUrl), sqldb.WithLogger(s.logger))
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SqlStore) Save(dataCells ...*collector.DataCell) error {
	for _, cell := range dataCells {
		name := cell.GetTableName()
		if _, ok := s.Table[name]; !ok {
			//创建表
			columnNames := getFields(cell)
			err := s.db.CreateTable(sqldb.TableData{
				TableName:   name,
				ColumnNames: columnNames,
				AutoKey:     true,
			})
			if err != nil {
				s.logger.Error("create table failed", zap.Error(err))
			}
			s.Table[name] = struct{}{}
		}
		if len(s.dataDocker) >= s.BatchCount {
			err := s.Flush()
			if err != nil {
				s.logger.Error("flush db failed", zap.Error(err))
			}
		}
		s.dataDocker = append(s.dataDocker, cell)
	}
	return nil
}

func getFields(cell *collector.DataCell) []sqldb.Field {
	taskName := cell.GetTaskName()
	ruleName := cell.Data["Rule"].(string)
	fields := engine.GetFields(taskName, ruleName)

	var columnNames []sqldb.Field
	for _, field := range fields {
		columnNames = append(columnNames, sqldb.Field{
			Title: field,
			Type:  "MEDIUMTEXT",
		})
	}
	columnNames = append(columnNames,
		sqldb.Field{Title: "Url", Type: "VARCHAR(255)"},
		sqldb.Field{Title: "Time", Type: "VARCHAR(255)"},
	)
	return columnNames
}

func (s *SqlStore) Flush() error {
	if len(s.dataDocker) == 0 {
		return nil
	}
	args := make([]interface{}, 0)
	for _, cell := range s.dataDocker {
		ruleName := cell.Data["Rule"].(string)
		taskName := cell.Data["Task"].(string)
		fields := engine.GetFields(taskName, ruleName)
		data := cell.Data["Data"].(map[string]interface{})
		value := []string{}
		for _, field := range fields {
			v := data[field]
			switch v.(type) {
			case nil:
				value = append(value, "")
			case string:
				value = append(value, v.(string))
			default:
				j, err := json.Marshal(v)
				if err != nil {
					value = append(value, "")
				} else {
					value = append(value, string(j))
				}
			}
		}
		value = append(value, cell.Data["Url"].(string), cell.Data["Time"].(string))
		for _, v := range value {
			args = append(args, v)
		}
	}

	return s.db.Insert(sqldb.TableData{
		TableName:   s.dataDocker[0].GetTableName(),
		ColumnNames: getFields(s.dataDocker[0]),
		Args:        args,
		DataCount:   len(s.dataDocker),
	})
}
