package inject

// panicSafe can be replaced with new error handling; the default behavior is to panic. If panicSafe does not exit then you will get unexpected behavior
var HandleError func(err error)

func panicSafe(err error) {
	if HandleError != nil {
		HandleError(err)
	}

	panic(err.Error())
}
