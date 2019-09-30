package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/bon-ami/eztools"
	_ "github.com/go-sql-driver/mysql"
)

func getFile(db *sql.DB) {
	getInTran(db, false)
	fromN2 := eztools.PromptStr("Which dir to list on server or which file to get?([Enter]=root)")
	if len(fromN2) > 0 {
		toStr := eztools.PromptStr("Which dir to store the file?")
		if len(toStr) > 0 {
			fromN2 += "," + toStr
		}
	} else {
		url, err := eztools.GetPairStr(db, eztools.TblCHORE, "GoogleSvrFolder")
		if err != nil {
			eztools.LogErrPrintWtInfo("Database connection failure!", err)
			return
		}
		fromN2 = url + string(os.PathSeparator)
	}
	procNetNList(db, fromN2, "", "")
	getInTran(db, true)
}

func getRsrc(db *sql.DB) {
	getInTran(db, false)
	//get product, android & tool
	mp, err := choosePairOrAdd(db, false, true, eztools.TblPRODUCT, eztools.TblANDROID, eztools.TblTOOL)
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}

	//get tool
	tool, err := eztools.GetPairStr(db, eztools.TblTOOL, mp[eztools.TblTOOL])
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}

	//get bit
	sel := []string{eztools.FldBIT}
	res, err := eztools.Search(db, eztools.TblPRODFO,
		eztools.FldPRODUCT+"="+mp[eztools.TblPRODUCT]+" AND "+
			eztools.FldANDROID+"="+mp[eztools.TblANDROID], sel, "")
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if len(res) != 1 {
		eztools.LogPrint("For product " + mp[eztools.TblPRODUCT] +
			" and android " + mp[eztools.TblANDROID] + ", " +
			strconv.Itoa(len(res)) + " results found")
		return
	}
	bit, err := eztools.GetPairStr(db, eztools.TblBIT, res[0][0])
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}

	android, err := eztools.GetPairStr(db, eztools.TblANDROID, mp[eztools.TblANDROID])
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	procNetNList(db, tool, bit, android)
	getInTran(db, true)
}

func procNetNList(db *sql.DB, tool, bit, android string) {
	eztools.ShowStr("Please wait patiently for result... ")
	rep, err := procNet(db, tool, bit, android)
	if err != nil || rep == nil {
		eztools.LogErrPrint(err)
		return
	}
	defer rep.Body.Close()
	switch rep.StatusCode {
	case 200:
		eztools.ShowStrln("Requested successfully.")
	case 400:
		eztools.ShowStrln("Wrong request!")
	case 502:
		eztools.ShowStrln("Server path not found!")
	case 503:
		eztools.ShowStrln("Wrong server path config!")
	default:
		eztools.ShowStrln("Please check with Allen...")
		eztools.Log(strconv.Itoa(rep.StatusCode) + " got from server!")
	}
	body, err := ioutil.ReadAll(rep.Body)
	if err == nil && len(body) > 0 {
		eztools.ShowStrln(string(body))
	} else {
		eztools.ShowStrln("<none>")
	}
}

func procNet(db *sql.DB, tool, bit, android string) (rep *http.Response, err error) {
	url, err := eztools.GetPairStr(db, eztools.TblCHORE, "GoogleRes")
	if err != nil {
		return
	}
	req, err := http.NewRequest("GET", server+url, nil)
	if err != nil {
		return
	}
	q := req.URL.Query()
	if len(tool) > 0 {
		q.Add("tool", tool)
	}
	if len(bit) > 0 {
		q.Add("bit", bit)
	}
	if len(android) > 0 {
		q.Add("android", android)
	}
	req.URL.RawQuery = q.Encode()
	rep, err = http.DefaultClient.Do(req)
	return
}

func getInTran(db *sql.DB, hint bool) {
	if hint {
		eztools.ShowStrln("Please note following list may not be accurate, since more downloads may be discovered while server is processing. Try to get resource again to get a refreshed list and check transfer status.")
	}
	procNetNList(db, "", "", "")
}
