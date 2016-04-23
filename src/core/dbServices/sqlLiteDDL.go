package dbServices

import (
	"core/extensions"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strings"
)

type tableSchema struct {
	CId          int
	Name         string
	FieldType    string
	NotNull      int
	DefaultValue sql.NullString
	PrimaryKey   int
}

func createSQLiteTables(tables []tableDef) {

	for _, table := range tables {

		currentSchema := getSQLiteTableSchema(table)
		doWeAlter := isAlterRequired(table, currentSchema)

		if doWeAlter {
			renameSQLiteTable(table.Name, table.Name+"_alterTemp")
		}

		fmt.Printf("%+v\n", doWeAlter)
		sqlStmt := generateSQLiteTableCreate(table, []foreignKeyDef{})

		fmt.Println(sqlStmt)

		_, errDBExec := DB.Exec(sqlStmt)
		if errDBExec != nil {
			fmt.Println("Creation of table \"" + table.Name + "\" failed:  " + errDBExec.Error())
			continue
		}

		if doWeAlter {
			copySQLiteTableWithAlter(table.Name+"_alterTemp", table.Name, currentSchema, table)
			dropSQLiteTable(table.Name + "_alterTemp")
		}

		fmt.Println("Creation of table \"" + table.Name + "\" successful.")
	}
}

func isAlterRequired(table tableDef, existingSchema []tableSchema) bool {
	if len(existingSchema) == 0 {
		return false
	}

	if len(table.Fields) < len(existingSchema) {
		return true
	}

	if len(table.Fields) > len(existingSchema) {
		return true
	}

	for i, schemaRow := range existingSchema {

		field := table.Fields[i]
		if field.Name != schemaRow.Name {
			return true
		}
		if field.FieldType != schemaRow.FieldType {
			return true
		}
		if field.AllowNull == true && schemaRow.NotNull == 1 {
			return true
		}
		if field.AllowNull == false && schemaRow.NotNull == 0 {
			return true
		}
		if field.Primary == true && schemaRow.PrimaryKey == 0 {
			return true
		}
		if field.Primary == false && schemaRow.PrimaryKey == 1 {
			return true
		}
		if field.Default == "" && schemaRow.DefaultValue.Valid == true {
			return true
		}
		if (field.Default != "" && schemaRow.DefaultValue.Valid == true) && (field.Default != schemaRow.DefaultValue.String) {
			return true
		}
	}

	return false
}

func generateSQLiteTableCreate(table tableDef, keys []foreignKeyDef) string {
	sqlStmt := "create table if not exists "
	sqlStmt += table.Name + " "
	sqlStmt += "("

	for _, field := range table.Fields {
		sqlStmt += field.Name + " "
		sqlStmt += field.FieldType + " "
		if field.AllowNull == false {
			sqlStmt += "not null "
		}
		if field.Primary == true {
			sqlStmt += "primary key "
		}
		if field.IsUnique == true {
			sqlStmt += "unique "
		}
		if field.Check != "" {
			sqlStmt += "check(" + field.Check + ") "
		}
		if field.Collate != "" {
			sqlStmt += "collate " + field.Collate + " "
		}
		if field.Default != "" {
			fieldChar := getSQLiteFieldCharacter(field.FieldType)
			sqlStmt += "Default " + fieldChar + field.Default + fieldChar + " "
		}

		sqlStmt += ","
	}

	sqlStmt = extensions.TrimSuffix(sqlStmt, ",")

	sqlStmt += genSQLiteForeignKeyCreate(keys)

	sqlStmt += ");"
	fmt.Println(sqlStmt)

	return sqlStmt
}

func genSQLiteForeignKeyCreate(keys []foreignKeyDef) string {

	sqlStmt := ""
	for _, fk := range keys {
		cascades := ""
		if fk.OnDelete == true {
			cascades += " ON DELETE CASCADE"
		}
		if fk.OnUpdate == true {
			cascades += " ON UPDATE CASCADE"
		}

		tblFields := ""
		for _, f := range fk.Fields {
			tblFields += f + ","
		}

		tblFields = extensions.TrimSuffix(tblFields, ",")

		fkFields := ""
		for _, f := range fk.FKFields {
			fkFields += f + ","
		}

		fkFields = extensions.TrimSuffix(fkFields, ",")

		sqlStmt += ", FOREIGN KEY (" + tblFields + ") references " + fk.FKTable + "(" + fkFields + ") " + cascades
	}

	return sqlStmt
}

func createSQLiteIndexes(indexes []indexDef) {
	for _, index := range indexes {
		sqlStmt := "create "
		if index.IsUnique == true {
			sqlStmt += "unique "
		}

		sqlStmt += "index if not exists "
		sqlStmt += index.Name + " on "
		sqlStmt += index.TableName + " ("

		for _, field := range index.Fields {
			sqlStmt += field
			sqlStmt += ","
		}

		sqlStmt = extensions.TrimSuffix(sqlStmt, ",")

		sqlStmt += ");"

		_, errDBExec := DB.Exec(sqlStmt)
		if errDBExec != nil {
			fmt.Println("Creation of index \"" + index.Name + "\" failed:  " + errDBExec.Error())
			continue
		}

		fmt.Println("Creation of index \"" + index.Name + "\" successful.")
	}
}

