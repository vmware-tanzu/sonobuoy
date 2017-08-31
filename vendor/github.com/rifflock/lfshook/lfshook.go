// Package LFShook allows users to write to the logfiles using logrus.
package lfshook

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
)

// We are logging to file, strip colors to make the output more readable
var txtFormatter = &logrus.TextFormatter{DisableColors: true}

// Map for linking a log level to a log file
// Multiple levels may share a file, but multiple files may not be used for one level
type PathMap map[logrus.Level]string

// Alternatively map a log level to an io.Writer
type WriterMap map[logrus.Level]io.Writer

// Hook to handle writing to local log files.
type lfsHook struct {
	paths     PathMap
	writer    WriterMap
	levels    []logrus.Level
	lock      *sync.Mutex
	formatter logrus.Formatter
}

// Given a map with keys equal to log levels.
// We can generate our levels handled on the fly, and write to a specific file for each level.
// We can also write to the same file for all levels. They just need to be specified.
func NewHook(levelMap interface{}) *lfsHook {
	hook := &lfsHook{
		lock:      new(sync.Mutex),
		formatter: txtFormatter,
	}

	switch levelMap.(type) {
	case PathMap:
		hook.paths = levelMap.(PathMap)
		for level := range levelMap.(PathMap) {
			hook.levels = append(hook.levels, level)
		}
		break
	case WriterMap:
		hook.writer = levelMap.(WriterMap)
		for level := range levelMap.(WriterMap) {
			hook.levels = append(hook.levels, level)
		}
		break
	default:
		panic(fmt.Sprintf("unsupported level map type: %s", reflect.TypeOf(levelMap)))
	}

	return hook
}

// Replace the color stripped default formatter with a custom formatter
func (hook *lfsHook) SetFormatter(formatter logrus.Formatter) {
	hook.formatter = formatter

	switch hook.formatter.(type) {
	case *logrus.TextFormatter:
		textFormatter := hook.formatter.(*logrus.TextFormatter)
		textFormatter.DisableColors = true
	}
}

// Open the file, write to the file, close the file.
// Whichever user is running the function needs write permissions to the file or directory if the file does not yet exist.
func (hook *lfsHook) Fire(entry *logrus.Entry) error {
	if hook.writer != nil {
		return hook.ioWrite(entry)
	} else {
		return hook.fileWrite(entry)
	}
}

// Write a log line to an io.Writer
func (hook *lfsHook) ioWrite(entry *logrus.Entry) error {
	var (
		msg []byte
		err error
		ok  bool
	)

	hook.lock.Lock()
	defer hook.lock.Unlock()

	if _, ok = hook.writer[entry.Level]; !ok {
		err = fmt.Errorf("no writer provided for loglevel: %d", entry.Level)
		log.Println(err.Error())
		return err
	}

	// use our formatter instead of entry.String()
	msg, err = hook.formatter.Format(entry)

	if err != nil {
		log.Println("failed to generate string for entry:", err)
		return err
	}
	_, err = hook.writer[entry.Level].Write(msg)
	return err
}

// Write a log line directly to a file
func (hook *lfsHook) fileWrite(entry *logrus.Entry) error {
	var (
		fd   *os.File
		path string
		msg  []byte
		err  error
		ok   bool
	)

	hook.lock.Lock()
	defer hook.lock.Unlock()

	if path, ok = hook.paths[entry.Level]; !ok {
		err = fmt.Errorf("no file provided for loglevel: %d", entry.Level)
		log.Println(err.Error())
		return err
	}
	fd, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Println("failed to open logfile:", path, err)
		return err
	}
	defer fd.Close()

	// use our formatter instead of entry.String()
	msg, err = hook.formatter.Format(entry)

	if err != nil {
		log.Println("failed to generate string for entry:", err)
		return err
	}
	fd.Write(msg)
	return nil
}

// Return configured log levels
func (hook *lfsHook) Levels() []logrus.Level {
	return hook.levels
}
