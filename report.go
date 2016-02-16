/*------------------------------------------------------------------------------
-- DATE:	       February 6, 2016
--
-- Source File:	 report.go
--
-- REVISIONS: 	February 13, 2016 - Generalised reporting functionality
--
-- DESIGNER:	   Marc Vouve
--
-- PROGRAMMER:	 Marc Vouve
--
--
-- INTERFACE:
--	func generateReport(fname string, elements *list.List)
--  func generateHeaders(i interface{}, row *xlsx.Row)
--  func generateRow(i interface{}, row *xlsx.Row)
--
--
-- NOTES: This file generates reports in xlsx format from a list.List of interfaces
------------------------------------------------------------------------------*/
package main

import (
	"container/list"
	"log"
	"reflect"

	"github.com/tealeg/xlsx"
)

// NAXIMUM ALLOWED ROWS BY EXCEL. https://support.office.com/en-us/article/Excel-specifications-and-limits-1672b34d-7043-467e-8e27-269d656771c3
const ExcelMaxRows = 1048576

/*-----------------------------------------------------------------------------
-- FUNCTION:    generateReport
--
-- DATE:        February 6, 2016
--
-- REVISIONS:	  February 13, 2016 generalised for any list of interface{}s
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:		func generateReport(fname string, connections *list.List)
--     fname:   Name of the file
--  elements:   A list of structures to be reported
--
-- RETURNS: 		void
--
-- NOTES:			This function will only generate a report of up to ExcelMaxRows rows
--            They will all be on "Sheet 1" and a warning will be logged if more
--            data is present.
------------------------------------------------------------------------------*/
func generateReport(fname string, elements *list.List) {
	doc := xlsx.NewFile()
	if elements.Len() <= 0 {
		return
	}
	report, _ := doc.AddSheet("Sheet 1") // TODO: make this more generalised?
	generateHeaders(elements.Front().Value, report.AddRow())
	for e := elements.Front(); e != nil; e = e.Next() {
		generateRow(e.Value, report.AddRow())
		if report.MaxRow >= ExcelMaxRows {
			log.Println("Too many entries for report, stopping at ", report.MaxRow)
			break
		}
	}
	doc.Save(fname + ".xlsx")
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    generateHeaders
--
-- DATE:        February 13, 2016
--
-- REVISIONS:	 (DATE AND INFO)
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:		generateHeaders(i interface{}, row *xlsx.Row)
--         i:   the interface to generate headers from.
--       row:   a pointer for i's field names to be written to.
--
-- RETURNS: 		void
--
-- NOTES:			This function uses reflect to get the name of the element and print
--            the name of the element to a row.
------------------------------------------------------------------------------*/
func generateHeaders(i interface{}, row *xlsx.Row) {
	fields := reflect.ValueOf(i)
	for i := 0; i < fields.NumField(); i++ {
		cell := row.AddCell()
		cell.SetString(fields.Type().Field(i).Name)
	}
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    Generate Report
--
-- DATE:        February 13, 2016
--
-- REVISIONS:	 (DATE AND INFO)
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:		func generateRow(i interface{}, row *xlsx.Row)
--         i:   the interface to generate headers from.
--       row:   a pointer to a row from a spreadsheet for the i to be printed on.
--
-- RETURNS: 		void
--
-- NOTES:			This function uses reflect to get the name of the element and print
--            the name of the element to a row.
------------------------------------------------------------------------------*/
func generateRow(i interface{}, row *xlsx.Row) {
	fields := reflect.ValueOf(i)
	for i := 0; i < fields.NumField(); i++ {
		cell := row.AddCell()
		cell.SetValue(fields.Field(i).Interface())
	}
}