func createSQLiteForeignKeys(foreignKeys []foreignKeyTableDef, tables []tableDef) {
	for _, fk := range foreignKeys {

		renameSQLiteTable(fk.Table, fk.Table+"_temp")

		//Create the new Table with FK's
		for _, table := range tables {
			if table.Name == fk.Table {
				sqlStmt := generateSQLiteTableCreate(table, fk.Keys)
				_, errDBExec := DB.Exec(sqlStmt)
				if errDBExec != nil {
					fmt.Println("Creation of table with Foreign Keys Failed:  Table  \"" + fk.Table + "_temp" + "\" " + errDBExec.Error())
					return
				}
				fmt.Println("Creation of table \"" + fk.Table + "\" successful.")
				break
			}
		}

		copySQLiteTable(fk.Table+"_temp", fk.Table)
		dropSQLiteTable(fk.Table + "_temp")
	}
}

func renameSQLiteTable(original string, newName string) {

	sqlStmt := "ALTER TABLE " + original + " RENAME TO " + newName + ";"

	fmt.Println("Renaming table from " + original + " to " + newName)

	_, errDBExec := DB.Exec(sqlStmt)
	if errDBExec != nil {
		fmt.Println("Rename of table \"" + original + "\" failed:  " + errDBExec.Error())
	}
	fmt.Println("Renamed " + original + " to " + newName)
}

func copySQLiteTable(from string, to string) {

	sqlStmt := "INSERT INTO " + to + " SELECT * FROM " + from + ";"

	fmt.Println("Copying Table Data from " + from + " to " + to)

	_, errDBExec := DB.Exec(sqlStmt)
	if errDBExec != nil {
		fmt.Println("Copy of table \"" + from + "\" to " + "\"" + to + "\" failed:  " + errDBExec.Error())
	}
	fmt.Println("Copied Data from " + from + " to " + to + " successfully.")
}

func copySQLiteTableWithAlter(from string, to string, currentSchema []tableSchema, table tableDef) {

	sqlStmt := "INSERT INTO " + to + " ("
	sqlStmt2 := "SELECT "
	schemaLength := len(currentSchema)

	for i, field := range table.Fields {
		if schemaLength == i {
			break
		}
		sqlStmt += field.Name + ","
		sqlStmt2 += currentSchema[i].Name + ","
	}

	sqlStmt = extensions.TrimSuffix(sqlStmt, ",")
	sqlStmt2 = extensions.TrimSuffix(sqlStmt2, ",")

	sqlStmt += ") "
	sqlStmt2 += " FROM " + from + ";"

	sqlStmt += sqlStmt2

	fmt.Println("Copying Table Data from " + from + " to " + to)
	fmt.Println(sqlStmt)

	_, errDBExec := DB.Exec(sqlStmt)
	if errDBExec != nil {
		fmt.Println("Copy of table \"" + from + "\" to " + "\"" + to + "\" failed:  " + errDBExec.Error())
	}
	fmt.Println("Copied Data from " + from + " to " + to + " successfully.")
}

func dropSQLiteTable(tableName string) {

	sqlStmt := "DROP TABLE IF EXISTS " + tableName + ";"

	fmt.Println("Dropping table " + tableName)

	_, errDBExec := DB.Exec(sqlStmt)
	if errDBExec != nil {
		fmt.Println("Drop of Table \"" + tableName + "\" failed:  " + errDBExec.Error())
	}
	fmt.Println("Dropped Table " + tableName + " successfully.")
}

func getSQLiteFieldCharacter(value string) string {
	if strings.Contains(value, "CHAR") || strings.Contains(value, "TEXT") || strings.Contains(value, "CLOB") {
		return "'"
	}

	return ""
}

func getSQLiteTableSchema(table tableDef) []tableSchema {

	schemaRows := []tableSchema{}

	rows, err := DB.Query("PRAGMA table_info(" + table.Name + ");")
	if err != nil {
		fmt.Println("Prepare of DB Query Failed.  " + err.Error())
		return schemaRows
	}
	defer rows.Close()

	for rows.Next() {
		var schema tableSchema
		err = rows.Scan(&schema.CId, &schema.Name, &schema.FieldType, &schema.NotNull, &schema.DefaultValue, &schema.PrimaryKey)
		if err != nil {
			fmt.Println("Scan failed to load Table Schema for table \"" + table.Name + "\":  " + err.Error())
		}
		schemaRows = append(schemaRows, schema)
	}
	return schemaRows
}