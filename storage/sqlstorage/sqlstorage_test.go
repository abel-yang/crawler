package sqlstorage

import (
	"github.com/abel-yang/crawler/engine"
	"github.com/abel-yang/crawler/parse/doubanbook"
	"github.com/abel-yang/crawler/parse/doubanggroup"
	"github.com/abel-yang/crawler/spider"
	"github.com/abel-yang/crawler/sqldb"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {
	engine.Store.Add(doubanbook.DoubanbookTask)
	engine.Store.Add(doubanggroup.DoubangroupTask)
	engine.Store.AddJSTask(doubanggroup.DoubangroupjsTask)
}

type mysqldb struct {
}

func (m mysqldb) CreateTable(t sqldb.TableData) error {
	return nil
}

func (m mysqldb) Insert(t sqldb.TableData) error {
	return nil
}

func TestSQLStorage_Flush(t *testing.T) {
	type fields struct {
		dataDocker []*spider.DataCell
		options    options
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "empty", wantErr: false},
		{name: "no Rule filed", fields: fields{dataDocker: []*spider.DataCell{
			{Data: map[string]interface{}{"url": "<http://xxx.com>"}},
		}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SqlStorage{
				dataDocker: tt.fields.dataDocker,
				db:         mysqldb{},
				options:    tt.fields.options,
			}
			if err := s.Flush(); (err != nil) != tt.wantErr {
				t.Errorf("flush() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.NotNil(t, s.dataDocker)
		})
	}
}
