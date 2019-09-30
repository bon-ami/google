package main

import (
	//"bufio"
	"bytes"
	"database/sql"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/bon-ami/eztools"
	_ "github.com/go-sql-driver/mysql"
)

func getStrFromID(db *sql.DB, table, id string) string {
	str, err := eztools.GetPairStr(db, table, id)
	if err != nil {
		return ""
	}
	return str
}

func listReqExec(db *sql.DB, cri string, ch chan string) {
	sel := []string{
		eztools.FldTOOL,
		eztools.FldANDROID,
		eztools.FldVER,
		eztools.FldREQ,
		eztools.FldEXP}
	res, err := eztools.Search(db, eztools.TblGOOGLE, cri, sel, " ORDER BY "+eztools.FldEXP+" ASC,"+eztools.FldTOOL)
	if err != nil {
		eztools.LogErrPrint(err)
	} else {
		var info string
		for _, i := range res {
			if len(i) < 5 {
				eztools.LogPrint("NOT enough fields in table google!")
				break
			}
			info = eztools.FldTOOL + "=" + getStrFromID(db, eztools.TblTOOL, i[0]) + ", " +
				eztools.FldANDROID + "=" + getStrFromID(db, eztools.TblANDROID, i[1]) + ", " +
				eztools.FldVER + "=" + getStrFromID(db, eztools.TblVER, i[2]) + ", " +
				eztools.FldREQ + "=" + i[3] + ", " +
				eztools.FldEXP + "=" + i[4]
			ch <- info
		}
	}
	//since multiple instances are using same channel, we cannot close it
}

func listReq(db *sql.DB, cri string, ch chan string) {
	if len(cri) > 0 {
		cri += " AND "
	}
	cri += eztools.FldEXP
	listReqExec(db, cri+" IS NOT NULL", ch)
	listReqExec(db, cri+" IS NULL", ch)
	ch <- ""
}

func updateProdgle(db *sql.DB, cri, google string) (err error) {
	fields := []string{eztools.FldGOOGLE}
	values := []string{google}
	err = eztools.UpdateWtParams(db, eztools.TblPRODGLE, cri, fields, values, true)
	return
}

func addNewProdgle(db *sql.DB, google, product string) (err error) {
	fields := []string{eztools.FldGOOGLE, eztools.FldPRODUCT}
	values := []string{google, product}
	_, err = eztools.AddWtParams(db, eztools.TblPRODGLE, fields, values, true)
	return
}

func getGoogleID(db *sql.DB, product, tool, ver, android string) (googleID string, err error) {
	selStr := []string{eztools.FldID}
	cri := eztools.FldTOOL + "=" + tool + " AND " + eztools.FldVER + "=" + ver + " AND (" + eztools.FldANDROID + "=" + android + " OR " + eztools.FldANDROID + "=" + strconv.Itoa(eztools.DefID) + ")"
	googleIDs, err := eztools.Search(db, eztools.TblGOOGLE, cri, selStr, "")
	//googleIDs is the new one
	if err != nil {
		return
	}
	switch {
	case len(googleIDs) < 1:
		err = errors.New("NO definition with combination of such Android, tool, and version!")
		return
	case len(googleIDs) > 1:
		err = errors.New("More than 1 definition with combination of such Android, tool, and version!")
		return
	}
	googleID = googleIDs[0][0]
	return
}

func addOrUpdateProdgle(db *sql.DB, product, tool, ver, android string) (err error) {
	googleID, err := getGoogleID(db, product, tool, ver, android)
	if err != nil {
		return
	}
	selStr := []string{eztools.FldGOOGLE}
	cri := eztools.FldPRODUCT + "=" + product
	googleIDs, err := eztools.Search(db, eztools.TblPRODGLE, cri, selStr, "")
	//googleIDs are the old ones
	if err != nil {
		return
	}
	if len(googleIDs) > 0 {
		selStr[0] = eztools.FldID
		cri = ""
		for _, i := range googleIDs {
			if len(cri) > 0 {
				cri = cri + " OR "
			}
			cri = cri + eztools.FldID + "=" + i[0]
		}
		cri = eztools.FldANDROID + "=" + android + " AND " + eztools.FldTOOL + "=" + tool + " AND (" + cri + ")"
		var searched [][]string
		searched, err = eztools.Search(db, eztools.TblGOOGLE, cri, selStr, "")
		//searched is the old google ID to be replaced
		if err != nil {
			return
		}
		switch len(searched) {
		case 0:
			err = addNewProdgle(db, googleID, product)
		case 1:
			/* TODO: How to check for different versions
			cri = eztools.FldPRODUCT + " = " + product + " AND " + eztools.FldGOOGLE + " = " + searched[0][0]
			err = updateProdgle(db, cri, googleID)*/
		default: //TODO
			err = errors.New("multiple google tool definition for single product!")
		}
	} else {
		err = addNewProdgle(db, googleID, product)
	}
	return
}

//obsolete
func adbGtPrp(adbCmd, param string) (res string, err error) {
	cmd := exec.Command("adb", param)
	bytes, err := cmd.CombinedOutput()
	res = string(bytes[:])
	return
}

