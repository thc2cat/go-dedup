package main

// confirm displays a prompt `s` to the user and returns a bool indicating yes / no
import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// If the lowercased, trimmed input begins with anything other than 'y', it returns false
// It accepts an int `tries` representing the number of attempts before returning false
func confirm(s string, tries int) byte {
	r := bufio.NewReader(os.Stdin)

	for ; tries > 0; tries-- {
		fmt.Printf("%s", s)

		res, err := r.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		return byte(strings.ToLower(strings.TrimSpace(res))[0])
	}

	return byte('s')
}
