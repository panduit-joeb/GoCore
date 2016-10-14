package model

import (
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/DanielRenne/GoCore/core/dbServices"
	"github.com/asaskevich/govalidator"
	"github.com/fatih/camelcase"
)

const (
	TRANSACTION_DATATYPE_ORIGINAL = 1
	TRANSACTION_DATATYPE_NEW      = 2

	TRANSACTION_CHANGETYPE_INSERT = 1
	TRANSACTION_CHANGETYPE_UPDATE = 2
	TRANSACTION_CHANGETYPE_DELETE = 3

	MGO_RECORD_NOT_FOUND = "not found"

	VALIDATION_ERROR                   = "ValidationError"
	VALIDATION_ERROR_REQUIRED          = "ValidationErrorRequiredFieldMissing"
	VALIDATION_ERROR_EMAIL             = "ValidationErrorInvalidEmail"
	VALIDATION_ERROR_SPECIFIC_REQUIRED = "ValidationFieldSpecificRequired"
	VALIDATION_ERROR_SPECIFIC_EMAIL    = "ValidationFieldSpecificEmailRequired"
)

type modelEntity interface {
	Save() error
	Delete() error
	SaveWithTran(*Transaction) error
	Reflect() []Field
	JoinFields(string, *Query, int) error
}

type modelCollection interface {
	Rollback(transactionId string) error
}

type collection interface {
	Query() *Query
}

type tQueue struct {
	sync.RWMutex
	queue map[string]*transactionsToPersist
}

type transactionsToPersist struct {
	t             *Transaction
	newItems      []entityTransaction
	originalItems []entityTransaction
	startTime     time.Time
}

type entityTransaction struct {
	changeType int
	committed  bool
	entity     modelEntity
}

type Field struct {
	Name       string
	Label      string
	DataType   string
	IsView     bool
	Validation *dbServices.FieldValidation
}

var transactionQueue tQueue

func init() {
	transactionQueue.queue = make(map[string]*transactionsToPersist)
	go clearTransactionQueue()
}

func Q(k string, v interface{}) map[string]interface{} {
	return map[string]interface{}{k: v}
}

//Every 12 hours check the transactionQueue and remove any outstanding stale transactions > 48 hours old
func clearTransactionQueue() {

	transactionQueue.Lock()

	for key, value := range transactionQueue.queue {

		if time.Since(value.startTime).Hours() > 48 {
			delete(transactionQueue.queue, key)
		}
	}

	transactionQueue.Unlock()

	time.Sleep(12 * time.Hour)
	clearTransactionQueue()
}

func getBase64(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func decodeBase64(value string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}

	return string(data[:]), nil
}

func getNow() time.Time {
	return time.Now()
}

func removeDuplicates(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	if err.Error() == VALIDATION_ERROR || err.Error() == VALIDATION_ERROR_EMAIL {
		return true
	}
	return false
}

func validateFields(x interface{}, objectToUpdate interface{}, val reflect.Value) error {

	isError := false
	for key, value := range dbServices.GetValidationTags(x) {

		fieldValue := dbServices.GetReflectionFieldValue(key, objectToUpdate)
		validations := strings.Split(value, ",")

		if validations[0] != "" {
			if err := validateRequired(fieldValue, validations[0]); err != nil {
				dbServices.SetFieldValue("Errors."+key, val, VALIDATION_ERROR_SPECIFIC_REQUIRED)
				isError = true
			}
		}
		if validations[1] != "" {

			cleanup, err := validateType(fieldValue, validations[1])

			if err != nil {
				if err.Error() == VALIDATION_ERROR_EMAIL {
					dbServices.SetFieldValue("Errors."+key, val, VALIDATION_ERROR_SPECIFIC_EMAIL)
				}
				isError = true
			}

			if cleanup != "" {
				dbServices.SetFieldValue(key, val, cleanup)
			}

		}

	}

	if isError {
		return errors.New(VALIDATION_ERROR)
	}

	return nil
}

