/*
Copyright the Sonobuoy contributors 2022

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package discovery

import (
	"bufio"
	"io/fs"
	"os"
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
)

const (
	//String format for the regex to match all pod log files inside the podlog directory
	//Equivalent to a shell glob podlogs/*/logs/*.txt
	logFilePatternPodlogsString = `^podlogs/[^/]+/[^/]+/logs/.+\.txt`

	logPatternNameErrors         = "Errors"
	logPatternFailedString       = `[fF]ailed`
	logPatternErrorString        = `[eE]rror`
	logPatternErrorCodeString    = `^E[0-9]+`
	logPatternLevelErrorString   = `level=error`
	logPatternNameWarnings       = "Warnings"
	logPatternWarningString      = `[wW]arn`
	logPatternWarningCodeString  = `^W[0-9]+`
	logPatternLevelWarningString = `level=warn`
)

// LogPattern is a struct that defines a class of log patterns,
// a LogPatterns instance will contain one or more file name patterns, stored in filePathPattern
// and another list of patterns that are meant to be matched against the content of files whose name matches the filePathPattern.
// both filePathPattern and matchPatterns are one or more compiled regular expressions
//
// A match for a pattern defined in LogPatterns will happen if:
// a certain file has a path that matches at least one of the regex in filePathPattern
// and
// at least one of the lines in the content of this file matches at least one
// of the regex patterns in matchPattern
type LogPattern struct {
	filePathPattern []*regexp.Regexp
	matchPatterns   []*regexp.Regexp
}

// LogPatterns maps the name of a set of patterns to its components
// The LogPattern structure defines the components of a set of patterns
type LogPatterns map[string]LogPattern

type LogSummary map[string]LogHitCounter

type LogHitCounter map[string]int

// GetDefaultLogPatterns returns the default set of log patterns that can be used with ReadLogSummary
func GetDefaultLogPatterns() LogPatterns {
	return LogPatterns{
		logPatternNameErrors: LogPattern{
			[]*regexp.Regexp{
				regexp.MustCompilePOSIX(logFilePatternPodlogsString),
			},
			[]*regexp.Regexp{
				regexp.MustCompilePOSIX(logPatternFailedString),
				regexp.MustCompilePOSIX(logPatternErrorString),
				regexp.MustCompilePOSIX(logPatternErrorCodeString),
				regexp.MustCompilePOSIX(logPatternLevelErrorString),
			},
		},
		logPatternNameWarnings: LogPattern{
			[]*regexp.Regexp{
				regexp.MustCompilePOSIX(logFilePatternPodlogsString),
			},
			[]*regexp.Regexp{
				regexp.MustCompilePOSIX(logPatternWarningString),
				regexp.MustCompilePOSIX(logPatternWarningCodeString),
				regexp.MustCompilePOSIX(logPatternLevelWarningString),
			},
		},
	}
}

func getPatternNamesForfile(relFilePath string, patterns LogPatterns) []string {
	result := make([]string, 0)

	for patternName, pattern := range patterns {
		for _, fileNameRegex := range pattern.filePathPattern {
			if fileNameRegex.MatchString(relFilePath) {
				result = append(result, patternName)
			}
		}
	}
	return result
}

// ReadLogSummary will recursively scan the tarballRootDir
// looking for files with names matching pre-defined regular expressions
// and scanning the content of these files for pre-defined error matching regexes
// and counting the number of hits
// The return value is a map of results with key the type of condition
// and the values of the map are the file names and the hit count
// the patterns parameter is a list of LogPatterns objects that define what to scan for
// The GetDefaultLogPatterns can be used to obtain such list.
// Errors encountered while scanning the directory are logged but no error will be returned.
func ReadLogSummary(r *results.Reader, patterns LogPatterns) (LogSummary, error) {
	logSummary := make(LogSummary)

	findAndScanLogFiles := func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			patternsForFile := getPatternNamesForfile(filePath, patterns)
			//If the patternsForFile list is empty it means this file is not interesting
			if len(patternsForFile) == 0 {
				return nil
			}

			file, err := os.Open(filePath)
			if err != nil {
				logrus.Errorf("findAndScanLogFiles: ignoring file '%s' because scanning it failed: %s", filePath, err)
				return nil
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				for _, patternName := range patternsForFile {
					for _, pattern := range patterns[patternName].matchPatterns {
						if pattern.MatchString(line) {
							if _, ok := logSummary[patternName]; !ok {
								logSummary[patternName] = make(LogHitCounter)
							}
							logSummary[patternName][filePath]++
							//We can stop looping through patterns from this patternName for this line
							break
						}
					}
				}
			}

		}
		return nil
	}
	err := r.WalkFiles(findAndScanLogFiles)
	if err != nil {
		logrus.Errorf("Failed to scan log files: %s", err)
	}
	return logSummary, nil
}

// ReadLogSummaryWithDefaultPatterns is a wrapper to ReadLogSummary + GetDefaultLogPatterns
func ReadLogSummaryWithDefaultPatterns(r *results.Reader) (LogSummary, error) {
	return ReadLogSummary(r, GetDefaultLogPatterns())
}
