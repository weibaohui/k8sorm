package utils

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

func TruncateString(s string, length int) string {
	if utf8.RuneCountInString(s) <= length {
		return s
	}

	// Convert the string to a slice of runes
	runes := []rune(s)
	return string(runes[:length])
}

func ToInt(s string) int {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return id
}

func ToIntDefault(s string, i int) int {
	id, err := strconv.Atoi(s)
	if err != nil {
		return i
	}
	return id
}

func ToUInt(s string) uint {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	if id > math.MaxUint {
		return 0
	}
	return uint(id)
}

// ToIntSlice 将逗号分隔的数字字符串转换为 []int 切片
func ToIntSlice(ids string) []int {
	// 分割字符串
	strIds := strings.Split(ids, ",")
	var intIds []int

	// 遍历字符串数组并转换为整数
	for _, strId := range strIds {
		strId = strings.TrimSpace(strId) // 移除前后空格
		if id, err := strconv.Atoi(strId); err == nil {
			intIds = append(intIds, id)
		}
	}

	return intIds
}
func ToInt64Slice(ids string) []int64 {
	// 分割字符串
	strIds := strings.Split(ids, ",")
	var intIds []int64

	// 遍历字符串数组并转换为整数
	for _, strId := range strIds {
		strId = strings.TrimSpace(strId) // 移除前后空格
		if id, err := strconv.ParseInt(strId, 10, 64); err == nil {
			intIds = append(intIds, id)
		}
	}

	return intIds
}
func ToInt64(str string) int64 {
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
func IsTextFile(ob []byte) (bool, error) {

	n := len(ob)
	if n > 1024 {
		n = 1024
	}
	// 检查是否包含非文本字符
	if !utf8.Valid(ob[:n]) {
		return false, nil
	}

	// 检查是否包含空字节（\x00），空字节通常代表二进制文件
	if bytes.Contains(ob[:n], []byte{0}) {
		return false, nil
	}

	return true, nil
}

// SanitizeFileName 去除文件名中的非法字符，并替换为下划线 '_'
func SanitizeFileName(filename string) string {
	// 定义非法字符的正则表达式，包括 \ / : * ? " < > | 以及小括号 ()
	reg := regexp.MustCompile(`[\\/:*?"<>|()]+`)

	// 替换非法字符为下划线 '_'
	sanitizedFilename := reg.ReplaceAllString(filename, "_")

	return sanitizedFilename
}
func NormalizeNewlines(input string) string {
	// 将 Windows 风格的换行符 \r\n 替换为 Unix 风格 \n
	return strings.ReplaceAll(input, "\r\n", "\n")
}
func NormalizeToWindows(input string) string {
	// 先统一为 \n 再替换为 \r\n，防止重复替换出错
	unixNormalized := strings.ReplaceAll(input, "\r\n", "\n")
	return strings.ReplaceAll(unixNormalized, "\n", "\r\n")
}
