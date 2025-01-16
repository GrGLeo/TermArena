package communication

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

func DecodeRLE(rle []byte) ([20][50]int, error) {
  parts := bytes.Split(rle, []byte{0x00}) 
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
