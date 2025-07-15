package internal

import (
	"bytes"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Version 版本号，格式：更新日期(8位).更新次数(累加)
const Version = "2025.1"

// AppURL site
const AppURL = "https://gitee.com/chrisx/mysql-sync"

const timeFormatStd string = "2006-01-02 15:04:05"

// loadJsonFile load json
func loadJSONFile(jsonPath string, val any) error {
	bs, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(bs), "\n")
	var bf bytes.Buffer
	for _, line := range lines {
		lineNew := strings.TrimSpace(line)
		if (len(lineNew) > 0 && lineNew[0] == '#') || (len(lineNew) > 1 && lineNew[0:2] == "//") {
			continue
		}
		bf.WriteString(lineNew)
	}
	return json.Unmarshal(bf.Bytes(), &val)
}

func inStringSlice(str string, strSli []string) bool {
	for _, v := range strSli {
		if str == v {
			return true
		}
	}
	return false
}

func simpleMatch(patternStr string, str string, msg ...string) bool {
	str = strings.TrimSpace(str)
	patternStr = strings.TrimSpace(patternStr)
	if patternStr == str {
		log.Println("simple_match:suc,equal", msg, "patternStr:", patternStr, "str:", str)
		return true
	}
	pattern := "^" + strings.ReplaceAll(patternStr, "*", `.*`) + "$"
	match, err := regexp.MatchString(pattern, str)
	if err != nil {
		log.Println("simple_match:error", msg, "patternStr:", patternStr, "pattern:", pattern, "str:", str, "err:", err)
	}
	return match
}

// url解码以支持密码中包含特殊字符
func decodePass(pass string) string {
	// 使用url.QueryUnescape解码
	decodes, err := url.QueryUnescape(pass)
	if err != nil {
		log.Fatalf("URL解码错误:", err)
	}
	return string(decodes)
}
