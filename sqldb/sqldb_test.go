package sqldb

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSqlDB_CreateTable(t *testing.T) {
	sqldb, err := New(
		WithConnURL("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
	)

	assert.Nil(t, err)
	assert.NotNil(t, sqldb)
	// 测试对于无效的配置返回错误
	name := "test_create_table"
	var notValidTable = TableData{
		TableName: name,
		ColumnNames: []Field{
			{Title: "书名", Type: "notValid"},
			{Title: "URL", Type: "VARCHAR(255)"},
		},
		AutoKey: true,
	}

	// 测试对于无效的配置返回错误
	err = sqldb.CreateTable(notValidTable)
	assert.NotNil(t, err)

	// 测试对于有效的配置返回错误
	var validTable = TableData{
		TableName: name,
		ColumnNames: []Field{
			{Title: "书名", Type: "MEDIUMTEXT"},
			{Title: "URL", Type: "VARCHAR(255)"},
		},
		AutoKey: true,
	}

	//延迟删除表
	defer func() {
		err := sqldb.DropTable(validTable)
		assert.Nil(t, err)
	}()
	err = sqldb.CreateTable(validTable)
	assert.Nil(t, err)
}

func TestSqlDB_CreateTableDriver(t *testing.T) {
	type args struct {
		t TableData
	}
	name := "test_create_table"
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "create_not_valid_table",
		args: args{TableData{
			TableName: name,
			ColumnNames: []Field{
				{Title: "书名", Type: "notValid"},
				{Title: "URL", Type: "VARCHAR(255)"},
			},
			AutoKey: true,
		}},
		wantErr: true,
	},
		{
			name: "create_valid_table",
			args: args{TableData{
				TableName: name,
				ColumnNames: []Field{
					{Title: "书名", Type: "MEDIUMTEXT"},
					{Title: "URL", Type: "VARCHAR(255)"},
				},
				AutoKey: true,
			}},
			wantErr: false,
		},
		{
			name: "create_valid_table_with_primary_key",
			args: args{TableData{
				TableName: name,
				ColumnNames: []Field{
					{Title: "书名", Type: "MEDIUMTEXT"},
					{Title: "URL", Type: "VARCHAR(255)"},
				},
				AutoKey: true,
			}},
			wantErr: true,
		},
	}

	sqldb, err := New(WithConnURL("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"))
	for _, tt := range tests {
		err = sqldb.CreateTable(tt.args.t)
		if tt.wantErr {
			assert.NotNil(t, err, tt.name)
		} else {
			assert.Nil(t, err, tt.name)
		}
		sqldb.DropTable(tt.args.t)
	}
}

func TestSqldb_InsertTable(t *testing.T) {
	type args struct {
		t TableData
	}
	tableName := "test_create_table"
	columnNames := []Field{{Title: "书名", Type: "MEDIUMTEXT"}, {Title: "price", Type: "TINYINT"}}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "insert_data",
			args: args{TableData{
				TableName:   tableName,
				ColumnNames: columnNames,
				Args:        []interface{}{"book1", 2},
				DataCount:   1,
			}},
			wantErr: false,
		},
		{
			name: "insert_multi_data",
			args: args{TableData{
				TableName:   tableName,
				ColumnNames: columnNames,
				Args:        []interface{}{"book3", 88.88, "book4", 99.99},
				DataCount:   2,
			}},
			wantErr: false,
		},
		{
			name: "insert_multi_data_wrong_count",
			args: args{TableData{
				TableName:   tableName,
				ColumnNames: columnNames,
				Args:        []interface{}{"book3", 88.88, "book4", 99.99},
				DataCount:   1,
			}},
			wantErr: true,
		},
		{
			name: "insert_wrong_data_type",
			args: args{TableData{
				TableName:   tableName,
				ColumnNames: columnNames,
				Args:        []interface{}{"book2", "rrr"},
				DataCount:   1,
			}},
			wantErr: false,
		},
	}

	sqldb, err := New(
		WithConnURL("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
	)
	err = sqldb.CreateTable(tests[0].args.t)
	defer sqldb.DropTable(tests[0].args.t)
	assert.Nil(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = sqldb.Insert(tt.args.t)
			if tt.wantErr {
				assert.NotNil(t, err, tt.name)
			} else {
				assert.Nil(t, err, tt.name)
			}
		})
	}
}
