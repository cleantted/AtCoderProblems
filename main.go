package main

import (
	"database/sql"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"regexp"
	"strconv"
	"strings"
)

type Submit struct {
	Id           int
	ProblemId    string
	User         string
	Language     string
	SourceLength int
	Status       string
	ExecTime     int
	CreatedAt    string
}

func (s *Submit) IdStr() string {
	return strconv.Itoa(s.Id)
}

func GetContestUrls() []string {
	x := []string{}
	doc, _ := goquery.NewDocument("http://atcoder.jp/")
	rep := regexp.MustCompile(`^https*://([a-z0-9\-]*)\.contest\.atcoder.*$`)
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, _ := s.Attr("href")
		if rep.Match([]byte(url)) {
			url = rep.ReplaceAllString(url, "$1")
			x = append(x, url)
		}
	})
	return x
}

func GetProblemSet(contest string) []string {
	set := make(map[string]bool)
	url := "http://" + contest + ".contest.atcoder.jp/assignments"
	doc, _ := goquery.NewDocument(url)
	rep := regexp.MustCompile(`^/tasks/([0-9_a-z]*)$`)
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, _ := s.Attr("href")
		if rep.Match([]byte(url)) {
			url = rep.ReplaceAllString(url, "$1")
			set[url] = true
		}
	})

	x := []string{}
	for key, _ := range set {
		x = append(x, key)
	}
	return x
}

func GetSubmissions(contest string, i int) ([]Submit, int) {
	x := []Submit{}
	max := 1

	url := "http://" + contest + ".contest.atcoder.jp/submissions/all/" + strconv.Itoa(i)
	doc, _ := goquery.NewDocument(url)

	allrep := regexp.MustCompile(`^/submissions/all/([0-9]*)$`)
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, _ := s.Attr("href")
		if allrep.Match([]byte(url)) {
			url = allrep.ReplaceAllString(url, "$1")
			page, _ := strconv.Atoi(url)
			if page > max {
				max = page
			}
		}
	})

	rep := regexp.MustCompile(`^/submissions/([0-9]*)$`)
	prep := regexp.MustCompile(`^/tasks/([0-9_a-z]*)$`)
	urep := regexp.MustCompile(`^/users/([0-9_a-zA-Z]*)$`)

	doc.Find("tbody").Each(func(_ int, s *goquery.Selection) {
		s.Find("tr").Each(func(_ int, s *goquery.Selection) {
			var key int
			var problem_id string
			var user_name string
			s.Find("a").Each(func(_ int, t *goquery.Selection) {
				url, _ := t.Attr("href")
				if rep.Match([]byte(url)) {
					url = rep.ReplaceAllString(url, "$1")
					key, _ = strconv.Atoi(url)
				} else if prep.Match([]byte(url)) {
					problem_id = prep.ReplaceAllString(url, "$1")
				} else if urep.Match([]byte(url)) {
					user_name = urep.ReplaceAllString(url, "$1")
				}
			})

			data := []string{}
			s.Find("td").Each(func(_ int, s *goquery.Selection) {
				data = append(data, s.Text())
			})
			length, _ := strconv.Atoi(strings.Replace(data[5], " Byte", "", -1))
			t := Submit{
				Id:           key,
				ProblemId:    problem_id,
				User:         user_name,
				Language:     data[3],
				SourceLength: length,
				Status:       data[6],
				CreatedAt:    data[0],
				ExecTime:     0,
			}
			if len(data) == 10 {
				exec_time, _ := strconv.Atoi(strings.Replace(data[7], " ms", "", -1))
				t.ExecTime = exec_time
			}
			x = append(x, t)
		})
	})

	return x, max
}

func GetMySQL(user string, pass string) (db *sql.DB) {
	server := user + ":" + pass + "@/atcoder"
	db, err := sql.Open("mysql", server)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func NewRecord(table, column, key string, db *sql.DB) bool {
	query, args, _ := sq.Select(column).From(table).Where(sq.Eq{column: key}).ToSql()
	row, _ := db.Query(query, args...)
	defer row.Close()
	for row.Next() {
		return false
	}
	return true
}

func main() {
	var u string
	var p string
	fmt.Scan(&u)
	fmt.Scan(&p)
	db := GetMySQL(u, p)
	defer db.Close()

	urls := GetContestUrls()
	for _, contest := range urls {
		if NewRecord("contests", "id", contest, db) {
			problems := GetProblemSet(contest)
			if len(problems) == 0 {
				continue
			}
			query, args, _ := sq.Insert("contests").Columns("id").Values(contest).ToSql()
			db.Exec(query, args...)
			q := sq.Insert("problems").Columns("id", "contest")
			for _, problem := range problems {
				if NewRecord("problems", "id", problem, db) {
					q = q.Values(problem, contest)
				}
			}
			query, args, _ = q.ToSql()
			db.Exec(query, args...)
		}
	}

	contest := "abc031"
	m := 1
	for i := 1; i <= m; i++ {
		submissions, max := GetSubmissions(contest, i)
		if max > m {
			m = max
		}
		q := sq.Insert("submissions")
		q = q.Columns(
			"id", "problem_id", "contest_id",
			"user_name", "status", "source_length", "language",
			"exec_time", "created_time")
		for _, s := range submissions {
			if NewRecord("submissions", "id", s.IdStr(), db) {
				q = q.Values(
					s.Id, s.ProblemId, contest,
					s.User, s.Status, s.SourceLength, s.Language,
					s.ExecTime, s.CreatedAt)
			}
		}
		query, args, _ := q.ToSql()
		db.Exec(query, args...)
		fmt.Println(i)
	}
}
