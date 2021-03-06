// Debug functions.
package core

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/DanielRenne/GoCore/core/extensions"
	"github.com/DanielRenne/GoCore/core/serverSettings"
	"github.com/davidrenne/reflections"
	"github.com/go-errors/errors"
)

type core_debug struct{}

var core_logger = log.New(os.Stdout, "", 0)

var TransactionLog string
var Debug = core_debug{}
var Logger = core_logger
var TransactionLogMutex *sync.RWMutex

func init() {
	TransactionLogMutex = &sync.RWMutex{}
}

// Nop is a dummy function that can be called in source files where
// other debug functions are constantly added and removed.
// That way import "github.com/ungerik/go-start/debug" won't cause an error when
// no other debug function is currently used.
// Arbitrary objects can be passed as arguments to avoid "declared and not used"
// error messages when commenting code out and in.
// The result is a nil interface{} dummy value.
func (self *core_debug) Nop(dummiesIn ...interface{}) (dummyOut interface{}) {
	return nil
}

func (self *core_debug) CallStackInfo(skip int) (info string) {
	pc, file, line, ok := runtime.Caller(skip)
	if ok {
		funcName := runtime.FuncForPC(pc).Name()
		info += fmt.Sprintf("In function %s()", funcName)
	}
	for i := 0; ok; i++ {
		info += fmt.Sprintf("\n%s:%d", file, line)
		_, file, line, ok = runtime.Caller(skip + i)
	}
	return info
}

func (self *core_debug) PrintCallStack() {
	debug.PrintStack()
}

func (self *core_debug) LogCallStack() {
	log.Print(self.Stack())
}

func (self *core_debug) Stack() string {
	return string(debug.Stack())
}

func (self *core_debug) formatValue(value interface{}) string {
	return fmt.Sprintf("\n     Type: %T\n    Value: %v\nGo Syntax: %#v", value, value, value)
}

func (self *core_debug) formatCallstack(skip int) string {
	return fmt.Sprintf("\nCallstack: %s", self.CallStackInfo(skip+1))
}

func (self *core_debug) FormatSkip(skip int, value interface{}) string {
	return self.formatValue(value) + self.formatCallstack(skip+1)
}

func (self *core_debug) Format(value interface{}) string {
	return self.FormatSkip(2, value)
}

func (self *core_debug) DumpQuiet(values ...interface{}) {
	// uncomment below to find your callers to quiet
	self.Print("Silently not dumping " + extensions.IntToString(len(values)) + " values")
	//Logger.Println("DumpQuiet has " + extensions.IntToString(len(values)) + " parameters called")
	//Logger.Println("")
	//self.ThrowAndPrintError()
}

func IsZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

func IsZeroOfUnderlyingType2(x interface{}) bool {
	return x == reflect.Zero(reflect.TypeOf(x)).Interface()
}

func (self *core_debug) HandleError(err error) (s string) {
	if err != nil {
		// notice that we're using 1, so it will actually log the where
		// the error happened, 0 = this function, we don't want that.
		_, fn, line, _ := runtime.Caller(1)
		fileNameParts := strings.Split(fn, "/")
		return fmt.Sprintf("  Error Info: %s Line %d. ErrorType: %v", fileNameParts[len(fileNameParts)-1], line, err)
	}
	return ""
}

func (self *core_debug) ErrLineAndFile(err error) (s string) {
	if err != nil {
		// notice that we're using 1, so it will actually log the where
		// the error happened, 0 = this function, we don't want that.
		_, fn, line, _ := runtime.Caller(1)
		fileNameParts := strings.Split(fn, "/")
		return fmt.Sprintf("%s Line %d", fileNameParts[len(fileNameParts)-1], line)
	}
	return ""
}