func updatePrjDB(db *sql.DB, prod, version string) (err error) {
	selStr := []string{eztools.FldID}
	searched, err := eztools.Search(db, eztools.TblPRODUCT,
		eztools.FldSTR+"=\""+prod+"\"", selStr, "")
	if err != nil {
		err = errors.New(err.Error() + "... New project " + prod + " found through adb!")
		return
	}
	if len(searched) != 1 {
		err = errors.New("No single result... New project " + prod + " found through adb!")
		return
	}
	// SP = 2018-11-01 ro.build.version.security_patch
	// android = 9 / 8.1.0 ro.build.version.release
	// version = 8.1_201806.go / 8.0_r6
	valArr := strings.Split(version, "_")
	var android, ver string
	if len(valArr) <= 1 {
		err = errors.New("Invalid GMS version " + ver + " found through adb!")
		return
	}
	prodID := searched[0][0]
	android = valArr[0]
	ver = valArr[1]
	ver, err = tranVer(ver)
	if err != nil {
		err = errors.New(err.Error() + "... Version " + version + " failed to be parsed! (android=" + android + ",ver=" + valArr[1])
		return
	}
	if len(ver) < 1 {
		err = errors.New("Version " + version + " failed to be parsed! (android=" + android + ",ver=" + valArr[1])
		return
	}
	searched, err = eztools.Search(db, eztools.TblVER,
		eztools.FldSTR+"=\""+ver+"\"", selStr, "")
	if err != nil {
		err = errors.New(err.Error() + "... New GMS version " +
			ver + " found through adb!")
		return
	}
	if len(searched) != 1 {
		err = errors.New("No single result... New GMS version " +
			ver + " found through adb!")
		return
	}
	verID := searched[0][0]
	searched, err = eztools.Search(db, eztools.TblANDROID,
		eztools.FldSTR+"=\""+android+"\"", selStr, "")
	if err != nil {
		err = errors.New(err.Error() + "... New Android version " +
			android + " found through adb!")
		return
	}
	if len(searched) != 1 {
		err = errors.New("No single result... New Android version " +
			android + " found through adb!")
		return
	}
	andID := searched[0][0]
	searched, err = eztools.Search(db, eztools.TblTOOL,
		eztools.FldSTR+"=\"GMS\"", selStr, "")
	if err != nil {
		err = errors.New(err.Error() + "... GMS table NOT found!")
		return
	}
	if len(searched) != 1 {
		err = errors.New("No single result... GMS table NOT found!")
		return
	}
	err = addOrUpdateProdgle(db, prodID, searched[0][0], verID, andID)
	return
}

func wait4Cmd(done chan error, cmd *exec.Cmd) {
	err := cmd.Wait()
	if err != nil {
		done <- err
	} else {
		done <- errors.New("")
	}
}

func updatePrjBG(db *sql.DB, ci chan string) {
	const (
		adbCmd      = "adb"
		adbPrmShell = "shell"
		adbPrmProp  = "getprop"
	)
	params := [...][3]string{
		{"wait-for-device", "", ""},
		{adbPrmShell, adbPrmProp, "ro.product.board"},
		{adbPrmShell, adbPrmProp, "ro.com.google.gmsversion"}}
	adbRes := make([]string, len(params))
	var paramSlice []string
ADBCMDLOOP:
	for i, param := range params {
		paramSlice = nil
		for parami, param1 := range param {
			if param1 == "" {
				paramSlice = param[:parami]
				break
			}
		}
		if paramSlice == nil {
			paramSlice = param[:]
		}
		eztools.Log("ADB cmd: " + adbCmd + " " + strings.Join(paramSlice, " "))
		cmd := exec.Command(adbCmd, paramSlice...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		if cmd.Start() != nil {
			eztools.LogPrint("ADB: failed to prepare " + param[0])
			break
		}
		done := make(chan error, 1)
		defer close(done)
		go wait4Cmd(done, cmd)
		select {
		case <-ci:
			if err := cmd.Process.Kill(); err != nil {
				eztools.LogPrint("failed to kill ADB process: " + err.Error())
			} else {
				eztools.Log("ADB process killed as timeout reached")
			}
			<-done //wait for wait4Cmd
			break ADBCMDLOOP
		case err := <-done:
			if err != nil {
				if err.Error() == "" {
					eztools.Log("ADB process finished without error")
					bytes := buf.Bytes()
					if len(bytes) > 0 {
						adbRes[i] = string(bytes[:])
						if index := strings.Index(adbRes[i], "\r\n"); index >= 0 {
							adbRes[i] = adbRes[i][:index]
						} else if index := strings.Index(adbRes[i], "\n"); index >= 0 {
							adbRes[i] = adbRes[i][:index]
						}
					}
				} else {
					eztools.LogPrint("ADB process finished with error = " + err.Error())
				}
			} else {
				eztools.Log("ADB process finished without return value!")
			}
		}
	}
	eztools.Log("ADB results: " + strings.Join(adbRes, "; "))
	if len(adbRes) > 2 && len(adbRes[1]) > 0 && len(adbRes[2]) > 0 {
		if err := updatePrjDB(db, adbRes[1], adbRes[2]); err != nil && eztools.Debugging {
			eztools.LogErr(err)
		}
	}
}

// 201806.go -> Jun ; r6 -> 6
func tranVer(in string) (out string, err error) {
	if strings.HasPrefix(in, "r") {
		out = in[1 : len(in)-1]
	} else {
		valArr := strings.Split(in, ".")
		//if len(valArr) > 1 {
		tm, err := time.Parse("200601", valArr[0])
		if err == nil {
			out = tm.Month().String()[:3]
		}
		//}
	}
	return
}
