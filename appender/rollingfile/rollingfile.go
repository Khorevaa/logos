package rollingfile

import (
	"github.com/khorevaa/logos/internal/common"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type RollingFile struct {
	zapcore.WriteSyncer
}

type Config struct {
	// FileName is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	FileName string `logos-config:"file_name" logos-validate:"required"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `logos-config:"max_size" logos-validate:"min=1"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `logos-config:"max_age"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `logos-config:"max_backups"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `logos-config:"local_time"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `logos-config:"compress"`
}

func New(v *common.Config) (zapcore.WriteSyncer, error) {
	cfg := Config{
		MaxSize: 500,
	}
	if err := v.Unpack(&cfg); err != nil {
		return nil, err
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 7
	}
	w := &lumberjack.Logger{
		Filename:   cfg.FileName,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		LocalTime:  cfg.LocalTime,
		Compress:   cfg.Compress,
	}
	return &RollingFile{zapcore.AddSync(w)}, nil
}
