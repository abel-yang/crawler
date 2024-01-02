package sqldb

import (
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"strings"
)

type DBer interface {
	CreateTable(t TableData) error
	Insert(t TableData) error
}

type Field struct {
	Title string
	Type  string
}

type TableData struct {
	TableName   string
	ColumnNames []Field       //标题字段
	Args        []interface{} //数据
	DataCount   int           //插入数据的数据
	AutoKey     bool
}

type SqlDB struct {
	options
	db *sql.DB
}

func (d *SqlDB) OpenDB() error {
	db, err := sql.Open("mysql", d.sqlUrl)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(2048)
	db.SetMaxIdleConns(2048)
	if err := db.Ping(); err != nil {
		return err
	}
	d.db = db
	return nil
}

func New(opts ...Option) (*SqlDB, error) {
	options := defaultOption
	for _, opt := range opts {
		opt(&options)
	}
	d := &SqlDB{}
	d.options = options
	if err := d.OpenDB(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *SqlDB) CreateTable(t TableData) error {
	if len(t.ColumnNames) == 0 {
		return errors.New("column can not be empty")
	}
	dynamicSql := `CREATE TABLE IF NOT EXISTS ` + t.TableName + " ("
	if t.AutoKey {
		dynamicSql += `id INT(12) NOT NULL PRIMARY KEY AUTO_INCREMENT,`
	}
	for _, col := range t.ColumnNames {
		dynamicSql += col.Title + ` ` + col.Type + `,`
	}
	dynamicSql = dynamicSql[:len(dynamicSql)-1] + `) ENGINE=MyISAM DEFAULT CHARSET=UTF8MB4;`
	d.logger.Debug("create table", zap.String("dynamic_sql", dynamicSql))
	_, err := d.db.Exec(dynamicSql)
	return err
}

func (d *SqlDB) DropTable(t TableData) error {
	if len(t.ColumnNames) == 0 {
		return errors.New("column can not be empty")
	}
	dropSql := `DROP TABLE ` + t.TableName + ";"
	d.logger.Debug("drop table", zap.String("sql", dropSql))
	_, err := d.db.Exec(dropSql)
	return err
}

func (d *SqlDB) Insert(t TableData) error {
	if len(t.ColumnNames) == 0 {
		return errors.New("Column can not be empty")
	}

	dynamicSql := `INSERT INTO ` + t.TableName + `(`
	for _, col := range t.ColumnNames {
		dynamicSql += col.Title + ","
	}

	dynamicSql = dynamicSql[:len(dynamicSql)-1] + `) VALUES `
	//每一行记录值占位符
	blank := ",(" + strings.Repeat(",?", len(t.ColumnNames))[1:] + ")"
	dynamicSql += strings.Repeat(blank, t.DataCount)[1:] + `;`
	d.logger.Debug("insert table", zap.String("dynamicSql", dynamicSql))
	_, err := d.db.Exec(dynamicSql, t.Args)
	return err
}
