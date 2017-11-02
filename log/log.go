package log

import (
	"log"
	"os"

	"github.com/fatih/color"
)

var (
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

func init() {
	Info = log.New(os.Stdout,
		color.GreenString("[INFO] "),
		log.Ldate|log.Lshortfile)
	Warn = log.New(os.Stdout,
		color.YellowString("[WARN] "),
		log.Ldate|log.Lshortfile)

	Error = log.New(os.Stderr,
		color.RedString("[ERROR] "),
		log.Ldate|log.Lshortfile)
}
