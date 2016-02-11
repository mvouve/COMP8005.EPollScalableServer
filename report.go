package main

import (
	"container/list"

	"github.com/tealeg/xlsx"
)

func generateReport(reportName string, conenctions *list.List) {
	xlsx := xlsx.NewFile()
	sheet, _ := xlsx.AddSheet("Connections")
	for e := conenctions.Front(); e != nil; e = e.Next() {
		conn := e.Value.(connectionInfo)
		row := sheet.AddRow()
		hostNameCell := row.AddCell()
		hostNameCell.SetString(conn.hostName)
		numberOfRequestsCell := row.AddCell()
		numberOfRequestsCell.SetInt(conn.numberOfRequests)
		ammountOfDataCell := row.AddCell()
		ammountOfDataCell.SetInt(conn.ammountOfData)
	}

	xlsx.Save(reportName + ".xlsx")
}
