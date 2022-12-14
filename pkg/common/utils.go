package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ch "github.com/leprosus/golang-clickhouse"
)

type storedComment struct {
	cluster string
	comment string
}

func GetComment(comment string, cluster string) string {
	storingComment := fmt.Sprintf(`{"comment":"%v","cluster":"%v"}`, comment, cluster)
	storingComment = strings.Replace(storingComment, "'", "\\'", -1)
	return storingComment
}

func UnmarshalComment(storedComment string) (comment string, cluster string, err error) {
	storedComment = strings.Replace(storedComment, "\\'", "'", -1)

	byteStreamComment := []byte(storedComment)

	var dat map[string]interface{}

	if err := json.Unmarshal(byteStreamComment, &dat); err != nil {
		return "", "", err
	}
	comment = dat["comment"].(string)
	cluster = dat["cluster"].(string)

	return comment, cluster, err
}

func toString(result ch.Result, field string, errors *[]error) *string {

	value, err := result.String(field)
	if err != nil {
		*errors = append(*errors, err)
		return nil
	}
	if value == "\\N" {
		return nil
	}
	return &value
}

func toUint64(result ch.Result, field string, errors *[]error) *uint64 {

	value, err := result.String(field)
	if err != nil {
		*errors = append(*errors, err)
		return nil
	}
	if value == "\\N" {
		return nil
	}
	valueUint, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		*errors = append(*errors, err)
		return nil
	}
	return &valueUint
}

func GetClusterStatement(cluster string, defaultCluster string) (clusterStatement string, clusterToUse string) {
	clusterToUse = defaultCluster
	if cluster != "" {
		clusterToUse = cluster
	}
	clusterStatement = ""
	if clusterToUse != "" {
		clusterStatement = "ON CLUSTER " + clusterToUse
	}
	return clusterStatement, clusterToUse
}
