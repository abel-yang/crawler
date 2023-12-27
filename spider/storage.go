package spider

type Storage interface {
	Save(dcs ...*DataCell) error
}

type DataCell struct {
	Data map[string]interface{}
}

func (d *DataCell) GetTableName() string {
	return d.Data["table"].(string)
}

func (d *DataCell) GetTaskName() string {
	return d.Data["Task"].(string)
}
