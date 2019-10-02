package log

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type logFile struct {
	curFile      *os.File
	fileName     string
	sizeFlag     bool
	timeFlag     bool
	compressFlag bool
	filePath     string
	sizeValue    int64
	todayDate    string
	msgQueue     chan string
	closed       bool
	cnt          uint32
}

type Option func(file *logFile)

func NewLogFile(options ...Option) *logFile {
	logfile := &logFile{
		fileName: "",
		sizeFlag: false,
		timeFlag: false,
		closed:   false,
		msgQueue: make(chan string, 1000),
		cnt:      0,
	}

	for _, option := range options {
		option(logfile)
	}

	absPath, _ := filepath.Abs(os.Args[0])
	path := filepath.Dir(absPath)
	name := filepath.Base(absPath)
	logfile.todayDate = time.Now().Format("2006-01-02")
	//
	if logfile.fileName == "" {
		logfile.fileName = name + ".log"
	}

	if logfile.filePath == "" {
		logfile.filePath = path + string(filepath.Separator) + "log" + string(filepath.Separator)
	}

	os.MkdirAll(logfile.filePath, os.ModePerm)

	file, err := os.OpenFile(logfile.filePath+logfile.fileName,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}

	logfile.curFile = file

	go logfile.worker()

	return logfile
}

//设置文件名
func FileName(fileName string) Option {
	return func(file *logFile) {
		file.fileName = fileName
	}
}

//设置文件路径
func FilePath(path string) Option {
	return func(file *logFile) {
		var slash string = string(os.PathSeparator)
		Path := strings.TrimRight(path, slash)
		dir, _ := filepath.Abs(Path)
		file.filePath = dir + slash
	}
}

//设置文件切割大小
func FileSize(size int) Option {
	return func(file *logFile) {
		file.sizeFlag = true
		file.sizeValue = int64(size) * 1024 * 1024
	}
}

//按照天来切割
func FileTime(flag bool) Option {
	return func(file *logFile) {
		file.timeFlag = true
	}
}

func FileCompress(flag bool) Option {
	return func(file *logFile) {
		file.compressFlag = flag
	}
}

//
func (f *logFile) Write(p []byte) (n int, err error) {
	str := (*string)(unsafe.Pointer(&p))
	f.msgQueue <- (*str)
	//fmt.Println(*str)
	return len(p), nil
}

//切割文件
func (f *logFile) doRotate() {

	defer func() {
		rec := recover()
		if rec != nil {
			fmt.Println("doRotate %v", rec)
		}
	}()

	if f.curFile == nil {
		fmt.Println("doRotate curFile nil,return")
		return
	}
	prefile := f.curFile
	_, err := prefile.Stat()
	var prefileName string = ""
	if err == nil {
		filePath := f.filePath + f.fileName
		f.closed = true
		err := prefile.Close()
		if err != nil {
			fmt.Println("doRotate close err", err.Error())
		}
		y, m, d := time.Now().Date()
		f.cnt++
		prefileName = filePath + "." + fmt.Sprintf("%.4d%.2d%.2d", y, m, d) + strconv.FormatInt(int64(f.cnt), 10)
		err = os.Rename(filePath, prefileName)
	}

	if f.fileName != "" {
		nextFile, err := os.OpenFile(f.filePath+f.fileName,
			os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

		if err != nil {
			fmt.Println(err.Error())
		}
		f.closed = false
		f.curFile = nextFile
		nowDate := time.Now().Format("2006-01-02")
		f.todayDate = nowDate
	}

	if f.compressFlag == true {
		go f.compressFile(prefileName, prefileName+".gz")
	}
}

func (f *logFile) worker() {
	for {
		select {
		case msg := <-f.msgQueue:
			{
				if f.closed == false {
					f.curFile.WriteString(msg)
					if f.sizeFlag == true {
						curInfo, _ := os.Stat(f.filePath + f.fileName)
						if curInfo.Size() >= f.sizeValue {
							f.doRotate()
						}
					}
					nowDate := time.Now().Format("2006-01-02")
					if f.timeFlag == true &&
						nowDate != f.todayDate {
						f.doRotate()
					}
				}

			}
		}

	}
}

func (f *logFile) compressFile(Src string, Dst string) error {
	defer func() {
		rec := recover()
		if rec != nil {
			fmt.Println(rec)
		}
	}()
	newfile, err := os.Create(Dst)
	if err != nil {
		return err
	}
	defer newfile.Close()

	file, err := os.Open(Src)
	if err != nil {
		return err
	}

	zw := gzip.NewWriter(newfile)

	filestat, err := file.Stat()
	if err != nil {
		return nil
	}

	zw.Name = filestat.Name()
	zw.ModTime = filestat.ModTime()
	_, err = io.Copy(zw, file)
	if err != nil {
		return nil
	}

	zw.Flush()
	if err := zw.Close(); err != nil {
		return nil
	}
	file.Close()
	os.Remove(Src)
	return nil
}
