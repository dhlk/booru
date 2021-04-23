package booru

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

// TODO removal of failed index generation should probably be some kind of recovery

// database statements
const (
	StatementQueryTags        = "select tags.tag from tags order by tags.tag"
	StatementQueryTaggedPosts = `
select posts.id, posts.timestamp, posts.post from posts join relations on posts.id = relations.post join tags on relations.tag = tags.id where tags.tag=? order by posts.timestamp desc, posts.post desc`
	StatementQueryEveryPost = `select posts.id, posts.timestamp, posts.post from posts order by posts.timestamp desc, posts.post desc`
)

const globalIndexTag = "\000"
const tagIndexName = "tags"

func (b *Booru) indexPath(tag string) string {
	return filepath.Join(b.index, hex.EncodeToString([]byte(tag)))
}

func indexExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (b *Booru) indexStreamCancelable(tag string) CancelableStream {
	return func(ctx context.Context) <-chan Post {
		return b.indexStream(ctx, tag)
	}
}

func (b *Booru) indexStream(ctx context.Context, tag string) <-chan Post {
	resultFull := make(chan Post)

	go func(result chan<- Post) {
		defer close(resultFull)

		b.generateIndex(ctx, tag)

		index, err := os.Open(b.indexPath(tag))
		if err != nil {
			log.Printf("%v", err)
			return
		}
		defer index.Close()

		decoder := json.NewDecoder(index)

		for decoder.More() {
			var post Post
			err := decoder.Decode(&post)
			if err != nil {
				log.Printf("%v", err)
				return
			}

			select {
			case <-ctx.Done():
				return
			case result <- post:
			}
		}
	}(resultFull)

	return resultFull
}

func (b *Booru) generateIndex(ctx context.Context, tag string) (err error) {
	indexPath := b.indexPath(tag)
	if indexExists(indexPath) {
		return
	}

	var index *os.File
	if index, err = os.Create(indexPath); err != nil {
		return
	}
	defer index.Close()

	encoder := json.NewEncoder(index)

	var rows *sql.Rows
	if tag == globalIndexTag {
		rows, err = b.db.QueryContext(ctx, StatementQueryEveryPost)
	} else {
		rows, err = b.db.QueryContext(ctx, StatementQueryTaggedPosts, tag)
	}
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		if err = rows.Scan(&post.ID, &post.Time, &post.Post); err != nil {
			index.Close()
			os.Remove(indexPath)
			return
		}
		if err = encoder.Encode(post); err != nil {
			index.Close()
			os.Remove(indexPath)
			return
		}
	}

	if err = rows.Err(); err != nil {
		index.Close()
		os.Remove(indexPath)
		return
	}

	return
}

func (b *Booru) GenerateTagIndex(ctx context.Context) (err error) {
	indexPath := filepath.Join(b.index, tagIndexName)
	if indexExists(indexPath) {
		return
	}

	var rows *sql.Rows
	if rows, err = b.db.QueryContext(ctx, StatementQueryTags); err != nil {
		return
	}

	var index *os.File
	if index, err = os.Create(indexPath); err != nil {
		return
	}
	defer index.Close()

	encoder := json.NewEncoder(index)

	for rows.Next() {
		var tag string
		if err = rows.Scan(&tag); err != nil {
			index.Close()
			os.Remove(indexPath)
			return
		}
		if err = encoder.Encode(tag); err != nil {
			index.Close()
			os.Remove(indexPath)
			return
		}
	}

	if err = rows.Err(); err != nil {
		index.Close()
		os.Remove(indexPath)
		return
	}

	return
}

func (b *Booru) generateTagQuery(ctx context.Context, pattern string) (query string, err error) {
	var regex *regexp.Regexp
	if regex, err = regexp.Compile(pattern); err != nil {
		return
	}

	var index *os.File
	if index, err = os.Open(filepath.Join(b.index, tagIndexName)); err != nil {
		return
	}
	defer index.Close()

	decoder := json.NewDecoder(index)

	// TODO context - handle cancel
	for decoder.More() {
		var tag string
		if err = decoder.Decode(&tag); err != nil {
			return
		}

		if regex.MatchString(tag) {
			query = query + "~" + tag + " "
		}
	}

	if len(query) == 0 {
		err = errors.New("regex: no matching tags")
	}

	return
}

func (b *Booru) GenerateIndexes(ctx context.Context) (err error) {
	if err = b.generateIndex(ctx, globalIndexTag); err != nil {
		return
	}

	var rows *sql.Rows
	if rows, err = b.db.QueryContext(ctx, StatementQueryTags); err != nil {
		return
	}

	for rows.Next() {
		var tag string
		if err = rows.Scan(&tag); err != nil {
			return
		}
		if err = b.generateIndex(ctx, tag); err != nil {
			return
		}
	}

	if err = rows.Err(); err != nil {
		return
	}

	return
}