func validateRequired(value string, tagValue string) error {
	if tagValue == "true" {
		if value == "" {
			return errors.New(VALIDATION_ERROR_REQUIRED)
		}
		return nil
	}
	return nil
}

func validateType(value string, tagValue string) (string, error) {
	switch tagValue {
	case dbServices.VALIDATION_TYPE_EMAIL:
		return "", validateEmail(value)
	}
	return "", nil
}

func validateEmail(value string) error {
	if !govalidator.IsEmail(value) {
		return errors.New(VALIDATION_ERROR_EMAIL)
	}
	return nil
}

func getJoins(x reflect.Value, remainingRecursions string) (joins []join) {
	if remainingRecursions == "" {
		return
	}

	fields := strings.Split(remainingRecursions, ".")
	fieldName := fields[0]

	joinsField := x.FieldByName("Joins")

	if joinsField.Kind() != reflect.Struct {
		return
	}

	if fieldName == JOIN_ALL {
		for i := 0; i < joinsField.NumField(); i++ {

			typeField := joinsField.Type().Field(i)
			name := typeField.Name
			tagValue := typeField.Tag.Get("join")
			splitValue := strings.Split(tagValue, ",")
			var j join
			j.collectionName = splitValue[0]
			j.joinSchemaName = splitValue[1]
			j.joinFieldRefName = splitValue[2]
			j.joinFieldName = name
			j.joinSpecified = JOIN_ALL
			joins = append(joins, j)
		}
	} else {
		typeField, ok := joinsField.Type().FieldByName(fieldName)
		if ok == false {
			return
		}
		name := typeField.Name
		tagValue := typeField.Tag.Get("join")
		splitValue := strings.Split(tagValue, ",")
		var j join
		j.collectionName = splitValue[0]
		j.joinSchemaName = splitValue[1]
		j.joinFieldRefName = splitValue[2]
		j.joinFieldName = name
		j.joinSpecified = strings.Replace(remainingRecursions, fieldName+".", "", 1)
		joins = append(joins, j)
	}
	return
}

func IsZeroOfUnderlyingType(x interface{}) bool {
	return reflect.DeepEqual(x, reflect.Zero(reflect.TypeOf(x)).Interface())
}

func Reflect(obj interface{}) []Field {
	var ret []Field
	val := reflect.ValueOf(obj)

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		if typeField.Name != "Errors" && typeField.Name != "Joins" && typeField.Name != "Id" {
			if typeField.Name == "Views" {
				for f := 0; f < val.FieldByName("Views").NumField(); f++ {
					field := Field{}
					field.IsView = true
					name := val.FieldByName("Views").Type().Field(f).Name
					namePart := camelcase.Split(name)
					for x := 0; x < len(namePart); x++ {
						if x > 0 {
							namePart[x] = strings.ToLower(namePart[x])
						}
					}
					field.Name = val.FieldByName("Views").Type().Field(f).Name
					field.Label = strings.Join(namePart[:], " ")
					field.DataType = val.FieldByName("Views").Type().Field(f).Type.Name()
					validate := val.FieldByName("Views").Type().Field(f).Tag.Get("validate")
					if validate != "" {
						//core.Debug.Dump(validate)
						//parts := strings.Split(validate, ",")
						//field.Validation.Required = true//extensions.StringToBool(parts[0])
						//field.Validation.Type = parts[1]
						//field.Validation.Min = parts[2]
						//field.Validation.Max = parts[3]
						//field.Validation.Length = parts[4]
						//field.Validation.LengthMax = parts[4]
						//field.Validation.LengthMin = parts[5]
					}
					ret = append(ret, field)
				}
			} else {
				field := Field{}
				validate := typeField.Tag.Get("validate")
				if validate != "" {
					//core.Debug.Dump(validate)
					//parts := strings.Split(validate, ",")
					//core.Debug.Dump(extensions.StringToBool(parts[0]))
					//field.Validation.Required = extensions.StringToBool(parts[0])
					//field.Validation.Type = parts[1]
					//field.Validation.Min = parts[2]
					//field.Validation.Max = parts[3]
					//field.Validation.Length = parts[4]
					//field.Validation.LengthMax = parts[4]
					//field.Validation.LengthMin = parts[5]
				}
				name := typeField.Name
				namePart := camelcase.Split(name)
				for x := 0; x < len(namePart); x++ {
					if x > 0 {
						namePart[x] = strings.ToLower(namePart[x])
					}
				}
				field.Name = typeField.Name
				field.Label = strings.Join(namePart[:], " ")
				field.DataType = typeField.Type.Name()
				ret = append(ret, field)
			}
		}
	}
	return ret
}

