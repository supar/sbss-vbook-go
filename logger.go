package main

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
)

// RFC5424 log message levels.
// 0       Emergency: system is unusable
// 1       Alert: action must be taken immediately
// 2       Critical: critical conditions
// 3       Error: error conditions
// 4       Warning: warning conditions
// 5       Notice: normal but significant condition
// 6       Informational: informational messages
// 7       Debug: debug-level messages
const (
	LevelEmergency = iota
	LevelAlert
	LevelCritical
	LevelError
	LevelWarning
	LevelNotice
	LevelInformational
	LevelDebug
)

// Logger interface
type LogIface interface {
	Critical(v ...interface{})
	Error(v ...interface{})
	LogIfaceInfo
}

type LogIfaceInfo interface {
	Warn(v ...interface{})
	Notice(v ...interface{})
	Info(v ...interface{})
	Debug(v ...interface{})
}

// Расширение стандарной библиотеки log
type Log struct {
	*log.Logger

	// Уровень логирования
	levelLog int
	syslog   *syslog.Writer
}

// Инициализируй логер
func NewLogger(name string, level int) (logger *Log) {
	var (
		err  error
		flag = log.Ldate | log.Ltime | log.Lmicroseconds
		prog = name + " "
	)

	logger = &Log{
		Logger: log.New(os.Stdout, prog, flag),
	}

	if level == 0 {
		if logger.syslog, err = syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, prog); err != nil {
			logger.Critical(err)
		} else {
			logger.SetOutput(logger.syslog)
		}
	}

	return
}

// Установи уровень логирования
func (this *Log) SetLevel(l int) {
	this.levelLog = l
}

// Поверь вхождение запрашиваемого уровня в допустимую
// границу логирования
func (this *Log) level(l int) bool {
	for i := LevelEmergency; i <= LevelDebug; i++ {
		if i == l {
			if i > this.levelLog {
				return false
			}
		}
	}
	return true
}

// Единая точка обработки входиящих сообщений согласно
// их уровню и установленной границы логирования
// К сообщению добавляется префикс описание уровня сообщения
func (this *Log) print(level int, v []interface{}) {
	var (
		ln int

		msg,
		prefix string
	)

	ln = len(v)

	if ln == 0 {
		return
	}

	prefix = getPrefix(level)

	switch v[0].(type) {
	case string:
		prefix += v[0].(string)
		v = v[1:]

		msg = fmt.Sprintf(prefix, v...)

	default:
		v = append(v[:1], v[0:]...)
		v[0] = prefix

		msg = fmt.Sprint(v...)
	}

	if this.syslog == nil {
		this.Print(msg)

		return
	}

	switch level {
	case LevelEmergency:
		this.syslog.Emerg(msg)
	case LevelAlert:
		this.syslog.Alert(msg)
	case LevelCritical:
		this.syslog.Crit(msg)
	case LevelError:
		this.syslog.Err(msg)
	case LevelWarning:
		this.syslog.Warning(msg)
	case LevelNotice:
		this.syslog.Notice(msg)
	case LevelInformational:
		this.syslog.Info(msg)
	case LevelDebug:
		this.syslog.Debug(msg)
	}
}

// Префик описание числового значения уровня
func getPrefix(level int) string {
	var prefix = "error"

	switch level {
	case LevelEmergency:
		prefix = "emergency"
	case LevelAlert:
		prefix = "alert"
	case LevelCritical:
		prefix = "critical"
	case LevelError:
		prefix = "error"
	case LevelWarning:
		prefix = "warning"
	case LevelNotice:
		prefix = "notice"
	case LevelInformational:
		prefix = "info"
	case LevelDebug:
		prefix = "debug"
	}

	return "[" + prefix + "] "
}

func (this *Log) Emergency(v ...interface{}) {
	this.print(LevelEmergency, v)
	os.Exit(1)
}

func (this *Log) Alert(v ...interface{}) {
	this.print(LevelAlert, v)
}

func (this *Log) Critical(v ...interface{}) {
	this.print(LevelCritical, v)
	os.Exit(1)
}

func (this *Log) Error(v ...interface{}) {
	this.print(LevelError, v)
}

func (this *Log) Warn(v ...interface{}) {
	this.print(LevelWarning, v)
}

func (this *Log) Notice(v ...interface{}) {
	this.print(LevelNotice, v)
}

func (this *Log) Info(v ...interface{}) {
	this.print(LevelInformational, v)
}

func (this *Log) Debug(v ...interface{}) {
	this.print(LevelDebug, v)
}
