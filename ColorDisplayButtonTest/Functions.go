package main

import "strconv"

// Helper function
func inList(index int, exclusiveList []string) bool {
	if len(exclusiveList) == 1 && exclusiveList[0] == "" {
		return true
	}
	testInt := strconv.Itoa(index)
	for _, el := range exclusiveList {
		if el == testInt {
			return true
		}
	}
	return false
}
