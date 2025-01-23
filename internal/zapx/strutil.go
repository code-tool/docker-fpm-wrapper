package zapx

// LongestCommonPrefixOffset finds the longest common prefix string amongst a slice of strings.
func longestCommonPrefixOffset(strs []string) int {
	if len(strs) == 0 {
		return 0
	}

	// Take the first string as the reference
	result := len(strs[0])

	for i := 1; i < len(strs); i++ {
		// Reduce the prefix until it matches the current string
		for j := 0; j < result; j++ {
			if j >= len(strs[i]) || strs[i][j] != strs[0][j] {
				result = j
				break
			}
		}
		if result == 0 {
			break
		}
	}

	return result
}
