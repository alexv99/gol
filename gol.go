//
// MIT License
//
// Copyright (c) 2017 Alex Vauthey
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

package gol

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DEBUG = 0
const INFO = 1
const WARN = 2
const ERROR = 3
const FATAL = 5

const NUM_LOGGING_ROUTINES = 5

var levels = map[int]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var running bool = false

var aLoglevel int = INFO           // Log level
var aLogFolder string = "/var/log" // Path to gol file
var aLogMaxSize int64 = 1024       // in KB
var aLogMaxAge int = 10            // File older than MaxAge days will be deleted automatically
var aLogSuffix int = 0
var aLogName = "application.log"

var pLogFolder string = "/var/log" // Path to gol file
var pLogMaxSize int64 = 1024       // in KB
var pLogMaxAge int = 10            // File older than MaxAge will be deleted automatically
var pLogSuffix int = 0
var pLogName = "access.log"

var startStopMutex = sync.Mutex{}
var aFileRotateLock = sync.RWMutex{}
var pFileRotateLock = sync.RWMutex{}

var appLogChan chan string
var publicLogChan chan string

var appLogFile *os.File
var publicLogFile *os.File

var currentDate = time.Now().Local().Format("2006-01-02")

var logToStdOut = true

var showLineNumbers = true

var wg sync.WaitGroup

var aRotateCounter int
var pRotateCounter int

func Start() error {

	startStopMutex.Lock()
	defer startStopMutex.Unlock()

	if running {
		return nil
	}

	appLogChan = make(chan string, 1000)
	publicLogChan = make(chan string)

	var err error

	aLogSuffix = 0
	appLogFile, err = openLogFile(aLogFolder, aLogName)
	if err != nil {
		return err
	}

	publicLogFile, err = openLogFile(pLogFolder, pLogName)
	if err != nil {
		return err
	}

	running = true

	for i := 0; i < NUM_LOGGING_ROUTINES; i++ {
		go appLogWrite(appLogChan)             // App log write routine
		go publicAccessLogWrite(publicLogChan) // Public access log write routine
	}

	go purgeFiles(aLogFolder, aLogName, aLogMaxAge) // App log purge routine
	go purgeFiles(pLogFolder, pLogName, pLogMaxAge) // Public log purge routine

	return nil
}

func Stop() {

	startStopMutex.Lock()
	defer startStopMutex.Unlock()

	running = false

	close(appLogChan)
	close(publicLogChan)

	wg.Wait()
}

func Debug(v ...interface{}) {

	if !running {
		return
	}

	if s := decorateAppLogEntry(DEBUG, v); s != "" {
		appLogChan <- s
	}
}

func Info(v ...interface{}) {

	if !running {
		return
	}

	if s := decorateAppLogEntry(INFO, v); s != "" {
		appLogChan <- s
	}
}

func Warn(v ...interface{}) {

	if !running {
		return
	}

	if s := decorateAppLogEntry(WARN, v); s != "" {
		appLogChan <- s
	}
}

func Error(v ...interface{}) {

	if !running {
		return
	}

	if s := decorateAppLogEntry(ERROR, v); s != "" {
		appLogChan <- s
	}
}

// Logs the message synchronously and terminates the app with exit code 1.
func Fatal(v ...interface{}) {
	if !running {
		return
	}

	if message := decorateAppLogEntry(FATAL, v); message != "" {
		doAppLogWrite(message)
		os.Exit(1)
	}
}

func Public(req http.Request, statusCode int, contentLength int, duration time.Duration) {
	publicLogChan <- decoratePublicAccessLogEntry(req, statusCode, contentLength, duration)
}

func SetAppLogFolder(path string) {
	aLogFolder = path
}

func SetAppLogMaxSize(size int64) {
	aLogMaxSize = size
}

func SetAppLogMaxAge(age int) {
	aLogMaxAge = age
}

func SetPublicLogFolder(path string) {
	pLogFolder = path
}

func SetPublicLogMaxSize(size int64) {
	pLogMaxSize = size
}

func SetPublicLogMaxAge(age int) {
	pLogMaxAge = age
}

func LogToStdout(b bool) {
	logToStdOut = b
}

func ShowLineNumbers(b bool) {
	showLineNumbers = b
}

func SetAppLogLevel(level int) {
	if level != DEBUG && level != INFO && level != WARN && level != ERROR {
		log.Fatal("Ivalid gol level " + string(level))
	}
	aLoglevel = level
}

func appLogWrite(appDataChannel chan string) {

	wg.Add(1)
	defer wg.Done()

	var more bool = true
	var msg string = ""

	for more {
		msg, more = <-appDataChannel
		if msg != "" {
			err := doAppLogWrite(msg)

			if err != nil {
				log.Println("Unable to log message ["+msg+"]", err)
			}
		}
	}
}

func publicAccessLogWrite(publicDataChannel chan string) {

	wg.Add(1)
	defer wg.Done()

	var more bool = true
	var msg string = ""

	for more {
		msg, more = <-publicDataChannel
		if msg != "" {
			err := doPublicAccessLogWrite(msg)

			if err != nil {
				log.Println("Unable to log message ["+msg+"]", err)
			}
		}
	}
}

