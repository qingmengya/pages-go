package utils

// 传入字符串,将字符串中问号以此替换
func Replace(format string, values ...interface{}) (result string) {
	//计算所有?所在的索引位置
	var index_marks []int
	flag := 0
	for i, c := range format {
		str_c := string(c)
		if str_c == "?" {
			index_marks = append(index_marks, i)
			result += values[flag].(string)
			flag++
		} else {
			result += str_c
		}
	}
	return
}
