package booru

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"
)

// database statements
const (
	StatementQueryPost     = "select id, timestamp, post from posts where post = ?"
	StatementQueryPostTags = "select tags.id, tags.tag from relations join tags on relations.tag = tags.id where relations.post = ?"
)

type Tag struct {
	ID  int64
	Tag string
}

func (t Tag) String() string {
	return t.Tag
}

type Tags []Tag

func (t Tags) Len() int {
	return len(t)
}

func (t Tags) Less(i, j int) bool {
	return strings.Compare(t[i].Tag, t[j].Tag) == -1
}

func (t Tags) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t Tags) String() string {
	var str = ""
	for i, tag := range t {
		if i == 0 {
			str = tag.String()
			continue
		}
		str += " " + tag.String()
	}
	return str
}

type Post struct {
	ID   int64
	Time time.Time
	Post string
	Tags Tags
}

// semantics like strings.Compare
type PostCompare func(Post, Post) int

func ComparePostAscending(a, b Post) int {
	if a.ID == b.ID {
		return 0
	}

	if a.Time.Before(b.Time) {
		return -1
	} else if a.Time.After(b.Time) {
		return 1
	}

	if a.Post < b.Post {
		return -1
	} else if b.Post < a.Post {
		return 1
	}

	// unreachable if data isn't bad
	panic(nil)
}

func ComparePostDescending(a, b Post) int {
	return ComparePostAscending(b, a)
}

func (b *Booru) GetPostTags(ctx context.Context, transaction *sql.Tx, id int64) (tags Tags, err error) {
	if transaction == nil {
		if transaction, err = b.db.BeginTx(ctx, nil); err != nil {
			return
		}
		defer transaction.Rollback()
	}

	var rows *sql.Rows
	if rows, err = transaction.QueryContext(ctx, StatementQueryPostTags, id); err != nil {
		return
	}

	for rows.Next() {
		var tag Tag
		if err = rows.Scan(&tag.ID, &tag.Tag); err != nil {
			return
		}

		tags = append(tags, tag)
	}

	if err = rows.Err(); err != nil {
		return
	}

	sort.Sort(tags)

	return
}

func (b *Booru) GetPost(ctx context.Context, resource string) (post Post, err error) {
	var transaction *sql.Tx
	transaction, err = b.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer transaction.Rollback()

	// get the post itself
	var row *sql.Row
	row = transaction.QueryRowContext(ctx, StatementQueryPost, resource)
	if err = row.Scan(&post.ID, &post.Time, &post.Post); err != nil {
		return
	}

	if post.Tags, err = b.GetPostTags(ctx, transaction, post.ID); err != nil {
		return
	}

	transaction.Commit()

	return
}
