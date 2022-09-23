package common

import (
	"encoding/json"

	ch "github.com/leprosus/golang-clickhouse"
)

type DataSourceDbsItem struct {
	DbName       string `json:"db_name"`
	Engine       string `json:"engine"`
	DataPath     string `json:"data_path"`
	MetadataPath string `json:"metadata_path"`
	Uuid         string `json:"uuid"`
	Comment      string `json:"comment"`
}

func MapDbData(result ch.Result) (map[string]string, error) {

	name, err := result.String("name")
	if err != nil {
		return nil, err
	}
	engine, err := result.String("engine")
	if err != nil {
		return nil, err
	}
	data_path, err := result.String("data_path")
	if err != nil {
		return nil, err
	}
	metadata_path, err := result.String("metadata_path")
	if err != nil {
		return nil, err
	}
	uuid, err := result.String("uuid")
	if err != nil {
		return nil, err
	}
	comment, err := result.String("comment")
	if err != nil {
		return nil, err
	}

	input := DataSourceDbsItem{
		DbName:       name,
		Engine:       engine,
		DataPath:     data_path,
		MetadataPath: metadata_path,
		Uuid:         uuid,
		Comment:      comment,
	}

	var db map[string]string
	streamBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(streamBytes, &db)
	if err != nil {
		return nil, err
	}
	return db, nil
}
