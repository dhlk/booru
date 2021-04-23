package booru

import "context"

func drainTo(ctx context.Context, source <-chan Post, sink chan<- Post) {
	for post := range source {
		select {
		case <-ctx.Done():
			return
		case sink <- post:
		}
	}
}

func Skip(ctx context.Context, c <-chan Post, count int64) <-chan Post {
	result := make(chan Post)
	go func() {
		defer close(result)

		read := int64(0)
		for post := range c {
			read++
			if read > count {
				select {
				case <-ctx.Done():
					return
				case result <- post:
				}
				break
			}
		}

		drainTo(ctx, c, result)
	}()
	return result
}

func Limit(ctx context.Context, c <-chan Post, count int64) <-chan Post {
	result := make(chan Post)

	go func() {
		defer close(result)

		wrote := int64(0)
		for post := range c {
			wrote++
			if wrote > count {
				break
			}
			select {
			case <-ctx.Done():
				return
			case result <- post:
			}
		}
	}()

	return result
}

// TODO and needs a subcontext to cancel feeds into it
func and(ctx context.Context, l, r <-chan Post, compare func(Post, Post) int) <-chan Post {
	result := make(chan Post)
	go func() {
		defer close(result)

		lr, lok := <-l
		rr, rok := <-r
		for lok && rok {
			order := compare(lr, rr)
			if order == 0 {
				select {
				case <-ctx.Done():
					return
				case result <- lr:
					lr, lok = <-l
					rr, rok = <-r
				}
			} else if order == -1 {
				lr, lok = <-l
			} else if order == 1 {
				rr, rok = <-r
			}
		}
	}()
	return result
}

func And(ctx context.Context, q ...<-chan Post) <-chan Post {
	if len(q) == 0 {
		panic(q)
	}

	if len(q) == 1 {
		return q[0]
	}

	result := and(ctx, q[0], q[1], ComparePostDescending)
	for _, qr := range q[2:] {
		result = and(ctx, result, qr, ComparePostDescending)
	}
	return result
}

func or(ctx context.Context, l, r <-chan Post, compare func(Post, Post) int) <-chan Post {
	result := make(chan Post)
	go func() {
		defer close(result)

		lr, lok := <-l
		rr, rok := <-r
		for lok && rok {
			order := compare(lr, rr)
			if order == 0 {
				select {
				case <-ctx.Done():
					return
				case result <- lr:
					lr, lok = <-l
					rr, rok = <-r
				}
			} else if order == -1 {
				select {
				case <-ctx.Done():
					return
				case result <- lr:
					lr, lok = <-l
				}
			} else if order == 1 {
				select {
				case <-ctx.Done():
					return
				case result <- rr:
					rr, rok = <-r
				}
			}
		}

		if lok {
			select {
			case <-ctx.Done():
				return
			case result <- lr:
				drainTo(ctx, l, result)
			}
		}
		if rok {
			select {
			case <-ctx.Done():
				return
			case result <- rr:
				drainTo(ctx, r, result)
			}
		}
	}()
	return result
}

func Or(ctx context.Context, q ...<-chan Post) <-chan Post {
	if len(q) == 0 {
		panic(q)
	}

	if len(q) == 1 {
		return q[0]
	}

	result := or(ctx, q[0], q[1], ComparePostDescending)
	for _, qr := range q[2:] {
		result = or(ctx, result, qr, ComparePostDescending)
	}
	return result
}

func Not(ctx context.Context, query, everything <-chan Post, compare func(Post, Post) int) <-chan Post {
	result := make(chan Post)
	go func() {
		defer close(result)

		qr, qok := <-query
		er, eok := <-everything
		for qok {
			order := compare(qr, er)
			if order == 0 {
				qr, qok = <-query
				er, eok = <-everything
			} else if order == -1 {
				qr, qok = <-query
			} else if order == 1 {
				select {
				case <-ctx.Done():
					return
				case result <- er:
					er, eok = <-everything
				}
			}
		}

		if eok {
			select {
			case <-ctx.Done():
				return
			case result <- er:
				drainTo(ctx, everything, result)
			}
		}
	}()
	return result
}
