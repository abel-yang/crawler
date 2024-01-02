package sqlstorage

import (
	"encoding/json"
	"errors"
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/spider"
	"github.com/abel-yang/crawler/sqldb"
	"go.uber.org/zap"
)

type SqlStorage struct {
	dataDocker  []*spider.DataCell //分批输出结果缓存
	columnNames []sqldb.Field      // 标题字段
	db          sqldb.DBer
	Table       map[string]struct{}
	options
}

func New(opts ...Option) (*SqlStorage, error) {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	s := &SqlStorage{}
	s.options = options
	s.Table = make(map[string]struct{})
	var err error
	s.db, err = sqldb.New(sqldb.WithConnURL(s.sqlUrl), sqldb.WithLogger(s.logger))
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SqlStorage) Save(dataCells ...*spider.DataCell) error {
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

func getFields(cell *spider.DataCell) []sqldb.Field {
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

func (s *SqlStorage) Flush() error {
	if len(s.dataDocker) == 0 {
		return nil
	}
	args := make([]interface{}, 0)
	var ruleName string
	var taskName string
	var ok bool
	for _, cell := range s.dataDocker {
		if ruleName, ok = cell.Data["Rule"].(string); !ok {
			return errors.New("no rule field")
		}
		if taskName, ok = cell.Data["Task"].(string); !ok {
			return errors.New("no task field")
		}
		fields := engine.GetFields(taskName, ruleName)
		data := cell.Data["Data"].(map[string]interface{})
		var value []string
		for _, field := range fields {
			v := data[field]
			switch v := v.(type) {
			case nil:
				value = append(value, "")
			case string:
				value = append(value, v)
			default:
				j, err := json.Marshal(v)
				if err != nil {
					value = append(value, "")
				} else {
					value = append(value, string(j))
				}
			}
		}
		if v, ok := cell.Data["URL"].(string); ok {
			value = append(value, v)
		}
		if v, ok := cell.Data["Time"].(string); ok {
			value = append(value, v)
		}
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
