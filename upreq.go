package main

import (
	"database/sql"
	"errors"
	"github.com/bon-ami/eztools"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
)

// uniq is valid only when add is true
func choosePairOrAdd(db *sql.DB, add, uniq bool, tbl ...string) (
	res map[string]string, err error) {
	if tbl == nil {
		return
	}
	var id int
	res = make(map[string]string)
	for _, i := range tbl {
		if add {
			id, err = eztools.ChoosePairOrAdd(db, i, uniq)
		} else {
			id, err = eztools.ChoosePair(db, i)
		}
		if err != nil {
			return
		}
		if id == eztools.InvalidID {
			err = errors.New("Invalid ID")
			return
		}
		if id == 0 { //zero value not allowed
			eztools.LogPrint("NO default allowed for " + i)
			err = errors.New("Zero Value")
			return
		}
		res[i] = strconv.Itoa(id)
	}
	return
}

func addReqExec(db *sql.DB, date, android, tool, ver string) error {
	_, err := eztools.AddWtParams(db, eztools.TblGOOGLE,
		[]string{eztools.FldTOOL, eztools.FldANDROID,
			eztools.FldVER, eztools.FldREQ},
		[]string{tool, android, ver, date}, false)
	if err != nil {
		eztools.LogPrint("FAILED to add new item to table google!")
		return err
	} else {
		eztools.LogPrint("Item added to table google")
	}
	return nil
}

func modReq(db *sql.DB, id string) {
	searched, err := eztools.Search(db, eztools.TblGOOGLE,
		eztools.FldID+"="+id,
		[]string{eztools.FldID, eztools.FldANDROID,
			eztools.FldTOOL, eztools.FldVER,
			eztools.FldREQ, eztools.FldEXP},
		"")
	if err != nil {
		eztools.LogErrPrint(err)
		return
	}
	if len(searched) != 1 {
		eztools.LogPrint("No single record found for ID " + id)
		return
	}
	switch eztools.PromptStr("What to do with ID " + id +
		", Android " +
		getStrFromID(db, eztools.TblANDROID, searched[0][1]) +
		", Tool " +
		getStrFromID(db, eztools.TblTOOL, searched[0][2]) +
		", Version " +
		getStrFromID(db, eztools.TblVER, searched[0][3]) +
		", Requirement " +
		searched[0][4] +
		", Expiry " +
		searched[0][5] + "? Delete, Modify, or Pass?") {
	case "D", "d":
		switch eztools.PromptStr("Delete ID " + id + "?") {
		case "Y", "y":
			eztools.DeleteWtID(db, eztools.TblGOOGLE, id)
		}
	case "M", "m":
		m, err := choosePairOrAdd(db, true, true,
			eztools.TblTOOL, eztools.TblANDROID, eztools.TblVER)
		if err != nil {
			eztools.LogErrPrint(err)
			return
		}
		req := eztools.GetDate("Requirement date=")
		exp := eztools.GetDate("Expiry date=")
		eztools.UpdateWtParams(db, eztools.TblGOOGLE,
			eztools.FldID+"="+id,
			[]string{eztools.FldANDROID, eztools.FldTOOL,
				eztools.FldVER, eztools.FldREQ, eztools.FldEXP},
			[]string{m[eztools.TblANDROID], m[eztools.TblTOOL],
				m[eztools.TblVER], req, exp}, false)
	}
}

//update existing, adding new
func upNaddReq(db *sql.DB, date, android, tool, ver string) error {
	//update existing, adding expiry
	cri := eztools.FldANDROID + "=" + android + " AND " +
		eztools.FldTOOL + "=" + tool + " AND " +
		eztools.FldEXP + " IS NULL"
	searched, err := eztools.Search(db, eztools.TblGOOGLE, cri,
		[]string{eztools.FldID, eztools.FldREQ},
		"")
	if err != nil {
		eztools.LogErrPrint(err)
		return err
	}
	switch len(searched) {
	case 0:
		eztools.ShowStrln("No former versions detected.")
	case 1:
		if eztools.TranDate(searched[0][1]) == date {
			eztools.ShowStrln("Same existing item found.")
			return nil
		} else {
			cri = eztools.FldID + "=" + searched[0][0]
			eztools.UpdateWtParams(db, eztools.TblGOOGLE, cri,
				[]string{eztools.FldEXP}, []string{date}, false)
		}
	default:
		eztools.ShowStrln("TODO: multiple items with empty expiry")
	}

	return addReqExec(db, date, android, tool, ver)
}

