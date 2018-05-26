package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	goflags "github.com/jessevdk/go-flags"
)

type FlagsStruct struct {
	Recursive bool `short:"r"`
	Verbose   bool `short:"v"`
}

var flags FlagsStruct

func main() {
	var (
		args []string
		err  error
	)
	parser := goflags.NewParser(&flags, goflags.HelpFlag|goflags.PassDoubleDash|goflags.IgnoreUnknown)
	if args, err = parser.ParseArgs(os.Args[1:]); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	switch len(args) {
	case 2:
		var (
			src         = args[0]
			dst         = args[1]
			dstFile     *os.File
			srcFile     *os.File
			srcFileInfo os.FileInfo
			dstFileInfo os.FileInfo
		)

		if srcFile, err = os.Open(src); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], src, err)
			os.Exit(1)
		} else if srcFileInfo, err = srcFile.Stat(); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], src, err)
			os.Exit(1)
		} else if srcFileInfo.IsDir() && !flags.Recursive {
			fmt.Fprintf(os.Stderr, "%s %s: is a directory (not copied).\n", os.Args[0], src)
			os.Exit(1)
		}
		defer srcFile.Close()

		if dstFileInfo, err = os.Stat(dst); err != nil {
			if os.IsNotExist(err) {
				if srcFileInfo.IsDir() {
					if err = os.Mkdir(dst, 0770); err != nil {
						fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dst, err)
						os.Exit(1)
					}
					if dstFile, err = os.Open(dst); err != nil {
						fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dst, err)
						os.Exit(1)
					}
				} else if dstFile, err = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0660); err != nil {
					fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dst, err)
					os.Exit(1)
				}
				if dstFileInfo, err = dstFile.Stat(); err != nil {
					fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dst, err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dst, err)
				os.Exit(1)
			}
		} else if dstFileInfo.IsDir() {
			if dstFile, err = os.Open(dst); err != nil {
				fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dst, err)
				os.Exit(1)
			}
		} else {
			if dstFile, err = os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0660); err != nil {
				fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dst, err)
				os.Exit(1)
			}
		}

		if dstFileInfo.IsDir() && srcFileInfo.IsDir() {
			copyDirToDir(dstFile, srcFile)
		} else if dstFileInfo.IsDir() && !srcFileInfo.IsDir() {
			copyFileToDir(dstFile, srcFile)
		} else if !dstFileInfo.IsDir() && srcFileInfo.IsDir() {
			fmt.Fprintf(os.Stderr, "%s %s: Not a directory\n", os.Args[0], dst)
			os.Exit(1)
		} else {
			copyFileToFile(dstFile, srcFile)
		}
	default:
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}

func copyDirToDir(dstDir, srcDir *os.File) {
	err := filepath.Walk(srcDir.Name(), func(srcFilePath string, srcFileInfo os.FileInfo, err error) error {
		var (
			dstFileInfo      os.FileInfo
			srcFile, dstFile *os.File
		)

		relativePath := strings.TrimLeft(srcFilePath, srcDir.Name())
		relativePath = strings.TrimLeft(relativePath, "/")
		dstFilePath := filepath.Join(dstDir.Name(), relativePath)

		if srcFileInfo.IsDir() {
			if dstFileInfo, err = os.Stat(dstFilePath); err != nil {
				if os.IsNotExist(err) {
					if err = os.Mkdir(dstFilePath, 0770); err != nil {
						fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dstFilePath, err)
						return filepath.SkipDir
					}
				} else {
					fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dstFilePath, err)
					return filepath.SkipDir
				}
			} else if !dstFileInfo.IsDir() {
				fmt.Fprintf(os.Stderr, "%s %s: Not a directory\n", os.Args[0], dstFilePath)
				return filepath.SkipDir
			}
		} else {
			if srcFile, err = os.Open(srcFilePath); err != nil {
				fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], srcFilePath, err)
				return nil
			}
			defer srcFile.Close()

			if dstFileInfo, err = os.Stat(dstFilePath); err != nil {
				if os.IsNotExist(err) {
					if dstFile, err = os.OpenFile(dstFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0660); err != nil {
						fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dstFilePath, err)
						return nil
					}
				} else {
					fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dstFilePath, err)
					return nil
				}
			} else if dstFileInfo.IsDir() {
				fmt.Fprintf(os.Stderr, "%s %s: cannot overwrite directory %s with non-directory %s\n", os.Args[0], dstFilePath, srcFilePath)
				return nil
			} else if dstFile, err = os.OpenFile(dstFilePath, os.O_WRONLY|os.O_TRUNC, 0660); err != nil {
				fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dstFilePath, err)
				return nil
			}
			defer dstFile.Close()

			copyFileToFile(dstFile, srcFile)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dstDir.Name(), err)
	}
}

func copyFileToDir(dstDir, srcFile *os.File) {
	fileName := filepath.Base(srcFile.Name())
	dstFilePath := filepath.Join(dstDir.Name(), fileName)
	if dstFile, err := os.OpenFile(dstFilePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0660); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], dstFilePath, err)
	} else {
		copyFileToFile(dstFile, srcFile)
	}
}

func copyFileToFile(dstFile, srcFile *os.File) {
	var (
		srcFileInfo os.FileInfo
		err         error
	)
	if flags.Verbose {
		fmt.Printf("%s -> %s\n", srcFile.Name(), dstFile.Name())
	}

	var statfs syscall.Statfs_t
	if err = syscall.Statfs(srcFile.Name(), &statfs); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], srcFile.Name(), err)
		return
	}

	if srcFileInfo, err = srcFile.Stat(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", os.Args[0], srcFile.Name(), err)
		return
	}

	buf := make([]byte, statfs.Bsize)
	for copied := int64(0); copied < srcFileInfo.Size(); copied += int64(len(buf)) {
		if srcFileInfo.Size()-copied < int64(statfs.Bsize) {
			buf = buf[0:(srcFileInfo.Size() - copied)]
		}
		if _, err = srcFile.ReadAt(buf, copied); err != nil {
			// Ignore IO Error, and leave dstFile a hole
		} else if _, err = dstFile.WriteAt(buf, copied); err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: Write Error: %s\n", os.Args[0], dstFile.Name(), err)
		}
	}
}
