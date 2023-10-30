package common

// breaks up a slice into chunks of size chunkSize or as close to it as possible
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	var chunks [][]T

	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
