package booru

import (
	"context"
	"database/sql"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/dhlk/booru/parse"
)

var nothing CancelableStream = func(context.Context) <-chan Post {
	c := make(chan Post)
	close(c)
	return c
}

var compare = ComparePostDescending

func (b *Booru) query(query string) (result CancelableStream, err error) {
	// parse the query
	var tree *parse.Tree
	if tree, err = parse.Parse(query); err != nil {
		log.Printf("%v", err)
		return
	}

	result = b.queryForNode(tree.Root)
	return
}

func (b *Booru) Query(ctx context.Context, query string, page, length int64) (posts []Post, err error) {
	fwdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var results CancelableStream
	if results, err = b.query(query); err != nil {
		return
	}
	selection := Limit(Skip(results, page*length), length)(fwdCtx)

	// open the transaction and add tag data
	var transaction *sql.Tx
	if transaction, err = b.db.BeginTx(ctx, nil); err != nil {
		log.Printf("%v", err)
		return
	}
	defer transaction.Rollback()

	for post := range selection {
		if post.Tags, err = b.GetPostTags(ctx, transaction, post.ID); err != nil {
			log.Printf("%v", err)
			return
		}
		posts = append(posts, post)
	}
	err = transaction.Commit()

	return
}

func (b *Booru) queryForNode(node parse.Node) CancelableStream {
	switch node.Type() {
	case parse.NodeCond:
		return b.queryConditionalNode(node.(*parse.CondNode))
	case parse.NodeLess:
		return b.queryLessNode(node.(*parse.LessNode))
	case parse.NodeWord:
		return b.queryWordNode(node.(*parse.WordNode))
	}

	// should be unreachable
	panic(nil)
}

func (b *Booru) queryConditionalNode(cond *parse.CondNode) CancelableStream {
	orArr := make([]CancelableStream, len(cond.Or))
	for i, or := range cond.Or {
		orArr[i] = b.queryForNode(or)
	}

	andArr := make([]CancelableStream, len(cond.And))
	for i, and := range cond.And {
		andArr[i] = b.queryForNode(and)
	}
	if len(orArr) > 0 {
		andArr = append(andArr, Union(orArr, compare))
	}

	if len(andArr) == 0 {
		return b.queryEveryPost()
	}
	return Intersection(andArr, compare)
}

func (b *Booru) queryLessNode(less *parse.LessNode) CancelableStream {
	return Complement(b.queryForNode(less.Less), b.queryEveryPost(), compare)
}

func (b *Booru) queryWordNode(word *parse.WordNode) CancelableStream {
	tag := string(word.Word)

	// load baseline subquery
	if strings.HasPrefix(tag, "baseline:") {
		baseline := strings.Replace(tag, "baseline:", "", -1)
		query, err := ioutil.ReadFile(filepath.Join(b.baseline, baseline))
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}

		result, err := b.query(string(query))
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}
		return result
	}

	// load regex subquery
	if strings.HasPrefix(tag, "regex:") {
		if err := b.GenerateTagIndex(context.TODO()); err != nil {
			log.Printf("%v", err)
			return nothing
		}

		regex := strings.Replace(tag, "regex:", "", -1)
		query, err := b.generateTagQuery(context.TODO(), regex)
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}

		result, err := b.query(string(query))
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}
		return result
	}

	return b.indexStreamCancelable(string(word.Word))
}

func (b *Booru) queryEveryPost() CancelableStream {
	return b.indexStreamCancelable(globalIndexTag)
}
