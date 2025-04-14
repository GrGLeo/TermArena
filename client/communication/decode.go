package communication

import (
	"errors"
	"strconv"
	"strings"
)

// DecodeRLE decodes a Run-Length Encoded (RLE) byte slice into a 20x50 grid of integers.
//
// The RLE format consists of pairs of "value:count" separated by "|". Each pair represents
// a value repeated 'count' times in the grid.
//
// Parameters:
//   - rle: A byte slice containing the RLE-encoded data.
//
// Returns:
//   - A 20x50 grid of integers representing the decoded data.
//   - An error if the RLE data is malformed or cannot be decoded.
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

    for range count {
      decoded = append(decoded, value)
    }
  }
  var grid [20][50]int
  for i := range 20 {
    copy(grid[i][:], decoded[i*50:(i+1)*50])
  }

  return grid, nil
}


// DecodeDeltas converts a slice of 3-byte arrays into a slice of 3-integer arrays.
// Each 3-byte array represents a delta with the following format:
//   - delta[0]: X-coordinate (byte)
//   - delta[1]: Y-coordinate (byte)
//   - delta[2]: Value (byte)
//
// Parameters:
//   - deltas: A slice of 3-byte arrays, where each array represents a delta.
//
// Returns:
//   - A slice of 3-integer arrays, where each array contains the decoded X, Y, and Value.
func DecodeDeltas(deltas [][3]byte) [][3]int {
  var decodedDeltas [][3]int 
  for _, delta := range deltas {
    decodedDelta := [3]int{int(delta[0]), int(delta[1]), int(delta[2])}
    decodedDeltas = append(decodedDeltas, decodedDelta)
  }
  return decodedDeltas
}
