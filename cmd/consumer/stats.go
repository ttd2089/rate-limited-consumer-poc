package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ttd2089/rate-limited-consumer-poc/internal/metrics"
	"golang.org/x/exp/maps"
)

func newStatsServer(
	addr string,
	stats *metrics.Count,
	wwwDir string,
) (*http.Server, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		data := stats.Data()
		accept := strings.Split(r.Header.Get("Accept"), ",")
		if slices.Contains(accept, "application/json") {
			serveJSON(data, w)
			return
		}
		serveHTML(wwwDir, data, w)
	})

	staticDir := filepath.Join(wwwDir, "static")
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Printf("error: listen and serve: %w", err)
		}
	}()

	return srv, nil
}

func serveJSON(data map[string]metrics.TimeBuckets, w http.ResponseWriter) {
	body, err := json.MarshalIndent(data, "", "   ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func serveHTML(wwwDir string, data map[string]metrics.TimeBuckets, w http.ResponseWriter) {
	params := make(map[string]struct {
		Points string
	}, len(data))

	orderedElements := maps.Keys(data)
	slices.Sort(orderedElements)
	for _, element := range orderedElements {
		buckets := data[element]

		orderedBucketTimes := maps.Keys(buckets)
		slices.SortFunc(orderedBucketTimes, func(a time.Time, b time.Time) int {
			delta := a.Sub(b).Nanoseconds()
			if delta < 0 {
				return -1
			}
			if delta > 0 {
				return 1
			}
			return 0
		})

		max := slices.Max(maps.Values(buckets))

		// HACK: I don't know how to make each panel the same height when they have different max
		// values without affecting the scaling of other components in the panel (like the legend)
		// so normalize the Y coordinates instead.
		verticalScalingFactor := float64(100) / float64(max)

		sb := strings.Builder{}
		for i, k := range orderedBucketTimes {
			count := buckets[k]

			// HACK: This right aligns the polyline by figuring out how much empty space there is
			// (the difference between the figure width and the number of data points) and shifting
			// the existing data points to the right that distance.
			x := (300 - len(buckets)) + i

			// HACK: SVG Y coordinates put y=0 at the top of the figure. This inverts our values to
			// compensate.
			y := max - count

			// HACK: Scale Y coordinate.
			y = int(float64(y) * verticalScalingFactor)

			if sb.Len() > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(strconv.Itoa(x))
			sb.WriteString(",")
			sb.WriteString(strconv.Itoa(y))
		}
		params[element] = struct {
			Points string
		}{Points: sb.String()}
	}

	t := template.New("t")
	t, err := t.ParseFiles(filepath.Join(wwwDir, "templates", "page.html"))
	if err != nil {
		fmt.Printf("error: parse HTML template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := t.ExecuteTemplate(w, "page.html", params); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
