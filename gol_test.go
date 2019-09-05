package github.com/alexv99/gol

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"sync"
	"net/http"
)

func TestAppLogWrite(t *testing.T) {
	removeLogFiles(".", t)

	SetAppLogFolder(".")
	SetPublicLogFolder(".")
	LogToStdout(false)


	err := Start()

	if err != nil {
		log.Println(err)
		t.Fatal()
	}

	defer Stop()

	path := "./application.log"

	SetAppLogLevel(ERROR)

	Debug("debug1")
	if fileContains(path, "debug1", t) {
		t.Fail()
	}
	Info("info1")
	if fileContains(path, "info1", t) {
		t.Fail()
	}
	Warn("warning1")
	if fileContains(path, "warning1", t) {
		t.Fail()
	}
	Error("error1")
	if !fileContains(path, "error1", t) {
		t.Fail()
	}

	SetAppLogLevel(WARN)

	Debug("debug2")
	if fileContains(path, "debug2", t) {
		t.Fail()
	}
	Info("info2")
	if fileContains(path, "info2", t) {
		t.Fail()
	}
	Warn("warning2")
	if !fileContains(path, "warning2", t) {
		t.Fail()
	}
	Error("error2")
	if !fileContains(path, "error2", t) {
		t.Fail()
	}

	SetAppLogLevel(INFO)

	Debug("debug3")
	if fileContains(path, "debug3", t) {
		t.Fail()
	}
	Info("info3")
	if !fileContains(path, "info3", t) {
		t.Fail()
	}
	Warn("warning3")
	if !fileContains(path, "warning3", t) {
		t.Fail()
	}
	Error("error3")
	if !fileContains(path, "error3", t) {
		t.Fail()
	}

	SetAppLogLevel(DEBUG)

	Debug("debug4")
	if !fileContains(path, "debug4", t) {
		t.Fail()
	}
	Info("info4")
	if !fileContains(path, "info4", t) {
		t.Fail()
	}
	Warn("warning4")
	if !fileContains(path, "warning4", t) {
		t.Fail()
	}
	Error("error4")
	if !fileContains(path, "error4", t) {
		t.Fail()
	}
}

func TestPublicLogWrite(t *testing.T) {
	removeLogFiles(".", t)

	SetAppLogFolder(".")
	SetPublicLogFolder(".")
	LogToStdout(false)

	err := Start()
	defer Stop()

	if err != nil {
		log.Println(err)
		t.Fatal()
	}

	method := "GET"
	url := "http://www.deal.com/abc?p=xys"
	code := "200"

	req , err := http.NewRequest(method , url, nil)

	Public(*req,200, 10, 1*time.Millisecond)

	path := "./access.log"

	if !fileContains(path,method, t) {
		log.Println("Missing method from public access log entry")
		t.FailNow()
	}

	if !fileContains(path,url, t) {
		log.Println("Missing URL from public access log entry")
		t.FailNow()
	}

	if !fileContains(path,code, t) {
		log.Println("Missing http return code from public access log entry")
		t.FailNow()
	}

	if !fileContains(path,"1ms", t) {
		log.Println("Missing duration from public access log entry")
		t.FailNow()
	}
}

func TestAppLogRotate(t *testing.T) {
	removeLogFiles(".", t)

	SetAppLogFolder(".")
	SetPublicLogFolder(".")
	SetAppLogMaxSize(100)
	LogToStdout(false)

	err := Start()

	if err != nil {
		log.Println(err)
		t.Fatal()
	}

	defer Stop()

	path := "./application.log"

	SetAppLogLevel(INFO)
	LogToStdout(false)

	for j := 0; j < 50000; j++ {
		Info("Hello " + strconv.Itoa(j))
	}

	if !fileExists(path, t) {
		t.Fail()
	}

	for i :=0 ; i < 4; i++ {
		path = "./" + time.Now().Local().Format("2006-01-02") + "-" + strconv.Itoa(i) + "-application.log"
		if !fileExists(path, t) {
			t.Fail()
		}
	}
}