func (self *core_debug) Dump(valuesOriginal ...interface{}) {
	t := time.Now()
	l := "!!!!!!!!!!!!! DEBUG " + t.Format("2006-01-02 15:04:05.000000") + "!!!!!!!!!!!!!\n\n"
	Logger.Println(l)

	serverSettings.WebConfigMutex.RLock()
	if serverSettings.WebConfig.Application.ReleaseMode == "development" {
		TransactionLogMutex.Lock()
		TransactionLog += l
		TransactionLogMutex.Unlock()
	}
	serverSettings.WebConfigMutex.RUnlock()
	for _, value := range valuesOriginal {
		l := self.DumpBase(value)
		Logger.Print(l)
		serverSettings.WebConfigMutex.RLock()
		if serverSettings.WebConfig.Application.ReleaseMode == "development" {
			TransactionLogMutex.Lock()
			TransactionLog += l
			TransactionLogMutex.Unlock()
		}
		serverSettings.WebConfigMutex.RUnlock()
	}
	l = self.ThrowAndPrintError()
	Logger.Print(l)

	serverSettings.WebConfigMutex.RLock()
	if serverSettings.WebConfig.Application.ReleaseMode == "development" {
		TransactionLogMutex.Lock()
		TransactionLog += l
		TransactionLogMutex.Unlock()
	}
	serverSettings.WebConfigMutex.RUnlock()
	l = "!!!!!!!!!!!!! ENDDEBUG " + t.Format("2006-01-02 15:04:05.000000") + "!!!!!!!!!!!!!"
	Logger.Println(l)
	serverSettings.WebConfigMutex.RLock()
	if serverSettings.WebConfig.Application.ReleaseMode == "development" {
		TransactionLogMutex.Lock()
		TransactionLog += l
		TransactionLogMutex.Unlock()
	}
	serverSettings.WebConfigMutex.RUnlock()
}

func (self *core_debug) GetDump(valuesOriginal ...interface{}) (output string) {
	for _, value := range valuesOriginal {
		output += self.DumpBase(value)
	}
	//output += self.ThrowAndPrintError()
	return output
}

func (self *core_debug) GetDumpWithInfo(valuesOriginal ...interface{}) (output string) {
	t := time.Now()
	return self.GetDumpWithInfoAndTimeString(t.String(), valuesOriginal...)
}

func (self *core_debug) GetDumpWithInfoAndTimeString(timeStr string, valuesOriginal ...interface{}) (output string) {
	l := "\n!!!!!!!!!!!!! DEBUG " + timeStr + "!!!!!!!!!!!!!\n\n"
	output += l

	serverSettings.WebConfigMutex.RLock()
	if serverSettings.WebConfig.Application.ReleaseMode == "development" {
		TransactionLogMutex.Lock()
		TransactionLog += l
		TransactionLogMutex.Unlock()
	}
	serverSettings.WebConfigMutex.RUnlock()

	for _, value := range valuesOriginal {
		output += self.DumpBase(value) + "\n"
	}

	serverSettings.WebConfigMutex.RLock()
	if serverSettings.WebConfig.Application.ReleaseMode == "development" {
		TransactionLogMutex.Lock()
		TransactionLog += output
		TransactionLogMutex.Unlock()
	}
	serverSettings.WebConfigMutex.RUnlock()

	l = self.ThrowAndPrintError()
	output += l
	serverSettings.WebConfigMutex.RLock()
	if serverSettings.WebConfig.Application.ReleaseMode == "development" {
		TransactionLogMutex.Lock()
		TransactionLog += l
		TransactionLogMutex.Unlock()
	}
	serverSettings.WebConfigMutex.RUnlock()
	l = "!!!!!!!!!!!!! ENDDEBUG " + timeStr + "!!!!!!!!!!!!!\n"
	output += l
	serverSettings.WebConfigMutex.RLock()
	if serverSettings.WebConfig.Application.ReleaseMode == "development" {
		TransactionLogMutex.Lock()
		TransactionLog += l
		TransactionLogMutex.Unlock()
	}
	serverSettings.WebConfigMutex.RUnlock()
	return output
}