func doAppLogWrite(msg string) (err error) {

	aRotateCounter++

	if aRotateCounter <= 10 {
		aRotateCounter = 0
		aFileRotateLock.Lock()
		if needRotation(appLogFile, aLogMaxSize) {
			appLogFile.Close()
			newLogFile, err := rotate(aLogFolder, aLogName, &aLogSuffix)
			if err != nil {
				log.Println("ERROR - Rotation required and unable to create file ", err)
			} else {
				appLogFile = newLogFile
			}
		}
		aFileRotateLock.Unlock()
	}

	if logToStdOut {
		log.Print(msg)
	}

	aFileRotateLock.RLock()
	appLogFile.Write([]byte(msg))
	aFileRotateLock.RUnlock()

	return nil
}

func doPublicAccessLogWrite(msg string) (err error) {

	pRotateCounter++

	if pRotateCounter <= 10 {
		pRotateCounter = 0
		pFileRotateLock.Lock()
		if needRotation(publicLogFile, pLogMaxSize) {
			publicLogFile.Close()
			newLogFile, err := rotate(pLogFolder, pLogName, &pLogSuffix)
			if err != nil {
				log.Println("ERROR - Rotation required and unable to create file ", err)
			} else {
				publicLogFile = newLogFile
			}
		}
		pFileRotateLock.Unlock()
	}

	if logToStdOut {
		log.Print(msg)
	}

	pFileRotateLock.RLock()
	publicLogFile.Write([]byte(msg))
	pFileRotateLock.RUnlock()

	return nil
}

func needRotation(f *os.File, maxSize int64) bool {

	fileInfo, err := f.Stat()

	if err != nil {
		log.Println("ERROR - Unable to stat file "+f.Name(), err)
		return false
	}

	if fileInfo.Size() > (maxSize * 1024) { // Max size reached
		return true
	}

	return false
}

func purgeFiles(folder string, suffix string, maxAge int) {

	for running {

		then := time.Now().AddDate(0, 0, 0-maxAge)
		files, err := ioutil.ReadDir(folder)
		if err != nil {
			log.Println("ERROR: Purge routine unable to read directory ["+folder+"]", err)
		}
		for _, f := range files {
			if strings.HasSuffix(f.Name(), suffix) {
				if f.ModTime().Before(then) {
					path := folder + "/" + f.Name()
					err := os.Remove(path)
					if err != nil {
						log.Println("ERROR: Purge routine unable to remove file ["+path+"]", err)
					} else {
						log.Println("Purge routine removed file [" + path + "]")
					}
				}
			}
		}
		time.Sleep(1 * time.Minute)
	}
}

func openLogFile(folder string, aLogName string) (logFile *os.File, err error) {

	os.MkdirAll(folder, 0744)

	fileName := folder + "/" + aLogName

	logFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.FileMode(0644))
	if err != nil {
		return nil, err
	}

	return logFile, err
}

func rotate(folder string, fileName string, fileNumber *int) (logFile *os.File, err error) {

	now := time.Now().Local().Format("2006-01-02")

	os.MkdirAll(folder, 0744)

	var rotated bool = false

	for !rotated {
		archiveFilePath := folder + "/" + now + "-" + strconv.Itoa(*fileNumber) + "-" + fileName
		currentFilePath := folder + "/" + fileName

		_, err = os.Stat(archiveFilePath)

		if os.IsNotExist(err) {
			err = os.Rename(currentFilePath, archiveFilePath)

			if err != nil {
				log.Println("Error while rotating, unable to rename [" + currentFilePath + "] to [" + archiveFilePath + "]")
				return nil, err
			}

			logFile, err = os.OpenFile(currentFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.FileMode(0644))

			if err != nil {
				log.Println("Error while rotating, unable to create/open [" + fileName + "]")
				return nil, err
			}

			rotated = true

		} else if err != nil {
			log.Println("Error while rotating, unable to stat ["+archiveFilePath+"]", err)
			return nil, err
		}
		*fileNumber++
	}

	return logFile, nil
}

func decorateAppLogEntry(level int, v []interface{}) string {

	if aLoglevel > level {
		return ""
	}

	msg := time.Now().Format("2006-01-02 15:04:05") + " " + levels[level] + " " + fmt.Sprint(v)

	if showLineNumbers {
		_, file, line, _ := runtime.Caller(2)
		msg += " at " + file + ":" + strconv.Itoa(line) + "\n"
	}

	return msg
}
func decoratePublicAccessLogEntry(r http.Request, status int, contentLength int, d time.Duration) string {
	ns := int64(d)
	μs := int64(d / time.Microsecond)
	ms := int64(d / time.Millisecond)

	fromIp := r.Header.Get("X-Forwarded-For")

	if strings.TrimSpace(fromIp) == "" {
		fromIp = r.RemoteAddr
	}

	message := time.Now().Format("2006-01-02 15:04:05") + " "
	message += r.Method + " " + fmt.Sprint(r.URL) + " " + r.Proto + " from [" + fromIp + "] with agent [" + r.Header.Get("User-Agent") + "]"

	if ms > 0 {
		message += " in " + strconv.FormatInt(ms, 10) + "ms => " + strconv.Itoa(status)
	} else if μs > 0 {
		message += " in " + strconv.FormatInt(μs, 10) + "μs => " + strconv.Itoa(status)
	} else {
		// Very fast computer ;)
		message += " in " + strconv.FormatInt(ns, 10) + "ns => " + strconv.Itoa(status)
	}

	message += " with " + strconv.Itoa(contentLength) + " bytes \n"

	return message
}
