package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	configFile   = flag.String("c", "conf.json", "config file location")
	parsedconfig = conf{}
	Random       *os.File
	lineRe       = regexp.MustCompile(`^(.+):([0-9]+).([0-9]+),([0-9]+).([0-9]+) ([0-9]+) ([0-9]+)$`)
)

type conf struct {
	ListenPort string `json:"listenPort"`
	ShieldURL  string `json:"shieldServerURL"`
}

type CoverageResult struct {
	Percent float64
}

type Profile struct {
	FileName string
	Mode     string
	Blocks   []ProfileBlock
}

type byFileName []*Profile

func (p byFileName) Len() int           { return len(p) }
func (p byFileName) Less(i, j int) bool { return p[i].FileName < p[j].FileName }
func (p byFileName) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// ProfileBlock represents a single block of profiling data.
type ProfileBlock struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	NumStmt, Count      int
}

type blocksByStart []ProfileBlock

func (b blocksByStart) Len() int      { return len(b) }
func (b blocksByStart) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b blocksByStart) Less(i, j int) bool {
	bi, bj := b[i], b[j]
	return bi.StartLine < bj.StartLine || bi.StartLine == bj.StartLine && bi.StartCol < bj.StartCol
}

func init() {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		log.Fatal(err)
	}
	Random = f
}

func percentCovered(p *Profile) float64 {
	var total, covered int64
	for _, b := range p.Blocks {
		total += int64(b.NumStmt)
		if b.Count > 0 {
			covered += int64(b.NumStmt)
		}
	}
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}

// ParseProfiles parses profile data in the specified file and returns a
// Profile for each source file described therein.
func ParseProfiles(fileName string) ([]*Profile, error) {
	pf, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer pf.Close()

	files := make(map[string]*Profile)
	buf := bufio.NewReader(pf)
	// First line is "mode: foo", where foo is "set", "count", or "atomic".
	// Rest of file is in the format
	//	encoding/base64/base64.go:34.44,37.40 3 1
	// where the fields are: name.go:line.column,line.column numberOfStatements count
	s := bufio.NewScanner(buf)
	mode := ""
	for s.Scan() {
		line := s.Text()
		if mode == "" {
			const p = "mode: "
			if !strings.HasPrefix(line, p) || line == p {
				return nil, fmt.Errorf("bad mode line: %v", line)
			}
			mode = line[len(p):]
			continue
		}
		m := lineRe.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("line %q doesn't match expected format: %v", m, lineRe)
		}
		fn := m[1]
		p := files[fn]
		if p == nil {
			p = &Profile{
				FileName: fn,
				Mode:     mode,
			}
			files[fn] = p
		}
		p.Blocks = append(p.Blocks, ProfileBlock{
			StartLine: toInt(m[2]),
			StartCol:  toInt(m[3]),
			EndLine:   toInt(m[4]),
			EndCol:    toInt(m[5]),
			NumStmt:   toInt(m[6]),
			Count:     toInt(m[7]),
		})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	for _, p := range files {
		sort.Sort(blocksByStart(p.Blocks))
	}
	// Generate a sorted slice.
	profiles := make([]*Profile, 0, len(files))
	for _, profile := range files {
		profiles = append(profiles, profile)
	}
	sort.Sort(byFileName(profiles))
	return profiles, nil
}

func toInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func httpErrorf(w http.ResponseWriter, format string, a ...interface{}) {
	err := fmt.Errorf(format, a...)
	log.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func statusColor(coveragePct string) string {
	pctInt := toInt(coveragePct)

	if pctInt <= 30 {
		return "red"
	}

	if pctInt > 30 && pctInt <= 75 {
		return "yellow"
	}

	if pctInt > 75 {
		return "green"
	}

	return "blue"

}

func uploadHandler(config conf) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not supported", http.StatusMethodNotAllowed)
			return
		}
		repoName := r.URL.Query().Get("repo")

		if _, err := os.Stat(repoName); err != nil {
			if os.IsNotExist(err) {
				log.Printf("Directory for %s does not exist, creating", repoName)
				if err := os.MkdirAll(repoName, 0755); err != nil {
					log.Printf("error creating directory for %s:  %s", repoName, err)
					return
				}
			} else {
				log.Println("error inspecting: ", err)
			}
		}

		reader, err := r.MultipartReader()
		if err != nil {
			httpErrorf(w, "error creating multipart reader: %s", err)
			return
		}
		var dst *os.File
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if part.FileName() == "" {
				continue
			}

			dst, err = os.Create(filepath.Join(repoName, "coverage.out"))
			if err != nil {
				httpErrorf(w, "error creating coverage file: %s", err)
				return
			}
			defer dst.Close()
			if _, err := io.Copy(dst, part); err != nil {
				httpErrorf(w, "error writing coverage file: %s", err)
				return
			}
		}

		profs, err := ParseProfiles(dst.Name())
		if err != nil {
			log.Println("Error parsing profile file: ", err)
		}

		percentCovered := percentCovered(profs[0])

		// redirect to shields API server
		roundedFloat := fmt.Sprintf("%.0f", percentCovered)
		log.Println("Coverage percent: ", roundedFloat)
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.Redirect(w, r, config.ShieldURL+"/coverage-"+roundedFloat+"%25-"+statusColor(roundedFloat)+".svg", http.StatusSeeOther)
	})
}

func displayHandler(config conf) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not supported", http.StatusMethodNotAllowed)
			return
		}
		repoName := r.URL.Query().Get("repo")
		if repoName == "" {
			http.Redirect(w, r, config.ShieldURL+"/coverage-NaN-red.svg", http.StatusSeeOther)
			return
		}
		profs, err := ParseProfiles(filepath.Join(repoName, "coverage.out"))
		if err != nil {
			httpErrorf(w, "Error parsing profile file: %s", err)
			return
		}

		percentCovered := percentCovered(profs[0])

		// get image from shields API server
		roundedFloat := fmt.Sprintf("%.0f", percentCovered)
		log.Println("Coverage percent: ", roundedFloat)
		log.Println("URL: ", config.ShieldURL+"/coverage-"+roundedFloat+"%25-"+statusColor(roundedFloat)+".svg")
		reqImg, err := http.Get(config.ShieldURL + "/coverage-" + roundedFloat + "%25-" + statusColor(roundedFloat) + ".svg")
		if err != nil {
			httpErrorf(w, "Error loading SVG from shield: %s", err)
			return
		}
		w.Header().Set("Content-Type", reqImg.Header.Get("Content-Type"))
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		if _, err = io.Copy(w, reqImg.Body); err != nil {
			httpErrorf(w, "Error writing SVG to ResponseWriter: %s", err)
			return
		}
		reqImg.Body.Close()
	})
}

func main() {
	flag.Parse()
	file, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal("unable to read config file, exiting...")
	}
	if err := json.Unmarshal(file, &parsedconfig); err != nil {
		log.Fatal("unable to marshal config file, exiting...")
	}

	http.Handle("/upload", uploadHandler(parsedconfig))
	http.Handle("/display", displayHandler(parsedconfig))

	log.Println("running without SSL enabled")
	log.Fatal(http.ListenAndServe(":"+parsedconfig.ListenPort, nil))

}
