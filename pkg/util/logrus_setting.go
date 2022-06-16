package util

import (
	"io"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

func InitLogrus(log_fp, log_lvl, log_fmt string) {
	log_fp = PathSanitize(log_fp)
	if log_fp == "" {
		logrus.SetOutput(os.Stdout)
	} else {
		log_f, e := os.OpenFile(log_fp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if e != nil {
			logrus.WithFields(
				logrus.Fields{
					"filepath_target": log_fp,
					"error":           e,
					"where":           GetCurrentFunctionInfo(),
				}).Fatalf("Unable to create log file", log_fp, e)
		}
		logrus.SetOutput(io.MultiWriter(log_f, os.Stdout))
	}

	switch log_lvl {
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		logrus.WithFields(
			logrus.Fields{
				"arg_name":  "loglovel",
				"arg_value": log_lvl,
			}).Panicf("Wrong argument")
	}

	switch log_fmt {
	case "text":
		logrus.SetFormatter(&nested.Formatter{
			TimestampFormat:  "2006-01-02 15:04:05.000",
			NoColors:         false,
			HideKeys:         true,
			NoFieldsColors:   false,
			NoFieldsSpace:    false,
			ShowFullLevel:    false,
			NoUppercaseLevel: false,
			TrimMessages:     true,
		})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
		})
	default:
		logrus.WithFields(
			logrus.Fields{
				"arg_name":  "logformat",
				"arg_value": log_fmt,
			}).Panicf("Wrong argument")
	}
}
