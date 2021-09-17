package cover

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type CoverInfo struct {
	DAList   map[int]int    // key: line_no , val: execute times
	SF       string         // file path
	FNList   map[string]int // key : function name ,val : start line
	FNDAList map[string]int // key : function name, val : execute times
	FNF      int            // function counts
	FNH      int            // function execute counts
	LF       int            // line counts
	LH       int            // line execute counts
}

// 通过 .info 文件，解析成 coverinfo 列表
func Decode(file_path string) []*CoverInfo {
	f, err := os.Open(file_path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	br := bufio.NewReader(f)

	cover_info_list := []*CoverInfo{}
	cover_info := &CoverInfo{}

	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		match_sf, _ := regexp.Match("^SF:.*", a)
		if match_sf {
			// if have SF , create a new coverinfo
			cover_info = &CoverInfo{}
			cover_info_list = append(cover_info_list, cover_info)
			line_info := Separate(string(a))
			cover_info.SF = line_info[0]
		}
		match_da, _ := regexp.Match("^DA:.*", a)
		if match_da {
			line_info := Separate(string(a))
			if len(line_info) < 2 {
				continue
			}
			line_no_str := line_info[0]
			exe_times_str := line_info[1]
			line_no, _ := strconv.Atoi(line_no_str)
			exe_times, _ := strconv.Atoi(exe_times_str)
			if cover_info.DAList == nil {
				cover_info.DAList = make(map[int]int)
			}
			cover_info.DAList[line_no] = exe_times
		}

		match_fn, _ := regexp.Match("^FN:.*", a)
		if match_fn {
			line_info := Separate(string(a))
			start_line, _ := strconv.Atoi(line_info[0])
			func_name := line_info[1]
			if cover_info.FNList == nil {
				cover_info.FNList = make(map[string]int)
			}
			cover_info.FNList[func_name] = start_line
		}
		match_fnda, _ := regexp.Match("^FNDA:.*", a)
		if match_fnda {
			line_info := Separate(string(a))
			execute_times, _ := strconv.Atoi(line_info[0])
			func_name := line_info[1]
			if cover_info.FNDAList == nil {
				cover_info.FNDAList = make(map[string]int)
			}
			cover_info.FNDAList[func_name] = execute_times
		}
		match_fnf, _ := regexp.Match("^FNF:.*", a)
		if match_fnf {
			line_info := Separate(string(a))
			func_counts, _ := strconv.Atoi(line_info[0])
			cover_info.FNF = func_counts
		}
		match_fnh, _ := regexp.Match("^FNH:.*", a)
		if match_fnh {
			line_info := Separate(string(a))
			func_exe_counts, _ := strconv.Atoi(line_info[0])
			cover_info.FNH = func_exe_counts
		}
		match_lf, _ := regexp.Match("^LF:.*", a)
		if match_lf {
			line_info := Separate(string(a))
			line_counts, _ := strconv.Atoi(line_info[0])
			cover_info.LF = line_counts
		}
		match_lh, _ := regexp.Match("^LH:.*", a)
		if match_lh {
			line_info := Separate(string(a))
			line_exe_counts, _ := strconv.Atoi(line_info[0])
			cover_info.LH = line_exe_counts
		}
	}
	return cover_info_list
}

func Separate(s string) []string {
	tmp_list := strings.Split(s, ":")
	return strings.Split(tmp_list[1], ",")
}
