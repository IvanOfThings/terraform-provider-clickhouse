package datasources

type CHDatabase struct {
	Name         string `json:"name" ch:"name"`
	Engine       string `json:"engine" ch:"engine"`
	DataPath     string `json:"data_path" ch:"data_path"`
	MetadataPath string `json:"metadata_path" ch:"metadata_path"`
	Uuid         string `json:"uuid" ch:"uuid"`
	Comment      string `json:"comment" ch:"comment"`
}