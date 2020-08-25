package chronicle

import (
	"github.com/dapixio/fio.etl/logging"
	"log"
)

var (
	elog *log.Logger
	ilog *log.Logger
	dlog *log.Logger
)

func init() {
	elog, ilog, dlog = logging.Setup("[fioetl-consumer] ")
}