func JoinEntity(collectionQ *Query, y interface{}, j join, id string, manyItems interface{}, fieldToSet reflect.Value, remainingRecursions string, q *Query, endRecursion bool, recursionCount int) (err error) {
	if IsZeroOfUnderlyingType(fieldToSet.Interface()) {

		if j.isMany {
			err = collectionQ.Filter(Q(j.joinForeignFieldName, id)).All(y)
		} else {
			if j.joinForeignFieldName == "" {
				err = collectionQ.ById(id, y)
			} else {
				err = collectionQ.Filter(Q(j.joinForeignFieldName, id)).One(y)
			}
		}

		if err == nil {
			if endRecursion == false && recursionCount > 0 {
				recursionCount--

				in := []reflect.Value{}
				in = append(in, reflect.ValueOf(remainingRecursions))
				in = append(in, reflect.ValueOf(q))
				in = append(in, reflect.ValueOf(recursionCount))

				if j.isMany {

					myArray := reflect.ValueOf(y).Elem()
					for i := 0; i < myArray.Len(); i++ {
						s := myArray.Index(i)
						err = CallMethod(s.Interface(), "JoinFields", in)
					}
				} else {
					err = CallMethod(y, "JoinFields", in)
				}

			}
			if err == nil {
				if j.isMany {
					var ji AccountJoinItems
					fieldToSet.Set(reflect.ValueOf(&ji))

					itemsField := fieldToSet.Elem().FieldByName("Items")
					countField := fieldToSet.Elem().FieldByName("Count")
					itemsField.Set(reflect.ValueOf(y))
					countField.Set(reflect.ValueOf(reflect.ValueOf(y).Elem().Len()))
				} else {
					fieldToSet.Set(reflect.ValueOf(y))
				}

				if q.renderViews {
					err = q.processViews(y)
					if err != nil {
						return
					}
				}

			}
		}
	} else {
		if endRecursion == false && recursionCount > 0 {
			recursionCount--
			method := fieldToSet.MethodByName("JoinFields")
			in := []reflect.Value{}
			in = append(in, reflect.ValueOf(remainingRecursions))
			in = append(in, reflect.ValueOf(q))
			in = append(in, reflect.ValueOf(recursionCount))
			values := method.Call(in)
			if values[0].Interface() == nil {
				err = nil
				return
			}
			err = values[0].Interface().(error)
		}
	}
	return
}

func CallMethod(i interface{}, methodName string, in []reflect.Value) (err error) {
	var ptr reflect.Value
	var value reflect.Value
	var finalMethod reflect.Value

	value = reflect.ValueOf(i)

	// if we start with a pointer, we need to get value pointed to
	// if we start with a value, we need to get a pointer to that value
	if value.Type().Kind() == reflect.Ptr {
		ptr = value
		value = ptr.Elem()
	} else {
		ptr = reflect.New(reflect.TypeOf(i))
		temp := ptr.Elem()
		temp.Set(value)
	}

	// check for method on value
	method := value.MethodByName(methodName)
	if method.IsValid() {
		finalMethod = method
	}
	// check for method on pointer
	method = ptr.MethodByName(methodName)
	if method.IsValid() {
		finalMethod = method
	}

	if finalMethod.IsValid() {
		values := finalMethod.Call(in)
		if values[0].Interface() == nil {
			err = nil
			return
		}
		err = values[0].Interface().(error)
		return
	}

	// return or panic, method not found of either type
	return nil
}

// Start of autogenerated code....
