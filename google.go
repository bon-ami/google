package main

import (
	"database/sql"
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/bon-ami/eztools"
	_ "github.com/go-sql-driver/mysql"
)

var (
	ver    string
	server string
)

func main() {
	var (
		err            error
		db             *sql.DB
		paramH, paramD bool
		paramL         string
	)
	var readonly bool
	if ver != "dev" {
		readonly = true
	}
	if !readonly {
		paramD = true
	}
	paramL = "google.log"
	flag.BoolVar(&paramH, "h", false, "Print info message")
	flag.BoolVar(&paramD, "d", paramD, "Print debug messages")
	flag.StringVar(&paramL, "l", paramL, "to specify the name of Log file")
	flag.Parse()
	if paramH {
		eztools.ShowStrln("V0.1 current requirements on the bottom when shown.")
		eztools.ShowStrln("V1.0 auto upgrade.")
		eztools.ShowStrln("V1.1 adb messages hidden.")
		eztools.ShowStrln("V2.0 Resource request supported for PAB. Most empty answers return by default.")
		eztools.ShowStrln("V2.1 Resource request supported for STS.")
		eztools.ShowStrln("V2.2 Resource request supported for CTS, GTS.")
		eztools.ShowStrln("V2.3 Resource request supported for any files. Transfer progress supported.")
		//eztools.ShowStrln("V1.1 Open Source")
		flag.Usage()
		return
	}
	if paramD {
		eztools.Debugging = true
	}

	_, week := time.Now().ISOWeek()
	eztools.ShowStrln("V" + ver + ". Now it is week " + strconv.Itoa(week))
	db, err = eztools.Connect()
	if err != nil {
		eztools.LogErrFatal(err)
	}
	defer db.Close()

	// log file
	if len(paramL) > 0 {
		stat, err := os.Stat(paramL)
		if err == nil {
			if stat.Size() > 1024*1024 {
				switch eztools.PromptStr("Remove " + paramL + " because it is too big?([Enter]=y)") {
				case "y", "Y", "Yes", "YES", "yes", "":
					os.Remove(paramL) //TODO: backup before removal?
				}
			}
		}
		file, err := os.OpenFile(paramL, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			defer file.Close()
			eztools.InitLogger(file)
		} else {
			eztools.ShowStrln("Failed to open log file")
		}
	}

	var serverGot bool
	upch := make(chan bool)
	svch := make(chan string)
	go eztools.AppUpgrade(db, "Google", ver, &svch, upch)

	var readonlyStr string
	if !readonly {
		readonlyStr = "update"
	} else {
		readonlyStr = "list"
	}
	choices := []string{"quit", //0
		readonlyStr + " projects' statuses", //1
		readonlyStr + " requirements",       //2
		"get resource or list trasfers",     //3
		"list/get files on/from server"}     //4
	/*if !readonly {
		choices = append(choices, "get resource") //3
	}*/
	eztools.ShowStrln("checking for server...")

	serverGot = <-upch
	if serverGot {
		server = <-svch
	}
	cadb := make(chan string, 1)
	defer func() {
		cadb <- "STOP"
	}()
	go updatePrjBG(db, cadb)

EXIT:
	for {
		ch := make(chan string)
		// to make it fast for upReq, listReq for all actions
		go listReq(db, "", ch)
		// try to check phones' statuses without intervention
		c := eztools.ChooseStrings(choices)
		switch c {
		case 0, eztools.InvalidID:
			break EXIT
		case 1:
			upPrj(db, readonly)
		case 2:
			upReq(db, ch, readonly)
		case 3:
			getRsrc(db)
		case 4:
			getFile(db)
		default:
			eztools.ShowStrln("impossible choice: " + strconv.Itoa(c))
		}
		for i := 0; i < 30; i++ {
			eztools.ShowStr("-")
		}
		eztools.ShowStrln("")
	}

	if serverGot {
		eztools.ShowStrln("waiting for update check to end...")
		<-upch
	}
	return
}
