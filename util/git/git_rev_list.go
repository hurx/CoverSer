package git

import (
	"CoverSer/util/cmd"
	"bufio"
	"os"
)

func decodeGitRevList(filepath_ string) (string, error) {
	fo, err := os.Open(filepath_)
	if err != nil {
		return "", err
	}
	defer fo.Close()
	reader := bufio.NewReader(fo)
	bytes, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	if len(bytes) == 0 {
		return "", nil
	}
	return string(bytes), nil
}

func GetLastCommitId(dir string) (commitId string, err error) {
	tempLogFile := cmd.GitPreCommit(dir)
	return decodeGitRevList(tempLogFile)
}
