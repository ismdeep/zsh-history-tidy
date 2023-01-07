package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Record struct {
	UnixTime string
	Command  string
}

func Unmarshal(line string) (*Record, error) {
	if len(line) <= 2 {
		return nil, errors.New("bad request")
	}

	if line[:2] != ": " {
		return nil, errors.New("bad request")
	}

	line = line[2:]

	unixTime := line[:strings.Index(line, ":")]
	if len(unixTime) != 10 {
		return nil, errors.New("invalid time format")
	}

	line = line[11:]

	if len(line) <= 2 {
		return nil, errors.New("invalid line format")
	}
	if line[:2] != "0;" {
		return nil, errors.New("invalid line format")
	}

	return &Record{
		UnixTime: unixTime,
		Command:  strings.TrimSpace(line[2:]),
	}, nil
}

func (r *Record) Marshal() string {
	return fmt.Sprintf(": %v:0;%v", r.UnixTime, r.Command)
}

func main() {
	v := viper.New()

	mainCommand := &cobra.Command{
		Use:   "zsh-history-tidy",
		Short: "zsh-history-tidy",
		Run: func(cmd *cobra.Command, args []string) {
			filePath := v.GetString("from")

			outputPath := ""
			if v.GetBool("overwrite") {
				outputPath = filePath
			} else {
				// Save
				userHomeDir, err := os.UserHomeDir()
				if err != nil {
					panic(err)
				}

				folder := fmt.Sprintf("%v/.mathematician42/cache/zsh-history-tidy/%v", userHomeDir, time.Now().UnixNano())
				if err := os.MkdirAll(folder, 0750); err != nil {
					panic(err)
				}
				outputPath = fmt.Sprintf("%v/zsh_history", folder)
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				panic(err)
			}

			lines := strings.Split(string(content), "\n")

			s := make(map[string]bool)
			cnt := 0
			var records []Record
			for _, line := range lines {
				r, err := Unmarshal(line)
				if err != nil {
					continue
				}

				if _, ok := s[r.Command]; ok {
					continue
				}

				s[r.Command] = true

				records = append(records, *r)

				cnt++
			}

			f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0750)
			if err != nil {
				panic(err)
			}

			// 排序
			sortField := v.GetString("sort")
			switch sortField {
			case "command":
				sort.Slice(records, func(i, j int) bool {
					return strings.Compare(records[i].Command, records[j].Command) <= 0
				})
			case "time":
				sort.Slice(records, func(i, j int) bool {
					return strings.Compare(records[i].UnixTime, records[j].UnixTime) <= 0
				})
			default:
				panic(fmt.Sprintf("invalid sort field. [%v]", sortField))
			}

			// 写入文件
			for _, record := range records {
				if _, err := f.WriteString(fmt.Sprintf("%v\n", record.Marshal())); err != nil {
					panic(err)
				}
			}

			fmt.Println("Output:", outputPath)
			fmt.Println("cnt:", cnt)
		},
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	mainCommand.Flags().String("from", fmt.Sprintf("%v/Documents/Data/zsh_history", userHomeDir), "from zsh history file")
	mainCommand.Flags().Bool("overwrite", false, "overwrite source file")
	mainCommand.Flags().StringP("sort", "s", "time", "--sort, -s <sort field>, e.g. time, command")
	_ = v.BindPFlags(mainCommand.Flags())

	if err := mainCommand.Execute(); err != nil {
		os.Exit(1)
	}
}
