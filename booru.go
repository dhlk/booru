package booru

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

// database statements
const (
	StatementCreatePosts     = "create table posts (id integer not null primary key autoincrement, timestamp timestamp not null, post text unique not null)"
	StatementCreateTags      = "create table tags (id integer not null primary key autoincrement, tag text unique not null)"
	StatementCreateRelations = "create table relations (post integer not null, tag integer not null, primary key (post, tag))"

	StatementInsertPost = "insert into posts (post, timestamp) values (?, ?)"
	StatementInsertTag  = "insert into tags (tag) values (?)"

	StatementAssociateTag   = "insert into relations (post, tag) values (?, ?)"
	StatementUnAssociateTag = "delete from relations where post = ? and tag = ?"

	StatementPostValidID = "select count(*) from posts where id = ?"
	StatementTagValidID  = "select count(*) from posts where id = ?"

	StatementQueryTagID = "select id from tags where tag = ?"
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

func (b *Booru) NewPost(ctx context.Context, post string, timestamp time.Time) (id int64, err error) {
	// start the transaction
	var transaction *sql.Tx
	if transaction, err = b.db.BeginTx(ctx, nil); err != nil {
		return
	}
	defer transaction.Rollback()

	var res sql.Result
	res, err = transaction.ExecContext(ctx, StatementInsertPost, post, timestamp)
	if err != nil {
		// check if this is a uniqueness violation (duplicate post)
		if strings.HasPrefix(err.Error(), "UNIQUE") {
			id, err = 0, ErrorDuplicatePost
		}

		return
	}

	if id, err = res.LastInsertId(); err != nil {
		return
	}

	err = transaction.Commit()
	return
}

func (b *Booru) validatePostID(ctx context.Context, transaction *sql.Tx, id int64) (err error) {
	row := transaction.QueryRowContext(ctx, StatementPostValidID, id)

	var res int64
	err = row.Scan(&res)
	if err != nil {
		return
	} else if res != 1 {
		return ErrorInvalidPostID
	}

	return nil
}

func (b *Booru) validateTagID(ctx context.Context, transaction *sql.Tx, id int64) (err error) {
	row := transaction.QueryRowContext(ctx, StatementTagValidID, id)

	var res int64
	err = row.Scan(&res)
	if err != nil {
		return
	} else if res != 1 {
		return ErrorInvalidPostID
	}

	return nil
}

func (b *Booru) getTagExisting(ctx context.Context, transaction *sql.Tx, tag string) (id int64, err error) {
	row := transaction.QueryRowContext(ctx, StatementQueryTagID, tag)
	err = row.Scan(&id)
	return
}

func (b *Booru) getTag(ctx context.Context, transaction *sql.Tx, tag string) (id int64, err error) {
	id, err = b.getTagExisting(ctx, transaction, tag)

	if err != nil {
		if err == sql.ErrNoRows {
			// create the tag
			var res sql.Result
			if res, err = transaction.ExecContext(ctx, StatementInsertTag, tag); err != nil {
				return
			}

			var ins int64
			if ins, err = res.LastInsertId(); err != nil {
				return
			}
			id = ins
		} else {
			return
		}
	}

	return
}

func (b *Booru) TagPost(ctx context.Context, postID int64, tag string) error {
	transaction, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	err = b.validatePostID(ctx, transaction, postID)
	if err != nil {
		return err
	}

	tagID, err := b.getTag(ctx, transaction, tag)
	if err != nil {
		return err
	}

	_, err = transaction.ExecContext(ctx, StatementAssociateTag, postID, tagID)
	if err != nil {
		// check if this is a uniqueness violation (duplicate tag)
		// this is a shit check, but w/e legacy software
		if strings.HasPrefix(err.Error(), "UNIQUE") {
			return ErrorDuplicateTag
		}

		return err
	}

	return transaction.Commit()
}

func (b *Booru) UnTagPost(ctx context.Context, postID int64, tag string) error {
	transaction, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	err = b.validatePostID(ctx, transaction, postID)
	if err != nil {
		return err
	}

	tagID, err := b.getTag(ctx, transaction, tag)
	if err != nil {
		return err
	}

	_, err = transaction.ExecContext(ctx, StatementUnAssociateTag, postID, tagID)
	if err != nil {
		return err
	}

	return transaction.Commit()
}
