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

var nothing <-chan Post = func() <-chan Post {
	c := make(chan Post)
	close(c)
	return c
}()

func (b *Booru) query(ctx context.Context, query string) (result <-chan Post, err error) {
	// parse the query
	var tree *parse.Tree
	if tree, err = parse.Parse(query); err != nil {
		log.Printf("%v", err)
		return
	}

	result = b.queryForNode(ctx, tree.Root)
	return
}

func (b *Booru) Query(ctx context.Context, query string, page, length int64) (posts []Post, err error) {
	fwdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var results <-chan Post
	if results, err = b.query(fwdCtx, query); err != nil {
		return
	}
	selection := Limit(fwdCtx, Skip(fwdCtx, results, page*length), length)

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

func (b *Booru) queryForNode(ctx context.Context, node parse.Node) <-chan Post {
	switch node.Type() {
	case parse.NodeCond:
		return b.queryConditionalNode(ctx, node.(*parse.CondNode))
	case parse.NodeLess:
		return b.queryLessNode(ctx, node.(*parse.LessNode))
	case parse.NodeWord:
		return b.queryWordNode(ctx, node.(*parse.WordNode))
	}

	// should be unreachable
	panic(nil)
}

func (b *Booru) queryConditionalNode(ctx context.Context, cond *parse.CondNode) <-chan Post {
	orArr := make([]<-chan Post, len(cond.Or))
	for i, or := range cond.Or {
		orArr[i] = b.queryForNode(ctx, or)
	}

	andArr := make([]<-chan Post, len(cond.And))
	for i, and := range cond.And {
		andArr[i] = b.queryForNode(ctx, and)
	}
	if len(orArr) > 0 {
		andArr = append(andArr, Or(ctx, orArr...))
	}

	if len(andArr) == 0 {
		return b.queryEveryPost(ctx)
	}
	return And(ctx, andArr...)
}

func (b *Booru) queryLessNode(ctx context.Context, less *parse.LessNode) <-chan Post {
	return Not(ctx, b.queryForNode(ctx, less.Less), b.queryEveryPost(ctx), ComparePostDescending)
}

func (b *Booru) queryWordNode(ctx context.Context, word *parse.WordNode) <-chan Post {
	tag := string(word.Word)

	// load baseline subquery
	if strings.HasPrefix(tag, "baseline:") {
		baseline := strings.Replace(tag, "baseline:", "", -1)
		query, err := ioutil.ReadFile(filepath.Join(b.baseline, baseline))
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}

		result, err := b.query(ctx, string(query))
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}
		return result
	}

	// load regex subquery
	if strings.HasPrefix(tag, "regex:") {
		if err := b.GenerateTagIndex(ctx); err != nil {
			log.Printf("%v", err)
			return nothing
		}

		regex := strings.Replace(tag, "regex:", "", -1)
		query, err := b.generateTagQuery(ctx, regex)
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}

		result, err := b.query(ctx, string(query))
		if err != nil {
			log.Printf("%v", err)
			return nothing
		}
		return result
	}

	return b.indexStream(ctx, string(word.Word))
}

func (b *Booru) queryEveryPost(ctx context.Context) <-chan Post {
	return b.indexStream(ctx, globalIndexTag)
}
