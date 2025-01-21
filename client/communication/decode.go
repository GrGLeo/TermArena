package communication

import (
	"errors"
	"strconv"
	"strings"
)

/*
DecodeRLE decodes a Run-Length Encoded (RLE) byte slice into a 20x50 integer grid.

The input RLE format is expected to be a slice of byte where each segment consists of a value and a count, separated by a colon (":"),

and multiple segments are separated by a pipe ("|").

For example, "1:10|0:40" represents 10 occurrences of the value 1 followed by 40 occurrences of the value 0.

This function returns the decoded 20x50 grid or an error if the decoding process fails.
*/
func DecodeRLE(rle []byte) ([20][50]int, error) {
  parts := strings.Split(string(rle), "|") 
  var decoded []int

  for _, part := range parts {
    subParts := strings.SplitN(string(part), ":", 2)
    if len(subParts) != 2 {
      return [20][50]int{}, errors.New("Failed to decode RLE")
    }
     value, err := strconv.Atoi(subParts[0])
    if err != nil {
      return [20][50]int{}, err
    }
    count, err := strconv.Atoi(subParts[1])
    if err != nil {
      return [20][50]int{}, err
    }

    for i := 0; i < count; i++ {
      decoded = append(decoded, value)
    }
  }
  var grid [20][50]int
  for i := 0; i < 20; i++ {
    copy(grid[i][:], decoded[i*50:(i+1)*50])
  }

  return grid, nil
}
