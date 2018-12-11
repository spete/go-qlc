/*
 * Copyright (c) 2018 QLC Chain Team
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package log

import (
	"fmt"
	"github.com/json-iterator/go"
	"github.com/qlcchain/go-qlc/common/util"
	"github.com/qlcchain/go-qlc/config"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logfile = "qlc.log"
)

var (
	once      sync.Once
	lumlog    lumberjack.Logger
	logger, _ = zap.NewDevelopment()
)

func Init(config *config.Config) error {
	var initErr error
	once.Do(func() {
		logFolder := config.LogDir()
		err := util.CreateDirIfNotExist(logFolder)
		if err != nil {
			initErr = err
		}
		logfile := filepath.Join(logFolder, logfile)
		lumlog = lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    10, // megabytesÒ
			MaxBackups: 10,
			MaxAge:     28, // days
			Compress:   true,
			LocalTime:  true,
		}
		var logCfg zap.Config
		bytes, _ := jsoniter.Marshal(config.LogConfig)
		err = jsoniter.Unmarshal(bytes, &logCfg)
		if err != nil {
			initErr = err
			fmt.Println(err)
		}
		logCfg.EncoderConfig = zap.NewProductionEncoderConfig()
		logger, _ = logCfg.Build(zap.Hooks(lumberjackZapHook))
	})

	return initErr
}

//NewLogger create logger by name
func NewLogger(name string) *zap.SugaredLogger {
	return logger.Sugar().Named(name)
}

func lumberjackZapHook(e zapcore.Entry) error {
	_, err := lumlog.Write([]byte(fmt.Sprintf("%s %s [%s] %s %s\n", e.Time.Format(time.RFC3339Nano), e.Level.CapitalString(), e.LoggerName, e.Caller.TrimmedPath(), e.Message)))
	return err
}