func upOnlyReq(db *sql.DB, date, id string) error {
	err := eztools.UpdateWtParams(db, eztools.TblGOOGLE,
		eztools.FldID+"="+id,
		[]string{eztools.FldREQ}, []string{id}, false)
	if err != nil {
		eztools.LogErrPrint(err)
		return err
	}
	return nil
}

func upReq(db *sql.DB, ch chan string, readonly bool) {
	var info string
	eztools.ShowStrln("Current google regulations.")
	for info = <-ch; len(info) > 0; info = <-ch {
		eztools.ShowStrln(info)
	}
	if readonly {
		return
	}
	date := eztools.GetDate("Enter the must-use date. (Invalid to modify existing.)")
	modify := false
	if date == "NULL" {
		/*switch eztools.PromptStr("NO date provided. Y/y to modify existing items. ") {
		case "Y", "y":
			modify(db)
		default:
			return
		}*/
		modify = true
	}
	for {
		m, err := choosePairOrAdd(db, !modify, true,
			eztools.TblTOOL, eztools.TblANDROID, eztools.TblVER)
		if err != nil {
			eztools.LogErrPrint(err)
			return
		}
		//selStrArr := []string{eztools.FldANDROID, eztools.FldTOOL, eztools.FldVER}
		//selStr := selStrArr[:]
		selStr := []string{eztools.FldANDROID, eztools.FldTOOL,
			eztools.FldVER, eztools.FldID}
		cri := eztools.FldTOOL + "=" + m[eztools.TblTOOL] + " AND " +
			eztools.FldVER + "=" + m[eztools.TblVER]
		updateAll := true
		if m[eztools.TblANDROID] != strconv.Itoa(eztools.AllID) {
			cri += " AND " + eztools.FldANDROID + "=" +
				m[eztools.TblANDROID]
			updateAll = false
		}
		//list exact tool and version (and android, if specified) in existence
		searched, err := eztools.Search(db, eztools.TblGOOGLE,
			cri, selStr, "")
		if err != nil {
			eztools.LogErrPrint(err)
			return
		}
		if len(searched) > 0 {
			eztools.ShowStrln(strconv.Itoa(len(searched)) +
				" results found. No new items will be added.")
			//update all in existence without adding any new, in case of updateAll
			for _, i := range searched {
				if eztools.Debugging {
					eztools.ShowStrln("changing " +
						getStrFromID(db,
							eztools.TblANDROID,
							i[0]))
				}
				switch modify {
				case false:
					if upOnlyReq(db, date, i[3]) != nil {
						return
					}
				case true:
					modReq(db, i[3])
				}
			}
		} else if !modify {
			if updateAll {
				if eztools.Debugging {
					eztools.ShowStrln("adding for all...")
				}
				selStr[0] = eztools.FldID
				selStr = selStr[:1]
				searched, err = eztools.Search(db,
					eztools.TblANDROID, "", selStr, "")
				if err != nil {
					eztools.LogErrPrint(err)
					return
				}
				for _, i := range searched {
					if upNaddReq(db, date, i[0],
						m[eztools.TblTOOL],
						m[eztools.FldVER]) != nil {
						return
					}
				}
			} else if upNaddReq(db, date, m[eztools.TblANDROID],
				m[eztools.TblTOOL],
				m[eztools.FldVER]) != nil {
				return
			}
		} else {
			eztools.ShowStrln("No such items found")
		}
	}
}
