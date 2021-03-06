package mongoStubs

var TransactionObjects string

func init() {
	TransactionObjects = `
package model

import (
	// "encoding/base64"
	"encoding/json"
	"errors"
	"github.com/DanielRenne/GoCore/core/dbServices"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"log"
)

type TransactionObjects struct{}

var mongoTransactionObjectsCollection *mgo.Collection

func init() {
	go func() {

		for {
			mdb := dbServices.ReadMongoDB()
			if mdb != nil {
				log.Println("Building Indexes for MongoDB collection TransactionObjects:")
				mongoTransactionObjectsCollection = mdb.C("TransactionObjects")
				ci := mgo.CollectionInfo{ForceIdIndex: true}
				mongoTransactionObjectsCollection.Create(&ci)
				var obj TransactionObjects
				obj.Index()
				return
			}
			<-dbServices.WaitForDatabase()
		}
	}()
}

type TransactionObject struct {
	Id         bson.ObjectId ` + "`" + `json:"id" bson:"_id,omitempty"` + "`" + `
	TId        string        ` + "`" + `json:"tId" dbIndex:"index" bson:"tId"` + "`" + `
	DataType   int           ` + "`" + `json:"dataType" bson:"dataType"` + "`" + `
	Collection string        ` + "`" + `json:"collection" bson:"collection"` + "`" + `
	Data       string        ` + "`" + `json:"data" bson:"data"` + "`" + `
	ChgType    int           ` + "`" + `json:"chgType" bson:"chgType"` + "`" + `
}

func (self *TransactionObjects) Single(field string, value interface{}) (retObj TransactionObject, e error) {
	if field == "id" {
		query := mongoTransactionObjectsCollection.FindId(bson.ObjectIdHex(value.(string)))
		e = query.One(&retObj)
		return
	}
	m := make(bson.M)
	m[field] = value
	query := mongoTransactionObjectsCollection.Find(m)
	e = query.One(&retObj)
	return
}

func (obj *TransactionObjects) Search(field string, value interface{}) (retObj []TransactionObject, e error) {
	var query *mgo.Query
	if field == "id" {
		query = mongoTransactionObjectsCollection.FindId(bson.ObjectIdHex(value.(string)))
	} else {
		m := make(bson.M)
		m[field] = value
		query = mongoTransactionObjectsCollection.Find(m)
	}

	e = query.All(&retObj)
	return
}

func (obj *TransactionObjects) SearchAdvanced(field string, value interface{}, limit int, skip int) (retObj []TransactionObject, e error) {
	var query *mgo.Query
	if field == "id" {
		query = mongoTransactionObjectsCollection.FindId(bson.ObjectIdHex(value.(string)))
	} else {
		m := make(bson.M)
		m[field] = value
		query = mongoTransactionObjectsCollection.Find(m)
	}

	if limit == 0 && skip == 0 {
		e = query.All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 && skip > 0 {
		e = query.Limit(limit).Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 {
		e = query.Limit(limit).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if skip > 0 {
		e = query.Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	return
}

func (obj *TransactionObjects) All() (retObj []TransactionObject, e error) {
	e = mongoTransactionObjectsCollection.Find(bson.M{}).All(&retObj)
	if len(retObj) == 0 {
		retObj = []TransactionObject{}
	}
	return
}

func (obj *TransactionObjects) AllAdvanced(limit int, skip int) (retObj []TransactionObject, e error) {
	if limit == 0 && skip == 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 && skip > 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).Limit(limit).Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).Limit(limit).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if skip > 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	return
}

func (obj *TransactionObjects) AllByIndex(index string) (retObj []TransactionObject, e error) {
	e = mongoTransactionObjectsCollection.Find(bson.M{}).Sort(index).All(&retObj)
	if len(retObj) == 0 {
		retObj = []TransactionObject{}
	}
	return
}

func (obj *TransactionObjects) AllByIndexAdvanced(index string, limit int, skip int) (retObj []TransactionObject, e error) {
	if limit == 0 && skip == 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).Sort(index).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 && skip > 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).Sort(index).Limit(limit).Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).Sort(index).Limit(limit).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if skip > 0 {
		e = mongoTransactionObjectsCollection.Find(bson.M{}).Sort(index).Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	return
}

func (obj *TransactionObjects) Range(min, max, field string) (retObj []TransactionObject, e error) {
	var query *mgo.Query
	m := make(bson.M)
	m[field] = bson.M{"$gte": min, "$lte": max}
	query = mongoTransactionObjectsCollection.Find(m)
	e = query.All(&retObj)
	return
}

func (obj *TransactionObjects) RangeAdvanced(min, max, field string, limit int, skip int) (retObj []TransactionObject, e error) {
	var query *mgo.Query
	m := make(bson.M)
	m[field] = bson.M{"$gte": min, "$lte": max}
	query = mongoTransactionObjectsCollection.Find(m)
	if limit == 0 && skip == 0 {
		e = query.All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 && skip > 0 {
		e = query.Limit(limit).Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if limit > 0 {
		e = query.Limit(limit).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	if skip > 0 {
		e = query.Skip(skip).All(&retObj)
		if len(retObj) == 0 {
			retObj = []TransactionObject{}
		}
		return
	}
	return
}

func (obj *TransactionObjects) Index() error {
	for key, value := range dbServices.GetDBIndexes(TransactionObject{}) {
		index := mgo.Index{
			Key:        []string{key},
			Unique:     false,
			Background: true,
		}

		if value == "unique" {
			index.Unique = true
		}

		err := mongoTransactionObjectsCollection.EnsureIndex(index)
		if err != nil {
			log.Println("Failed to create index for Transaction." + key + ":  " + err.Error())
		} else {
			log.Println("Successfully created index for Transaction." + key)
		}
	}
	return nil
}

func (obj *TransactionObjects) New() *TransactionObject {
	return &TransactionObject{}
}

func (self *TransactionObject) Save() error {
	if mongoTransactionObjectsCollection == nil {
		return errors.New("Collection TransactionObjects not initialized")
	}
	objectId := bson.NewObjectId()
	if self.Id != "" {
		objectId = self.Id
	}
	changeInfo, err := mongoTransactionObjectsCollection.UpsertId(objectId, &self)
	if err != nil {
		log.Println("Failed to upsertId for TransactionObject:  " + err.Error())
		return err
	}
	if changeInfo.UpsertedId != nil {
		self.Id = changeInfo.UpsertedId.(bson.ObjectId)
	}
	return nil
}

func (self *TransactionObject) Delete() error {
	return mongoTransactionObjectsCollection.Remove(self)
}

func (obj *TransactionObject) JSONString() (string, error) {
	bytes, err := json.Marshal(obj)
	return string(bytes), err
}

func (obj *TransactionObject) JSONBytes() ([]byte, error) {
	return json.Marshal(obj)
}
`
}
