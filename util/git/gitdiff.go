package git

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"CoverSer/util/cmd"
)

type GitDiff struct {
	FilePath     string
	SrcDel       []int
	DisAdd       []int
	DisDeltaInfo []DisDelta
}

//type GitFile struct {
//StartLine int
//EndLine   int
//}

// 解析 git diff 生成的文件
func DecodeGitFile(file_path string) []*GitDiff {
	f, err := os.Open(file_path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	br := bufio.NewReader(f)

	var gitdiff_list []*GitDiff

	var cur_gitdiff *GitDiff

	src_cur_line := 0
	dis_cur_line := 0

	// 1 start to count number, 0 not
	flag := 0

	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		match, _ := regexp.Match("^diff --git.*", a)
		if match {
			flag = 0
			// if a new file , create another gitdiff
			cur_gitdiff = &GitDiff{}
			tmp_list := strings.Split(string(a), " ")
			file_path := tmp_list[2]
			cur_gitdiff.FilePath = file_path
			gitdiff_list = append(gitdiff_list, cur_gitdiff)
			continue
		}

		match, _ = regexp.Match("^@@.*@@.*", a)
		if match {
			flag = 1
			tmp_list := strings.Split(string(a), " ")

			src_list := strings.Split(string(tmp_list[1]), ",")
			src_start := src_list[0][1:]

			dis_list := strings.Split(string(tmp_list[2]), ",")
			dis_start := dis_list[0][1:]

			// if get @@ label , cur line set to diff start line-1
			src_cur_line, _ = strconv.Atoi(src_start)
			src_cur_line -= 1
			dis_cur_line, _ = strconv.Atoi(dis_start)
			dis_cur_line -= 1
			continue
		}
		match_p, _ := regexp.Match("^+++.*", a)
		match_s, _ := regexp.Match("^---.*", a)
		if flag == 1 {
			if string(string(a)[0]) == "+" && !(match_p || match_s) {
				dis_cur_line += 1
				cur_gitdiff.DisAdd = append(cur_gitdiff.DisAdd, dis_cur_line)
			} else if string(string(a)[0]) == "-" && !(match_p || match_s) {
				src_cur_line += 1
				cur_gitdiff.SrcDel = append(cur_gitdiff.SrcDel, src_cur_line)
			} else {
				dis_cur_line += 1
				src_cur_line += 1
			}
		}
	}

	return gitdiff_list
}

//var (
//	fileDifReg = regexp.MustCompile(`^diff --git (.+) (.+)$`)
//	sourceLineReg = regexp.MustCompile(`^(---|\+\+\+) (.+)$`)
//	EffectLineReg = regexp.MustCompile(`^@@ (.+),(.+) (.+),(.+) @@`)
//)
//
//func getAbsInt(num_ string)int{
//	data,_ := strconv.Atoi(num_)
//	if data < 0{ // 由于这个值不会很大 所以不存在 -math.MinInt64
//		return -data
//	}
//	return data
//}
//
//func DecodeGitFileNew(filePath string) []*GitDiff {
//	f, err := os.Open(filePath)
//	if err != nil {
//		panic(err)
//	}
//	defer f.Close()
//
//	br := bufio.NewReader(f)
//
//	var gitdiff_list []*GitDiff
//
//	var cur_gitdiff *GitDiff
//
//	src_cur_line := 0
//	dis_cur_line := 0
//
//	for {
//		line_, _, c := br.ReadLine()
//		if c == io.EOF {
//			break
//		}
//		lineStr := string(line_)
//		m := fileDifReg.FindStringSubmatch(lineStr) //diff --git
//		if len(m) > 0 {
//			cur_gitdiff = &GitDiff{}
//			fmt.Printf("find 1match:%s\n", m)
//			for index_, m_ := range m {
//				fmt.Printf("1index: %d:%s\n", index_, m_)
//			}
//			cur_gitdiff.FilePath = strings.Replace(m[2], "b/","",1) // 最新commit的文件名
//			continue
//		}
//		if strings.HasPrefix(lineStr, "index "){ // no means line: index b4468c8..ca2851b 100644
//			continue
//		}
//		m2 := sourceLineReg.FindStringSubmatch(lineStr) // ---/+++ a/DoTask/dotask.go
//		if len(m2) > 0 {
//			continue
//		}
//		m3 := EffectLineReg.FindStringSubmatch(lineStr) // @@ -25,11 +25,12 @@
//		if len(m3) > 0 {
//			src_cur_line = getAbsInt(m3[1])
//			dis_cur_line = getAbsInt(m3[3])
//			continue
//		}
//		// 下面就是代码行
//		if cur_gitdiff.FilePath != "" && len(cur_gitdiff.DisAdd) == 0 && len(cur_gitdiff.SrcDel) == 0 {
//			if strings.HasPrefix(lineStr, "+"){
//				cur_gitdiff.DisAdd = append(cur_gitdiff.DisAdd, dis_cur_line)
//				dis_cur_line +=1
//			}else if strings.HasPrefix(lineStr, "-"){
//				cur_gitdiff.SrcDel = append(cur_gitdiff.SrcDel, dis_cur_line)
//				dis_cur_line +=1
//			}else{
//				dis_cur_line += 1
//				src_cur_line += 1
//			}
//		}
//	}
//
//	return gitdiff_list
//}

type DisDelta struct {
	Start  int
	Length int
}

// 通过 git diff 中增加的行列表，计算出新增行的开始和步长，用于计算不同 commit 之间的覆盖行号变化
func GetDisDelta(dis_list []int) []DisDelta {
	start := 0
	length := 0
	disdelta_list := []DisDelta{}
	for k, v := range dis_list {
		if k == 0 {
			start = v
			length = 1
		} else {
			if (dis_list[k] - dis_list[k-1]) == 1 {
				length += 1
			} else {
				disdelta := DisDelta{
					Start:  start,
					Length: length,
				}
				disdelta_list = append(disdelta_list, disdelta)
				start = v
				length = 1
			}

		}
	}
	disdelta := DisDelta{
		Start:  start,
		Length: length,
	}
	disdelta_list = append(disdelta_list, disdelta)
	return disdelta_list
}

// 通过两个 commitid 获取到 git diff 的信息
func GetGitDiff(workspace string, commit_1 string, commit_2 string) []*GitDiff {
	git_diff_file := cmd.GitDiff(workspace, commit_1, commit_2)
	git_diff_list := DecodeGitFile(git_diff_file)
	return git_diff_list
}
