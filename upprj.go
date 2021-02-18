package main

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/bon-ami/eztools"
	_ "github.com/go-sql-driver/mysql"
)

func qryPrj(db *sql.DB, product string) {
	var (
		selStr []string
		info   string
	)
	selStr = make([]string, 1)
	selStr[0] = eztools.FldGOOGLE
	res, err := eztools.Search(db, eztools.TblPRODGLE, eztools.FldPRODUCT+"="+product, selStr, "")
	if err != nil {
		return
	}
	if len(res) < 1 {
		eztools.ShowStrln("NO results found.")
		return
	}
	ch := make(chan string)
	j := 0
	for _, i := range res {
		go listReq(db, eztools.FldID+"="+i[0], ch)
		j++
	}
	for {
		info = <-ch
		if len(info) > 0 {
			eztools.ShowStrln(info)
		} else {
			j--
			if j == 0 {
				break
			}
		}
	}
}

// cri: criteria when modifying, nil if adding only. [0]=bit, [1]=android, [2]=phase
func addOrModProdFo(db *sql.DB, product string, cri *[]string) (err error) {
	mb, err := choosePairOrAdd(db, true, true,
		eztools.TblBIT, eztools.TblANDROID, eztools.TblPHASE)
	if err != nil {
		return
	}
	if len(mb[eztools.TblBIT]) < 1 ||
		len(mb[eztools.TblANDROID]) < 1 ||
		len(mb[eztools.TblPHASE]) < 1 {
		return errors.New("empty value(s)")
	}
	fields := []string{eztools.FldBIT, eztools.FldPRODUCT, eztools.FldANDROID, eztools.FldPHASE}
	values := []string{mb[eztools.TblBIT], product,
		mb[eztools.TblANDROID], mb[eztools.TblPHASE]}
	if cri == nil {
		_, err = eztools.AddWtParams(db, eztools.TblPRODFO, fields, values, false)
	} else {
		if len(*cri) < 3 {
			return errors.New("Not enough criteria!")
		}
		criStr := eztools.FldPRODUCT + "=\"" + product + "\""
		for i := 0; i < 3; i++ {
			if len((*cri)[i]) > 0 {
				criStr += " AND "
				switch i {
				case 0:
					criStr += eztools.FldBIT
				case 1:
					criStr += eztools.FldANDROID
				case 2:
					criStr += eztools.FldPHASE
				}
				criStr += "=\"" + (*cri)[i] + "\""
			}
		}
		err = eztools.UpdateWtParams(db, eztools.TblPRODFO,
			criStr, fields, values, false)
	}
	if err != nil {
		return
	}
	return
}

func upPrj(db *sql.DB, readonly bool) {
	mp, err := choosePairOrAdd(db, !readonly, false, eztools.TblPRODUCT)
	if err != nil {
		return
	}
	qryPrj(db, mp[eztools.FldPRODUCT])
	if readonly {
		return
	}
	selStr := []string{eztools.FldBIT, eztools.FldANDROID, eztools.FldPHASE}
	searched, err := eztools.Search(db, eztools.TblPRODFO,
		eztools.FldPRODUCT+"="+mp[eztools.FldPRODUCT],
		selStr, "")
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if len(searched) < 1 {
		if err := addOrModProdFo(db, mp[eztools.FldPRODUCT], nil); err != nil {
			eztools.LogErrPrint(err)
			return
		}
	} else {
		for i := 0; i < len(searched); i++ {
			eztools.ShowStrln(strconv.Itoa(i) + ": bit " +
				getStrFromID(db, eztools.TblBIT, searched[i][0]) +
				", android " +
				getStrFromID(db, eztools.TblANDROID, searched[i][1]) +
				", phase " +
				getStrFromID(db, eztools.TblPHASE, searched[i][2]))
		}
	}
	answer := eztools.PromptStr("Input a number of above to change an item. Input \"a\" to add one item. Input nothing to skip product maintenance.")
	switch answer {
	case "a":
		err = addOrModProdFo(db, mp[eztools.FldPRODUCT], nil)
	case "":
	default:
		var ans int
		ans, err = strconv.Atoi(answer)
		if err == nil && ans >= 0 && ans < len(searched) {
			err = addOrModProdFo(db, mp[eztools.FldPRODUCT], &searched[ans])
		}
	}
	if err != nil {
		eztools.LogErrPrint(err)
	}
	mo, err := choosePairOrAdd(db, false, false, eztools.TblTOOL, eztools.TblANDROID, eztools.TblVER)
	if err != nil {
		return
	}
	err = addOrUpdateProdgle(db, mp[eztools.FldPRODUCT], mo[eztools.TblTOOL], mo[eztools.TblVER], mo[eztools.TblANDROID])
	if err != nil {
		eztools.LogErrPrint(err)
	}
}
