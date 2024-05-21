package ringbuf

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {

	t.Run("Len", func(t *testing.T) {
		testCases := []struct {
			name        string
			capacity    int
			adds        int
			expectedLen int
		}{
			{
				name:        "is zero when nothing is added",
				capacity:    3,
				adds:        0,
				expectedLen: 0,
			},
			{
				name:        "increases when item is added",
				capacity:    3,
				adds:        1,
				expectedLen: 1,
			},
			{
				name:        "increases to capacity",
				capacity:    3,
				adds:        3,
				expectedLen: 3,
			},
			{
				name:        "does not increase beyond capacity",
				capacity:    3,
				adds:        4,
				expectedLen: 3,
			},
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				b := New[time.Time](tt.capacity)
				for i := 0; i < tt.adds; i++ {
					b.Push(time.Now())
				}
				assert.Equal(t, tt.expectedLen, b.Len())
			})
		}
	})

	t.Run("Get", func(t *testing.T) {

		t.Run("writes have not wrapped", func(t *testing.T) {
			first := time.Now()
			expected := []time.Time{
				first,
				first.Add(time.Second),
				first.Add(2 * time.Second),
			}

			b := New[time.Time](len(expected))
			for _, expected := range expected {
				b.Push(expected)
			}

			for i, expected := range expected {
				t.Run(fmt.Sprintf("index in range/b[%d]", i), func(t *testing.T) {
					actual, err := b.Get(i)
					assert.NoError(t, err)
					assert.Equal(t, expected, actual)
				})
			}
		})

		t.Run("writes have wrapped", func(t *testing.T) {
			first := time.Now()
			expected := []time.Time{
				first,
				first.Add(time.Second),
				first.Add(2 * time.Second),
			}
			unexpected := []time.Time{
				first.Add(3 * time.Second),
				first.Add(4 * time.Second),
				first.Add(5 * time.Second),
				first.Add(6 * time.Second),
				first.Add(7 * time.Second),
				first.Add(8 * time.Second),
				first.Add(9 * time.Second),
			}

			b := New[time.Time](len(expected))
			for _, unexpected := range unexpected {
				b.Push(unexpected)
			}
			for _, expected := range expected {
				b.Push(expected)
			}

			for i, expected := range expected {
				t.Run(fmt.Sprintf("index in range/b[%d]", i), func(t *testing.T) {
					actual, err := b.Get(i)
					assert.NoError(t, err)
					assert.Equal(t, expected, actual)
				})
			}
		})
	})
}