func (self *core_debug) DumpBase(values ...interface{}) (output string) {
	//golog "github.com/DanielRenne/GoCore/core/log"
	//defer golog.TimeTrack(time.Now(), "Dump")
	var jsonString string
	var err error
	var structKeys []string
	if Logger != nil {
		for _, value := range values {
			isAllJSON := true
			var kind string
			kind = strings.TrimSpace(fmt.Sprintf("%T", value))
			var pieces = strings.Split(kind, " ")
			if pieces[0] == "struct" || strings.Index(pieces[0], "model.") != -1 || strings.Index(pieces[0], "viewModel.") != -1 {
				kind = reflections.ReflectKind(value)
				structKeys, err = reflections.FieldsDeep(value)
				if err == nil {
					for _, field := range structKeys {
						jsonString, err = reflections.GetFieldTag(value, field, "json")
						if err != nil {
							isAllJSON = false
						}
						if jsonString == "" {
							isAllJSON = false
						}
					}
				} else {
					isAllJSON = false
				}
			} else {
				isAllJSON = false
			}
			if isAllJSON || kind == "map" || kind == "bson.M" || kind == "slice" {
				var rawBytes []byte
				rawBytes, err = json.MarshalIndent(value, "", "\t")
				if err == nil {
					output += fmt.Sprintf("#### %-39s ####\n%+v", kind, string(rawBytes[:]))
				}
			} else {
				if strings.TrimSpace(kind) == "string" {
					var stringVal = value.(string)
					position := strings.Index(stringVal, "Desc->")
					if position == -1 {
						output += "#### + " + kind + " " + stringVal + "[len:" + extensions.IntToString(len(stringVal)) + "]####" + "\n"
						// for _, tmp := range strings.Split(stringVal, "\\n") {
						// 	output += "\n" + tmp
						// }
					} else {
						output += stringVal[6:] + " --> "
					}
				} else {
					output += fmt.Sprintf("#### %-39s ####\n%+v", kind, value)
				}
			}
		}
	}
	return output
}

func (self *core_debug) ThrowAndPrintError() (output string) {

	serverSettings.WebConfigMutex.RLock()
	ok := serverSettings.WebConfig.Application.CoreDebugStackTrace
	serverSettings.WebConfigMutex.RUnlock()
	if ok {
		output += "\n"
		errorInfo := self.ThrowError()
		stack := strings.Split(errorInfo.ErrorStack(), "\n")
		if len(stack) >= 8 {
			output += "\nDump Caller:"
			output += "\n---------------"
			//output += strings.Join(stack, ",")
			output += "\n golines ==> " + strings.TrimSpace(stack[6])
			output += "\n         ==> " + strings.TrimSpace(stack[7])
			output += "\n         ==> " + strings.TrimSpace(stack[8])
			output += "\n         ==> " + strings.TrimSpace(stack[9])
			output += "\n         ==> " + strings.TrimSpace(stack[10])
			if len(stack) >= 12 {
				output += "\n         ==> " + strings.TrimSpace(stack[11])
			}
			if len(stack) >= 13 {
				output += "\n         ==> " + strings.TrimSpace(stack[12])
			}
			if len(stack) >= 14 {
				output += "\n         ==> " + strings.TrimSpace(stack[13])
			}
			if len(stack) >= 15 {
				output += "\n         ==> " + strings.TrimSpace(stack[14])
			}
			if len(stack) >= 16 {
				output += "\n         ==> " + strings.TrimSpace(stack[15])
			}
			output += "\n---------------"
			output += "\n"
			output += "\n"
		}
	}
	return output
}

func (self *core_debug) ThrowError() *errors.Error {
	return errors.Errorf("Debug Dump")
}

func (self *core_debug) Print(values ...interface{}) {
	if Logger != nil {
		Logger.Print(values...)
	}
}

func (self *core_debug) Printf(format string, values ...interface{}) {
	if Logger != nil {
		Logger.Printf(format, values...)
	}
}
