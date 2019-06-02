package debug

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

func Debugf(format string, a ...interface{}) {
	Debug(fmt.Sprintf(format, a...))
}

func Debug(a ...interface{}) {
	if viper.GetBool("verbose") {
		fmt.Fprintln(os.Stderr, a...)
	}
}
