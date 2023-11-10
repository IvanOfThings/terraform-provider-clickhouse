package common

func MapArrayInterfaceToArrayOfStrings(in []interface{}) []string {
	ret := make([]string, 0)
	for _, s := range in {
		ret = append(ret, s.(string))
	}
	return ret
}
