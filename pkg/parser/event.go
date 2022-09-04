// Package parser implements structs and functions to parse events from pdf content.

package parser

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// RawEvent contains data, position in pdf file, and initial date to normalize event dates.
// It is retrieved from input pdf.
type RawEvent struct {
	data        string
	position    pdf.Point
	initialDate time.Time
}

// Event is retrieved from RawEvent. It is contained in output json.
type Event struct {
	Title    string      `json:"title"`
	Teacher  string      `json:"teacher"`
	Type     string      `json:"type"`
	Subgroup string      `json:"subgroup"`
	Location string      `json:"location"`
	Dates    []EventDate `json:"dates"`
}

// getRawEvents takes slice of pdf.Text, forms slice of RawEvent and returns it.
func GetRawEvents(texts []pdf.Text, initialDate time.Time) []RawEvent {
	rawEvents := make([]RawEvent, 0)
	var rawEvent RawEvent
	for i, text := range texts {
		if rawEvent.data == "" {
			rawEvent.position = pdf.Point{X: text.X, Y: text.Y}
		} else if text.Y != texts[i-1].Y {
			rawEvent.data += " "
		}
		rawEvent.data += text.S
		if text.S == "]" {
			rawEvent.initialDate = initialDate
			rawEvents = append(rawEvents, rawEvent)
			rawEvent.data = ""
		}
	}
	return rawEvents
}

// parseEvent parses *RawEvent and returns *Event.
func parseEvent(raw *RawEvent) (*Event, error) {
	// Parse type from data.
	const (
		lecture = "лекции"
		seminar = "семинар"
		lab     = "лабораторные занятия"
	)
	typeRegexp := regexp.MustCompile(fmt.Sprintf(`(%s|%s|%s)\.`, lecture, seminar, lab))
	typeIndexes := typeRegexp.FindStringIndex(raw.data)
	if typeIndexes == nil {
		return nil, errors.New("schedule event type is not found")
	}
	eventTypes := map[string]string{
		lecture: "lecture",
		seminar: "seminar",
		lab:     "lab",
	}
	eventType := eventTypes[raw.data[typeIndexes[0]:typeIndexes[1]-1]]

	// Parse title and teacher from data.
	var eventTitle, eventTeacher string

	stringsBeforeType := strings.Split(raw.data[:typeIndexes[0]-1], ". ")
	if len(stringsBeforeType) == 1 {
		eventTitle = stringsBeforeType[0][:len(stringsBeforeType)-1]
	} else {
		eventTitle = stringsBeforeType[0]
		eventTeacher = stringsBeforeType[1]
	}

	// Parse dates from data and position.
	var (
		eventDates      []EventDate
		datesStartIndex int
		err             error
	)
	if eventType == "lab" {
		eventDates, datesStartIndex, err = parseDates(raw, 1)
	} else {
		eventDates, datesStartIndex, err = parseDates(raw, 0)
	}
	if err != nil {
		return nil, fmt.Errorf("parseDates error: %w", err)
	}

	// Parse subgroup and location from data.
	var eventSubgroup, eventLocation string

	stringsAfterType := strings.Split(raw.data[typeIndexes[1]+1:datesStartIndex-2], ". ")
	if len(stringsAfterType) == 2 {
		eventSubgroup = strings.Trim(stringsAfterType[0], "()")
		eventLocation = stringsAfterType[1]
	} else {
		eventLocation = stringsAfterType[0]
	}

	return &Event{
		eventTitle,
		eventTeacher,
		eventType,
		eventSubgroup,
		eventLocation,
		eventDates,
	}, nil
}

// parseEvents takes slice of RawEvent, forms slice of Event and returns it.
func ParseEvents(rawEvents []RawEvent) ([]Event, error) {
	events := make([]Event, 0)
	for i, rawEvent := range rawEvents {
		log.Printf("<--- %v --->\n", rawEvent)
		event, err := parseEvent(&rawEvent)
		if err != nil {
			return nil, fmt.Errorf("parse events[%d]: %w", i, err)
		}
		log.Printf("<--- %v --->\n\n", *event)
		events = append(events, *event)
	}
	return events, nil
}
