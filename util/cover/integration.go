package cover

import (
	"sort"
	"strings"

	"CoverSer/util/git"
)

// 通过覆盖率执行的行号 和 gitdiff 信息，生成新的覆盖率执行行号。
// 返回 新、旧两个版本的覆盖率行号
func GetNewDAList(da_list []int, gitdiff git.GitDiff) (src_da_list []int, dis_da_list []int) {
	delta := 0
	src_del_list := gitdiff.SrcDel
	dis_delta_list := gitdiff.DisDeltaInfo

	src_da_list = []int{}
	dis_da_list = []int{}

	for _, da_v := range da_list {
		flag := 1
		for _, src_del := range src_del_list {
			if da_v == src_del {
				flag = 0
				break
			}
		}
		if flag == 0 {
			continue
		}

		for dis_delta_i, dis_delta := range dis_delta_list {
			if (da_v + delta) >= dis_delta.Start {
				delta += dis_delta.Length
				dis_delta_list = dis_delta_list[dis_delta_i+1:]
			}
		}
		src_da_list = append(src_da_list, da_v)
		dis_da_list = append(dis_da_list, (da_v + delta))
	}
	return src_da_list, dis_da_list
}

// 将 coverinfo 的行号取出来
func GetDAList(c CoverInfo) []int {
	da_list := []int{}
	for k, _ := range c.DAList {
		da_list = append(da_list, k)
	}
	sort.Ints(da_list)
	return da_list
}

//  将新、旧覆盖率行号整合到一个文件中
func Integrate(c1 CoverInfo, c2 CoverInfo, src_da_list []int, dis_da_list []int) CoverInfo {
	if c2.SF == "" {
		c2 = c1
		c2.DAList = map[int]int{}
	}
	for i, _ := range dis_da_list {
		src_line_no := src_da_list[i]
		dis_line_no := dis_da_list[i]
		if _, ok := c2.DAList[dis_line_no]; ok {
			c2.DAList[dis_line_no] += c1.DAList[src_line_no]
		} else {
			if c2.DAList == nil {
				c2.DAList = make(map[int]int)
			}
			c2.DAList[dis_line_no] = c1.DAList[src_line_no]
		}
	}
	return c2
}

// 通过两个commitid 和 git diff的内容生成新的覆盖率文件
func CombineCoverInfo(c1 CoverInfo, c2 CoverInfo, git_diff git.GitDiff) CoverInfo {
	da_list := GetDAList(c1)
	src_da_list, dis_da_list := GetNewDAList(da_list, git_diff)
	NewCover := Integrate(c1, c2, src_da_list, dis_da_list)
	return NewCover
}

// every file create a LineUpInfo
type LineUpInfo struct {
	CoverInfo1 CoverInfo
	CoverInfo2 CoverInfo
	GitDiff    git.GitDiff
}

// map key is file_path, values are cover infos and git diff
// 将两次commit的覆盖率信息和gitdiff信息按文件整合到一起，方便计算
func LineUp(project string, cover_info_list_1 []*CoverInfo, cover_info_list_2 []*CoverInfo, git_diff_list []*git.GitDiff) map[string]*LineUpInfo {
	line_up_info := map[string]*LineUpInfo{}

	for _, cover_info := range cover_info_list_1 {
		cover_file_path := cover_info.SF
		file_path_list := strings.Split(cover_file_path, project)
		file_path := file_path_list[len(file_path_list)-1]
		if line_up_info[file_path] == nil {
			line_up_info[file_path] = &LineUpInfo{}
		}
		line_up_info[file_path].CoverInfo1 = *cover_info
	}
	for _, cover_info := range cover_info_list_2 {
		cover_file_path := cover_info.SF
		file_path_list := strings.Split(cover_file_path, project)
		file_path := file_path_list[len(file_path_list)-1]
		if line_up_info[file_path] == nil {
			line_up_info[file_path] = &LineUpInfo{}
		}
		line_up_info[file_path].CoverInfo2 = *cover_info
	}

	for _, git_diff := range git_diff_list {
		git_file_path := git_diff.FilePath
		file_path := git_file_path[1:]
		if line_up_info[file_path] == nil {
			line_up_info[file_path] = &LineUpInfo{}
		}
		line_up_info[file_path].GitDiff = *git_diff
	}
	return line_up_info
}

// from cover info list  and git diff get coverinfo
// 项目名称、两次覆盖率数据、gitdiff信息生成出整合后的覆盖率文件
func CombineCoverInfos(project string, cover_info_list_1 []*CoverInfo, cover_info_list_2 []*CoverInfo, git_diff_list []*git.GitDiff) []*CoverInfo {
	line_up_info := LineUp(project, cover_info_list_1, cover_info_list_2, git_diff_list)

	var cover_list []*CoverInfo
	//cover_list := []CoverInfo{}
	for _, info := range line_up_info {
		if info.CoverInfo1.SF == "" && info.CoverInfo2.SF == "" {
			continue
		}
		cover_info := CombineCoverInfo(info.CoverInfo1, info.CoverInfo2, info.GitDiff)
		cover_list = append(cover_list, &cover_info)
	}
	return cover_list
}
