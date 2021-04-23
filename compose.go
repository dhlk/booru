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

type CancelableStream func(context.Context) <-chan Post

func Skip(in CancelableStream, count int64) CancelableStream {
	return func(ctx context.Context) <-chan Post {
		return skip(ctx, in, count)
	}
}

func skip(ctx context.Context, in CancelableStream, count int64) <-chan Post {
	result := make(chan Post)

	go func(out chan<- Post) {
		defer close(result)

		fwdCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		input := in(fwdCtx)

		read := int64(0)
		for post := range input {
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

		drainTo(ctx, input, result)
	}(result)

	return result
}

func Limit(in CancelableStream, count int64) CancelableStream {
	return func(ctx context.Context) <-chan Post {
		return limit(ctx, in, count)
	}
}

func limit(ctx context.Context, in CancelableStream, count int64) <-chan Post {
	result := make(chan Post)

	go func(out chan<- Post) {
		defer close(result)

		fwdCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		wrote := int64(0)
		for post := range in(fwdCtx) {
			wrote++
			if wrote > count {
				return
			}
			select {
			case <-ctx.Done():
				return
			case result <- post:
			}
		}
	}(result)

	return result
}

func Intersection(in []CancelableStream, compare PostCompare) CancelableStream {
	return func(ctx context.Context) <-chan Post {
		return intersection(ctx, in, compare)
	}
}

func intersection(ctx context.Context, in []CancelableStream, compare PostCompare) <-chan Post {
	if len(in) == 0 {
		return nothing(ctx)
	} else if len(in) == 1 {
		return in[0](ctx)
	}

	result := make(chan Post)

	go func(out chan<- Post) {
		defer close(result)

		fwdCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		inputs := make([]<-chan Post, len(in))
		post := make([]Post, len(in))
		ok := make([]bool, len(in))

		for i := range inputs {
			inputs[i] = in[i](fwdCtx)
		}

		for i := range inputs {
			post[i], ok[i] = <-inputs[i]
			if !ok[i] {
				return
			}
		}

		for {
			for i := range inputs {
				if i+1 == len(inputs) {
					select {
					case <-ctx.Done():
						return
					case out <- post[0]:
						post[0], ok[0] = <-inputs[0]
					}
					break
				}

				order := compare(post[i], post[i+1])
				if order == 0 {
					continue
				} else if order == -1 {
					post[i], ok[i] = <-inputs[i]
					if !ok[i] {
						return
					}
					break
				} else if order == 1 {
					post[i+1], ok[i+1] = <-inputs[i+1]
					if !ok[i+1] {
						return
					}
					break
				}
			}
		}
	}(result)

	return result
}

func Union(in []CancelableStream, compare PostCompare) CancelableStream {
	return func(ctx context.Context) <-chan Post {
		return union(ctx, in, compare)
	}
}

func union(ctx context.Context, in []CancelableStream, compare PostCompare) <-chan Post {
	if len(in) == 0 {
		return nothing(ctx)
	} else if len(in) == 1 {
		return in[0](ctx)
	}

	result := make(chan Post)

	go func(out chan<- Post) {
		defer close(result)

		fwdCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		inputs := make([]<-chan Post, len(in))
		post := make([]Post, len(in))
		ok := make([]bool, len(in))

		for i := range inputs {
			inputs[i] = in[i](fwdCtx)
		}

		for i := range inputs {
			post[i], ok[i] = <-inputs[i]
		}

		for {
			min := -1

			for i, iok := range ok {
				if iok {
					min = i
					break
				}
			}

			if min == -1 {
				return
			}

			for i := range inputs {
				if !ok[i] || i == min {
					continue
				}

				order := compare(post[min], post[i])
				if order == 0 {
					post[i], ok[i] = <-inputs[i]
				} else if order == -1 {
					continue
				} else if order == 1 {
					min = i
				}
			}

			select {
			case <-ctx.Done():
				return
			case out <- post[min]:
				post[min], ok[min] = <-inputs[min]
			}
		}
	}(result)

	return result
}

// assumes in is a strict subset of space!
func Complement(in, space CancelableStream, compare PostCompare) CancelableStream {
	return func(ctx context.Context) <-chan Post {
		return complement(ctx, in, space, compare)
	}
}

func complement(ctx context.Context, in, postSpace CancelableStream, compare PostCompare) <-chan Post {
	result := make(chan Post)

	go func(out chan<- Post) {
		defer close(result)

		fwdCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		input := in(fwdCtx)
		space := postSpace(fwdCtx)

		iv, iok := <-input
		sv, sok := <-space

		for iok {
			order := compare(iv, sv)
			if order == 0 {
				iv, iok = <-input
				sv, sok = <-space
			} else if order == -1 {
				iv, iok = <-input
			} else if order == 1 {
				select {
				case <-ctx.Done():
					return
				case out <- sv:
					sv, sok = <-space
				}
			}
		}

		if sok {
			select {
			case <-ctx.Done():
				return
			case out <- sv:
				drainTo(ctx, space, out)
			}
		}
	}(result)

	return result
}
