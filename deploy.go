package main

/////////////
// Imports //
/////////////
import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"net/http"

	"github.com/r4wm/kjvapi"
)

/////////////
// Structs //
/////////////
//Book Name of Book and how many chapters contained in that book.
type Book struct {
	Name     string
	Chapters int
}

// KJVMapping static mapping containing books and number of chapters per book.
type KJVMapping struct {
	Books []Book
}

type response struct {
	Text string `json:"text"`
}

//////////
// Vars //
//////////
var DB *sql.DB
var Mapping KJVMapping

///////////////
// Functions //
///////////////
// helloWorld basic handler function
func helloWorld(w http.ResponseWriter, r *http.Request) {
	basicResponse := &response{Text: "Hello World " + r.URL.Path[:]}
	jsonResponse, _ := json.Marshal(basicResponse)

	w.Header().Set("Content-Type", "application/json")

	fmt.Fprintf(w, string(jsonResponse))
}

// GetBooks retrieve list of books from the kjv db
func GetBooks(w http.ResponseWriter, r *http.Request) {
	jsonResponse, err := json.Marshal(Mapping)

	if err != nil {
		log.Fatal("Could not marshal books")
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(jsonResponse))
}

// GetChapter print the book, chapter and verses in json format
func GetChapter(w http.ResponseWriter, r *http.Request) {
	log.Printf("%#v\n", r)

	var verses []kjvapi.KJVVerse

	book, ok := r.URL.Query()["book"]
	if !ok || len(book[0]) < 1 {
		log.Println("Url param book is missing.")
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("406 - book param not found."))
		return
	}

	book[0] = strings.ToUpper(book[0])

	chapter, ok := r.URL.Query()["chapter"]
	if !ok {
		log.Println("Url param chapter is missing.")
		w.WriteHeader(http.StatusNotAcceptable)
		w.Write([]byte("406 - chapter param not found."))
		return
	}

	stmt := fmt.Sprintf("select verse, text from kjv where book='%s' and chapter=%v", book[0], chapter[0])

	rows, err := DB.Query(stmt)
	defer rows.Close()

	if err != nil {
		log.Println(err)
		log.Printf("database: %#v\n", DB)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("400 - Could not query such a request: "))
		return
	}

	var verse int
	var text string

	for rows.Next() {
		rows.Scan(&verse, &text)
		verses = append(verses, kjvapi.KJVVerse{Verse: verse, Text: text})
	}
	i, err := strconv.Atoi(chapter[0])
	if err != nil {
		log.Printf("Could not convert %s to int.", chapter[0])
	}

	bkResult := kjvapi.KJVBook{
		Book: book[0],
		Chapters: []kjvapi.KJVChapter{
			kjvapi.KJVChapter{Chapter: i, Verses: verses}}}

	response, _ := json.Marshal(bkResult)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, string(response))
}

func main() {
	fmt.Printf("Mapping: %#v\n", Mapping)

	/////////////////
	// Args	       //
	/////////////////
	createDB := flag.Bool("createDB", false, "create database")
	dbPath := flag.String("dbPath", "", "path to datebase")
	flag.Parse()

	if len(*dbPath) == 0 {
		log.Fatalf("Must provide dbPath")
	}

	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		if *createDB == false {
			log.Fatalf("database file does not exist: %s\n", *dbPath)
		} else {
			kjvapi.CreateKJVDB(*dbPath)
		}
	}

	fmt.Println("dbPath: ", *dbPath)
	fmt.Println("createDB: ", *createDB)
	////////////////////////////////
	// Database Connection	      //
	////////////////////////////////
	DB, _ = sql.Open("sqlite3", *dbPath)
	fmt.Println(fmt.Sprintf("%T\n", DB))
	log.Printf("Running server using database at: %s\n", *dbPath)

	/////////////////////////////
	// Populate Mapping	   //
	/////////////////////////////
	// Cant do this part in an init() cause it will run before main and we havent spec'd the db from args
	// TODO: Maybe make db location fixed..
	// populate the Book struct
	rows, _ := DB.Query("select distinct book from kjv")
	defer rows.Close()

	for rows.Next() {
		var bookName string
		rows.Scan(&bookName)
		book := Book{Name: bookName}

		chaptersQuery := fmt.Sprintf("select max(chapter) from kjv where book=\"%s\"", bookName)
		fmt.Println(chaptersQuery)
		rowsForChapterCount, err := DB.Query(chaptersQuery)
		defer rowsForChapterCount.Close()

		if err != nil {
			log.Fatalf("Failed query on %s\n", chaptersQuery)
		}

		for rowsForChapterCount.Next() {
			err := rowsForChapterCount.Scan(&book.Chapters)
			if err != nil {
				log.Fatalf("Could not get %s from db.\n", bookName)
			}

			Mapping.Books = append(Mapping.Books, book)
		}

	}
	fmt.Printf("Mapping: %#v\n", Mapping)

	/////////////////////
	// Handlers	   //
	/////////////////////
	http.HandleFunc("/", helloWorld)
	http.HandleFunc("/get_books", GetBooks)
	http.HandleFunc("/get_chapter", GetChapter)
	http.ListenAndServe(":8000", nil)
}
