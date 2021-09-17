package cover

import (
	"fmt"
	"os"
	"strconv"

	"CoverSer/util/git"
)

// 根据 CoverInfo 列表计算出覆盖率
func CoverRate(cover_info_list []*CoverInfo) float64 {
	covered_line := 0
	all_line := 0
	for _, cover_info := range cover_info_list {
		onefile_covered_line := 0
		onefile_all_line := 0
		for _, da := range cover_info.DAList {
			if da > 0 {
				onefile_covered_line += 1
				covered_line += 1
			}
			onefile_all_line += 1
			all_line += 1
		}
		cover_info.LH = onefile_covered_line
		cover_info.LF = onefile_all_line
	}
	rate := float64(covered_line) / float64(all_line)
	rate_2, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", rate), 64)
	return rate_2
}

// 将 coverinfo 的列表结构转换成 .info 文件，输出到 file_path 文件中
func EncodeInfo(file_path string, cover_info_list []*CoverInfo) {
	file, err := os.OpenFile(file_path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("open file error", err.Error())
	}
	defer file.Close()
	for _, cover_info := range cover_info_list {
		file.Write([]byte("TN:\n"))
		sf := "SF:" + cover_info.SF + "\n"
		file.Write([]byte(sf))
		for fn_name, fn_line := range cover_info.FNList {
			fn := "FN:" + strconv.Itoa(fn_line) + "," + fn_name + "\n"
			file.Write([]byte(fn))
		}
		for fnda_name, fnda_times := range cover_info.FNDAList {
			fnda := "FNDA:" + strconv.Itoa(fnda_times) + "," + fnda_name + "\n"
			file.Write([]byte(fnda))
		}

		fnf := "FNF:" + strconv.Itoa(cover_info.FNF) + "\n"
		file.Write([]byte(fnf))
		fnh := "FNH:" + strconv.Itoa(cover_info.FNH) + "\n"
		file.Write([]byte(fnh))
		for da_line, da_times := range cover_info.DAList {
			da := "DA:" + strconv.Itoa(da_line) + "," + strconv.Itoa(da_times) + "\n"
			file.Write([]byte(da))
		}
		lf := "LF:" + strconv.Itoa(cover_info.LF) + "\n"
		file.Write([]byte(lf))
		lh := "LH:" + strconv.Itoa(cover_info.LH) + "\n"
		file.Write([]byte(lh))

		file.Write([]byte("end_of_record\n"))
	}
}

// 根据 项目名称 、 coverinfo 的列表 和 commitid 生成增量覆盖率 (相较于master)
func GetIncreaseCover(git_diff_list []*git.GitDiff, project string, cover_info_list []*CoverInfo, commitid string) (float64, []*CoverInfo) {
	cover_info_nouse := []*CoverInfo{}
	covered_lines := 0
	all_lines := 0
	line_up_info_list := LineUp(project, cover_info_list, cover_info_nouse, git_diff_list)
	cover_info_incre_list := []*CoverInfo{}
	for _, line_up_info := range line_up_info_list {
		cover_info_incre := line_up_info.CoverInfo1
		cover_info_incre.DAList = map[int]int{}
		for _, v := range line_up_info.GitDiff.DisAdd {
			all_lines += 1
			if _, ok := line_up_info.CoverInfo1.DAList[v]; ok {
				if line_up_info.CoverInfo1.DAList[v] != 0 {
					covered_lines += 1
					cover_info_incre.DAList[v] = line_up_info.CoverInfo1.DAList[v]
				} else {
					cover_info_incre.DAList[v] = 0
				}
			} else {
				cover_info_incre.DAList[v] = 0
			}
		}
		// DA list 大于预置的1行，才算有覆盖率数据，否则为空， 不加入队列
		if len(cover_info_incre.DAList) > 0 {
			cover_info_incre_list = append(cover_info_incre_list, &cover_info_incre)
		}
	}
	// 减掉覆盖率插桩的行
	covered_lines = covered_lines - 3
	all_lines = all_lines - 5
	if covered_lines < 0 || all_lines < 0 {
		return 0, cover_info_incre_list
	}
	cover_rate := float64(covered_lines) / float64(all_lines)
	cover_rate, _ = strconv.ParseFloat(fmt.Sprintf("%.4f", cover_rate), 64)
	return cover_rate, cover_info_incre_list
}
