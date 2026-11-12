package service

import (
 "strings"

 "gorm.io/gorm"
 "gorm.io/gorm/clause"
)

// ApplyColumnSort orders a query by one of its actual mapped database columns.
// The selected column is resolved through GORM schema metadata, so user input
// cannot introduce SQL expressions or additional clauses.
func ApplyColumnSort(tx *gorm.DB, key, direction string) *gorm.DB {
 if tx==nil || tx.Statement==nil {return tx}
 key=strings.TrimSpace(key)
 direction=strings.ToLower(strings.TrimSpace(direction))
 if key=="" || (direction!="asc" && direction!="desc"){return tx}
 stmt:=tx.Statement
 if stmt.Schema==nil {
  if stmt.Model==nil || stmt.Parse(stmt.Model)!=nil {return tx}
 }
 field:=stmt.Schema.LookUpField(key)
 if field==nil || field.DBName=="" || !field.Readable {return tx}
 desc:=direction=="desc"
 sortColumns:=[]clause.OrderByColumn{{
  Column:clause.Column{Table:clause.CurrentTable,Name:field.DBName},
  Desc:desc,
 }}
 if primary:=stmt.Schema.PrioritizedPrimaryField;primary!=nil&&primary.DBName!=""&&primary.DBName!=field.DBName {
  sortColumns=append(sortColumns,clause.OrderByColumn{
   Column:clause.Column{Table:clause.CurrentTable,Name:primary.DBName},Desc:false,
  })
 }
 // Assign the clause directly to replace default ordering, including order
 // clauses installed by existing list services.
 if stmt.Clauses==nil {stmt.Clauses=map[string]clause.Clause{}}
 stmt.Clauses["ORDER BY"]=clause.Clause{Name:"ORDER BY",Expression:clause.OrderBy{Columns:sortColumns}}
 return tx
}
