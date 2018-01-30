package database

import (
	log "github.com/sirupsen/logrus"

	"strings"

	"fmt"

	common "github.com/tb0hdan/torpedo_common"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MongoDB struct {
	logger   *log.Logger
	DBURI    string
	Database string
}

type TorpedoStats struct {
	ProcessedMessagesTotal int64
}

func (mdb *MongoDB) GetSession() (session *mgo.Session, err error) {
	session, err = mgo.Dial(mdb.DBURI)
	if err != nil {
		mdb.logger.Panic(err)
	}
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	return
}

func (mdb *MongoDB) GetCollection(collectionName string) (session *mgo.Session, collection *mgo.Collection, err error) {
	session, err = mdb.GetSession()
	if err != nil {
		return
	}
	collection = session.DB(mdb.Database).C(collectionName)
	return
}

func (mdb *MongoDB) GetUpdateTotalMessages(step int64) (count int64) {
	session, collection, err := mdb.GetCollection("messagestats")
	if err != nil {
		mdb.logger.Printf("GetUpdateTotalMessages failed with: %+v\n", err)
		return
	}
	defer session.Close()
	result := TorpedoStats{}
	err = collection.Find(bson.M{}).One(&result)
	if err != nil {
		mdb.logger.Printf("No stats available: %+v\n", err)
		count = 1
		err = collection.Insert(&result)
		if err != nil {
			mdb.logger.Fatal(err)
		}
	} else {
		count = result.ProcessedMessagesTotal
	}
	result = TorpedoStats{ProcessedMessagesTotal: count + step}
	err = collection.Update(bson.M{}, result)
	if err != nil {
		mdb.logger.Printf("Failed to update stats: %+v\n", err)
	}
	return
}

func New(db_uri, database string) (mongodb *MongoDB) {
	var server string
	cu := &common.Utils{}
	// Handle empty DB name
	if database == "" {
		database = "torpedobot"
	}
	//
	if db_uri == "" {
		server = "localhost"
	} else if strings.HasPrefix(db_uri, "mongodb://") {
		server = db_uri
		// override DB name
		creds := strings.Split(db_uri, "mongodb://")[1]
		if len(strings.Split(creds, "/")) == 2 {
			database = strings.Split(creds, "/")[1]
			// non-empty db param (auth db)
		} else if database != "" {
			//
			server += fmt.Sprintf("/%s", database)
		}
	} else {
		// fallback to host and use database parameter
		server = db_uri
	}
	mongodb = &MongoDB{DBURI: server,
		Database: database}
	mongodb.logger = cu.NewLog("MongoDB")
	mongodb.logger.Printf("MongoDB connector: server `%s` - database `%s`", server, database)
	return
}