func TestPublicLogRotate(t *testing.T) {
	removeLogFiles(".", t)

	SetAppLogFolder(".")
	SetPublicLogFolder(".")
	SetPublicLogMaxSize(100)
	LogToStdout(false)

	err := Start()

	if err != nil {
		log.Println(err)
		t.Fatal()
	}

	defer Stop()


	SetAppLogLevel(INFO)
	LogToStdout(false)

	method := "GET"
	code := 200

	for j := 0; j < 10000; j++ {
		url := "http://www.deal.com/abc?p=xyz" + strconv.Itoa(j)
		req , _ := http.NewRequest(method , url, nil )
		Public(*req, code, 10, 1*time.Millisecond)
	}


	path := "./access.log"
	if !fileExists(path, t) {
		t.Fail()
	}

	for i :=0 ; i < 4; i++ {
		path = "./" + time.Now().Local().Format("2006-01-02") + "-" + strconv.Itoa(i) + "-access.log"
		if !fileExists(path, t) {
			t.Fail()
		}
	}
}

func TestAppLogMultiThreaded(t *testing.T) {

	removeLogFiles(".", t)

	SetAppLogFolder(".")
	SetPublicLogFolder(".")
	SetAppLogMaxSize(100)

	err := Start()

	if err != nil {
		log.Println(err)
		t.Fatal()
	}

	SetAppLogLevel(INFO)
	LogToStdout(false)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(j int) {
			for k := 0; k < 100; k++ {
				Info("Hello {" + strconv.Itoa(j) + "," + strconv.Itoa(k) + "}")
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	Stop()

	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			s := "{" + strconv.Itoa(i) + "," + strconv.Itoa(j) + "}"
			if !filesContains(".", s, t) {
				log.Println("Missing log record: " + s)
				t.FailNow()
			}
		}
	}
}

func TestPublicLogMultiThreaded(t *testing.T) {

	removeLogFiles(".", t)

	SetAppLogFolder(".")
	SetPublicLogFolder(".")
	SetPublicLogMaxSize(100)
	LogToStdout(false)

	err := Start()

	if err != nil {
		log.Println(err)
		t.Fatal()
	}

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(j int) {
			for k := 0; k < 100; k++ {
				req , _ := http.NewRequest("GET" , "http://www.deal.com?i=" + strconv.Itoa(j) + "&j=" + strconv.Itoa(k), nil )
				Public(*req, 200, 10, 1*time.Millisecond)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	Stop()

	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			s := "http://www.deal.com?i=" + strconv.Itoa(i) + "&j=" + strconv.Itoa(j)
			if !filesContains(".", s, t) {
				log.Println("Missing log record: " + s)
				t.FailNow()
			}
		}
	}
}

func removeLogFiles(path string, t *testing.T) {

	files, err := ioutil.ReadDir(path)

	if err != nil {
		log.Println("Unable to read dir  " + path, err)
		t.FailNow()
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(),".log") {
			err := os.Remove(path + "/" + f.Name())
			if err != nil {
				t.Fatal("Unable to remove log files before test", err)
			}
		}
	}
}

func fileExists(path string, t *testing.T) bool {

	for i := 0; i < 100; i++ {

		_, err := os.Stat(path)

		if err != nil {
			if !os.IsNotExist(err) {
				t.Fatal("Unable to check file existence "+path, err)
			}
			time.Sleep(1 * time.Millisecond)
		} else {
			return true
		}
	}
	return false
}

func fileContains(path string, s string, t *testing.T) bool {

	if fileExists(path, t) {
		for i := 0; i < 100; i++ {

			b, err := ioutil.ReadFile(path)

			if err != nil {
				log.Println("Unable to check file "+path+" contains "+s, err)
				t.FailNow()
			}

			fileContent := string(b)

			if strings.Contains(fileContent, s) {
				return true
			}
			time.Sleep(1 * time.Millisecond)
		}
	}
	return false
}

func filesContains(path string, s string, t *testing.T) bool {

	files, err := ioutil.ReadDir(path)

	if err != nil {
		log.Println("Unable to read dir  " + path, err)
		t.FailNow()
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".log") {
			b, err := ioutil.ReadFile(f.Name())

			if err != nil {
				log.Println("Unable to check file "+ f.Name() + " contains " + s, err)
				t.FailNow()
			}

			fileContent := string(b)

			if strings.Contains(fileContent, s) {
				return true
			}
		}
	}
	return false
}
