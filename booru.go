package booru

import (
	"database/sql"
	"errors"
)

// database statements
const (
	StatementCreatePosts     = "create table posts (id integer not null primary key autoincrement, timestamp timestamp not null, post text unique not null)"
	StatementCreateTags      = "create table tags (id integer not null primary key autoincrement, tag text unique not null)"
	StatementCreateRelations = "create table relations (post integer not null, tag integer not null, primary key (post, tag))"
)

// errors
var (
	ErrorDuplicatePost = errors.New("duplicate post")
	ErrorDuplicateTag  = errors.New("duplicate tag")
	ErrorInvalidPostID = errors.New("invalid post id")
)

type Booru struct {
	db       *sql.DB
	index    string
	baseline string
}

func New(db *sql.DB, index, baseline string) *Booru {
	return &Booru{db, index, baseline}
}

var createStatements = []string{StatementCreatePosts, StatementCreateTags, StatementCreateRelations}

// Initialize the database with all the tables used by the booru.
func (b *Booru) InitDB() (err error) {
	var transaction *sql.Tx
	transaction, err = b.db.Begin()
	if err != nil {
		return
	}

	for _, create := range createStatements {
		_, err = transaction.Exec(create)
		if err != nil {
			transaction.Rollback()
			return
		}
	}

	transaction.Commit()

	return
}

func (b *Booru) Close() error {
	return b.db.Close()
}
