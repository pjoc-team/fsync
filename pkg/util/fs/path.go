package fs

import "strings"

// FileName return file name of path
func FileName(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}
