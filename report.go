package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type CoverProfile struct {
	Mode     string
	Packages map[string]*Package

	Total   int
	Covered int
}

func (c *CoverProfile) Coverage() float64 {
	if c == nil {
		return math.NaN()
	}

	return float64(c.Covered) / float64(c.Total)
}

type Package struct {
	Name   string
	Blocks []Block

	Total   int
	Covered int
}

func (p *Package) Coverage() float64 {
	if p == nil {
		return math.NaN()
	}

	return float64(p.Covered) / float64(p.Total)
}

type Block struct {
	Filename       string
	Start          Position
	End            Position
	StatementCount int
	HitCount       int
}

type Position struct {
	Line   int
	Column int
}

func LoadCoverProfile(filename string) (*CoverProfile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("missing header")
	}
	header := scanner.Text()
	if !strings.HasPrefix(header, "mode: ") {
		return nil, fmt.Errorf("profile must start with [mode: ] header")
	}

	profile := &CoverProfile{
		Mode:     strings.TrimPrefix(header, "mode: "),
		Packages: map[string]*Package{},
	}

	line := 0
	for scanner.Scan() {
		line++
		match := lineRe.FindStringSubmatch(scanner.Text())
		if match == nil {
			return nil, fmt.Errorf("malformed coverage line: %s", scanner.Text())
		}

		// note: format of each coverage line https://github.com/golang/tools/blob/0cf4e2708ac840da8674eb3947b660a931bd2c1f/cover/profile.go#L51-L54
		path := match[1]
		pkgName := filepath.Dir(path)
		fileName := filepath.Base(path)
		startLine, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, fmt.Errorf("invalid startLine on line %d: %w", line, err)
		}
		startCol, err := strconv.Atoi(match[3])
		if err != nil {
			return nil, fmt.Errorf("invalid startCol on line %d: %w", line, err)
		}
		endLine, err := strconv.Atoi(match[4])
		if err != nil {
			return nil, fmt.Errorf("invalid endLine on line %d: %w", line, err)
		}
		endCol, err := strconv.Atoi(match[5])
		if err != nil {
			return nil, fmt.Errorf("invalid endCol on line %d: %w", line, err)
		}
		statementCount, err := strconv.Atoi(match[6])
		if err != nil {
			return nil, fmt.Errorf("invalid statementCount on line %d: %w", line, err)
		}
		hitCount, err := strconv.Atoi(match[7])
		if err != nil {
			return nil, fmt.Errorf("invalid hitCount on line %d: %w", line, err)
		}

		pkgData := profile.Packages[pkgName]
		if pkgData == nil {
			// package not yet seen - create new struct
			pkgData = &Package{
				Name: pkgName,
			}
			profile.Packages[pkgName] = pkgData
		}

		// increment statement and coverage (hit) counts at both a package and overall profile level
		pkgData.Total += statementCount
		profile.Total += statementCount
		if hitCount > 0 {
			pkgData.Covered += statementCount
			profile.Covered += statementCount
		}

		pkgData.Blocks = append(pkgData.Blocks, Block{
			Filename: fileName,
			Start: Position{
				Line:   startLine,
				Column: startCol,
			},
			End: Position{
				Line:   endLine,
				Column: endCol,
			},
			StatementCount: statementCount,
			HitCount:       hitCount,
		})
	}

	return profile, scanner.Err()
}

var lineRe = regexp.MustCompile(`^([^:]*):(\d*)\.(\d*),(\d*)\.(\d*) (\d*) (\d*)$`)
